package middleware

import (
	"context"
	"net/http"
	"strings"

	"personal_finance_backend/auth"

	"github.com/gin-gonic/gin"
)

func FirebaseAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		parts := strings.Split(authorization, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication Required"})
			c.Abort()
			return
		}

		user, err := auth.VerifyIDToken(context.Background(), parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Firebase token"})
			c.Abort()
			return
		}

		c.Set("firebase_user", user)
		c.Next()
	}
}
