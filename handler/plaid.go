package handler

import (
	"errors"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"personal_finance_backend/auth"
	"personal_finance_backend/database"

	"github.com/gin-gonic/gin"
	"github.com/plaid/plaid-go/v12/plaid"
)

// --- DTOs ---

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
	Balance        float64 `json:"balance"`
	Income         float64 `json:"income"`
	Expenses       float64 `json:"expenses"`
	Savings        float64 `json:"savings"`
	RecurringBills float64 `json:"recurringBills"`
}

// --- Handler Receiver Struct (Dependency Injection) ---

type PlaidHandler struct {
	plaidClient *plaid.APIClient
	store       database.Store // assuming database.Store is your interface, or concrete type pointer
}

func NewPlaidHandler(client *plaid.APIClient, store database.Store) *PlaidHandler {
	return &PlaidHandler{
		plaidClient: client,
		store:       store,
	}
}

// --- HTTP Handlers ---

func (h *PlaidHandler) CreateLinkToken(c *gin.Context) {
	var reqBody CreateLinkTokenRequest
	_ = c.ShouldBindJSON(&reqBody)

	userID := reqBody.UserID
	if userID == "" {
		userID = "personal-finance-user"
	}

	user := plaid.NewLinkTokenCreateRequestUser(userID)
	request := plaid.NewLinkTokenCreateRequest(
		"Personal Finance Backend",
		"en",
		[]plaid.CountryCode{plaid.COUNTRYCODE_US},
		*user,
	)
	request.SetProducts([]plaid.Products{plaid.PRODUCTS_TRANSACTIONS})

	resp, _, err := h.plaidClient.PlaidApi.LinkTokenCreate(c.Request.Context()).
		LinkTokenCreateRequest(*request).
		Execute()

	if err != nil {
		handlePlaidError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"linkToken": resp.GetLinkToken()})
}

func (h *PlaidHandler) SetAccessToken(c *gin.Context) {
	var payload SetAccessTokenRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		handlePlaidError(c, err)
		return
	}

	exchangeReq := plaid.NewItemPublicTokenExchangeRequest(payload.PublicToken)
	resp, _, err := h.plaidClient.PlaidApi.ItemPublicTokenExchange(c.Request.Context()).
		ItemPublicTokenExchangeRequest(*exchangeReq).
		Execute()
	if err != nil {
		handlePlaidError(c, err)
		return
	}

	user, ok := h.extractAuthUser(c)
	if !ok {
		return
	}

	if err := h.store.UpdatePlaidAccessToken(user.UID, resp.GetAccessToken()); err != nil {
		handlePlaidError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "plaid access token stored",
		"accessToken": resp.GetAccessToken(),
	})
}

func (h *PlaidHandler) GetTransactions(c *gin.Context) {
	accessToken, ok := h.getAccessTokenForUser(c)
	if !ok {
		return
	}

	req := plaid.NewTransactionsSyncRequest(accessToken)
	resp, _, err := h.plaidClient.PlaidApi.TransactionsSync(c.Request.Context()).
		TransactionsSyncRequest(*req).
		Execute()
	if err != nil {
		handlePlaidError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"transactions": mapPlaidTransactions(resp.GetAdded())})
}

func (h *PlaidHandler) GetOverviewSummary(c *gin.Context) {
	accessToken, ok := h.getAccessTokenForUser(c)
	if !ok {
		return
	}

	ctx := c.Request.Context()

	// 1. Fetch Balances
	balanceReq := plaid.NewAccountsBalanceGetRequest(accessToken)
	balanceResp, _, err := h.plaidClient.PlaidApi.AccountsBalanceGet(ctx).
		AccountsBalanceGetRequest(*balanceReq).
		Execute()
	if err != nil {
		handlePlaidError(c, err)
		return
	}

	var currentBalance float64
	for _, account := range balanceResp.GetAccounts() {
		currentBalance += *account.GetBalances().Current.Get()
	}

	// 2. Sync Transactions
	transactionsReq := plaid.NewTransactionsSyncRequest(accessToken)
	transactionsResp, _, err := h.plaidClient.PlaidApi.TransactionsSync(ctx).
		TransactionsSyncRequest(*transactionsReq).
		Execute()
	if err != nil {
		handlePlaidError(c, err)
		return
	}

	income, expenses := summarizeTransactions(transactionsResp.GetAdded())

	// 3. Fetch Recurring Bills
	recurringReq := plaid.NewTransactionsRecurringGetRequest(accessToken, nil)
	recurringResp, _, err := h.plaidClient.PlaidApi.TransactionsRecurringGet(ctx).
		TransactionsRecurringGetRequest(*recurringReq).
		Execute()
	if err != nil {
		handlePlaidError(c, err)
		return
	}

	recurringBills := summarizeRecurringBills(recurringResp.GetOutflowStreams())

	c.JSON(http.StatusOK, OverviewSummaryDTO{
		Balance:        currentBalance,
		Income:         income,
		Expenses:       expenses,
		Savings:        income - expenses,
		RecurringBills: recurringBills,
	})
}

