package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type BudgetAllocation struct {
	ID          int32           `json:"id"`
	WorkspaceID int32           `json:"workspaceId"`
	CategoryID  int32           `json:"categoryId"`
	Year        int             `json:"year"`
	Month       int             `json:"month"`
	Amount      decimal.Decimal `json:"amount"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

type BudgetCategoryWithAllocation struct {
	CategoryID   int32           `json:"categoryId"`
	CategoryName string          `json:"categoryName"`
	Allocated    decimal.Decimal `json:"allocated"`
}

// BudgetStatus represents the budget health status
type BudgetStatus string

const (
	BudgetStatusHealthy BudgetStatus = "healthy" // < 80%
	BudgetStatusWarning BudgetStatus = "warning" // 80-100%
	BudgetStatusOver    BudgetStatus = "over"    // > 100%
)

// BudgetProgress represents a category's budget with spending progress
type BudgetProgress struct {
	CategoryID   int32           `json:"categoryId"`
	CategoryName string          `json:"categoryName"`
	Allocated    decimal.Decimal `json:"allocated"`
	Spent        decimal.Decimal `json:"spent"`
	Remaining    decimal.Decimal `json:"remaining"`
	Percentage   decimal.Decimal `json:"percentage"` // 0-100+
	Status       BudgetStatus    `json:"status"`
}

// MonthlyBudgetSummary contains budget progress for all categories in a month
type MonthlyBudgetSummary struct {
	Year                    int               `json:"year"`
	Month                   int               `json:"month"`
	TotalAllocated          decimal.Decimal   `json:"totalAllocated"`
	TotalSpent              decimal.Decimal   `json:"totalSpent"`
	TotalRemaining          decimal.Decimal   `json:"totalRemaining"`
	Categories              []*BudgetProgress `json:"categories"`
	Initialized             bool              `json:"initialized"`
	CopiedFromPreviousMonth bool              `json:"copiedFromPreviousMonth"`
	IsHistorical            bool              `json:"isHistorical"`
}

// CategorySpending represents spending for a single category
type CategorySpending struct {
	CategoryID int32           `json:"categoryId"`
	Spent      decimal.Decimal `json:"spent"`
}

// CategoryTransaction represents a transaction for budget tracking
type CategoryTransaction struct {
	ID              int32           `json:"id"`
	Name            string          `json:"name"`
	Amount          decimal.Decimal `json:"amount"`
	TransactionDate string          `json:"transactionDate"`
	AccountName     string          `json:"accountName"`
}

// CategoryTransactionsResponse contains transactions for a specific category
type CategoryTransactionsResponse struct {
	CategoryID   int32                  `json:"categoryId"`
	CategoryName string                 `json:"categoryName"`
	Transactions []*CategoryTransaction `json:"transactions"`
}

type BudgetAllocationRepository interface {
	Upsert(allocation *BudgetAllocation) (*BudgetAllocation, error)
	UpsertBatch(allocations []*BudgetAllocation) error
	GetByMonth(workspaceID int32, year, month int) ([]*BudgetAllocation, error)
	GetByCategory(workspaceID int32, categoryID int32, year, month int) (*BudgetAllocation, error)
	Delete(workspaceID int32, categoryID int32, year, month int) error
	GetCategoriesWithAllocations(workspaceID int32, year, month int) ([]*BudgetCategoryWithAllocation, error)
	GetSpendingByCategory(workspaceID int32, year, month int) ([]*CategorySpending, error)
	GetCategoryTransactions(workspaceID int32, categoryID int32, year, month int) ([]*CategoryTransaction, error)
	CountAllocationsForMonth(workspaceID int32, year, month int) (int64, error)
	CopyAllocationsToMonth(workspaceID int32, fromYear, fromMonth, toYear, toMonth int) error
}
