package handler

import (
	"context"
	"errors"
	"log"
	"math"
	"net/http"
	"personal_finance_backend/auth"
	"personal_finance_backend/database"
	"personal_finance_backend/plaid_config"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/plaid/plaid-go/v12/plaid"
)

type CreateLinkTokenRequest struct {
	UserID string `json:"userId"`
}

type SetAccessTokenRequest struct {
	PublicToken string `json:"publicToken" binding:"required"`
}

type TransactionDTO struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Category     string  `json:"category"`
	Date         string  `json:"date"`
	Amount       float64 `json:"amount"`
	Type         string  `json:"type"`
	Recurring    bool    `json:"recurring"`
	Frequency    string  `json:"frequency,omitempty"`
	NextDate     string  `json:"nextDate,omitempty"`
	Status       string  `json:"status,omitempty"`
	DaysUntilDue *int    `json:"daysUntilDue,omitempty"`
}

type OverviewSummaryDTO struct {
	Balance  float64 `json:"balance"`
	Income   float64 `json:"income"`
	Expenses float64 `json:"expenses"`
	Savings  float64 `json:"savings"`
}

func inferTransactionType(amount float64) string {
	switch {
	case amount < 0:
		return "income"
	case amount > 0:
		return "expense"
	default:
		return "transfer"
	}
}

func normalizeFrequency(frequency plaid.RecurringTransactionFrequency) string {
	switch frequency {
	case plaid.RECURRINGTRANSACTIONFREQUENCY_WEEKLY:
		return "weekly"
	case plaid.RECURRINGTRANSACTIONFREQUENCY_BIWEEKLY:
		return "biweekly"
	case plaid.RECURRINGTRANSACTIONFREQUENCY_SEMI_MONTHLY:
		return "semi_monthly"
	case plaid.RECURRINGTRANSACTIONFREQUENCY_MONTHLY:
		return "monthly"
	case plaid.RECURRINGTRANSACTIONFREQUENCY_ANNUALLY:
		return "yearly"
	default:
		return ""
	}
}

func calculateNextDate(lastDateStr string, frequency plaid.RecurringTransactionFrequency) (string, time.Time, error) {
	if strings.TrimSpace(lastDateStr) == "" {
		return "", time.Time{}, errors.New("empty last date")
	}

	lastDate, err := time.Parse("2006-01-02", lastDateStr)
	if err != nil {
		return "", time.Time{}, err
	}

	var nextDate time.Time
	switch frequency {
	case plaid.RECURRINGTRANSACTIONFREQUENCY_WEEKLY:
		nextDate = lastDate.AddDate(0, 0, 7)
	case plaid.RECURRINGTRANSACTIONFREQUENCY_BIWEEKLY:
		nextDate = lastDate.AddDate(0, 0, 14)
	case plaid.RECURRINGTRANSACTIONFREQUENCY_SEMI_MONTHLY:
		nextDate = lastDate.AddDate(0, 0, 15)
	case plaid.RECURRINGTRANSACTIONFREQUENCY_MONTHLY:
		nextDate = lastDate.AddDate(0, 1, 0)
	case plaid.RECURRINGTRANSACTIONFREQUENCY_ANNUALLY:
		nextDate = lastDate.AddDate(1, 0, 0)
	default:
		nextDate = lastDate.AddDate(0, 1, 0)
	}

	return nextDate.Format("2006-01-02"), nextDate, nil
}

func determineRecurringStatusAndNextDate(stream plaid.TransactionStream) (status string, nextDateStr string, daysUntilDue *int) {
	// 1. If stream is inactive, mark as paid
	if !stream.GetIsActive() {
		return "paid", stream.GetLastDate(), nil
	}

	// 2. Resolve Last Date (or First Date as fallback)
	lastDateStr := stream.GetLastDate()
	if strings.TrimSpace(lastDateStr) == "" {
		lastDateStr = stream.GetFirstDate()
	}

	if strings.TrimSpace(lastDateStr) == "" {
		return "upcoming", "", nil
	}

	// 3. Compute next date based on LastDate + Frequency interval
	nextDateStr, nextDateTime, err := calculateNextDate(lastDateStr, stream.GetFrequency())
	if err != nil {
		return "upcoming", lastDateStr, nil
	}

	// 4. Standardize dates to UTC midnight for exact day calculation
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	targetDate := time.Date(nextDateTime.Year(), nextDateTime.Month(), nextDateTime.Day(), 0, 0, 0, 0, time.UTC)

	// Calculate difference in full days
	days := int(targetDate.Sub(today).Hours() / 24)

	// 5. Categorization logic
	switch {
	case days < 0:
		status = "past_due"
	case days <= 7:
		status = "due_soon"
	default: // days >= 8
		status = "upcoming"
	}

	return status, nextDateStr, &days
}

