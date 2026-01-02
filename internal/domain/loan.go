package domain

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

var (
	ErrLoanNotFound        = errors.New("loan not found")
	ErrLoanItemNameEmpty   = errors.New("loan item name is required")
	ErrLoanItemNameTooLong = errors.New("loan item name must be 200 characters or less")
	ErrLoanAmountInvalid   = errors.New("loan amount must be positive")
	ErrLoanMonthsInvalid   = errors.New("number of months must be at least 1")
	ErrLoanProviderInvalid = errors.New("loan provider is required")
)

type Loan struct {
	ID                int32           `json:"id"`
	WorkspaceID       int32           `json:"workspaceId"`
	ProviderID        int32           `json:"providerId"`
	ItemName          string          `json:"itemName"`
	TotalAmount       decimal.Decimal `json:"totalAmount"`
	NumMonths         int32           `json:"numMonths"`
	PurchaseDate      time.Time       `json:"purchaseDate"`
	InterestRate      decimal.Decimal `json:"interestRate"`
	MonthlyPayment    decimal.Decimal `json:"monthlyPayment"`
	FirstPaymentYear  int32           `json:"firstPaymentYear"`
	FirstPaymentMonth int32           `json:"firstPaymentMonth"`
	Notes             *string         `json:"notes,omitempty"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
	DeletedAt         *time.Time      `json:"deletedAt,omitempty"`
}

// LoanWithStats includes loan data plus payment statistics
type LoanWithStats struct {
	Loan
	LastPaymentYear  int32           `json:"lastPaymentYear"`
	LastPaymentMonth int32           `json:"lastPaymentMonth"`
	TotalCount       int32           `json:"totalCount"`
	PaidCount        int32           `json:"paidCount"`
	RemainingBalance decimal.Decimal `json:"remainingBalance"`
	Progress         float64         `json:"progress"` // Calculated: paidCount/totalCount * 100
}

// LoanFilter defines the filter options for listing loans
type LoanFilter string

const (
	LoanFilterAll       LoanFilter = "all"
	LoanFilterActive    LoanFilter = "active"    // remaining_balance > 0
	LoanFilterCompleted LoanFilter = "completed" // remaining_balance = 0
)

func (l *Loan) Validate() error {
	if l.ItemName == "" {
		return ErrLoanItemNameEmpty
	}
	if len(l.ItemName) > 200 {
		return ErrLoanItemNameTooLong
	}
	if l.TotalAmount.LessThanOrEqual(decimal.Zero) {
		return ErrLoanAmountInvalid
	}
	if l.NumMonths < 1 {
		return ErrLoanMonthsInvalid
	}
	if l.ProviderID <= 0 {
		return ErrLoanProviderInvalid
	}
	return nil
}

// IsActive returns true if the loan still has remaining payments based on current year/month
func (l *Loan) IsActive(currentYear, currentMonth int) bool {
	lastPaymentYear, lastPaymentMonth := l.GetLastPaymentYearMonth()
	if lastPaymentYear > currentYear {
		return true
	}
	if lastPaymentYear == currentYear && lastPaymentMonth >= currentMonth {
		return true
	}
	return false
}

// GetLastPaymentYearMonth calculates the year and month of the final payment
func (l *Loan) GetLastPaymentYearMonth() (year, month int) {
	// Calculate months to add (num_months - 1 since first payment is already counted)
	monthsToAdd := int(l.NumMonths) - 1
	totalMonths := int(l.FirstPaymentMonth) - 1 + monthsToAdd // 0-indexed
	year = int(l.FirstPaymentYear) + totalMonths/12
	month = (totalMonths % 12) + 1 // Back to 1-indexed
	return
}

type LoanRepository interface {
	Create(loan *Loan) (*Loan, error)
	CreateTx(tx interface{}, loan *Loan) (*Loan, error) // Transactional create
	GetByID(workspaceID int32, id int32) (*Loan, error)
	GetAllByWorkspace(workspaceID int32) ([]*Loan, error)
	GetActiveByWorkspace(workspaceID int32, currentYear, currentMonth int) ([]*Loan, error)
	GetCompletedByWorkspace(workspaceID int32, currentYear, currentMonth int) ([]*Loan, error)
	Update(loan *Loan) (*Loan, error)
	SoftDelete(workspaceID int32, id int32) error
	CountActiveLoansByProvider(workspaceID int32, providerID int32, currentYear, currentMonth int) (int64, error)
	// Stats methods - joins with loan_payments for aggregated data
	GetAllWithStats(workspaceID int32) ([]*LoanWithStats, error)
	GetActiveWithStats(workspaceID int32) ([]*LoanWithStats, error)
	GetCompletedWithStats(workspaceID int32) ([]*LoanWithStats, error)
}
