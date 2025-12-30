package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type TransactionType string

const (
	TransactionTypeIncome  TransactionType = "income"
	TransactionTypeExpense TransactionType = "expense"
)

type CCSettlementIntent string

const (
	CCSettlementThisMonth CCSettlementIntent = "this_month"
	CCSettlementNextMonth CCSettlementIntent = "next_month"
)

type Transaction struct {
	ID                 int32               `json:"id"`
	WorkspaceID        int32               `json:"workspaceId"`
	AccountID          int32               `json:"accountId"`
	Name               string              `json:"name"`
	Amount             decimal.Decimal     `json:"amount"`
	Type               TransactionType     `json:"type"`
	TransactionDate    time.Time           `json:"transactionDate"`
	IsPaid             bool                `json:"isPaid"`
	CCSettlementIntent *CCSettlementIntent `json:"ccSettlementIntent,omitempty"`
	Notes              *string             `json:"notes,omitempty"`
	CreatedAt          time.Time           `json:"createdAt"`
	UpdatedAt          time.Time           `json:"updatedAt"`
	DeletedAt          *time.Time          `json:"deletedAt,omitempty"`
}

type TransactionFilters struct {
	AccountID *int32
	StartDate *time.Time
	EndDate   *time.Time
	Type      *TransactionType
	Page      int32
	PageSize  int32
}

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type PaginatedTransactions struct {
	Data       []*Transaction `json:"data"`
	Page       int32          `json:"page"`
	PageSize   int32          `json:"pageSize"`
	TotalItems int64          `json:"totalItems"`
	TotalPages int32          `json:"totalPages"`
}

type TransactionRepository interface {
	Create(transaction *Transaction) (*Transaction, error)
	GetByID(workspaceID int32, id int32) (*Transaction, error)
	GetByWorkspace(workspaceID int32, filters *TransactionFilters) (*PaginatedTransactions, error)
	TogglePaid(workspaceID int32, id int32) (*Transaction, error)
}
