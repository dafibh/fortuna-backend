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

// CCState represents the lifecycle state of a credit card transaction
type CCState string

const (
	CCStatePending CCState = "pending"
	CCStateBilled  CCState = "billed"
	CCStateSettled CCState = "settled"
)

// SettlementIntent represents when a CC transaction should be settled
type SettlementIntent string

const (
	SettlementIntentImmediate SettlementIntent = "immediate"
	SettlementIntentDeferred  SettlementIntent = "deferred"
)

type Transaction struct {
	ID                     int32               `json:"id"`
	WorkspaceID            int32               `json:"workspaceId"`
	AccountID              int32               `json:"accountId"`
	Name                   string              `json:"name"`
	Amount                 decimal.Decimal     `json:"amount"`
	Type                   TransactionType     `json:"type"`
	TransactionDate        time.Time           `json:"transactionDate"`
	IsPaid                 bool                `json:"isPaid"`
	CCSettlementIntent     *CCSettlementIntent `json:"ccSettlementIntent,omitempty"`
	Notes                  *string             `json:"notes,omitempty"`
	TransferPairID         *uuid.UUID          `json:"transferPairId,omitempty"`
	CategoryID             *int32              `json:"categoryId,omitempty"`
	CategoryName           *string             `json:"categoryName,omitempty"`
	IsCCPayment            bool                `json:"isCcPayment"`
	RecurringTransactionID *int32              `json:"recurringTransactionId,omitempty"`
	CreatedAt              time.Time           `json:"createdAt"`
	UpdatedAt              time.Time           `json:"updatedAt"`
	DeletedAt              *time.Time          `json:"deletedAt,omitempty"`

	// CC Lifecycle (v2)
	CCState          *CCState          `json:"ccState"`          // 'pending' | 'billed' | 'settled' | null
	BilledAt         *time.Time        `json:"billedAt"`         // when marked as billed
	SettledAt        *time.Time        `json:"settledAt"`        // when settlement completed
	SettlementIntent *SettlementIntent `json:"settlementIntent"` // 'immediate' | 'deferred' | null

	// Recurring/Projection (v2)
	Source      string `json:"source"`      // 'manual' | 'recurring'
	TemplateID  *int32 `json:"templateId"`  // FK to recurring_templates, nullable
	IsProjected bool   `json:"isProjected"` // true = future projection
	IsModified  bool   `json:"isModified"`  // true if projected instance differs from template
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
	// CC Lifecycle (v2)
	CCState          *CCState
	BilledAt         *time.Time
	SettledAt        *time.Time
	SettlementIntent *SettlementIntent
	// Recurring/Projection (v2)
	Source      string
	TemplateID  *int32
	IsProjected bool
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

// CreateCCPaymentRequest represents a request to create a CC payment transaction
type CreateCCPaymentRequest struct {
	CCAccountID     int32           `json:"ccAccountId" validate:"required"`
	Amount          decimal.Decimal `json:"amount" validate:"required"`
	TransactionDate time.Time       `json:"transactionDate" validate:"required"`
	SourceAccountID *int32          `json:"sourceAccountId,omitempty"` // Optional bank account
	Notes           string          `json:"notes,omitempty"`
}

// CCPaymentResponse represents the response after creating a CC payment
type CCPaymentResponse struct {
	CCTransaction     *Transaction `json:"ccTransaction"`
	SourceTransaction *Transaction `json:"sourceTransaction,omitempty"` // Only if source provided
}

// CCMetrics holds aggregated CC transaction metrics for dashboard display
type CCMetrics struct {
	Pending     decimal.Decimal `json:"pending"`     // Sum of pending CC transactions
	Outstanding decimal.Decimal `json:"outstanding"` // Sum of billed CC transactions with deferred intent (balance to settle)
	Purchases   decimal.Decimal `json:"purchases"`   // Sum of all CC transactions (pending + billed + settled)
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
	GetCCMetrics(workspaceID int32, startDate, endDate time.Time) (*CCMetrics, error)
	BatchToggleToBilled(workspaceID int32, ids []int32) ([]*Transaction, error)

	// Projection management (v2)
	GetProjectionsByTemplate(workspaceID int32, templateID int32) ([]*Transaction, error)
	DeleteProjectionsByTemplate(workspaceID int32, templateID int32) error
	DeleteProjectionsBeyondDate(workspaceID int32, templateID int32, date time.Time) error
	OrphanActualsByTemplate(workspaceID int32, templateID int32) error
}
