package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/artem-cherevko/a-scam-bot-v2/internal/config"
	"github.com/artem-cherevko/a-scam-bot-v2/internal/database"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var ErrUserNotFound = errors.New("user not found")

type UserResponse struct {
	AddedScammers uint    `json:"added_scammers"`
	ID            int64   `json:"id"`
	PhotoID       *string `json:"photo_id"`
	Role          string  `json:"role"`
	TgID          int64   `json:"tg_id"`
	UserName      *string `json:"user_name"`
	UserSearched  uint    `json:"user_searched"`
	Warns         uint16  `json:"warns"`
	ScammerReason *string `json:"scammer_reason"`
}

func StartBot(cfg *config.Config, ctx context.Context) error {
	var app *BotApp

	opts := []bot.Option{
		bot.WithMiddlewares(showMessageWithUserID),
		bot.WithDefaultHandler(func(
			ctx context.Context,
			b *bot.Bot,
			update *models.Update,
		) {
			if app != nil {
				app.HandleGuarantorFSMMessage(ctx, b, update)
			}
		}),
	}

	b, err := bot.New(cfg.BOT_TOKEN, opts...)
	if err != nil {
		return err
	}

	app = NewBotApp(b, cfg.API_ENDPOINT)
	app.RegisterGuarantorFSM()

	_, err = b.DeleteWebhook(ctx, &bot.DeleteWebhookParams{
		DropPendingUpdates: true,
	})
	if err != nil {
		return err
	}

	// /start
	b.RegisterHandler(
		bot.HandlerTypeMessageText,
		"/start",
		bot.MatchTypePrefix,
		func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if update.Message == nil {
				return
			}

			payload := struct {
				TelegramID int64  `json:"telegram_id"`
				Username   string `json:"username"`
			}{
				TelegramID: update.Message.From.ID,
				Username:   update.Message.From.Username,
			}

			body, err := json.Marshal(payload)
			if err != nil {
				log.Printf("failed to marshal register payload: %v", err)
				return
			}

			resp, err := apiRequest(
				ctx,
				http.MethodPost,
				cfg.API_ENDPOINT+"/api/register",
				update.Message.From.ID,
				body,
			)
			if err != nil {
				log.Printf("register request failed: %v", err)
				return
			}

			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
				log.Printf("api returned status: %s", resp.Status)
				return
			}

			text := `
♧ <b>Добро пожаловать в</b> <a href="https://t.me/asbas3">A-Scam</a>

Привет, ты попал в бота от проекта
As-Scam

⚠️ Если вас обманули, вы можете
слить скамера в <a href="https://t.me/asbas3">предложку</a>

ⓘ У нас есть чат для <a href="https://t.me/c/asbas3/1">Поиска
гарантов</a>. Там всегда найдёте
гаранта.

🔎 Проверить на скам: <code>/check</code>
<code>@Тег_Человека</code>
Проверить себя: <code>/me</code>
`

			_, err = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    update.Message.Chat.ID,
				Text:      text,
				ParseMode: "HTML",
			})
			if err != nil {
				log.Printf("failed to send message: %v", err)
			}
		},
	)

	// /check
	b.RegisterHandler(
		bot.HandlerTypeMessageText,
		"/check",
		bot.MatchTypePrefix,
		func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if update.Message == nil {
				return
			}

			identifier := strings.TrimSpace(
				strings.TrimPrefix(update.Message.Text, "/check"),
			)

			if identifier == "" {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:    update.Message.Chat.ID,
					Text:      "Please provide a Telegram ID or username after the /check command.",
					ParseMode: "HTML",
				})
				return
			}

			user, err := getUser(
				ctx,
				cfg.API_ENDPOINT,
				identifier,
				update.Message.From.ID,
				true,
			)

			// Пользователь отсутствует в БД.
			// Всё равно показываем карточку, но роль = "Нет в базе".
			if errors.Is(err, ErrUserNotFound) {
				text := formatUnknownUser(identifier)

				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   text,
				})
				return
			}

			if err != nil {
				log.Printf("failed to get user: %v", err)

				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Произошла ошибка при проверке пользователя.",
				})
				return
			}

			text := formatUser(user)

			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   text,
			})
		},
	)

	b.RegisterHandler(bot.HandlerTypeMessageText, "/me", bot.MatchTypeExact, func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil {
			return
		}

		resp, err := apiRequest(
			ctx,
			http.MethodGet,
			cfg.API_ENDPOINT+"/api/me",
			update.Message.From.ID,
			nil,
		)
		if err != nil {
			log.Printf("failed to get user info: %v", err)
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Произошла ошибка при получении информации о пользователе.",
			})
			return
		}

		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Вы отсутствуете в базе данных. Пожалуйста, зарегистрируйтесь с помощью команды /start.",
			})
			return
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf(
				"api returned status: %s",
				resp.Status,
			)
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Произошла ошибка при получении информации о пользователе.",
			})
			return
		}

		var user UserResponse

		if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Произошла ошибка при получении информации о пользователе.",
			})
			return
		}

		text := formatUser(&user)

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   text,
		})
	})

	// Обновления роли пользователя (ТОЛЬКО АДМИНЫ)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/set_role", bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *models.Update) {
		me, err := getUser(ctx, cfg.API_ENDPOINT, strconv.FormatInt(update.Message.From.ID, 10), update.Message.From.ID, false)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.From.ID,
				Text:   "Вас нету в базе данных. Напишите /start для регистрации",
			})
			return
		}

		if !canSetRole(me.Role) {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.From.ID,
				Text:   "Вам не доступна эта команда.",
			})
			return
		}

		identifier := strings.TrimSpace(
			strings.TrimPrefix(update.Message.Text, "/set_role"),
		)

		if identifier == "" {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    update.Message.Chat.ID,
				Text:      "Please provide a Telegram ID or username after the /set_role command.",
				ParseMode: "HTML",
			})
			return
		}

		user, err := getUser(ctx, cfg.API_ENDPOINT, identifier, update.Message.From.ID, false)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.From.ID,
				Text:   "Пользователь не обнаружен для обновления роли.",
			})
			return
		}

		kb := SetRoleKb(user.TgID, me.Role)

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      update.Message.Chat.ID,
			Text:        fmt.Sprintf("Выберите роль для пользователя: %s", *user.UserName),
			ReplyMarkup: kb,
		})
	})

	// /add_trainee <telegram_id>
	b.RegisterHandler(
		bot.HandlerTypeMessageText,
		"/add_trainee",
		bot.MatchTypePrefix,
		func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if update.Message == nil {
				return
			}

			me, err := getUser(
				ctx,
				cfg.API_ENDPOINT,
				strconv.FormatInt(update.Message.From.ID, 10),
				update.Message.From.ID,
				false,
			)
			if err != nil {
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Вас нет в базе данных. Напишите /start.",
				})
				return
			}

			// Гарант и топ-гарант могут добавлять trainee.
			if me.Role != string(database.Guarantor) &&
				me.Role != string(database.TopGuarantor) {
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Эта команда доступна только гарантами.",
				})
				return
			}

			args := strings.TrimSpace(
				strings.TrimPrefix(update.Message.Text, "/add_trainee"),
			)

			targetID, err := strconv.ParseInt(args, 10, 64)
			if err != nil {
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Формат: /add_trainee <telegram_id>",
				})
				return
			}

			resp, err := apiRequest(
				ctx,
				http.MethodPost,
				fmt.Sprintf(
					cfg.API_ENDPOINT+"/api/guarantor/trainee/%d",
					targetID,
				),
				update.Message.From.ID,
				nil,
			)
			if err != nil {
				log.Printf("add trainee request failed: %v", err)

				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Ошибка при обращении к API.",
				})
				return
			}

			defer resp.Body.Close()

			switch resp.StatusCode {
			case http.StatusOK:
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text: fmt.Sprintf(
						"✅ Trainee <code>%d</code> добавлен.",
						targetID,
					),
					ParseMode: models.ParseModeHTML,
				})

			case http.StatusConflict:
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Этот trainee уже добавлен.",
				})

			case http.StatusBadRequest:
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Пользователь не является trainee.",
				})

			case http.StatusNotFound:
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Trainee не найден.",
				})

			default:
				log.Printf("add trainee API returned %s", resp.Status)

				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Не удалось добавить trainee.",
				})
			}
		},
	)

	// /add_g @us|id top|basic
	b.RegisterHandler(bot.HandlerTypeMessageText, "/add_g", bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *models.Update) {
		me, err := getUser(
			ctx,
			cfg.API_ENDPOINT,
			strconv.FormatInt(update.Message.From.ID, 10),
			update.Message.From.ID,
			false,
		)

		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.From.ID,
				Text:   "Вас нету в базе данных. Напишите /start для регистрации",
			})
			return
		}

		if !canSetRole(me.Role) {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.From.ID,
				Text:   "Вам не доступна эта команда.",
			})
			return
		}
		text := strings.TrimSpace(update.Message.Text)
		args := strings.TrimSpace(strings.TrimPrefix(text, "/add_g"))
		parts := strings.Fields(args)

		if len(parts) < 2 {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.From.ID,
				Text:   "Не верный формат. Пример: /add_g @user|id top|basic",
			})
			return
		}

		identifier := parts[0]
		rank := parts[1]

		var body struct {
			UserID   string  `json:"user_id"`
			UserName *string `json:"user_name,omitempty"`
			Rank     string  `json:"rank"`
		}

		if _, err := strconv.ParseInt(identifier, 10, 64); err == nil {
			body.UserID = identifier
		} else {
			body.UserID = "0"
			username := strings.TrimPrefix(identifier, "@")
			body.UserName = &username
		}
		body.Rank = rank

		marshaled, err := json.Marshal(body)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.From.ID,
				Text:   "Не удалось создать тело запроса.",
			})
			return
		}

		resp, err := apiRequest(ctx, "POST", cfg.API_ENDPOINT+"/api/admin/guarantor/add", update.Message.From.ID, marshaled)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.From.ID,
				Text:   "Ошибка при обращении к API.",
			})
			return
		}

		if resp.StatusCode != http.StatusCreated {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.From.ID,
				Text:   "Не удалось создать гаранта.",
			})
			return
		} else {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.From.ID,
				Text:   "Гарант успешно создан!",
			})
			return
		}
	})

	// /add @us|id 0-1 (0 = скамер, 1 = сом. персона) причина
	b.RegisterHandler(
		bot.HandlerTypeMessageText,
		"/add",
		bot.MatchTypePrefix,
		func(ctx context.Context, b *bot.Bot, update *models.Update) {
			me, err := getUser(
				ctx,
				cfg.API_ENDPOINT,
				strconv.FormatInt(update.Message.From.ID, 10),
				update.Message.From.ID,
				false,
			)

			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.From.ID,
					Text:   "Вас нету в базе данных. Напишите /start для регистрации",
				})
				return
			}

			if !isAdminRole(me.Role) {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.From.ID,
					Text:   "Вам не доступна эта команда.",
				})
				return
			}

			text := strings.TrimSpace(update.Message.Text)
			args := strings.TrimSpace(strings.TrimPrefix(text, "/add"))
			parts := strings.Fields(args)

			if len(parts) < 3 {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.From.ID,
					Text: "Не верный формат.\n\n" +
						"Пример:\n" +
						"/add @us|id 0-1 причина\n\n" +
						"0 — скамер\n" +
						"1 — сомнительный персонаж",
				})
				return
			}

			identifier := parts[0]
			role := parts[1]
			reason := strings.Join(parts[2:], " ")

			var body struct {
				UserID   string         `json:"user_id"`
				UserName *string        `json:"user_name,omitempty"`
				Role     database.Roles `json:"role"`
				Reason   string         `json:"reason"`
			}

			// Определяем роль.
			switch role {
			case "0":
				body.Role = database.Scammer

			case "1":
				body.Role = database.DodgyCharacter

			default:
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.From.ID,
					Text:   "Данная роль отсутствует для выбора.",
				})
				return
			}

			body.Reason = reason

			// Определяем, что передали:
			// Telegram ID или username.
			if id, err := strconv.ParseInt(identifier, 10, 64); err == nil {
				body.UserID = strconv.FormatInt(id, 10)
			} else {
				body.UserID = "0"

				username := strings.TrimPrefix(identifier, "@")

				if username == "" {
					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: update.Message.From.ID,
						Text:   "Некорректный username.",
					})
					return
				}

				body.UserName = &username
			}

			// Превращаем body в JSON.
			marshaled, err := json.Marshal(body)
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.From.ID,
					Text:   "Произошла ошибка при подготовке тела запроса.",
				})
				return
			}

			// Один endpoint.
			// API само решает:
			// пользователь существует → UPDATE
			// пользователя нет → CREATE
			_, err = apiRequest(
				ctx,
				"PUT",
				cfg.API_ENDPOINT+"/api/admin/scammer",
				update.Message.From.ID,
				marshaled,
			)

			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.From.ID,
					Text:   "Произошла ошибка при добавлении пользователя.",
				})
				return
			}

			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Успешно!",
			})
		},
	)
	// /complete id @username
	b.RegisterHandler(
		bot.HandlerTypeMessageText,
		"/complete",
		bot.MatchTypePrefix,
		func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if update.Message == nil {
				return
			}

			// Проверяем администратора.
			me, err := getUser(
				ctx,
				cfg.API_ENDPOINT,
				strconv.FormatInt(update.Message.From.ID, 10),
				update.Message.From.ID,
				false,
			)

			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Вас нету в базе данных. Напишите /start для регистрации",
				})
				return
			}

			if !isAdminRole(me.Role) {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Вам не доступна эта команда.",
				})
				return
			}

			text := strings.TrimSpace(update.Message.Text)
			args := strings.TrimSpace(strings.TrimPrefix(text, "/complete"))
			parts := strings.Fields(args)

			if len(parts) < 2 {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text: "Неверный формат.\n\n" +
						"Пример:\n" +
						"/complete 1381931882 @username",
				})
				return
			}

			identifier := parts[0]
			username := strings.TrimPrefix(parts[1], "@")

			// Сейчас /complete работает именно с Telegram ID.
			id, err := strconv.ParseInt(identifier, 10, 64)
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Первым параметром должен быть Telegram ID.",
				})
				return
			}

			body := struct {
				UserID   string `json:"user_id"`
				UserName string `json:"user_name"`
			}{
				UserID:   strconv.FormatInt(id, 10),
				UserName: username,
			}

			marshaled, err := json.Marshal(body)
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Произошла ошибка при подготовке запроса.",
				})
				return
			}

			resp, err := apiRequest(
				ctx,
				http.MethodPut,
				cfg.API_ENDPOINT+"/api/admin/user/complete",
				update.Message.From.ID,
				marshaled,
			)

			if err != nil {
				log.Printf("complete user request failed: %v", err)

				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Произошла ошибка при обращении к API.",
				})
				return
			}

			defer resp.Body.Close()

			switch resp.StatusCode {
			case http.StatusOK:
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text: fmt.Sprintf(
						"✅ Пользователь дополнен.\n\n"+
							"🪪 ID: %d\n"+
							"👤 Username: @%s",
						id,
						username,
					),
				})

			case http.StatusNotFound:
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Пользователь не найден в базе данных.",
				})

			case http.StatusConflict:
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "У этого пользователя username уже заполнен.",
				})

			case http.StatusBadRequest:
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Некорректные данные.",
				})

			default:
				log.Printf("complete API returned status: %s", resp.Status)

				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Произошла ошибка при дополнении пользователя.",
				})
			}
		},
	)
	// /warn @username|id
	b.RegisterHandler(
		bot.HandlerTypeMessageText,
		"/warn",
		bot.MatchTypePrefix,
		func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if update.Message == nil {
				return
			}

			me, err := getUser(
				ctx,
				cfg.API_ENDPOINT,
				strconv.FormatInt(update.Message.From.ID, 10),
				update.Message.From.ID,
				false,
			)

			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Вас нету в базе данных. Напишите /start для регистрации",
				})
				return
			}

			if !canSetRole(me.Role) {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Вам не доступна эта команда.",
				})
				return
			}

			identifier := strings.TrimSpace(
				strings.TrimPrefix(update.Message.Text, "/warn"),
			)

			if identifier == "" {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Пример: /warn 1381931882",
				})
				return
			}