func (h *PlaidHandler) GetRecurringBills(c *gin.Context) {
	accessToken, ok := h.getAccessTokenForUser(c)
	if !ok {
		return
	}

	req := plaid.NewTransactionsRecurringGetRequest(accessToken, nil)
	resp, _, err := h.plaidClient.PlaidApi.TransactionsRecurringGet(c.Request.Context()).
		TransactionsRecurringGetRequest(*req).
		Execute()
	if err != nil {
		handlePlaidError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"recurringBills": mapPlaidRecurringStreams(resp.GetOutflowStreams())})
}

// --- Context & Auth Helpers ---

func (h *PlaidHandler) extractAuthUser(c *gin.Context) (*auth.VerifiedFirebaseUser, bool) {
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

func (h *PlaidHandler) getAccessTokenForUser(c *gin.Context) (string, bool) {
	user, ok := h.extractAuthUser(c)
	if !ok {
		return "", false
	}

	userRecord, ok := h.store.GetUser(user.UID)
	if !ok || strings.TrimSpace(userRecord.PlaidAccessToken) == "" {
		log.Printf("[ERROR] 400 Bad Request: no plaid access token for UID %s", user.UID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "no plaid access token for this user"})
		return "", false
	}

	return userRecord.PlaidAccessToken, true
}

// --- Pure Domain / Mapping Helpers ---

func handlePlaidError(c *gin.Context, err error) {
	if plaidErr, pErr := plaid.ToPlaidError(err); pErr == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error_type":      plaidErr.GetErrorType(),
			"error_code":      plaidErr.GetErrorCode(),
			"error_message":   plaidErr.GetErrorMessage(),
			"display_message": plaidErr.GetDisplayMessage(),
		})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

func determineRecurringStatusAndNextDate(stream plaid.TransactionStream) (string, string, *int) {
	if !stream.GetIsActive() {
		return "paid", stream.GetLastDate(), nil
	}

	lastDateStr := stream.GetLastDate()
	if strings.TrimSpace(lastDateStr) == "" {
		lastDateStr = stream.GetFirstDate()
	}

	if strings.TrimSpace(lastDateStr) == "" {
		return "upcoming", "", nil
	}

	nextDateStr, nextDateTime, err := calculateNextDate(lastDateStr, stream.GetFrequency())
	if err != nil {
		return "upcoming", lastDateStr, nil
	}

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	targetDate := time.Date(nextDateTime.Year(), nextDateTime.Month(), nextDateTime.Day(), 0, 0, 0, 0, time.UTC)

	days := int(targetDate.Sub(today).Hours() / 24)

	var status string
	switch {
	case days < 0:
		status = "past_due"
	case days <= 7:
		status = "due_soon"
	default:
		status = "upcoming"
	}

	return status, nextDateStr, &days
}

func mapPlaidTransactions(transactions []plaid.Transaction) []TransactionDTO {
	mapped := make([]TransactionDTO, 0, len(transactions))
	for _, txn := range transactions {
		category := resolveCategory(txn.GetCategory(), txn.GetMerchantName(), txn.GetName())

		date := txn.GetDate()
		if strings.TrimSpace(date) == "" {
			date = txn.GetAuthorizedDate()
		}

		mapped = append(mapped, TransactionDTO{
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
	return mapped
}

func mapPlaidRecurringStreams(streams []plaid.TransactionStream) []TransactionDTO {
	mapped := make([]TransactionDTO, 0, len(streams))
	for _, stream := range streams {
		category := resolveCategory(stream.GetCategory(), stream.GetMerchantName(), stream.GetDescription())

		date := stream.GetLastDate()
		if strings.TrimSpace(date) == "" {
			date = stream.GetFirstDate()
		}

		amount := stream.LastAmount.GetAmount()
		if amount == 0 {
			amount = stream.AverageAmount.GetAmount()
		}

		status, nextDate, daysUntilDue := determineRecurringStatusAndNextDate(stream)

		mapped = append(mapped, TransactionDTO{
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
	return mapped
}

func resolveCategory(categories []string, fallback1, fallback2 string) string {
	if len(categories) > 0 && strings.TrimSpace(categories[0]) != "" {
		return categories[0]
	}
	if strings.TrimSpace(fallback1) != "" {
		return fallback1
	}
	if strings.TrimSpace(fallback2) != "" {
		return fallback2
	}
	return "Uncategorized"
}

func summarizeTransactions(transactions []plaid.Transaction) (income float64, expenses float64) {
	for _, txn := range transactions {
		amount := txn.GetAmount()
		if amount < 0 {
			income += math.Abs(amount)
		} else if amount > 0 {
			expenses += amount
		}
	}
	return income, expenses
}

func summarizeRecurringBills(streams []plaid.TransactionStream) float64 {
	var total float64
	for _, stream := range streams {
		amount := stream.LastAmount.GetAmount()
		if amount == 0 {
			amount = stream.AverageAmount.GetAmount()
		}
		total += math.Abs(amount)
	}
	return total
}