func mapPlaidTransactions(transactions []plaid.Transaction) []TransactionDTO {
	mappedTransactions := make([]TransactionDTO, 0, len(transactions))
	for _, txn := range transactions {
		category := ""
		if len(txn.GetCategory()) > 0 {
			category = txn.GetCategory()[0]
		}
		if strings.TrimSpace(category) == "" {
			category = txn.GetMerchantName()
		}
		if strings.TrimSpace(category) == "" {
			category = txn.GetName()
		}
		if strings.TrimSpace(category) == "" {
			category = "Uncategorized"
		}

		date := txn.GetDate()
		if strings.TrimSpace(date) == "" {
			date = txn.GetAuthorizedDate()
		}

		mappedTransactions = append(mappedTransactions, TransactionDTO{
			ID:        txn.GetTransactionId(),
			Name:      txn.GetName(),
			Category:  category,
			Date:      date,
			Amount:    txn.GetAmount(),
			Type:      inferTransactionType(txn.GetAmount()),
			Recurring: false,
			Status:    "paid",
		})
	}

	return mappedTransactions
}

func mapPlaidRecurringStreams(streams []plaid.TransactionStream) []TransactionDTO {
	mappedStreams := make([]TransactionDTO, 0, len(streams))
	for _, stream := range streams {
		category := ""
		if len(stream.GetCategory()) > 0 {
			category = stream.GetCategory()[0]
		}
		if strings.TrimSpace(category) == "" {
			category = stream.GetMerchantName()
		}
		if strings.TrimSpace(category) == "" {
			category = stream.GetDescription()
		}
		if strings.TrimSpace(category) == "" {
			category = "Uncategorized"
		}

		date := stream.GetLastDate()
		if strings.TrimSpace(date) == "" {
			date = stream.GetFirstDate()
		}

		amount := stream.LastAmount.GetAmount()
		if amount == 0 {
			amount = stream.AverageAmount.GetAmount()
		}

		status, nextDate, daysUntilDue := determineRecurringStatusAndNextDate(stream)

		mappedStreams = append(mappedStreams, TransactionDTO{
			ID:           stream.GetStreamId(),
			Name:         stream.GetDescription(),
			Category:     category,
			Date:         date,
			Amount:       amount,
			Type:         "expense",
			Recurring:    true,
			Frequency:    normalizeFrequency(stream.GetFrequency()),
			NextDate:     nextDate,
			Status:       status,
			DaysUntilDue: daysUntilDue,
		})
	}

	return mappedStreams
}

func summarizeTransactions(transactions []plaid.Transaction) (income float64, expenses float64) {
	for _, txn := range transactions {
		amount := txn.GetAmount()
		if amount < 0 {
			income += math.Abs(amount)
			continue
		}
		if amount > 0 {
			expenses += amount
		}
	}
	return income, expenses
}

func CreateLinkToken(c *gin.Context) {
	var reqBody CreateLinkTokenRequest

	// Bind optional request body if provided by frontend
	_ = c.ShouldBindJSON(&reqBody)

	userID := reqBody.UserID
	if userID == "" {
		// Fallback to default if not provided
		userID = "personal-finance-user"
	}

	// Build the Plaid User object
	user := plaid.NewLinkTokenCreateRequestUser(userID)

	// Create Link Token request using Plaid SDK constants
	request := plaid.NewLinkTokenCreateRequest(
		"Personal Finance Backend",
		"en",
		[]plaid.CountryCode{plaid.COUNTRYCODE_US},
		*user,
	)
	request.SetProducts([]plaid.Products{plaid.PRODUCTS_TRANSACTIONS})

	// Execute API call
	resp, _, err := plaid_config.Client.PlaidApi.LinkTokenCreate(context.Background()).LinkTokenCreateRequest(*request).Execute()

	if err != nil {
		// Extract actual detailed Plaid error
		if plaidErr, pErr := plaid.ToPlaidError(err); pErr == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error_type":      plaidErr.GetErrorType(),
				"error_code":      plaidErr.GetErrorCode(),
				"error_message":   plaidErr.GetErrorMessage(),
				"display_message": plaidErr.GetDisplayMessage(),
			})
			return
		}

		// Generic fallback error
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

	mappedTransactions := mapPlaidTransactions(resp.GetAdded())
	c.JSON(http.StatusOK, gin.H{"transactions": mappedTransactions})
}