user, err := getUser(
			ctx,
			cfg.API_ENDPOINT,
			identifier,
			update.Message.From.ID,
			false,
		)

		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Пользователь не найден.",
			})
			return
		}

		resp, err := apiRequest(
			ctx,
			http.MethodPut,
			cfg.API_ENDPOINT+"/api/admin/user/warn/"+strconv.FormatInt(user.TgID, 10),
				update.Message.From.ID,
				nil,
			)

			if err != nil {
				log.Printf("warn request failed: %v", err)

				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Произошла ошибка при выдаче варна.",
				})
				return
			}

			defer resp.Body.Close()

			var response struct {
				Status   string         `json:"status"`
				Warns    uint16         `json:"warns"`
				Role     database.Roles `json:"role"`
				Punished bool           `json:"punished"`
				Error    string         `json:"error"`
			}

			_ = json.NewDecoder(resp.Body).Decode(&response)

			switch resp.StatusCode {
			case http.StatusOK:
				if response.Punished {
					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: update.Message.Chat.ID,
						Text: fmt.Sprintf(
							"🚫 Пользователь получил 3/3 варна.\n\n"+
								"👤 %s\n"+
								"🪪 ID: %d\n"+
								"⚙️ Роль снята.\n"+
								"🔄 Варны сброшены: 0/3",
							formatUserName(user),
							user.TgID,
						),
					})
					return
				}

				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text: fmt.Sprintf(
						"⚠️ Варн выдан.\n\n"+
							"👤 %s\n"+
							"🪪 ID: %d\n"+
							"⚠️ Варны: %d/3",
						formatUserName(user),
						user.TgID,
						response.Warns,
					),
				})

			case http.StatusForbidden:
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Вам нельзя выдать варн этому пользователю.",
				})

			default:
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Не удалось выдать варн.",
				})
			}
		},
	)
	// /unwarn @username|id
	b.RegisterHandler(
		bot.HandlerTypeMessageText,
		"/unwarn",
		bot.MatchTypePrefix,
		func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if update.Message == nil {
				return
			}

			me, err := getUser(
				ctx,
				cfg.API_ENDPOINT,
				strconv.FormatInt(update.Message.From.ID, 10),
				update.Message.From.ID,
				false,
			)

			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Вас нету в базе данных. Напишите /start для регистрации",
				})
				return
			}

			if !canSetRole(me.Role) {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Вам не доступна эта команда.",
				})
				return
			}

			identifier := strings.TrimSpace(
				strings.TrimPrefix(update.Message.Text, "/unwarn"),
			)

			if identifier == "" {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Пример: /unwarn 1381931882",
				})
				return
			}

