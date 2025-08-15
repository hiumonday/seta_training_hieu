package middleware

import (
	"log"
	"net/http"
	"strings"

	"go_service/internal/auth"
	"go_service/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AuthMiddleware(db *gorm.DB) gin.HandlerFunc {
	userService := services.NewUserService()
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			log.Printf("Token validation failed: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		log.Printf("Token validated successfully for user ID: %s", claims.UserID.String())

		// Fetch user info from database to get role
		userResponse, err := userService.GetUserByID(claims.UserID.String())
		if err != nil {
			log.Printf("Failed to fetch user data: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Fail to fetch user data"})
			c.Abort()
			return
		}

		if userResponse == nil || userResponse.User.ID == "" {
			log.Printf("User data is empty for ID: %s", claims.UserID.String())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		log.Printf("User found: ID=%s, Role=%s", userResponse.User.ID, userResponse.User.Role)

		c.Set("user_id", claims.UserID)
		c.Set("role", userResponse.User.Role)
		c.Next()
	}
}