func GetOverviewSummary(c *gin.Context) {
	firebaseUser, exists := c.Get("firebase_user")
	if !exists {
		log.Printf("[ERROR GetOverviewSummary] 401 Unauthorized: firebase user not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "firebase user not found in context"})
		return
	}
	user := firebaseUser.(*auth.VerifiedFirebaseUser)

	userRecord, ok := database.DefaultStore.GetUser(user.UID)
	if !ok || strings.TrimSpace(userRecord.PlaidAccessToken) == "" {
		log.Printf("[ERROR GetOverviewSummary] 400 Bad Request: no plaid access token for UID %s", user.UID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "no plaid access token for this user"})
		return
	}

	ctx := context.Background()

	// 1. Fetch Balances
	balanceReq := plaid.NewAccountsBalanceGetRequest(userRecord.PlaidAccessToken)
	balanceResp, _, err := plaid_config.Client.PlaidApi.AccountsBalanceGet(ctx).AccountsBalanceGetRequest(*balanceReq).Execute()
	if err != nil {
		if plaidErr, parseErr := plaid.ToPlaidError(err); parseErr == nil {
			log.Printf("[ERROR GetOverviewSummary - Balance] Plaid Error [%s]: %s", plaidErr.ErrorCode, plaidErr.ErrorMessage)
			c.JSON(http.StatusBadRequest, gin.H{
				"error_code":    plaidErr.ErrorCode,
				"error_message": plaidErr.ErrorMessage,
				"error_type":    plaidErr.ErrorType,
			})
			return
		}
		log.Printf("[ERROR GetOverviewSummary - Balance]: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var currentBalance float64
	for _, account := range balanceResp.GetAccounts() {
		balances := account.GetBalances()
		currentBalance += balances.GetCurrent()
	}

	// 2. Sync Transactions
	transactionsReq := plaid.NewTransactionsSyncRequest(userRecord.PlaidAccessToken)
	transactionsResp, _, err := plaid_config.Client.PlaidApi.TransactionsSync(ctx).TransactionsSyncRequest(*transactionsReq).Execute()
	if err != nil {
		if plaidErr, parseErr := plaid.ToPlaidError(err); parseErr == nil {
			log.Printf("[ERROR GetOverviewSummary - TransactionsSync] Plaid Error [%s]: %s", plaidErr.ErrorCode, plaidErr.ErrorMessage)
			c.JSON(http.StatusBadRequest, gin.H{
				"error_code":    plaidErr.ErrorCode,
				"error_message": plaidErr.ErrorMessage,
				"error_type":    plaidErr.ErrorType,
			})
			return
		}
		log.Printf("[ERROR GetOverviewSummary - TransactionsSync]: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	income, expenses := summarizeTransactions(transactionsResp.GetAdded())

	// 3. Fetch Recurring Bills
	recurringReq := plaid.NewTransactionsRecurringGetRequest(userRecord.PlaidAccessToken, nil)
	recurringResp, _, err := plaid_config.Client.PlaidApi.TransactionsRecurringGet(ctx).TransactionsRecurringGetRequest(*recurringReq).Execute()
	if err != nil {
		if plaidErr, parseErr := plaid.ToPlaidError(err); parseErr == nil {
			log.Printf("[ERROR GetOverviewSummary - Recurring] Plaid Error [%s]: %s", plaidErr.ErrorCode, plaidErr.ErrorMessage)
			c.JSON(http.StatusBadRequest, gin.H{
				"error_code":    plaidErr.ErrorCode,
				"error_message": plaidErr.ErrorMessage,
				"error_type":    plaidErr.ErrorType,
			})
			return
		}
		log.Printf("[ERROR GetOverviewSummary - Recurring]: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var recurringBills float64
	for _, stream := range recurringResp.GetOutflowStreams() {
		amount := stream.LastAmount.GetAmount()
		if amount == 0 {
			amount = stream.AverageAmount.GetAmount()
		}
		recurringBills += math.Abs(amount)
	}

	c.JSON(http.StatusOK, gin.H{
		"balance":        currentBalance,
		"income":         income,
		"expenses":       expenses,
		"savings":        income - expenses,
		"recurringBills": recurringBills,
	})
}

func GetRecurringBills(c *gin.Context) {
	firebaseUser, exists := c.Get("firebase_user")
	if !exists {
		log.Printf("[ERROR GetRecurringBills] 401 Unauthorized: firebase user not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "firebase user not found in context"})
		return
	}
	user := firebaseUser.(*auth.VerifiedFirebaseUser)

	userRecord, ok := database.DefaultStore.GetUser(user.UID)
	if !ok || strings.TrimSpace(userRecord.PlaidAccessToken) == "" {
		log.Printf("[ERROR GetRecurringBills] 400 Bad Request: no plaid access token for UID %s", user.UID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "no plaid access token for this user"})
		return
	}

	ctx := context.Background()
	req := plaid.NewTransactionsRecurringGetRequest(userRecord.PlaidAccessToken, nil)

	resp, _, err := plaid_config.Client.PlaidApi.TransactionsRecurringGet(ctx).TransactionsRecurringGetRequest(*req).Execute()
	if err != nil {
		if plaidErr, parseErr := plaid.ToPlaidError(err); parseErr == nil {
			log.Printf("[ERROR GetRecurringBills] Plaid API Error [%s]: %s (Type: %s)", plaidErr.ErrorCode, plaidErr.ErrorMessage, plaidErr.ErrorType)
			c.JSON(http.StatusBadRequest, gin.H{
				"error_code":    plaidErr.ErrorCode,
				"error_message": plaidErr.ErrorMessage,
				"error_type":    plaidErr.ErrorType,
			})
			return
		}

		log.Printf("[ERROR GetRecurringBills] 400 Bad Request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"recurringBills": mapPlaidRecurringStreams(resp.GetOutflowStreams())})
}
