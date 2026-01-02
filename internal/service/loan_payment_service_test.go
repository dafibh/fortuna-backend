package service

import (
	"testing"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestGetPaymentsByLoanID_Success(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(paymentRepo, loanRepo)

	workspaceID := int32(1)
	loanID := int32(10)

	// Setup loan exists check
	loanRepo.Loans[loanID] = &domain.Loan{
		ID:          loanID,
		WorkspaceID: workspaceID,
	}

	// Setup payments
	payments := []*domain.LoanPayment{
		{ID: 1, LoanID: loanID, PaymentNumber: 1, Amount: decimal.NewFromInt(100)},
		{ID: 2, LoanID: loanID, PaymentNumber: 2, Amount: decimal.NewFromInt(100)},
	}
	paymentRepo.ByLoanID[loanID] = payments

	result, err := svc.GetPaymentsByLoanID(workspaceID, loanID)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestGetPaymentsByLoanID_LoanNotFound(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(paymentRepo, loanRepo)

	workspaceID := int32(1)
	loanID := int32(999)

	result, err := svc.GetPaymentsByLoanID(workspaceID, loanID)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrLoanNotFound, err)
	assert.Nil(t, result)
}

func TestGetPaymentsByLoanID_WrongWorkspace(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(paymentRepo, loanRepo)

	loanID := int32(10)

	// Loan belongs to workspace 1
	loanRepo.Loans[loanID] = &domain.Loan{
		ID:          loanID,
		WorkspaceID: 1,
	}

	// Try to access with workspace 2
	result, err := svc.GetPaymentsByLoanID(2, loanID)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrLoanNotFound, err)
	assert.Nil(t, result)
}

func TestUpdatePaymentAmount_Success(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(paymentRepo, loanRepo)

	workspaceID := int32(1)
	loanID := int32(10)
	paymentID := int32(100)
	newAmount := decimal.NewFromInt(150)

	// Setup loan
	loanRepo.Loans[loanID] = &domain.Loan{
		ID:          loanID,
		WorkspaceID: workspaceID,
	}

	// Setup payment
	paymentRepo.Payments[paymentID] = &domain.LoanPayment{
		ID:     paymentID,
		LoanID: loanID,
		Amount: decimal.NewFromInt(100),
	}

	result, err := svc.UpdatePaymentAmount(workspaceID, loanID, paymentID, newAmount)
	assert.NoError(t, err)
	assert.Equal(t, newAmount, result.Amount)
}

func TestUpdatePaymentAmount_InvalidAmount(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(paymentRepo, loanRepo)

	result, err := svc.UpdatePaymentAmount(1, 10, 100, decimal.Zero)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrLoanPaymentAmountInvalid, err)
	assert.Nil(t, result)
}

func TestUpdatePaymentAmount_LoanNotFound(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(paymentRepo, loanRepo)

	result, err := svc.UpdatePaymentAmount(1, 999, 100, decimal.NewFromInt(150))
	assert.Error(t, err)
	assert.Equal(t, domain.ErrLoanNotFound, err)
	assert.Nil(t, result)
}

func TestUpdatePaymentAmount_PaymentNotFound(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(paymentRepo, loanRepo)

	workspaceID := int32(1)
	loanID := int32(10)

	// Setup loan
	loanRepo.Loans[loanID] = &domain.Loan{
		ID:          loanID,
		WorkspaceID: workspaceID,
	}

	result, err := svc.UpdatePaymentAmount(workspaceID, loanID, 999, decimal.NewFromInt(150))
	assert.Error(t, err)
	assert.Equal(t, domain.ErrLoanPaymentNotFound, err)
	assert.Nil(t, result)
}

func TestUpdatePaymentAmount_PaymentWrongLoan(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(paymentRepo, loanRepo)

	workspaceID := int32(1)
	loanID := int32(10)
	paymentID := int32(100)

	// Setup loan
	loanRepo.Loans[loanID] = &domain.Loan{
		ID:          loanID,
		WorkspaceID: workspaceID,
	}

	// Setup payment belonging to different loan
	paymentRepo.Payments[paymentID] = &domain.LoanPayment{
		ID:     paymentID,
		LoanID: 999, // Different loan
		Amount: decimal.NewFromInt(100),
	}

	result, err := svc.UpdatePaymentAmount(workspaceID, loanID, paymentID, decimal.NewFromInt(150))
	assert.Error(t, err)
	assert.Equal(t, domain.ErrLoanPaymentNotFound, err)
	assert.Nil(t, result)
}

