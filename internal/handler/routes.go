package handler

import (
	"github.com/dafibh/fortuna/fortuna-backend/internal/middleware"
	"github.com/labstack/echo/v4"
)

// RegisterRoutes sets up all API routes
func RegisterRoutes(e *echo.Echo, authMiddleware *middleware.AuthMiddleware, authHandler *AuthHandler, profileHandler *ProfileHandler, accountHandler *AccountHandler, transactionHandler *TransactionHandler, monthHandler *MonthHandler, dashboardHandler *DashboardHandler, budgetCategoryHandler *BudgetCategoryHandler, budgetHandler *BudgetHandler) {
	// API version 1
	api := e.Group("/api/v1")

	// Auth routes (protected)
	auth := api.Group("/auth")
	auth.Use(authMiddleware.Authenticate())
	auth.POST("/callback", authHandler.Callback)
	auth.GET("/me", authHandler.Me)
	auth.POST("/logout", authHandler.Logout)

	// Profile routes (protected)
	profile := api.Group("/profile")
	profile.Use(authMiddleware.Authenticate())
	profile.GET("", profileHandler.GetProfile)
	profile.PUT("", profileHandler.UpdateProfile)

	// Account routes (protected)
	accounts := api.Group("/accounts")
	accounts.Use(authMiddleware.Authenticate())
	accounts.POST("", accountHandler.CreateAccount)
	accounts.GET("", accountHandler.GetAccounts)
	accounts.PUT("/:id", accountHandler.UpdateAccount)
	accounts.DELETE("/:id", accountHandler.DeleteAccount)

	// Transaction routes (protected)
	transactions := api.Group("/transactions")
	transactions.Use(authMiddleware.Authenticate())
	transactions.POST("", transactionHandler.CreateTransaction)
	transactions.GET("", transactionHandler.GetTransactions)
	transactions.GET("/categories/recent", transactionHandler.GetRecentlyUsedCategories)
	transactions.PUT("/:id", transactionHandler.UpdateTransaction)
	transactions.DELETE("/:id", transactionHandler.DeleteTransaction)
	transactions.PATCH("/:id/toggle-paid", transactionHandler.TogglePaidStatus)
	transactions.PATCH("/:id/settlement-intent", transactionHandler.UpdateSettlementIntent)
	transactions.POST("/transfers", transactionHandler.CreateTransfer)

	// Month routes (protected)
	months := api.Group("/months")
	months.Use(authMiddleware.Authenticate())
	months.GET("/current", monthHandler.GetCurrent)
	months.GET("/:year/:month", monthHandler.GetByYearMonth)
	months.GET("", monthHandler.GetAllMonths)

	// Dashboard routes (protected)
	dashboard := api.Group("/dashboard")
	dashboard.Use(authMiddleware.Authenticate())
	dashboard.GET("/summary", dashboardHandler.GetSummary)

	// Budget Category routes (protected)
	budgetCategories := api.Group("/budget-categories")
	budgetCategories.Use(authMiddleware.Authenticate())
	budgetCategories.POST("", budgetCategoryHandler.CreateCategory)
	budgetCategories.GET("", budgetCategoryHandler.GetCategories)
	budgetCategories.PUT("/:id", budgetCategoryHandler.UpdateCategory)
	budgetCategories.DELETE("/:id", budgetCategoryHandler.DeleteCategory)
	budgetCategories.GET("/:id/can-delete", budgetCategoryHandler.CanDeleteCategory)

	// Budget Allocation routes (protected)
	budgets := api.Group("/budgets")
	budgets.Use(authMiddleware.Authenticate())
	budgets.GET("/:year/:month", budgetHandler.GetAllocations)
	budgets.PUT("/:year/:month", budgetHandler.SetAllocations)
	budgets.PUT("/:year/:month/:categoryId", budgetHandler.SetAllocation)
	budgets.GET("/:year/:month/:categoryId/transactions", budgetHandler.GetCategoryTransactions)
}
