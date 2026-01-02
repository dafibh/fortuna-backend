package handler

import (
	"github.com/dafibh/fortuna/fortuna-backend/internal/middleware"
	"github.com/labstack/echo/v4"
)

// RegisterRoutes sets up all API routes
func RegisterRoutes(e *echo.Echo, authMiddleware *middleware.AuthMiddleware, authHandler *AuthHandler, profileHandler *ProfileHandler, accountHandler *AccountHandler, transactionHandler *TransactionHandler, monthHandler *MonthHandler, dashboardHandler *DashboardHandler, budgetCategoryHandler *BudgetCategoryHandler, budgetHandler *BudgetHandler, ccHandler *CCHandler, recurringHandler *RecurringHandler, loanProviderHandler *LoanProviderHandler, loanHandler *LoanHandler, loanPaymentHandler *LoanPaymentHandler) {
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
	accounts.GET("/cc-summary", accountHandler.GetCCSummary)
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

	// Credit Card routes (protected)
	cc := api.Group("/cc")
	cc.Use(authMiddleware.Authenticate())
	cc.GET("/payable/breakdown", ccHandler.GetPayableBreakdown)
	cc.POST("/payments", ccHandler.CreateCCPayment)

	// Recurring Transactions routes (protected)
	recurring := api.Group("/recurring-transactions")
	recurring.Use(authMiddleware.Authenticate())
	recurring.POST("", recurringHandler.CreateRecurring)
	recurring.GET("", recurringHandler.GetRecurringTransactions)
	recurring.POST("/generate", recurringHandler.GenerateRecurring)
	recurring.GET("/:id", recurringHandler.GetRecurringTransaction)
	recurring.PUT("/:id", recurringHandler.UpdateRecurring)
	recurring.PATCH("/:id/toggle-active", recurringHandler.ToggleActive)
	recurring.DELETE("/:id", recurringHandler.DeleteRecurring)

	// Loan Provider routes (protected)
	loanProviders := api.Group("/loan-providers")
	loanProviders.Use(authMiddleware.Authenticate())
	loanProviders.POST("", loanProviderHandler.CreateLoanProvider)
	loanProviders.GET("", loanProviderHandler.GetLoanProviders)
	loanProviders.GET("/:id", loanProviderHandler.GetLoanProvider)
	loanProviders.PUT("/:id", loanProviderHandler.UpdateLoanProvider)
	loanProviders.DELETE("/:id", loanProviderHandler.DeleteLoanProvider)

	// Loan routes (protected)
	loans := api.Group("/loans")
	loans.Use(authMiddleware.Authenticate())
	loans.POST("", loanHandler.CreateLoan)
	loans.GET("", loanHandler.GetLoans)
	loans.POST("/preview", loanHandler.PreviewLoan)
	loans.GET("/:id", loanHandler.GetLoan)
	loans.GET("/:id/delete-check", loanHandler.GetDeleteCheck)
	loans.PUT("/:id", loanHandler.UpdateLoan)
	loans.DELETE("/:id", loanHandler.DeleteLoan)

	// Loan Payment routes (nested under loans)
	loans.GET("/:loanId/payments", loanPaymentHandler.GetPaymentsByLoanID)
	loans.PATCH("/:loanId/payments/:paymentId", loanPaymentHandler.UpdatePaymentAmount)
	loans.PUT("/:loanId/payments/:paymentId/toggle-paid", loanPaymentHandler.TogglePaymentPaid)
}