user, err := getUser(
			ctx,
			cfg.API_ENDPOINT,
			identifier,
			update.Message.From.ID,
			false,
		)

		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Пользователь не найден.",
			})
			return
		}

		resp, err := apiRequest(
			ctx,
			http.MethodPut,
			cfg.API_ENDPOINT+"/api/admin/user/unwarn/"+strconv.FormatInt(user.TgID, 10),
				update.Message.From.ID,
				nil,
			)

			if err != nil {
				log.Printf("unwarn request failed: %v", err)

				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Произошла ошибка при снятии варна.",
				})
				return
			}

			defer resp.Body.Close()

			var response struct {
				Status string `json:"status"`
				Warns  uint16 `json:"warns"`
				Error  string `json:"error"`
			}

			_ = json.NewDecoder(resp.Body).Decode(&response)

			switch resp.StatusCode {
			case http.StatusOK:
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text: fmt.Sprintf(
						"✅ Варн снят.\n\n"+
							"👤 %s\n"+
							"🪪 ID: %d\n"+
							"⚠️ Варны: %d/3",
						formatUserName(user),
						user.TgID,
						response.Warns,
					),
				})

			case http.StatusForbidden:
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Вам нельзя снять варн у этого пользователя.",
				})

			default:
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Не удалось снять варн.",
				})
			}
		},
	)

	// /remove_trainee <telegram_id>
	b.RegisterHandler(
		bot.HandlerTypeMessageText,
		"/remove_trainee",
		bot.MatchTypePrefix,
		func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if update.Message == nil {
				return
			}

			me, err := getUser(
				ctx,
				cfg.API_ENDPOINT,
				strconv.FormatInt(update.Message.From.ID, 10),
				update.Message.From.ID,
				false,
			)
			if err != nil {
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Вас нет в базе данных. Напишите /start.",
				})
				return
			}

			if me.Role != string(database.Guarantor) &&
				me.Role != string(database.TopGuarantor) {
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Эта команда доступна только гарантами.",
				})
				return
			}

			args := strings.TrimSpace(
				strings.TrimPrefix(update.Message.Text, "/remove_trainee"),
			)

			targetID, err := strconv.ParseInt(args, 10, 64)
			if err != nil {
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Формат: /remove_trainee <telegram_id>",
				})
				return
			}

			resp, err := apiRequest(
				ctx,
				http.MethodDelete,
				fmt.Sprintf(
					cfg.API_ENDPOINT+"/api/guarantor/trainee/%d",
					targetID,
				),
				update.Message.From.ID,
				nil,
			)
			if err != nil {
				log.Printf("remove trainee request failed: %v", err)

				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Ошибка при обращении к API.",
				})
				return
			}

			defer resp.Body.Close()

			switch resp.StatusCode {
			case http.StatusOK:
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text: fmt.Sprintf(
						"✅ Trainee <code>%d</code> удалён.",
						targetID,
					),
					ParseMode: models.ParseModeHTML,
				})

			case http.StatusNotFound:
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Этого trainee нет в вашем списке.",
				})

			default:
				log.Printf("remove trainee API returned %s", resp.Status)

				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Не удалось удалить trainee.",
				})
			}
		},
	)

	// /remove_trainee <telegram_id>
	b.RegisterHandler(
		bot.HandlerTypeMessageText,
		"/remove_trainee",
		bot.MatchTypePrefix,
		func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if update.Message == nil {
				return
			}

			text := strings.TrimSpace(update.Message.Text)
			args := strings.TrimSpace(strings.TrimPrefix(text, "/remove_trainee"))
			parts := strings.Fields(args)

			if len(parts) != 1 {
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Формат: /remove_trainee <telegram_id>",
				})
				return
			}

			targetID, err := strconv.ParseInt(parts[0], 10, 64)
			if err != nil {
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Telegram ID должен быть числом.",
				})
				return
			}

			resp, err := apiRequest(
				ctx,
				http.MethodDelete,
				fmt.Sprintf(
					cfg.API_ENDPOINT+"/api/guarantor/trainee/%s",
					url.PathEscape(strconv.FormatInt(targetID, 10)),
				),
				update.Message.From.ID,
				nil,
			)
			if err != nil {
				log.Printf("remove trainee request error: %v", err)

				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Ошибка при удалении trainee.",
				})
				return
			}

			defer resp.Body.Close()

			switch resp.StatusCode {
			case http.StatusOK:
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text: fmt.Sprintf(
						"✅ Trainee <code>%d</code> удалён.",
						targetID,
					),
					ParseMode: models.ParseModeHTML,
				})

			case http.StatusForbidden:
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Только гарант может удалять trainee.",
				})

			case http.StatusNotFound:
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Этого trainee нет в твоём списке.",
				})

			default:
				log.Printf("remove trainee API status: %s", resp.Status)

				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Не удалось удалить trainee.",
				})
			}
		},
	)

	b.RegisterHandler(bot.HandlerTypeMessageText, "/guarantors", bot.MatchTypeExact, func(ctx context.Context, b *bot.Bot, update *models.Update) {
		resp, err := apiRequest(ctx, "GET", cfg.API_ENDPOINT+"/api/guarantors", update.Message.From.ID, nil)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.From.ID,
				Text:   "Произошла ошибка при отправке запроса.",
			})
			return
		}
		if resp.StatusCode != http.StatusOK {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.From.ID,
				Text:   "Произошла ошибка при обработке запроса сервером.",
			})
			return
		} else {
			var response struct {
				Users      []database.User       `json:"users"`
				Guarantors []database.Guarantors `json:"guarantors"`
			}

			err = json.NewDecoder(resp.Body).Decode(&response)
			if err != nil {
				log.Println(err.Error())
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.From.ID,
					Text:   "Произошла ошибка при обработке запроса клиентом.",
				})
				return
			} else {
				var text strings.Builder

				hasTopGuarantors := false
				hasBasicGuarantors := false

				usersByID := make(map[int64]*database.User, len(response.Users))

				for i := range response.Users {
					user := &response.Users[i]
					usersByID[user.TgID] = user
				}

				fmt.Fprintln(&text, "🛡️ Список гарантов:")
				fmt.Fprintln(&text)

				fmt.Fprintln(&text, "💎 Топ гаранты:")

				for _, g := range response.Guarantors {
					if g.Rank != database.Ranks("top") {
						continue
					}

					hasTopGuarantors = true

					user := usersByID[g.TgUserID]

					username := "unknown"
					if user != nil && user.UserName != nil {
						username = *user.UserName
					}

					fmt.Fprintln(
						&text,
						formatGuarantor(g, username),
					)
					fmt.Fprintln(&text)
				}

				if !hasTopGuarantors {
					fmt.Fprintln(&text, "Не обнаружено.")
					fmt.Fprintln(&text)
				}

				fmt.Fprintln(&text, "🛡️ Гаранты:")

				for _, g := range response.Guarantors {
					if g.Rank == database.Ranks("top") {
						continue
					}

					hasBasicGuarantors = true

					user := usersByID[g.TgUserID]

					username := "unknown"
					if user != nil && user.UserName != nil {
						username = *user.UserName
					}

					fmt.Fprintln(
						&text,
						formatGuarantor(g, username),
					)
					fmt.Fprintln(&text)
				}

				if !hasBasicGuarantors {
					fmt.Fprintln(&text, "Не обнаружено.")
				}

				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:    update.Message.Chat.ID,
					Text:      text.String(),
					ParseMode: models.ParseModeHTML,
				})
			}
		}
	})

	// /del_g @us|id
	b.RegisterHandler(bot.HandlerTypeMessageText, "/del_g", bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *models.Update) {
		text := strings.TrimSpace(update.Message.Text)
		args := strings.TrimSpace(strings.TrimPrefix(text, "/del_g"))
		parts := strings.Fields(args)

		if len(parts) < 1 {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.From.ID,
				Text:   "Не верный формат. Пример: /del_g @us|id ",
			})
			return
		}

		identifier := parts[0]

		resp, err := apiRequest(ctx, "DELETE", fmt.Sprintf(cfg.API_ENDPOINT+"/api/admin/guarantors/remove/%s", identifier), update.Message.From.ID, nil)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Произошла ошибка при удалении гаранта.",
			})
			return
		}
		if resp.StatusCode != http.StatusOK {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Произошла ошибка при удалении гаранта.",
			})
			return
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Гарант удален!",
		})
	})

	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "set_role:", bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.CallbackQuery == nil {
			return
		}

		callback := update.CallbackQuery

		parts := strings.Split(update.CallbackQuery.Data, ":")

		if len(parts) != 3 {
			return
		}

		targetTGID, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return
		}

		role := database.Roles(parts[2])

		actorTGID := update.CallbackQuery.From.ID

		payload := struct {
			Role string `json:"role"`
		}{
			Role: string(role),
		}

		body, err := json.Marshal(payload)
		if err != nil {
			return
		}
		resp, err := apiRequest(ctx, "PUT", cfg.API_ENDPOINT+"/api/admin/user/role/"+
			url.PathEscape(strconv.FormatInt(targetTGID, 10)), actorTGID, body)
		if err != nil {
			log.Printf("update user role err: %v", err)

			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: callback.Message.Message.Chat.ID,
				Text:   "Произошла ошибка при обновлении роли.",
			})

			return
		}

		if resp.StatusCode != http.StatusOK {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: callback.Message.Message.Chat.ID,
				Text:   "Произошла ошибка при обновлении роли.",
			})

			return
		} else {
			_, err = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: callback.ID,
			})
			if err != nil {
				log.Printf("failed to answer callback: %v", err)
			}

			_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    callback.Message.Message.Chat.ID,
				MessageID: callback.Message.Message.ID,
				Text: fmt.Sprintf(
					"✅ <b>Роль пользователя обновлена</b>\n\n"+
						"🪪 ID: <code>%d</code>\n"+
						"⚙️ Новая роль: <b>%s</b>",
					targetTGID,
					formatRole(string(role)),
				),
				ParseMode: "HTML",
			})

			if err != nil {
				log.Printf("failed to edit message: %v", err)
			}
		}
	})

	b.Start(ctx)

	return nil
}

