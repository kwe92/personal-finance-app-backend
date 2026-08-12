package router

import (
	"personal_finance_backend/database"
	"personal_finance_backend/handler"
	"personal_finance_backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/plaid/plaid-go/v12/plaid"
)

// SetupRouter receives initialized dependencies needed by the handlers.
func SetupRouter(plaidClient *plaid.APIClient, store database.Store) *gin.Engine {
	router := gin.Default()
	router.Use(middleware.CORSMiddleware())

	// Initialize handlers with their dependencies
	plaidHandler := handler.NewPlaidHandler(plaidClient, store)

	protected := router.Group("/api")
	protected.Use(middleware.FirebaseAuthMiddleware())

	protected.GET("/health", handler.Health)

	// Firebase auth endpoint
	protected.POST("/verify-firebase-user", handler.VerifyFirebaseUser)

	// Plaid endpoints (bound to plaidHandler receiver methods)
	protected.POST("/plaid/create-link-token", plaidHandler.CreateLinkToken)
	protected.POST("/plaid/set-access-token", plaidHandler.SetAccessToken)
	protected.POST("/plaid/transactions", plaidHandler.GetTransactions)
	protected.GET("/plaid/overview-summary", plaidHandler.GetOverviewSummary)
	protected.GET("/plaid/recurring-bills", plaidHandler.GetRecurringBills)

	// Budget endpoints (uncounted until refactored)
	protected.GET("/budgets", handler.GetBudgets)
	protected.POST("/budgets", handler.CreateBudget)
	protected.PUT("/budgets/:id", handler.UpdateBudget)
	protected.DELETE("/budgets/:id", handler.DeleteBudget)

	// Pot endpoints (uncounted until refactored)
	protected.GET("/pots", handler.GetPots)
	protected.POST("/pots", handler.CreatePot)
	protected.PUT("/pots/:id", handler.UpdatePot)
	protected.DELETE("/pots/:id", handler.DeletePot)

	return router
}