func TestTogglePaymentPaid_MarkAsPaid(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(paymentRepo, loanRepo)

	workspaceID := int32(1)
	loanID := int32(10)
	paymentID := int32(100)

	// Setup loan
	loanRepo.Loans[loanID] = &domain.Loan{
		ID:          loanID,
		WorkspaceID: workspaceID,
	}

	// Setup payment
	paymentRepo.Payments[paymentID] = &domain.LoanPayment{
		ID:     paymentID,
		LoanID: loanID,
		Paid:   false,
	}

	result, err := svc.TogglePaymentPaid(workspaceID, loanID, paymentID, true)
	assert.NoError(t, err)
	assert.True(t, result.Paid)
	assert.NotNil(t, result.PaidDate)
}

func TestTogglePaymentPaid_MarkAsUnpaid(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(paymentRepo, loanRepo)

	workspaceID := int32(1)
	loanID := int32(10)
	paymentID := int32(100)

	// Setup loan
	loanRepo.Loans[loanID] = &domain.Loan{
		ID:          loanID,
		WorkspaceID: workspaceID,
	}

	// Setup payment as paid
	now := time.Now()
	paymentRepo.Payments[paymentID] = &domain.LoanPayment{
		ID:       paymentID,
		LoanID:   loanID,
		Paid:     true,
		PaidDate: &now,
	}

	result, err := svc.TogglePaymentPaid(workspaceID, loanID, paymentID, false)
	assert.NoError(t, err)
	assert.False(t, result.Paid)
	assert.Nil(t, result.PaidDate)
}

func TestTogglePaymentPaid_LoanNotFound(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(paymentRepo, loanRepo)

	result, err := svc.TogglePaymentPaid(1, 999, 100, true)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrLoanNotFound, err)
	assert.Nil(t, result)
}

func TestTogglePaymentPaid_PaymentNotFound(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(paymentRepo, loanRepo)

	workspaceID := int32(1)
	loanID := int32(10)

	loanRepo.Loans[loanID] = &domain.Loan{
		ID:          loanID,
		WorkspaceID: workspaceID,
	}

	result, err := svc.TogglePaymentPaid(workspaceID, loanID, 999, true)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrLoanPaymentNotFound, err)
	assert.Nil(t, result)
}

func TestGetPaymentsByMonth_Success(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(paymentRepo, loanRepo)

	workspaceID := int32(1)
	year := 2024
	month := 6

	// Setup monthly payments
	payments := []*domain.LoanPayment{
		{ID: 1, DueYear: 2024, DueMonth: 6},
		{ID: 2, DueYear: 2024, DueMonth: 6},
	}
	paymentRepo.SetPaymentsByMonth(workspaceID, year, month, payments)

	result, err := svc.GetPaymentsByMonth(workspaceID, year, month)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestGetUnpaidPaymentsByMonth_Success(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(paymentRepo, loanRepo)

	workspaceID := int32(1)
	year := 2024
	month := 6

	// Setup payments for the month (mix of paid and unpaid)
	payments := []*domain.LoanPayment{
		{ID: 1, DueYear: 2024, DueMonth: 6, Paid: false},
		{ID: 2, DueYear: 2024, DueMonth: 6, Paid: true},
	}
	paymentRepo.SetPaymentsByMonth(workspaceID, year, month, payments)

	result, err := svc.GetUnpaidPaymentsByMonth(workspaceID, year, month)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.False(t, result[0].Paid)
}

func TestTogglePaymentPaid_PaymentWrongLoan(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(paymentRepo, loanRepo)

	workspaceID := int32(1)
	loanID := int32(10)
	paymentID := int32(100)

	// Setup loan
	loanRepo.Loans[loanID] = &domain.Loan{
		ID:          loanID,
		WorkspaceID: workspaceID,
	}

	// Setup payment belonging to different loan
	paymentRepo.Payments[paymentID] = &domain.LoanPayment{
		ID:     paymentID,
		LoanID: 999, // Different loan
		Paid:   false,
	}

	result, err := svc.TogglePaymentPaid(workspaceID, loanID, paymentID, true)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrLoanPaymentNotFound, err)
	assert.Nil(t, result)
}