func formatUserName(user *UserResponse) string {
	if user.UserName != nil && *user.UserName != "" {
		return "@" + *user.UserName
	}

	return strconv.FormatInt(user.TgID, 10)
}

// formatUser форматирует найденного пользователя.
func formatUser(user *UserResponse) string {
	username := "—"

	if user.UserName != nil && *user.UserName != "" {
		username = "@" + *user.UserName
	}

	text := fmt.Sprintf(
		"👤 Имя: %s\n"+
			"🪪 ID: [%d]\n\n"+
			"⚙️ Роль: %s\n",
		username,
		user.TgID,
		formatRole(user.Role),
	)

	// Показываем причину только для скамера
	// или подозрительного пользователя.
	if user.Role == string(database.Scammer) ||
		user.Role == string(database.DodgyCharacter) {

		if user.ScammerReason != nil && *user.ScammerReason != "" {
			text += fmt.Sprintf(
				"🚨 Причина: %s\n",
				*user.ScammerReason,
			)
		}
	}

	if isAdminRole(user.Role) {
		text += fmt.Sprintf(
			"\n⚠️ Предупреждений: %d\n"+
				"🚨 Добавлено скамеров: %d\n",
			user.Warns,
			user.AddedScammers,
		)
	}

	text += fmt.Sprintf(
		"\n🔎 Пользователь искался: %d раз\n"+
			"📅 Последняя проверка [%s]",
		user.UserSearched,
		time.Now().Format("02.01.2006"),
	)

	return text
}

