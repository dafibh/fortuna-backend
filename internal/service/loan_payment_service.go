package service

import (
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/shopspring/decimal"
)

// LoanPaymentService handles loan payment business logic
type LoanPaymentService struct {
	paymentRepo domain.LoanPaymentRepository
	loanRepo    domain.LoanRepository
}

// NewLoanPaymentService creates a new LoanPaymentService
func NewLoanPaymentService(paymentRepo domain.LoanPaymentRepository, loanRepo domain.LoanRepository) *LoanPaymentService {
	return &LoanPaymentService{
		paymentRepo: paymentRepo,
		loanRepo:    loanRepo,
	}
}

// GetPaymentsByLoanID retrieves all payments for a loan, validating workspace ownership
func (s *LoanPaymentService) GetPaymentsByLoanID(workspaceID int32, loanID int32) ([]*domain.LoanPayment, error) {
	// Verify loan belongs to workspace
	_, err := s.loanRepo.GetByID(workspaceID, loanID)
	if err != nil {
		return nil, err
	}

	return s.paymentRepo.GetByLoanID(loanID)
}

// UpdatePaymentAmount updates the amount for a specific payment
func (s *LoanPaymentService) UpdatePaymentAmount(workspaceID int32, loanID int32, paymentID int32, amount decimal.Decimal) (*domain.LoanPayment, error) {
	// Validate amount
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrLoanPaymentAmountInvalid
	}

	// Verify loan belongs to workspace
	_, err := s.loanRepo.GetByID(workspaceID, loanID)
	if err != nil {
		return nil, err
	}

	// Verify payment belongs to loan
	payment, err := s.paymentRepo.GetByID(paymentID)
	if err != nil {
		return nil, err
	}
	if payment.LoanID != loanID {
		return nil, domain.ErrLoanPaymentNotFound
	}

	return s.paymentRepo.UpdateAmount(paymentID, amount)
}

// TogglePaymentPaid toggles the paid status of a payment
// If customPaidDate is provided and paid is true, uses that date; otherwise defaults to current date
func (s *LoanPaymentService) TogglePaymentPaid(workspaceID int32, loanID int32, paymentID int32, paid bool, customPaidDate *time.Time) (*domain.LoanPayment, error) {
	// Verify loan belongs to workspace
	_, err := s.loanRepo.GetByID(workspaceID, loanID)
	if err != nil {
		return nil, err
	}

	// Verify payment belongs to loan
	payment, err := s.paymentRepo.GetByID(paymentID)
	if err != nil {
		return nil, err
	}
	if payment.LoanID != loanID {
		return nil, domain.ErrLoanPaymentNotFound
	}

	var paidDate *time.Time
	if paid {
		if customPaidDate != nil {
			paidDate = customPaidDate
		} else {
			now := time.Now()
			paidDate = &now
		}
	}

	return s.paymentRepo.TogglePaid(paymentID, paid, paidDate)
}

// GetPaymentsByMonth retrieves all loan payments due in a specific month for a workspace
func (s *LoanPaymentService) GetPaymentsByMonth(workspaceID int32, year, month int) ([]*domain.LoanPayment, error) {
	return s.paymentRepo.GetByMonth(workspaceID, year, month)
}

// GetUnpaidPaymentsByMonth retrieves unpaid loan payments due in a specific month
func (s *LoanPaymentService) GetUnpaidPaymentsByMonth(workspaceID int32, year, month int) ([]*domain.LoanPayment, error) {
	return s.paymentRepo.GetUnpaidByMonth(workspaceID, year, month)
}
