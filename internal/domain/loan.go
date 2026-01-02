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
}
