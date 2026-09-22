package handler

import (
	"log"
	"net/http"
	"strings"

	"personal_finance_backend/auth"
	"personal_finance_backend/database"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	store database.Store
}

func NewSettingsHandler(store database.Store) *SettingsHandler {
	return &SettingsHandler{
		store: store,
	}
}

type UpdateUserNameRequest struct {
	DisplayName string `json:"displayName" binding:"required"`
}

type UpdatePasswordRequest struct {
	Password string `json:"password" binding:"required,min=6"`
}

func (h *SettingsHandler) UpdateName(c *gin.Context) {
	user, ok := h.extractAuthUser(c)
	if !ok {
		return
	}

	var payload UpdateUserNameRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	displayName := strings.TrimSpace(payload.DisplayName)
	if displayName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "display name cannot be empty"})
		return
	}

	if err := h.store.UpdateUserDisplayName(user.UID, displayName); err != nil {
		log.Printf("[ERROR UpdateName] Failed to update display name for UID %s: %v", user.UID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "user display name updated successfully",
		"updated": true,
	})
}

func (h *SettingsHandler) UpdatePassword(c *gin.Context) {
	user, ok := h.extractAuthUser(c)
	if !ok {
		return
	}

	var payload UpdatePasswordRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.UpdateUserPassword(user.UID, payload.Password); err != nil {
		log.Printf("[ERROR UpdatePassword] Failed to update password for UID %s: %v", user.UID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "user password updated successfully",
		"updated": true,
	})
}

func (h *SettingsHandler) extractAuthUser(c *gin.Context) (*auth.VerifiedFirebaseUser, bool) {
	val, exists := c.Get("firebase_user")
	if !exists {
		log.Printf("[ERROR] 401 Unauthorized: firebase user missing in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "firebase user not found in context"})
		return nil, false
	}

	user, ok := val.(*auth.VerifiedFirebaseUser)
	if !ok {
		log.Printf("[ERROR] 500 Internal Error: unexpected user type in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user context type"})
		return nil, false
	}

	return user, true
}
