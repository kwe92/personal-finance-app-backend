package handler

import (
	"log"
	"net/http"

	"personal_finance_backend/auth"
	"personal_finance_backend/database"

	"github.com/gin-gonic/gin"
)

// --- Handler Receiver Struct (Dependency Injection) ---

type BudgetHandler struct {
	store database.Store
}

func NewBudgetHandler(store database.Store) *BudgetHandler {
	return &BudgetHandler{
		store: store,
	}
}

// --- HTTP Handlers ---

func (h *BudgetHandler) GetBudgets(c *gin.Context) {
	user, ok := h.extractAuthUser(c)
	if !ok {
		return
	}

	budgets, err := h.store.GetBudgets(user.UID)
	if err != nil {
		log.Printf("[ERROR GetBudgets] Failed to fetch budgets for UID %s: %v", user.UID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"budgets": budgets})
}

func (h *BudgetHandler) CreateBudget(c *gin.Context) {
	user, ok := h.extractAuthUser(c)
	if !ok {
		return
	}

	var payload CreateBudgetRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	budget := database.Budget{
		Category:  payload.Category,
		Maximum:   payload.Maximum,
		Theme:     payload.Theme,
		Period:    payload.Period,
		StartDate: payload.StartDate,
	}

	createdBudget, err := h.store.CreateBudget(user.UID, budget)
	if err != nil {
		log.Printf("[ERROR CreateBudget] Failed to create budget for UID %s: %v", user.UID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "budget created successfully", "budget": createdBudget})
}

func (h *BudgetHandler) UpdateBudget(c *gin.Context) {
	user, ok := h.extractAuthUser(c)
	if !ok {
		return
	}

	budgetID := c.Param("id")

	var payload UpdateBudgetRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	budget := database.Budget{
		Category:  payload.Category,
		Maximum:   payload.Maximum,
		Theme:     payload.Theme,
		Period:    payload.Period,
		StartDate: payload.StartDate,
	}

	updatedBudget, err := h.store.UpdateBudget(user.UID, budgetID, budget)
	if err != nil {
		log.Printf("[ERROR UpdateBudget] Failed to update budget %s for UID %s: %v", budgetID, user.UID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "budget updated successfully",
		"budget":  updatedBudget,
	})
}

func (h *BudgetHandler) DeleteBudget(c *gin.Context) {
	user, ok := h.extractAuthUser(c)
	if !ok {
		return
	}

	budgetID := c.Param("id")

	if err := h.store.DeleteBudget(user.UID, budgetID); err != nil {
		log.Printf("[ERROR DeleteBudget] Failed to delete budget %s for UID %s: %v", budgetID, user.UID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "budget deleted successfully"})
}

// --- Context & Auth Helpers ---

func (h *BudgetHandler) extractAuthUser(c *gin.Context) (*auth.VerifiedFirebaseUser, bool) {
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

type CreateBudgetRequest struct {
	Category  string  `json:"category" binding:"required"`
	Maximum   float64 `json:"maximum" binding:"required"`
	Theme     string  `json:"theme" binding:"required"`
	Period    string  `json:"period"`
	StartDate string  `json:"startDate"`
}

type UpdateBudgetRequest struct {
	Category  string  `json:"category" binding:"required"`
	Maximum   float64 `json:"maximum" binding:"required"`
	Theme     string  `json:"theme" binding:"required"`
	Period    string  `json:"period"`
	StartDate string  `json:"startDate"`
}
