package handler

import (
	"log"
	"net/http"
	"personal_finance_backend/auth"
	"personal_finance_backend/database"

	"github.com/gin-gonic/gin"
)

type PreferenceHandler struct {
	store database.Store
}

func NewPreferenceHandler(store database.Store) *PreferenceHandler {
	return &PreferenceHandler{store: store}
}

func (h *PreferenceHandler) GetPreferences(c *gin.Context) {
	user, ok := h.extractAuthUser(c)
	if !ok {
		return
	}

	prefs, err := h.store.GetPreferences(user.UID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, prefs)
}

func (h *PreferenceHandler) UpdatePreferences(c *gin.Context) {
	user, ok := h.extractAuthUser(c)
	if !ok {
		return
	}

	var payload struct {
		MonthlySpendingTarget float64 `json:"monthlySpendingTarget"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prefs := database.UserPreferences{
		MonthlySpendingTarget: payload.MonthlySpendingTarget,
	}

	updated, err := h.store.UpdatePreferences(user.UID, prefs)
	if err != nil {
		log.Printf("[ERROR] UpdatePreferences: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updated)
}

func (h *PreferenceHandler) extractAuthUser(c *gin.Context) (*auth.VerifiedFirebaseUser, bool) {
	val, exists := c.Get("firebase_user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "firebase user not found"})
		return nil, false
	}
	user := val.(*auth.VerifiedFirebaseUser)
	return user, true
}
