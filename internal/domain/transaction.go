package domain

import (
	"time"

	"github.com/google/uuid"
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
	TransferPairID     *uuid.UUID          `json:"transferPairId,omitempty"`
	CategoryID         *int32              `json:"categoryId,omitempty"`
	CategoryName       *string             `json:"categoryName,omitempty"`
	CreatedAt          time.Time           `json:"createdAt"`
	UpdatedAt          time.Time           `json:"updatedAt"`
	DeletedAt          *time.Time          `json:"deletedAt,omitempty"`
}

// TransferResult represents the result of creating a transfer
type TransferResult struct {
	FromTransaction *Transaction `json:"fromTransaction"`
	ToTransaction   *Transaction `json:"toTransaction"`
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

type UpdateTransactionData struct {
	Name               string
	Amount             decimal.Decimal
	Type               TransactionType
	TransactionDate    time.Time
	AccountID          int32
	CCSettlementIntent *CCSettlementIntent
	Notes              *string
	CategoryID         *int32
}

// TransactionSummary holds aggregated transaction data for balance calculations
type TransactionSummary struct {
	AccountID         int32
	SumIncome         decimal.Decimal
	SumExpenses       decimal.Decimal
	SumUnpaidExpenses decimal.Decimal
}

// MonthlyTransactionSummary holds income/expense totals for a specific month
type MonthlyTransactionSummary struct {
	Year          int
	Month         int
	TotalIncome   decimal.Decimal
	TotalExpenses decimal.Decimal
}

// CCPayableSummaryRow holds settlement intent and total for CC payables query
type CCPayableSummaryRow struct {
	SettlementIntent CCSettlementIntent
	Total            decimal.Decimal
}

// RecentCategory holds recently used category info for suggestions
type RecentCategory struct {
	ID       int32     `json:"id"`
	Name     string    `json:"name"`
	LastUsed time.Time `json:"lastUsed"`
}

// CCPayableTransaction represents a single CC transaction in the payable breakdown
type CCPayableTransaction struct {
	ID               int32              `json:"id"`
	Name             string             `json:"name"`
	Amount           decimal.Decimal    `json:"amount"`
	TransactionDate  time.Time          `json:"transactionDate"`
	SettlementIntent CCSettlementIntent `json:"settlementIntent"`
	AccountID        int32              `json:"accountId"`
	AccountName      string             `json:"accountName"`
}

// CCPayableByAccount groups transactions by account for the payable breakdown
type CCPayableByAccount struct {
	AccountID    int32                  `json:"accountId"`
	AccountName  string                 `json:"accountName"`
	Total        decimal.Decimal        `json:"total"`
	Transactions []CCPayableTransaction `json:"transactions"`
}

// CCPayableBreakdown contains the full payable breakdown by settlement intent
type CCPayableBreakdown struct {
	ThisMonth      []CCPayableByAccount `json:"thisMonth"`
	NextMonth      []CCPayableByAccount `json:"nextMonth"`
	ThisMonthTotal decimal.Decimal      `json:"thisMonthTotal"`
	NextMonthTotal decimal.Decimal      `json:"nextMonthTotal"`
	GrandTotal     decimal.Decimal      `json:"grandTotal"`
}

type TransactionRepository interface {
	Create(transaction *Transaction) (*Transaction, error)
	GetByID(workspaceID int32, id int32) (*Transaction, error)
	GetByWorkspace(workspaceID int32, filters *TransactionFilters) (*PaginatedTransactions, error)
	TogglePaid(workspaceID int32, id int32) (*Transaction, error)
	UpdateSettlementIntent(workspaceID int32, id int32, intent CCSettlementIntent) (*Transaction, error)
	Update(workspaceID int32, id int32, data *UpdateTransactionData) (*Transaction, error)
	SoftDelete(workspaceID int32, id int32) error
	CreateTransferPair(fromTx, toTx *Transaction) (*TransferResult, error)
	SoftDeleteTransferPair(workspaceID int32, pairID uuid.UUID) error
	GetAccountTransactionSummaries(workspaceID int32) ([]*TransactionSummary, error)
	SumByTypeAndDateRange(workspaceID int32, startDate, endDate time.Time, txType TransactionType) (decimal.Decimal, error)
	GetMonthlyTransactionSummaries(workspaceID int32) ([]*MonthlyTransactionSummary, error)
	SumPaidExpensesByDateRange(workspaceID int32, startDate, endDate time.Time) (decimal.Decimal, error)
	SumUnpaidExpensesByDateRange(workspaceID int32, startDate, endDate time.Time) (decimal.Decimal, error)
	GetCCPayableSummary(workspaceID int32) ([]*CCPayableSummaryRow, error)
	GetRecentlyUsedCategories(workspaceID int32) ([]*RecentCategory, error)
	GetCCPayableBreakdown(workspaceID int32) ([]*CCPayableTransaction, error)
}