// formatUnknownUser форматирует пользователя,
// которого нет в базе данных.
func formatUnknownUser(identifier string) string {
	identifier = strings.TrimSpace(identifier)

	// Если искали по Telegram ID.
	if id, err := strconv.ParseInt(identifier, 10, 64); err == nil {
		return fmt.Sprintf(
			"👤 Имя: —\n"+
				"🪪 ID: [%d]\n\n"+
				"⚙️ Роль: Нет в базе\n\n"+
				"🔎 Пользователь искался: 0 раз\n"+
				"📅 Последняя проверка [%s]",
			id,
			time.Now().Format("02.01.2006"),
		)
	}

	// Если искали по username.
	username := "@" + strings.TrimPrefix(identifier, "@")

	return fmt.Sprintf(
		"👤 Имя: %s\n"+
			"🪪 ID: —\n\n"+
			"⚙️ Роль: Нет в базе\n\n"+
			"🔎 Пользователь искался: 0 раз\n"+
			"📅 Последняя проверка [%s]",
		username,
		time.Now().Format("02.01.2006"),
	)
}

func showMessageWithUserID(
	next bot.HandlerFunc,
) bot.HandlerFunc {
	return func(
		ctx context.Context,
		b *bot.Bot,
		update *models.Update,
	) {
		if update.Message != nil {
			log.Printf(
				"%d say: %s",
				update.Message.From.ID,
				update.Message.Text,
			)
		}

		next(ctx, b, update)
	}
}

