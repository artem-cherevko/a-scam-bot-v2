package api

import (
	"log"
	"strconv"

	"github.com/artem-cherevko/a-scam-bot-v2/internal/service"
	"github.com/gin-gonic/gin"
)

func Auth(userRepo *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the user ID from the request (e.g., from headers, query parameters, etc.)
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
			return
		}

		// Check if the user exists in the database
		id, err := strconv.ParseInt(userID, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(400, gin.H{"error": "Invalid user ID"})
			return
		}
		user, err := userRepo.GetUserByTelegramID(id)
		if err != nil || user == nil {
			log.Printf("auth failed: id=%d err=%v", id, err)
			c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
			return
		}

		// User is authenticated, proceed to the next handler
		c.Set("user", user)
		c.Next()
	}
}

func RequireAdmin(userRepo *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the user ID from the request (e.g., from headers, query parameters, etc.)
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
			return
		}

		// Check if the user exists in the database
		id, err := strconv.ParseInt(userID, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(400, gin.H{"error": "Invalid user ID"})
			return
		}
		user, err := userRepo.GetUserByTelegramID(id)
		if err != nil || user == nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
			return
		}

		// Check if the user is an admin
		if !user.IsAdmin() {
			c.AbortWithStatusJSON(403, gin.H{"error": "Forbidden"})
			return
		}

		// User is an admin, proceed to the next handler
		c.Set("user", user)
		c.Next()
	}
}
