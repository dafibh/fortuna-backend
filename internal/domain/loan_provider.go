package domain

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

// PaymentMode constants for loan provider billing behavior
const (
	PaymentModePerItem            = "per_item"
	PaymentModeConsolidatedMonthly = "consolidated_monthly"
)

var (
	ErrLoanProviderNotFound    = errors.New("loan provider not found")
	ErrLoanProviderHasLoans    = errors.New("loan provider has active loans")
	ErrLoanProviderNameExists  = errors.New("loan provider with this name already exists")
	ErrInvalidCutoffDay        = errors.New("cutoff day must be between 1 and 31")
	ErrInvalidInterestRate     = errors.New("interest rate must be non-negative")
	ErrInterestRateTooHigh     = errors.New("interest rate must be 100% or less")
	ErrLoanProviderNameEmpty   = errors.New("loan provider name is required")
	ErrLoanProviderNameTooLong = errors.New("loan provider name must be 100 characters or less")
	ErrInvalidPaymentMode      = errors.New("payment mode must be 'per_item' or 'consolidated_monthly'")
)

type LoanProvider struct {
	ID                  int32           `json:"id"`
	WorkspaceID         int32           `json:"workspaceId"`
	Name                string          `json:"name"`
	CutoffDay           int32           `json:"cutoffDay"`
	DefaultInterestRate decimal.Decimal `json:"defaultInterestRate"`
	PaymentMode         string          `json:"paymentMode"`
	CreatedAt           time.Time       `json:"createdAt"`
	UpdatedAt           time.Time       `json:"updatedAt"`
	DeletedAt           *time.Time      `json:"deletedAt,omitempty"`
}

func (lp *LoanProvider) Validate() error {
	if lp.Name == "" {
		return ErrLoanProviderNameEmpty
	}
	if lp.CutoffDay < 1 || lp.CutoffDay > 31 {
		return ErrInvalidCutoffDay
	}
	if lp.DefaultInterestRate.LessThan(decimal.Zero) {
		return ErrInvalidInterestRate
	}
	if lp.PaymentMode != "" && !IsValidPaymentMode(lp.PaymentMode) {
		return ErrInvalidPaymentMode
	}
	return nil
}

// IsValidPaymentMode checks if the given payment mode is valid
func IsValidPaymentMode(mode string) bool {
	return mode == PaymentModePerItem || mode == PaymentModeConsolidatedMonthly
}

type LoanProviderRepository interface {
	Create(provider *LoanProvider) (*LoanProvider, error)
	GetByID(workspaceID int32, id int32) (*LoanProvider, error)
	GetAllByWorkspace(workspaceID int32) ([]*LoanProvider, error)
	Update(provider *LoanProvider) (*LoanProvider, error)
	SoftDelete(workspaceID int32, id int32) error
	// HasActiveLoans will be implemented when loans table exists (Story 7-2)
	// HasActiveLoans(workspaceID int32, id int32) (bool, error)
}