func isAdminRole(role string) bool {
	switch database.Roles(role) {
	case database.Tech,
		database.Owner,
		database.CoOwner,
		database.SeniorAdmin,
		database.Admin,
		database.JuniorAdmin,
		database.Intern:
		return true

	default:
		return false
	}
}

func formatGuarantor(
	guarantor database.Guarantors,
	username string,
) string {
	channel := "-"
	if guarantor.ChanelUrl != nil && *guarantor.ChanelUrl != "" {
		channel = *guarantor.ChanelUrl
	}

	proofs := "-"
	if guarantor.ProofsCount != nil {
		proofs = *guarantor.ProofsCount
	}

	region := "-"
	if guarantor.Region != nil && *guarantor.Region != "" {
		region = *guarantor.Region
	}

	trainees := "-"
	if guarantor.TraineeIDs != nil && len(*guarantor.TraineeIDs) > 0 {
		parts := make([]string, 0, len(*guarantor.TraineeIDs))

		for _, id := range *guarantor.TraineeIDs {
			parts = append(parts, strconv.FormatInt(id, 10))
		}

		trainees = strings.Join(parts, ", ")
	}

	rank := "❓ Неизвестный"

	switch guarantor.Rank {
	case database.Ranks("basic"):
		rank = "🛡️ Гарант"
	case database.Ranks("top"):
		rank = "🔝 Топ гарант"
	}

	return fmt.Sprintf(
		"👤 TG: @%s\n"+
			"🆔 Telegram ID: <code>%d</code>\n"+
			"🏆 Ранг: %s\n"+
			"🌍 Регион: %s\n"+
			"📢 Канал: %s\n"+
			"📊 Пруфов: %s\n"+
			"👥 Trainee: %s",
		username,
		guarantor.TgUserID,
		rank,
		region,
		channel,
		proofs,
		trainees,
	)
}

func formatRole(role string) string {
	switch database.Roles(role) {
	case database.Tech:
		return "🛠️ Технический специалист"

	case database.Owner:
		return "👑 Владелец"

	case database.CoOwner:
		return "🤝 Совладелец"

	case database.SeniorAdmin:
		return "🔱 Старший администратор"

	case database.Admin:
		return "🛡️ Администратор"

	case database.JuniorAdmin:
		return "🔰 Младший администратор"

	case database.Intern:
		return "📚 Стажёр"

	case database.TopGuarantor:
		return "💎 Топ гарант"

	case database.Guarantor:
		return "🛡️ Гарант"

	case database.Trainee:
		return "🎓 Стажёр гаранта"

	case database.Scammer:
		return "🚨 Скамер"

	case database.DodgyCharacter:
		return "⚠️ Подозрительный пользователь"

	case database.RegularUser:
		return "👤 Пользователь"

	default:
		return "❓ Нет в базе"
	}
}

func canSetRole(role string) bool {
	switch database.Roles(role) {
	case database.Tech,
		database.Owner,
		database.CoOwner,
		database.SeniorAdmin:
		return true

	default:
		return false
	}
}
