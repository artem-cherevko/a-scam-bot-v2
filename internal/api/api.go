package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/artem-cherevko/a-scam-bot-v2/internal/config"
	"github.com/artem-cherevko/a-scam-bot-v2/internal/database"
	"github.com/artem-cherevko/a-scam-bot-v2/internal/repository"
	"github.com/artem-cherevko/a-scam-bot-v2/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

func RunApi(cfg *config.Config, db *gorm.DB) error {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	userRepo := repository.NewUserRepository(db)
	guarantorRepo := repository.NewGuarantorRepository(db)

	userService := service.NewUserService(userRepo, guarantorRepo)

	r.SetTrustedProxies([]string{"127.0.0.1"})

	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		api.POST("/register", func(c *gin.Context) {
			var userInput struct {
				TelegramID int64  `json:"telegram_id" binding:"required"`
				Username   string `json:"username" binding:"required"`
			}

			if err := c.ShouldBindJSON(&userInput); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			user := &database.User{
				TgID:     userInput.TelegramID,
				UserName: &userInput.Username,
				Role:     "user",
			}

			_, err := userService.GetUserByTelegramID(user.TgID)
			if err == nil {
				c.JSON(200, gin.H{"status": "User already exists"})
				return
			}

			err = userService.AddUser(user)
			log.Printf("Attempting to add user: %v", err)
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				c.JSON(400, gin.H{"error": "User already exists"})
				return
			} else if err != nil {
				c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to create user: %v", err)})
				return
			}

			c.JSON(201, gin.H{"message": "User registered successfully"})
		})
	}
	protected := api.Group("")
	protected.Use(Auth(userService))
	{
		protected.GET("/user/:telegram", func(c *gin.Context) {
			var (
				user *database.User
				err  error
			)
			value := c.Param("telegram")

			isCheck, err := strconv.ParseBool(c.DefaultQuery("isCheck", "false"))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid isCheck parameter",
				})
				return
			}

			// Если передан Telegram ID
			if id, parseErr := strconv.ParseInt(value, 10, 64); parseErr == nil {
				user, err = userService.GetUserByTelegramID(id)
			} else {
				// Если передан username
				username := strings.TrimPrefix(value, "@")
				user, err = userService.GetUserByName(username)
			}

			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "User not found",
				})
				return
			}

			if isCheck {
				if err := userService.UpdateUser(&database.User{ID: user.ID, UserSearched: user.UserSearched + 1}); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "Failed to update user search count",
					})
					return
				}
			}

			c.JSON(http.StatusOK, gin.H{
				"id":             user.ID,
				"tg_id":          user.TgID,
				"user_name":      user.UserName,
				"role":           user.Role,
				"warns":          user.Warns,
				"added_scammers": user.AddedScammers,
				"photo_id":       user.PhotoID,
				"user_searched":  user.UserSearched,
				"scammer_reason": user.ScammerReason,
			})
		})

		protected.GET("/me", func(c *gin.Context) {
			value, exists := c.Get("user")
			if !exists {
				return
			}

			user, ok := value.(*database.User)
			if !ok {
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"id":             user.ID,
				"tg_id":          user.TgID,
				"user_name":      user.UserName,
				"role":           user.Role,
				"warns":          user.Warns,
				"added_scammers": user.AddedScammers,
				"photo_id":       user.PhotoID,
				"user_searched":  user.UserSearched,
			})
		})

		protected.GET("/guarantors", func(c *gin.Context) {
			var guarantors []database.Guarantors
			guarantors, err := userService.GetAllGuarantors()
			if err != nil {
				log.Printf("Can't get all guarantors: %v", err)
				c.JSON(400, gin.H{
					"error": "can't get guarantors",
				})
				return
			}
			ids := make([]int64, len(guarantors))
			for i, g := range guarantors {
				ids[i] = g.TgUserID
			}

			users, err := userRepo.GetAllUserNamesByIDs(ids)
			if err != nil {
				log.Println(err)
				c.JSON(400, gin.H{
					"error": "can't get user names",
				})
				return
			}

			c.JSON(200, gin.H{"guarantors": guarantors, "users": users})
		})

		protected.POST("/guarantor/trainee/:telegram", func(c *gin.Context) {
			value, exists := c.Get("user")
			if !exists {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "actor not found in context",
				})
				return
			}

			actor, ok := value.(*database.User)
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "invalid actor type",
				})
				return
			}

			// Добавлять trainee может только гарант или топ гарант.
			if actor.Role != database.Guarantor &&
				actor.Role != database.TopGuarantor {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "only guarantors can manage trainees",
				})
				return
			}

			targetID, err := strconv.ParseInt(c.Param("telegram"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid telegram id",
				})
				return
			}

			// Нельзя добавить самого себя.
			if actor.TgID == targetID {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "you cannot add yourself as trainee",
				})
				return
			}

			// Проверяем target.
			target, err := userService.GetUserByTelegramID(targetID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					c.JSON(http.StatusNotFound, gin.H{
						"error": "trainee not found",
					})
					return
				}

				log.Printf("Can't get trainee %d: %v", targetID, err)

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "can't get trainee",
				})
				return
			}

			target.Role = database.Trainee

			if err := userService.UpdateUser(target); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "can't assign trainee role",
				})
				return
			}

			// Получаем самого гаранта.
			guarantor, err := guarantorRepo.GetGuarantorByID(actor.TgID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					c.JSON(http.StatusNotFound, gin.H{
						"error": "guarantor profile not found",
					})
					return
				}

				log.Printf("Can't get guarantor %d: %v", actor.TgID, err)

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "can't get guarantor",
				})
				return
			}

			// Проверяем дубликат.
			if guarantor.TraineeIDs != nil {
				for _, id := range *guarantor.TraineeIDs {
					if id == targetID {
						c.JSON(http.StatusConflict, gin.H{
							"error": "trainee already added",
						})
						return
					}
				}
			}

			// Добавляем trainee.
			var traineeIDs []int64

			if guarantor.TraineeIDs != nil {
				traineeIDs = append(
					traineeIDs,
					(*guarantor.TraineeIDs)...,
				)
			}

			traineeIDs = append(traineeIDs, targetID)

			ids := pq.Int64Array(traineeIDs)
			guarantor.TraineeIDs = &ids

			if err := userService.UpdateGuarantor(guarantor); err != nil {
				log.Printf(
					"Can't add trainee %d to guarantor %d: %v",
					targetID,
					actor.TgID,
					err,
				)

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "can't add trainee",
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"status":       "added",
				"trainee_id":   targetID,
				"guarantor_id": actor.TgID,
			})
		})

		protected.DELETE("/guarantor/trainee/:telegram", func(c *gin.Context) {
			value, exists := c.Get("user")
			if !exists {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "actor not found in context",
				})
				return
			}

			actor, ok := value.(*database.User)
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "invalid actor type",
				})
				return
			}

			// Удалять trainee может только гарант или топ гарант.
			if actor.Role != database.Guarantor &&
				actor.Role != database.TopGuarantor {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "only guarantors can manage trainees",
				})
				return
			}

			targetID, err := strconv.ParseInt(c.Param("telegram"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid telegram id",
				})
				return
			}

			target, err := userService.GetUserByTelegramID(targetID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					c.JSON(http.StatusNotFound, gin.H{
						"error": "target user not found",
					})
					return
				}

				log.Printf("Can't get target user %d: %v", targetID, err)

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "can't get target user",
				})
				return
			}

			guarantor, err := guarantorRepo.GetGuarantorByID(actor.TgID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					c.JSON(http.StatusNotFound, gin.H{
						"error": "guarantor profile not found",
					})
					return
				}

				log.Printf("Can't get guarantor %d: %v", actor.TgID, err)

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "can't get guarantor",
				})
				return
			}

			if guarantor.TraineeIDs == nil || len(*guarantor.TraineeIDs) == 0 {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "trainee not found in your list",
				})
				return
			}

			// Удаляем targetID.
			traineeIDs := make([]int64, 0, len(*guarantor.TraineeIDs))
			found := false

			for _, id := range *guarantor.TraineeIDs {
				if id == targetID {
					found = true
					continue
				}

				traineeIDs = append(traineeIDs, id)
			}

			if !found {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "trainee not found in your list",
				})
				return
			}

			if len(traineeIDs) == 0 {
				guarantor.TraineeIDs = nil
			} else {
				ids := pq.Int64Array(traineeIDs)
				guarantor.TraineeIDs = &ids
			}

			if err := userService.UpdateGuarantor(guarantor); err != nil {
				log.Printf(
					"Can't remove trainee %d from guarantor %d: %v",
					targetID,
					actor.TgID,
					err,
				)

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "can't remove trainee",
				})
				return
			}

			target.Role = database.RegularUser

			if err := userService.UpdateUser(target); err != nil {
				log.Printf(
					"Can't reset role for trainee %d: %v",
					targetID,
					err,
				)

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "trainee removed, but role reset failed",
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"status":       "removed",
				"trainee_id":   targetID,
				"guarantor_id": actor.TgID,
			})
		})

		adminGroup := protected.Group("/admin")
		adminGroup.Use(RequireAdmin(userService))
		{
			adminGroup.PUT("/guarantor/reset/:telegram", func(c *gin.Context) {
				targetID, err := strconv.ParseInt(c.Param("telegram"), 10, 64)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "invalid telegram id",
					})
					return
				}

				guarantor, err := guarantorRepo.GetGuarantorByID(targetID)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						c.JSON(http.StatusNotFound, gin.H{
							"error": "guarantor not found",
						})
						return
					}

					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "can't get guarantor",
					})
					return
				}

				if err := userService.ResetGuarantor(guarantor); err != nil {
					log.Printf(
						"Can't reset guarantor %d: %v",
						targetID,
						err,
					)

					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "can't reset guarantor",
					})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"status": "reset",
				})
			})
			adminGroup.PUT("/guarantor/edit/:telegram", func(c *gin.Context) {
				targetID, err := strconv.ParseInt(c.Param("telegram"), 10, 64)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "invalid telegram id",
					})
					return
				}

				var input struct {
					Field string `json:"field"`
					Value string `json:"value"`
				}

				if err := c.ShouldBindJSON(&input); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "please provide valid body",
					})
					return
				}

				input.Field = strings.TrimSpace(strings.ToLower(input.Field))
				input.Value = strings.TrimSpace(input.Value)

				if input.Value == "" {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "value cannot be empty",
					})
					return
				}

				switch input.Field {
				case "rank":
					if input.Value != "basic" && input.Value != "top" {
						c.JSON(http.StatusBadRequest, gin.H{
							"error": "rank must be basic or top",
						})
						return
					}

				case "proofs":
					proofs, err := strconv.ParseUint(input.Value, 10, 64)
					if err != nil {
						c.JSON(http.StatusBadRequest, gin.H{
							"error": "proofs must be a non-negative integer",
						})
						return
					}

					if proofs > uint64(^uint16(0)) {
						c.JSON(http.StatusBadRequest, gin.H{
							"error": "proofs value is too large",
						})
						return
					}

				case "channel":
					if !strings.HasPrefix(input.Value, "@") &&
						!strings.HasPrefix(input.Value, "https://t.me/") &&
						!strings.HasPrefix(input.Value, "http://t.me/") &&
						!strings.HasPrefix(input.Value, "https://telegram.me/") &&
						!strings.HasPrefix(input.Value, "http://telegram.me/") {
						c.JSON(http.StatusBadRequest, gin.H{
							"error": "invalid telegram channel",
						})
						return
					}

				case "region":
					if len([]rune(input.Value)) > 64 {
						c.JSON(http.StatusBadRequest, gin.H{
							"error": "region is too long",
						})
						return
					}

				default:
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "unsupported field",
					})
					return
				}

				guarantor, err := guarantorRepo.GetGuarantorByID(targetID)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						c.JSON(http.StatusNotFound, gin.H{
							"error": "guarantor not found",
						})
						return
					}

					log.Printf("Can't get guarantor %d: %v", targetID, err)

					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "can't get guarantor",
					})
					return
				}

				switch input.Field {
				case "rank":
					guarantor.Rank = database.Ranks(input.Value)

				case "channel":
					value := input.Value
					guarantor.ChanelUrl = &value

				case "proofs":
					proofs, err := strconv.ParseUint(input.Value, 10, 64)
					if err != nil {
						c.JSON(http.StatusBadRequest, gin.H{
							"error": "proofs must be a non-negative integer",
						})
						return
					}

					if proofs > 65535 {
						c.JSON(http.StatusBadRequest, gin.H{
							"error": "proofs value is too large",
						})
						return
					}

					value := strconv.FormatUint(proofs, 10)
					guarantor.ProofsCount = &value
				case "region":
					value := input.Value
					guarantor.Region = &value
				}

				if err := userService.UpdateGuarantor(guarantor); err != nil {
					log.Printf(
						"Can't update guarantor %d field %s: %v",
						targetID,
						input.Field,
						err,
					)

					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "can't update guarantor",
					})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"status": "updated",
					"field":  input.Field,
					"value":  input.Value,
				})
			})
			adminGroup.PUT("/user/complete", func(c *gin.Context) {
				var input struct {
					UserID   string `json:"user_id"`
					UserName string `json:"user_name"`
				}

				if err := c.ShouldBindJSON(&input); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "please provide valid body",
					})
					return
				}

				input.UserName = strings.TrimPrefix(
					strings.TrimSpace(input.UserName),
					"@",
				)

				if input.UserName == "" {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "username is required",
					})
					return
				}

				id, err := strconv.ParseInt(input.UserID, 10, 64)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "not valid user id",
					})
					return
				}

				user, err := userService.GetUserByTelegramID(id)
				if err != nil {
					if errors.Is(err, service.ErrUserNotFound) ||
						errors.Is(err, gorm.ErrRecordNotFound) {
						c.JSON(http.StatusNotFound, gin.H{
							"error": "user not found",
						})
						return
					}

					log.Printf("Can't get user: %v", err)

					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "can't get user",
					})
					return
				}

				// Уже заполнено — ничего не перезаписываем.
				if user.UserName != nil && *user.UserName != "" {
					c.JSON(http.StatusConflict, gin.H{
						"error":     "username already exists",
						"user_name": *user.UserName,
					})
					return
				}

				if err := userService.CompleteUser(user, input.UserName); err != nil {
					log.Printf("Can't complete user: %v", err)

					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "can't complete user",
					})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"status":    "completed",
					"user_id":   user.TgID,
					"user_name": user.UserName,
				})
			})
			adminGroup.PUT("/user/warn/:telegram", func(c *gin.Context) {
				value, exists := c.Get("user")
				if !exists {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "actor not found in context",
					})
					return
				}

				actor, ok := value.(*database.User)
				if !ok {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "invalid actor type",
					})
					return
				}

				targetID, err := strconv.ParseInt(c.Param("telegram"), 10, 64)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "invalid telegram id",
					})
					return
				}

				target, err := userService.GetUserByTelegramID(targetID)
				if err != nil {
					c.JSON(http.StatusNotFound, gin.H{
						"error": "user not found",
					})
					return
				}

				punished, err := userService.WarnUser(actor, target)
				if err != nil {
					log.Printf("Can't warn user: %v", err)

					c.JSON(http.StatusBadRequest, gin.H{
						"error": err.Error(),
					})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"status":   "warned",
					"warns":    target.Warns,
					"role":     target.Role,
					"punished": punished,
				})
			})
			adminGroup.PUT("/user/unwarn/:telegram", func(c *gin.Context) {
				value, exists := c.Get("user")
				if !exists {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "actor not found in context",
					})
					return
				}

				actor, ok := value.(*database.User)
				if !ok {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "invalid actor type",
					})
					return
				}

				targetID, err := strconv.ParseInt(c.Param("telegram"), 10, 64)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "invalid telegram id",
					})
					return
				}

				target, err := userService.GetUserByTelegramID(targetID)
				if err != nil {
					c.JSON(http.StatusNotFound, gin.H{
						"error": "user not found",
					})
					return
				}

				if err := userService.UnwarnUser(actor, target); err != nil {
					log.Printf("Can't unwarn user: %v", err)

					c.JSON(http.StatusBadRequest, gin.H{
						"error": err.Error(),
					})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"status": "unwarned",
					"warns":  target.Warns,
				})
			})
			adminGroup.POST("/guarantor/add", func(c *gin.Context) {
				var input struct {
					UserID   *string         `json:"user_id"`
					UserName *string         `json:"user_name"`
					Rank     *database.Ranks `json:"rank"`
				}
				if err := c.ShouldBindJSON(&input); err != nil {
					c.JSON(400, gin.H{
						"error": "please provide valid body",
					})
					return
				}

				if input.UserID != nil {
					id, err := strconv.ParseInt(*input.UserID, 10, 64)
					if err != nil {
						c.JSON(400, gin.H{
							"error": "please provide valid id",
						})
						return
					}

					user, err := userService.GetUserByTelegramID(id)
					if err != nil || user == nil {
						c.JSON(404, gin.H{
							"error": "user not found",
						})
						return
					}

					if input.Rank == nil {
						c.JSON(400, gin.H{
							"error": "rank is required",
						})
						return
					}

					guarantor := database.Guarantors{
						TgUserID: user.TgID,
						Rank:     *input.Rank,
					}

					if err := userService.AddGuarantor(&guarantor); err != nil {
						log.Printf("Can't create guarantor: %v", err)
						c.JSON(500, gin.H{
							"error": "can't create guarantor",
						})
						return
					}

					c.JSON(201, gin.H{"status": "created"})
					return
				}

				if input.UserName == nil {
					c.JSON(400, gin.H{
						"error": "username is required",
					})
					return
				}

				user, err := userService.GetUserByName(*input.UserName)
				if err != nil || user == nil {
					c.JSON(404, gin.H{
						"error": "user not found",
					})
					return
				}

				if input.Rank == nil {
					c.JSON(400, gin.H{
						"error": "rank is required",
					})
					return
				}

				guarantor := database.Guarantors{
					TgUserID: user.TgID,
					Rank:     *input.Rank,
				}

				if err := userService.AddGuarantor(&guarantor); err != nil {
					log.Printf("Can't create guarantor: %v", err)
					c.JSON(500, gin.H{
						"error": "can't create guarantor",
					})
					return
				}

				c.JSON(201, gin.H{"status": "created"})
			})
			adminGroup.DELETE("/guarantors/remove/:telegram", func(c *gin.Context) {
				value := c.Param("telegram")
				var (
					user *database.User
					err  error
				)

				if id, parseErr := strconv.ParseInt(value, 10, 64); parseErr == nil {
					user, err = userService.GetUserByTelegramID(id)
					if err != nil {
						c.JSON(400, gin.H{"error": "provide valid tg user name or id"})
						return
					}
				} else {
					username := strings.TrimPrefix(value, "@")
					user, err = userService.GetUserByName(username)
					if err != nil {
						c.JSON(400, gin.H{"error": "provide valid tg user name or id"})
						return
					}
				}

				if err := userService.DeleteGuarantor(user.TgID); err != nil {
					log.Println(err)
					c.JSON(500, gin.H{"error": "can't delete guarantor"})
					return
				}

				c.JSON(200, gin.H{"status": "removed"})
			})
			adminGroup.PUT("/scammer", func(c *gin.Context) {
				value, exists := c.Get("user")
				if !exists {
					c.JSON(500, gin.H{"error": "adder not found in context"})
					return
				}

				adder, ok := value.(*database.User)
				if !ok {
					c.JSON(500, gin.H{"error": "invalid adder type"})
					return
				}

				var input struct {
					UserName *string        `json:"user_name"`
					UserID   string         `json:"user_id"`
					Role     database.Roles `json:"role"`
					Reason   string         `json:"reason"`
				}

				if err := c.ShouldBindJSON(&input); err != nil {
					c.JSON(400, gin.H{
						"error": "please provide valid body",
					})
					return
				}

				switch input.Role {
				case database.Owner,
					database.CoOwner,
					database.SeniorAdmin,
					database.Admin,
					database.JuniorAdmin,
					database.Intern:

					c.JSON(400, gin.H{
						"error": "not able to add admin",
					})
					return
				}

				id, err := strconv.ParseInt(input.UserID, 10, 64)
				if err != nil {
					c.JSON(400, gin.H{
						"error": "not valid user id",
					})
					return
				}

				user, err := userService.GetUserByTelegramID(id)

				// ─────────────────────────
				// Пользователя ещё нет
				// ─────────────────────────

				if errors.Is(err, gorm.ErrRecordNotFound) {
					user = &database.User{
						TgID:          id,
						UserName:      input.UserName,
						Role:          input.Role,
						ScammerReason: &input.Reason,
					}

					if err := userService.AddUser(user); err != nil {
						log.Printf("Can't add user: %v", err)

						c.JSON(500, gin.H{
							"error": "can't add user",
						})
						return
					}

					if input.Role == database.Scammer ||
						input.Role == database.DodgyCharacter {

						if err := userService.IncrementAddedScammers(adder.TgID); err != nil {
							log.Printf(
								"Can't increment added scammers for %d: %v",
								adder.TgID,
								err,
							)

							c.JSON(500, gin.H{
								"error": "user added, but can't update adder statistics",
							})
							return
						}
					}

					c.JSON(201, gin.H{
						"status": "created",
					})
					return
				}

				// ─────────────────────────
				// Ошибка получения
				// ─────────────────────────

				if err != nil {
					log.Printf("Can't get user: %v", err)

					c.JSON(500, gin.H{
						"error": "can't get user",
					})
					return
				}

				// ─────────────────────────
				// Пользователь уже существует
				// ─────────────────────────

				oldRole := user.Role

				if input.UserName != nil {
					user.UserName = input.UserName
				}

				user.Role = input.Role

				if input.Reason != "" {
					user.ScammerReason = &input.Reason
				}

				if err := userService.UpdateUser(user); err != nil {
					log.Printf("Can't update user: %v", err)

					c.JSON(500, gin.H{
						"error": "can't update user",
					})
					return
				}

				// Считаем только переход в scammer/dodgy
				if (oldRole != database.Scammer &&
					oldRole != database.DodgyCharacter) &&
					(input.Role == database.Scammer ||
						input.Role == database.DodgyCharacter) {

					if err := userService.IncrementAddedScammers(adder.TgID); err != nil {
						log.Printf(
							"Can't increment added scammers for %d: %v",
							adder.TgID,
							err,
						)

						c.JSON(500, gin.H{
							"error": "user updated, but can't update adder statistics",
						})
						return
					}
				}

				c.JSON(200, gin.H{
					"status": "updated",
				})
			})
			adminGroup.PUT("/user/role/:telegram", func(c *gin.Context) {
				value := c.Param("telegram")
				var (
					user *database.User
					err  error
				)

				// Если передан Telegram ID
				if id, parseErr := strconv.ParseInt(value, 10, 64); parseErr == nil {
					user, err = userService.GetUserByTelegramID(id)
				} else {
					// Если передан username
					username := strings.TrimPrefix(value, "@")
					user, err = userService.GetUserByName(username)
				}

				if err != nil {
					c.JSON(http.StatusNotFound, gin.H{
						"error": "User not found",
					})
					return
				}

				var userInput struct {
					Role database.Roles `json:"role"`
				}
				if err := c.ShouldBindJSON(&userInput); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				if err := userService.UpdateUser(&database.User{ID: user.ID, Role: userInput.Role}); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
					return
				}

				c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
			})
		}
	}

	r.Run(fmt.Sprintf(":%s", cfg.API_PORT))
	return nil
}
