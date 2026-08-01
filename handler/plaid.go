package handler

import (
	"context"
	"net/http"
	"personal_finance_backend/auth"
	"personal_finance_backend/database"
	"personal_finance_backend/plaid_config"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/plaid/plaid-go/v12/plaid"
)

type SetAccessTokenRequest struct {
	PublicToken string `json:"publicToken" binding:"required"`
}

func CreateLinkToken(c *gin.Context) {
	request := plaid.NewLinkTokenCreateRequest(
		"Personal Finance Backend",
		"en",
		[]plaid.CountryCode{plaid.CountryCode("US")},
		*plaid.NewLinkTokenCreateRequestUser("personal-finance-user"),
	)
	request.SetProducts([]plaid.Products{plaid.Products("transactions")})

	resp, _, err := plaid_config.Client.PlaidApi.LinkTokenCreate(context.Background()).LinkTokenCreateRequest(*request).Execute()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"linkToken": resp.GetLinkToken()})
}

func SetAccessToken(c *gin.Context) {
	var payload SetAccessTokenRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	exchangeRequest := plaid.NewItemPublicTokenExchangeRequest(payload.PublicToken)
	resp, _, err := plaid_config.Client.PlaidApi.ItemPublicTokenExchange(ctx).ItemPublicTokenExchangeRequest(*exchangeRequest).Execute()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	firebaseUser, exists := c.Get("firebase_user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "firebase user not found in context"})
		return
	}
	user := firebaseUser.(*auth.VerifiedFirebaseUser)
	if err := database.DefaultStore.UpdatePlaidAccessToken(user.UID, resp.GetAccessToken()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "plaid access token stored", "accessToken": resp.GetAccessToken()})
}

func GetTransactions(c *gin.Context) {
	firebaseUser, exists := c.Get("firebase_user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "firebase user not found in context"})
		return
	}
	user := firebaseUser.(*auth.VerifiedFirebaseUser)

	userRecord, ok := database.DefaultStore.GetUser(user.UID)
	if !ok || strings.TrimSpace(userRecord.PlaidAccessToken) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no plaid access token for this user"})
		return
	}

	ctx := context.Background()
	req := plaid.NewTransactionsSyncRequest(userRecord.PlaidAccessToken)
	resp, _, err := plaid_config.Client.PlaidApi.TransactionsSync(ctx).TransactionsSyncRequest(*req).Execute()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transactions": resp.GetAdded()})
}
