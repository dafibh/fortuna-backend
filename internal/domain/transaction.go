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

// CCState represents the lifecycle state of a credit card transaction
// This is a computed/virtual state derived from billedAt and isPaid:
// - pending: billedAt IS NULL AND isPaid = false
// - billed: billedAt IS NOT NULL AND isPaid = false
// - settled: isPaid = true
type CCState string

const (
	CCStatePending CCState = "pending"
	CCStateBilled  CCState = "billed"
	CCStateSettled CCState = "settled"
)

// ComputeCCState derives the CC state from isPaid and billedAt
func ComputeCCState(isPaid bool, billedAt *time.Time) *CCState {
	if isPaid {
		state := CCStateSettled
		return &state
	}
	if billedAt != nil {
		state := CCStateBilled
		return &state
	}
	state := CCStatePending
	return &state
}

// SettlementIntent represents when a CC transaction should be settled
type SettlementIntent string

const (
	SettlementIntentImmediate SettlementIntent = "immediate"
	SettlementIntentDeferred  SettlementIntent = "deferred"
)

type Transaction struct {
	ID              int32           `json:"id"`
	WorkspaceID     int32           `json:"workspaceId"`
	AccountID       int32           `json:"accountId"`
	Name            string          `json:"name"`
	Amount          decimal.Decimal `json:"amount"`
	Type            TransactionType `json:"type"`
	TransactionDate time.Time       `json:"transactionDate"`
	IsPaid          bool            `json:"isPaid"`
	Notes           *string         `json:"notes,omitempty"`
	TransferPairID  *uuid.UUID      `json:"transferPairId,omitempty"`
	CategoryID      *int32          `json:"categoryId,omitempty"`
	CategoryName    *string         `json:"categoryName,omitempty"`
	IsCCPayment     bool            `json:"isCcPayment"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
	DeletedAt       *time.Time      `json:"deletedAt,omitempty"`

	// CC Lifecycle
	CCState          *CCState          `json:"ccState"`          // Computed: derived from billedAt and isPaid
	BilledAt         *time.Time        `json:"billedAt"`         // when marked as billed (null = pending)
	SettlementIntent *SettlementIntent `json:"settlementIntent"` // 'immediate' | 'deferred' | null

	// Recurring/Projection
	Source      string `json:"source"`      // 'manual' | 'recurring'
	TemplateID  *int32 `json:"templateId"`  // FK to recurring_templates, nullable
	IsProjected bool   `json:"isProjected"` // true = future projection
	IsModified  bool   `json:"isModified"`  // true if projected instance differs from template

	// Loan Integration (v2)
	LoanID *int32 `json:"loanId"` // FK to loans, nullable

	// Transaction Grouping
	GroupID   *int32  `json:"groupId"`
	GroupName *string `json:"groupName,omitempty"`
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
	CCStatus  *CCState // Filter by cc_state (pending, billed, settled)
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
	Name            string
	Amount          decimal.Decimal
	Type            TransactionType
	TransactionDate time.Time
	AccountID       int32
	Notes           *string
	CategoryID      *int32
	// CC Lifecycle
	IsPaid           bool // For settlement status
	BilledAt         *time.Time
	SettlementIntent *SettlementIntent
	// Recurring/Projection
	Source      string
	TemplateID  *int32
	IsProjected bool
}

// TransactionSummary holds aggregated transaction data for balance calculations
type TransactionSummary struct {
	AccountID         int32
	SumIncome         decimal.Decimal
	SumExpenses       decimal.Decimal // Paid expenses only (for regular accounts)
	SumUnpaidExpenses decimal.Decimal
	SumAllExpenses    decimal.Decimal // All expenses regardless of isPaid (for CC accounts)
}

// MonthlyTransactionSummary holds income/expense totals for a specific month
type MonthlyTransactionSummary struct {
	Year          int
	Month         int
	TotalIncome   decimal.Decimal
	TotalExpenses decimal.Decimal
}

// RecentCategory holds recently used category info for suggestions
type RecentCategory struct {
	ID       int32     `json:"id"`
	Name     string    `json:"name"`
	LastUsed time.Time `json:"lastUsed"`
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

// OverdueGroup groups overdue CC transactions by month
type OverdueGroup struct {
	Month         string          `json:"month"`         // "2025-11"
	MonthLabel    string          `json:"monthLabel"`    // "November"
	MonthsOverdue int             `json:"monthsOverdue"` // Number of months overdue
	TotalAmount   decimal.Decimal `json:"totalAmount"`
	ItemCount     int             `json:"itemCount"`
	Transactions  []*Transaction  `json:"transactions"`
}

// LoanTransactionStats holds paid/unpaid transaction counts for loan deletion confirmation
type LoanTransactionStats struct {
	PaidCount    int32           `json:"paidCount"`
	UnpaidCount  int32           `json:"unpaidCount"`
	PaidTotal    decimal.Decimal `json:"paidTotal"`
	UnpaidTotal  decimal.Decimal `json:"unpaidTotal"`
}

// LoanTrendDataRow represents aggregated loan transaction data for trend visualization
type LoanTrendDataRow struct {
	Year         int32           `json:"year"`
	Month        int32           `json:"month"`
	ProviderID   int32           `json:"providerId"`
	ProviderName string          `json:"providerName"`
	TotalAmount  decimal.Decimal `json:"totalAmount"`
	AllPaid      bool            `json:"allPaid"`
}

type TransactionRepository interface {
	Create(transaction *Transaction) (*Transaction, error)
	CreateBatchTx(tx interface{}, transactions []*Transaction) ([]*Transaction, error) // Batch create within DB transaction
	GetByID(workspaceID int32, id int32) (*Transaction, error)
	GetByWorkspace(workspaceID int32, filters *TransactionFilters) (*PaginatedTransactions, error)
	TogglePaid(workspaceID int32, id int32) (*Transaction, error)
	Update(workspaceID int32, id int32, data *UpdateTransactionData) (*Transaction, error)
	SoftDelete(workspaceID int32, id int32) error
	CreateTransferPair(fromTx, toTx *Transaction) (*TransferResult, error)
	SoftDeleteTransferPair(workspaceID int32, pairID uuid.UUID) error
	GetAccountTransactionSummaries(workspaceID int32) ([]*TransactionSummary, error)
	SumByTypeAndDateRange(workspaceID int32, startDate, endDate time.Time, txType TransactionType) (decimal.Decimal, error)
	GetMonthlyTransactionSummaries(workspaceID int32) ([]*MonthlyTransactionSummary, error)
	SumPaidExpensesByDateRange(workspaceID int32, startDate, endDate time.Time) (decimal.Decimal, error)
	SumUnpaidExpensesByDateRange(workspaceID int32, startDate, endDate time.Time) (decimal.Decimal, error)
	SumUnpaidExpensesForDisposable(workspaceID int32, startDate, endDate time.Time) (decimal.Decimal, error)
	SumDeferredCCByDateRange(workspaceID int32, startDate, endDate time.Time) (decimal.Decimal, error)
	GetRecentlyUsedCategories(workspaceID int32) ([]*RecentCategory, error)
	GetCCMetrics(workspaceID int32, startDate, endDate time.Time) (*CCMetrics, error)
	BatchToggleToBilled(workspaceID int32, ids []int32) ([]*Transaction, error)

	// Projection management
	GetProjectionsByTemplate(workspaceID int32, templateID int32) ([]*Transaction, error)
	DeleteProjectionsByTemplate(workspaceID int32, templateID int32) error
	DeleteProjectionsBeyondDate(workspaceID int32, templateID int32, date time.Time) error
	OrphanActualsByTemplate(workspaceID int32, templateID int32) error

	// Settlement operations
	GetByIDs(workspaceID int32, ids []int32) ([]*Transaction, error)
	BulkSettle(workspaceID int32, ids []int32) ([]*Transaction, error)
	GetDeferredForSettlement(workspaceID int32) ([]*Transaction, error)
	GetImmediateForSettlement(workspaceID int32, startDate, endDate time.Time) ([]*Transaction, error)
	GetPendingDeferredCC(workspaceID int32, startDate, endDate time.Time) ([]*Transaction, error)

	// AtomicSettle creates a transfer pair (expense and income) and settles CC transactions atomically
	// within a single database transaction. If any operation fails, all changes are rolled back.
	AtomicSettle(fromTx, toTx *Transaction, settleIDs []int32) (*Transaction, int, error)

	// Overdue detection
	GetOverdueCC(workspaceID int32) ([]*Transaction, error)

	// Aggregation operations (no pagination)
	GetByDateRangeForAggregation(workspaceID int32, startDate, endDate time.Time) ([]*Transaction, error)

	// Loan transaction operations (CL v2)
	GetLoanTransactionsByMonth(workspaceID int32, loanID int32, year, month int) ([]*Transaction, error)
	BulkMarkPaid(workspaceID int32, ids []int32) ([]*Transaction, error)
	// Get all transactions for a loan (for item-based modal)
	GetByLoanID(workspaceID int32, loanID int32) ([]*Transaction, error)
	// Loan deletion operations - orphan paid, delete unpaid
	OrphanPaidTransactionsByLoan(workspaceID int32, loanID int32) error
	DeleteUnpaidTransactionsByLoan(workspaceID int32, loanID int32) error
	GetLoanTransactionStats(workspaceID int32, loanID int32) (*LoanTransactionStats, error)
	// Loan edit cascade operations
	UpdatePayeesByLoan(workspaceID int32, loanID int32, newPayee string) (int64, error)
	HasPaidTransactionsByLoan(workspaceID int32, loanID int32) (bool, error)
	// Loan trend data aggregation
	GetLoanTrendData(workspaceID int32, startYear, startMonth, endYear, endMonth int32) ([]*LoanTrendDataRow, error)
}
