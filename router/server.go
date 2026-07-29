package router

import (
	"personal_finance_backend/handler"
	"personal_finance_backend/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	router := gin.Default()

	protected := router.Group("/api")
	protected.Use(middleware.FirebaseAuthMiddleware())

	protected.GET("/health", handler.Health)
	protected.POST("/verify-firebase-user", handler.VerifyFirebaseUser)
	protected.POST("/plaid/create-link-token", handler.CreateLinkToken)
	protected.POST("/plaid/set-access-token", handler.SetAccessToken)
	protected.POST("/plaid/transactions", handler.GetTransactions)

	return router
}
