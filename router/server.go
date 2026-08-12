package router

import (
	"personal_finance_backend/database"
	"personal_finance_backend/handler"
	"personal_finance_backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/plaid/plaid-go/v12/plaid"
)

func SetupRouter(plaidClient *plaid.APIClient, store database.Store) *gin.Engine {
	router := gin.Default()
	router.Use(middleware.CORSMiddleware())

	// Initialize handlers with injected dependencies
	plaidHandler := handler.NewPlaidHandler(plaidClient, store)
	budgetHandler := handler.NewBudgetHandler(store)
	potHandler := handler.NewPotHandler(store)

	protected := router.Group("/api")
	protected.Use(middleware.FirebaseAuthMiddleware())

	protected.GET("/health", handler.Health)

	// Firebase auth endpoint
	protected.POST("/verify-firebase-user", handler.VerifyFirebaseUser)

	// Plaid endpoints
	protected.POST("/plaid/create-link-token", plaidHandler.CreateLinkToken)
	protected.POST("/plaid/set-access-token", plaidHandler.SetAccessToken)
	protected.POST("/plaid/transactions", plaidHandler.GetTransactions)
	protected.GET("/plaid/overview-summary", plaidHandler.GetOverviewSummary)
	protected.GET("/plaid/recurring-bills", plaidHandler.GetRecurringBills)

	// Budget endpoints
	protected.GET("/budgets", budgetHandler.GetBudgets)
	protected.POST("/budgets", budgetHandler.CreateBudget)
	protected.PUT("/budgets/:id", budgetHandler.UpdateBudget)
	protected.DELETE("/budgets/:id", budgetHandler.DeleteBudget)

	// Pot endpoints
	protected.GET("/pots", potHandler.GetPots)
	protected.POST("/pots", potHandler.CreatePot)
	protected.PUT("/pots/:id", potHandler.UpdatePot)
	protected.DELETE("/pots/:id", potHandler.DeletePot)

	return router
}
