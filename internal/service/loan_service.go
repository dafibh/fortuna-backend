package service

import (
	"strings"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/shopspring/decimal"
)

// LoanService handles loan business logic
type LoanService struct {
	loanRepo     domain.LoanRepository
	providerRepo domain.LoanProviderRepository
}

// NewLoanService creates a new LoanService
func NewLoanService(loanRepo domain.LoanRepository, providerRepo domain.LoanProviderRepository) *LoanService {
	return &LoanService{
		loanRepo:     loanRepo,
		providerRepo: providerRepo,
	}
}

// CreateLoanInput contains input for creating a loan
type CreateLoanInput struct {
	ProviderID   int32
	ItemName     string
	TotalAmount  decimal.Decimal
	NumMonths    int32
	PurchaseDate time.Time
	InterestRate *decimal.Decimal // Optional override, uses provider default if nil
	Notes        *string
}

// CreateLoan creates a new loan with calculated values
func (s *LoanService) CreateLoan(workspaceID int32, input CreateLoanInput) (*domain.Loan, error) {
	// Validate item name
	itemName := strings.TrimSpace(input.ItemName)
	if itemName == "" {
		return nil, domain.ErrLoanItemNameEmpty
	}
	if len(itemName) > 200 {
		return nil, domain.ErrLoanItemNameTooLong
	}

	// Validate amount
	if input.TotalAmount.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrLoanAmountInvalid
	}

	// Validate months
	if input.NumMonths < 1 {
		return nil, domain.ErrLoanMonthsInvalid
	}

	// Validate provider exists
	if input.ProviderID <= 0 {
		return nil, domain.ErrLoanProviderInvalid
	}

	provider, err := s.providerRepo.GetByID(workspaceID, input.ProviderID)
	if err != nil {
		if err == domain.ErrLoanProviderNotFound {
			return nil, domain.ErrLoanProviderInvalid
		}
		return nil, err
	}

	// Use provided interest rate or default from provider
	interestRate := provider.DefaultInterestRate
	if input.InterestRate != nil {
		interestRate = *input.InterestRate
	}

	// Calculate monthly payment
	monthlyPayment := CalculateMonthlyPayment(input.TotalAmount, interestRate, int(input.NumMonths))

	// Calculate first payment month based on cutoff day
	firstPaymentYear, firstPaymentMonth := CalculateFirstPaymentMonth(input.PurchaseDate, int(provider.CutoffDay))

	loan := &domain.Loan{
		WorkspaceID:       workspaceID,
		ProviderID:        input.ProviderID,
		ItemName:          itemName,
		TotalAmount:       input.TotalAmount,
		NumMonths:         input.NumMonths,
		PurchaseDate:      input.PurchaseDate,
		InterestRate:      interestRate,
		MonthlyPayment:    monthlyPayment,
		FirstPaymentYear:  int32(firstPaymentYear),
		FirstPaymentMonth: int32(firstPaymentMonth),
		Notes:             input.Notes,
	}

	return s.loanRepo.Create(loan)
}

// PreviewLoanInput contains input for previewing loan calculations
type PreviewLoanInput struct {
	ProviderID   int32
	TotalAmount  decimal.Decimal
	NumMonths    int32
	PurchaseDate time.Time
	InterestRate *decimal.Decimal // Optional override, uses provider default if nil
}

// PreviewLoanResult contains the calculated values for a loan
type PreviewLoanResult struct {
	MonthlyPayment    decimal.Decimal
	FirstPaymentYear  int
	FirstPaymentMonth int
	InterestRate      decimal.Decimal
}

// PreviewLoan calculates loan values without creating the loan
func (s *LoanService) PreviewLoan(workspaceID int32, input PreviewLoanInput) (*PreviewLoanResult, error) {
	// Validate provider exists
	if input.ProviderID <= 0 {
		return nil, domain.ErrLoanProviderInvalid
	}

	provider, err := s.providerRepo.GetByID(workspaceID, input.ProviderID)
	if err != nil {
		if err == domain.ErrLoanProviderNotFound {
			return nil, domain.ErrLoanProviderInvalid
		}
		return nil, err
	}

	// Validate amount
	if input.TotalAmount.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrLoanAmountInvalid
	}

	// Validate months
	if input.NumMonths < 1 {
		return nil, domain.ErrLoanMonthsInvalid
	}

	// Use provided interest rate or default from provider
	interestRate := provider.DefaultInterestRate
	if input.InterestRate != nil {
		interestRate = *input.InterestRate
	}

	// Calculate monthly payment
	monthlyPayment := CalculateMonthlyPayment(input.TotalAmount, interestRate, int(input.NumMonths))

	// Calculate first payment month based on cutoff day
	firstPaymentYear, firstPaymentMonth := CalculateFirstPaymentMonth(input.PurchaseDate, int(provider.CutoffDay))

	return &PreviewLoanResult{
		MonthlyPayment:    monthlyPayment,
		FirstPaymentYear:  firstPaymentYear,
		FirstPaymentMonth: firstPaymentMonth,
		InterestRate:      interestRate,
	}, nil
}

// GetLoans retrieves all loans for a workspace
func (s *LoanService) GetLoans(workspaceID int32) ([]*domain.Loan, error) {
	return s.loanRepo.GetAllByWorkspace(workspaceID)
}

// GetActiveLoans retrieves active loans for a workspace
func (s *LoanService) GetActiveLoans(workspaceID int32, currentYear, currentMonth int) ([]*domain.Loan, error) {
	return s.loanRepo.GetActiveByWorkspace(workspaceID, currentYear, currentMonth)
}

// GetCompletedLoans retrieves completed loans for a workspace
func (s *LoanService) GetCompletedLoans(workspaceID int32, currentYear, currentMonth int) ([]*domain.Loan, error) {
	return s.loanRepo.GetCompletedByWorkspace(workspaceID, currentYear, currentMonth)
}

// GetLoanByID retrieves a loan by ID within a workspace
func (s *LoanService) GetLoanByID(workspaceID int32, id int32) (*domain.Loan, error) {
	return s.loanRepo.GetByID(workspaceID, id)
}

// DeleteLoan soft-deletes a loan
func (s *LoanService) DeleteLoan(workspaceID int32, id int32) error {
	// Verify loan exists before deleting
	_, err := s.loanRepo.GetByID(workspaceID, id)
	if err != nil {
		return err
	}
	return s.loanRepo.SoftDelete(workspaceID, id)
}

// CalculateMonthlyPayment calculates the monthly payment for a loan
// Formula: (totalAmount * (1 + interestRate/100)) / numMonths
func CalculateMonthlyPayment(totalAmount, interestRate decimal.Decimal, numMonths int) decimal.Decimal {
	if numMonths <= 0 {
		return decimal.Zero
	}
	multiplier := decimal.NewFromInt(1).Add(interestRate.Div(decimal.NewFromInt(100)))
	totalWithInterest := totalAmount.Mul(multiplier)
	return totalWithInterest.Div(decimal.NewFromInt(int64(numMonths))).Round(2)
}

// CalculateFirstPaymentMonth calculates the first payment year and month based on purchase date and cutoff day
// If purchase day < cutoff day → first payment in current month
// If purchase day >= cutoff day → first payment in next month
func CalculateFirstPaymentMonth(purchaseDate time.Time, cutoffDay int) (year, month int) {
	if purchaseDate.Day() < cutoffDay {
		return purchaseDate.Year(), int(purchaseDate.Month())
	}
	// Next month
	nextMonth := purchaseDate.AddDate(0, 1, 0)
	return nextMonth.Year(), int(nextMonth.Month())
}
