package handler

import (
	"log"
	"net/http"

	"personal_finance_backend/auth"
	"personal_finance_backend/database"

	"github.com/gin-gonic/gin"
)

// --- Handler Receiver Struct (Dependency Injection) ---

type PotHandler struct {
	store database.Store
}

func NewPotHandler(store database.Store) *PotHandler {
	return &PotHandler{
		store: store,
	}
}

// --- HTTP Handlers ---

func (h *PotHandler) GetPots(c *gin.Context) {
	user, ok := h.extractAuthUser(c)
	if !ok {
		return
	}

	pots, err := h.store.GetPots(user.UID)
	if err != nil {
		log.Printf("[ERROR GetPots] Failed to fetch pots for UID %s: %v", user.UID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pots": pots})
}

func (h *PotHandler) CreatePot(c *gin.Context) {
	user, ok := h.extractAuthUser(c)
	if !ok {
		return
	}

	var payload CreatePotRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pot := database.Pot{
		Name:   payload.Name,
		Target: payload.Target,
		Total:  payload.Total,
		Theme:  payload.Theme,
	}

	createdPot, err := h.store.CreatePot(user.UID, pot)
	if err != nil {
		log.Printf("[ERROR CreatePot] Failed to create pot for UID %s: %v", user.UID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "pot created successfully", "pot": createdPot})
}

func (h *PotHandler) UpdatePot(c *gin.Context) {
	user, ok := h.extractAuthUser(c)
	if !ok {
		return
	}

	potID := c.Param("id")

	var payload UpdatePotRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pot := database.Pot{
		Name:   payload.Name,
		Target: payload.Target,
		Total:  payload.Total,
		Theme:  payload.Theme,
	}

	updatedPot, err := h.store.UpdatePot(user.UID, potID, pot)
	if err != nil {
		log.Printf("[ERROR UpdatePot] Failed to update pot %s for UID %s: %v", potID, user.UID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "pot updated successfully",
		"pot":     updatedPot,
	})
}

func (h *PotHandler) DeletePot(c *gin.Context) {
	user, ok := h.extractAuthUser(c)
	if !ok {
		return
	}

	potID := c.Param("id")

	if err := h.store.DeletePot(user.UID, potID); err != nil {
		log.Printf("[ERROR DeletePot] Failed to delete pot %s for UID %s: %v", potID, user.UID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "pot deleted successfully"})
}

// --- Context & Auth Helpers ---

func (h *PotHandler) extractAuthUser(c *gin.Context) (*auth.VerifiedFirebaseUser, bool) {
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

// --- DTOs ---

type CreatePotRequest struct {
	Name   string  `json:"name" binding:"required"`
	Target float64 `json:"target" binding:"required"`
	Total  float64 `json:"total"`
	Theme  string  `json:"theme" binding:"required"`
}

type UpdatePotRequest struct {
	Name   string  `json:"name" binding:"required"`
	Target float64 `json:"target" binding:"required"`
	Total  float64 `json:"total"`
	Theme  string  `json:"theme" binding:"required"`
}
