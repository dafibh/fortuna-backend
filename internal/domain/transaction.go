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
	// V1 values (deprecated, migrated in 00025)
	CCSettlementThisMonth CCSettlementIntent = "this_month"
	CCSettlementNextMonth CCSettlementIntent = "next_month"

	// V2 values
	CCSettlementImmediate CCSettlementIntent = "immediate"
	CCSettlementDeferred  CCSettlementIntent = "deferred"
)

// CCState represents the lifecycle state of a CC transaction
type CCState string

const (
	CCStatePending CCState = "pending" // Purchased, not yet billed
	CCStateBilled  CCState = "billed"  // Appears in banking app outstanding
	CCStateSettled CCState = "settled" // Paid off
)

// TransactionSource indicates how the transaction was created
type TransactionSource string

const (
	TransactionSourceManual    TransactionSource = "manual"
	TransactionSourceRecurring TransactionSource = "recurring"
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
	IsCCPayment        bool                `json:"isCcPayment"`

	// V2 Fields - Recurring/Projection
	TemplateID  *int32             `json:"templateId,omitempty"`  // Renamed from RecurringTransactionID
	Source      TransactionSource  `json:"source"`                // 'manual' or 'recurring'
	IsProjected bool               `json:"isProjected"`           // true = future projection

	// V2 Fields - CC Lifecycle
	CCState   *CCState   `json:"ccState,omitempty"`   // pending, billed, settled (NULL for non-CC)
	BilledAt  *time.Time `json:"billedAt,omitempty"`  // When marked as billed
	SettledAt *time.Time `json:"settledAt,omitempty"` // When settlement completed

	// Metadata
	AccountName *string    `json:"accountName,omitempty"` // Joined from accounts table
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
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

// =====================================================
// V2 TYPES - CC LIFECYCLE & SETTLEMENT
// =====================================================

// CCMetrics holds CC transaction metrics for a specific month
type CCMetrics struct {
	TotalPurchases decimal.Decimal `json:"totalPurchases"` // All CC transactions (pending + billed + settled)
	Outstanding    decimal.Decimal `json:"outstanding"`    // Billed + deferred, not yet settled
	Pending        decimal.Decimal `json:"pending"`        // Not yet billed
}

// CCTransactionWithAccount includes account info for CC queries
type CCTransactionWithAccount struct {
	Transaction
	OriginYear  int `json:"originYear,omitempty"`
	OriginMonth int `json:"originMonth,omitempty"`
}

// SettlementRequest represents a request to settle CC transactions
type SettlementRequest struct {
	TransactionIDs    []int32 `json:"transactionIds" validate:"required,min=1"`
	SourceAccountID   int32   `json:"sourceAccountId" validate:"required"`
	TargetCCAccountID int32   `json:"targetCcAccountId" validate:"required"`
}

// SettlementResponse represents the result of a settlement operation
type SettlementResponse struct {
	TransferID   int32           `json:"transferId"`
	SettledCount int             `json:"settledCount"`
	TotalAmount  decimal.Decimal `json:"totalAmount"`
	SettledAt    time.Time       `json:"settledAt"`
}

// DeferredGroup represents a group of deferred CC transactions from a specific month
type DeferredGroup struct {
	Year         int                        `json:"year"`
	Month        int                        `json:"month"`
	MonthLabel   string                     `json:"monthLabel"` // e.g., "January 2026"
	Total        decimal.Decimal            `json:"total"`
	ItemCount    int                        `json:"itemCount"`
	IsOverdue    bool                       `json:"isOverdue"`
	Transactions []CCTransactionWithAccount `json:"transactions"`
}

// OverdueSummary represents a summary of overdue CC transactions for the warning banner
type OverdueSummary struct {
	HasOverdue  bool            `json:"hasOverdue"`
	TotalAmount decimal.Decimal `json:"totalAmount"`
	ItemCount   int             `json:"itemCount"`
	Groups      []DeferredGroup `json:"groups"`
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

	// V2 Projection methods
	DeleteProjectionsByTemplateID(workspaceID int32, templateID int32) (int64, error)
	OrphanActualsByTemplateID(workspaceID int32, templateID int32) (int64, error)
	CheckProjectionExists(workspaceID int32, templateID int32, year int, month int) (bool, error)
	DeleteProjectionsBeyondDate(workspaceID int32, templateID int32, endDate time.Time) (int64, error)
	GetProjectionsByTemplateID(workspaceID int32, templateID int32) ([]*Transaction, error)

	// V2 CC Lifecycle methods
	ToggleCCBilled(workspaceID int32, id int32) (*Transaction, error)
	UpdateCCState(workspaceID int32, id int32, state CCState) (*Transaction, error)
	GetPendingCCByMonth(workspaceID int32, year int, month int) ([]*CCTransactionWithAccount, error)
	GetBilledCCByMonth(workspaceID int32, year int, month int) ([]*CCTransactionWithAccount, error)
	GetCCMetricsByMonth(workspaceID int32, year int, month int) (*CCMetrics, error)
	BulkSettleTransactions(workspaceID int32, ids []int32) (int64, error)
	GetTransactionsByIDs(workspaceID int32, ids []int32) ([]*Transaction, error)
	GetDeferredCCByMonth(workspaceID int32) ([]*CCTransactionWithAccount, error)
	GetOverdueCC(workspaceID int32) ([]*CCTransactionWithAccount, error)
	UpdateAmount(workspaceID int32, id int32, amount decimal.Decimal) error
	GetExpensesByDateRange(workspaceID int32, startDate, endDate time.Time) ([]*Transaction, error)
}
