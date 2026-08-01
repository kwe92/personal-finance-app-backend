package handler

import (
	"context"
	"net/http"
	"personal_finance_backend/auth"

	"github.com/gin-gonic/gin"
)

type VerifyFirebaseUserRequest struct {
	IDToken string `json:"idToken" binding:"required"`
}

func VerifyFirebaseUser(c *gin.Context) {
	var payload VerifyFirebaseUserRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := auth.VerifyIDToken(context.Background(), payload.IDToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid firebase token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "firebase user verified",
		"uid":     user.UID,
		"email":   user.Email,
	})
}
