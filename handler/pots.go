package handler

import (
	"log"
	"net/http"
	"personal_finance_backend/auth"
	"personal_finance_backend/database"

	"github.com/gin-gonic/gin"
)

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

func GetPots(c *gin.Context) {
	firebaseUser, exists := c.Get("firebase_user")
	if !exists {
		log.Printf("[ERROR GetPots] 401 Unauthorized: firebase user not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "firebase user not found in context"})
		return
	}
	user := firebaseUser.(*auth.VerifiedFirebaseUser)

	pots, err := database.DefaultStore.GetPots(user.UID)
	if err != nil {
		log.Printf("[ERROR GetPots] Failed to fetch pots for UID %s: %v", user.UID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pots": pots})
}

func CreatePot(c *gin.Context) {
	firebaseUser, exists := c.Get("firebase_user")
	if !exists {
		log.Printf("[ERROR CreatePot] 401 Unauthorized: firebase user not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "firebase user not found in context"})
		return
	}
	user := firebaseUser.(*auth.VerifiedFirebaseUser)

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

	createdPot, err := database.DefaultStore.CreatePot(user.UID, pot)
	if err != nil {
		log.Printf("[ERROR CreatePot] Failed to create pot for UID %s: %v", user.UID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "pot created successfully", "pot": createdPot})
}

func UpdatePot(c *gin.Context) {
	firebaseUser, exists := c.Get("firebase_user")
	if !exists {
		log.Printf("[ERROR UpdatePot] 401 Unauthorized: firebase user not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "firebase user not found in context"})
		return
	}
	user := firebaseUser.(*auth.VerifiedFirebaseUser)

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

	updatedPot, err := database.DefaultStore.UpdatePot(user.UID, potID, pot)
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

func DeletePot(c *gin.Context) {
	firebaseUser, exists := c.Get("firebase_user")
	if !exists {
		log.Printf("[ERROR DeletePot] 401 Unauthorized: firebase user not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "firebase user not found in context"})
		return
	}
	user := firebaseUser.(*auth.VerifiedFirebaseUser)

	potID := c.Param("id")

	if err := database.DefaultStore.DeletePot(user.UID, potID); err != nil {
		log.Printf("[ERROR DeletePot] Failed to delete pot %s for UID %s: %v", potID, user.UID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "pot deleted successfully"})
}
