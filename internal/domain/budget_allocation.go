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

type BudgetAllocationRepository interface {
	Upsert(allocation *BudgetAllocation) (*BudgetAllocation, error)
	UpsertBatch(allocations []*BudgetAllocation) error
	GetByMonth(workspaceID int32, year, month int) ([]*BudgetAllocation, error)
	GetByCategory(workspaceID int32, categoryID int32, year, month int) (*BudgetAllocation, error)
	Delete(workspaceID int32, categoryID int32, year, month int) error
	GetCategoriesWithAllocations(workspaceID int32, year, month int) ([]*BudgetCategoryWithAllocation, error)
}
