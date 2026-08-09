package handler

import (
	"log"
	"net/http"
	"personal_finance_backend/auth"
	"personal_finance_backend/database"

	"github.com/gin-gonic/gin"
)

type UpdateBudgetRequest struct {
	Category  string  `json:"category" binding:"required"`
	Maximum   float64 `json:"maximum" binding:"required"`
	Theme     string  `json:"theme" binding:"required"`
	Period    string  `json:"period"`
	StartDate string  `json:"startDate"`
}

type CreateBudgetRequest struct {
	Category  string  `json:"category" binding:"required"`
	Maximum   float64 `json:"maximum" binding:"required"`
	Theme     string  `json:"theme" binding:"required"`
	Period    string  `json:"period"`
	StartDate string  `json:"startDate"`
}

func GetBudgets(c *gin.Context) {
	firebaseUser, exists := c.Get("firebase_user")
	if !exists {
		log.Printf("[ERROR GetBudgets] 401 Unauthorized: firebase user not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "firebase user not found in context"})
		return
	}
	user := firebaseUser.(*auth.VerifiedFirebaseUser)

	budgets, err := database.DefaultStore.GetBudgets(user.UID)
	if err != nil {
		log.Printf("[ERROR GetBudgets] Failed to fetch budgets for UID %s: %v", user.UID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"budgets": budgets})
}

func UpdateBudget(c *gin.Context) {
	firebaseUser, exists := c.Get("firebase_user")
	if !exists {
		log.Printf("[ERROR UpdateBudget] 401 Unauthorized: firebase user not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "firebase user not found in context"})
		return
	}
	user := firebaseUser.(*auth.VerifiedFirebaseUser)

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

	updatedBudget, err := database.DefaultStore.UpdateBudget(user.UID, budgetID, budget)
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

func DeleteBudget(c *gin.Context) {
	firebaseUser, exists := c.Get("firebase_user")
	if !exists {
		log.Printf("[ERROR DeleteBudget] 401 Unauthorized: firebase user not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "firebase user not found in context"})
		return
	}
	user := firebaseUser.(*auth.VerifiedFirebaseUser)

	budgetID := c.Param("id")

	if err := database.DefaultStore.DeleteBudget(user.UID, budgetID); err != nil {
		log.Printf("[ERROR DeleteBudget] Failed to delete budget %s for UID %s: %v", budgetID, user.UID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "budget deleted successfully"})
}

func CreateBudget(c *gin.Context) {
	firebaseUser, exists := c.Get("firebase_user")
	if !exists {
		log.Printf("[ERROR CreateBudget] 401 Unauthorized: firebase user not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "firebase user not found in context"})
		return
	}
	user := firebaseUser.(*auth.VerifiedFirebaseUser)

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

	createdBudget, err := database.DefaultStore.CreateBudget(user.UID, budget)
	if err != nil {
		log.Printf("[ERROR CreateBudget] Failed to create budget for UID %s: %v", user.UID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "budget created successfully", "budget": createdBudget})
}
