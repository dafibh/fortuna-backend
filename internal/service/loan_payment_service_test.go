package service

import (
	"context"
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
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)

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
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)

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
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)

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
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)

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
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)

	result, err := svc.UpdatePaymentAmount(1, 10, 100, decimal.Zero)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrLoanPaymentAmountInvalid, err)
	assert.Nil(t, result)
}

func TestUpdatePaymentAmount_LoanNotFound(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)

	result, err := svc.UpdatePaymentAmount(1, 999, 100, decimal.NewFromInt(150))
	assert.Error(t, err)
	assert.Equal(t, domain.ErrLoanNotFound, err)
	assert.Nil(t, result)
}

func TestUpdatePaymentAmount_PaymentNotFound(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)

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
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)

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
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)

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

	result, err := svc.TogglePaymentPaid(workspaceID, loanID, paymentID, true, nil)
	assert.NoError(t, err)
	assert.True(t, result.Paid)
	assert.NotNil(t, result.PaidDate)
}

func TestTogglePaymentPaid_WithCustomDate(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)

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

	customDate := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	result, err := svc.TogglePaymentPaid(workspaceID, loanID, paymentID, true, &customDate)
	assert.NoError(t, err)
	assert.True(t, result.Paid)
	assert.NotNil(t, result.PaidDate)
	assert.Equal(t, customDate, *result.PaidDate)
}

func TestTogglePaymentPaid_MarkAsUnpaid(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)

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

	result, err := svc.TogglePaymentPaid(workspaceID, loanID, paymentID, false, nil)
	assert.NoError(t, err)
	assert.False(t, result.Paid)
	assert.Nil(t, result.PaidDate)
}

func TestTogglePaymentPaid_LoanNotFound(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)

	result, err := svc.TogglePaymentPaid(1, 999, 100, true, nil)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrLoanNotFound, err)
	assert.Nil(t, result)
}

func TestTogglePaymentPaid_PaymentNotFound(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)

	workspaceID := int32(1)
	loanID := int32(10)

	loanRepo.Loans[loanID] = &domain.Loan{
		ID:          loanID,
		WorkspaceID: workspaceID,
	}

	result, err := svc.TogglePaymentPaid(workspaceID, loanID, 999, true, nil)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrLoanPaymentNotFound, err)
	assert.Nil(t, result)
}

func TestGetPaymentsByMonth_Success(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)

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
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)

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
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)

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

	result, err := svc.TogglePaymentPaid(workspaceID, loanID, paymentID, true, nil)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrLoanPaymentNotFound, err)
	assert.Nil(t, result)
}

// =============================================================================
// PayMonth Tests
// =============================================================================

func TestPayMonth_ProviderNotFound(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, providerRepo)

	ctx := context.Background()
	workspaceID := int32(1)
	providerID := int32(999) // Non-existent provider

	result, err := svc.PayMonth(ctx, workspaceID, providerID, "2026-01", []int32{1, 2, 3})
	assert.Error(t, err)
	assert.Equal(t, domain.ErrLoanProviderNotFound, err)
	assert.Nil(t, result)
}

func TestPayMonth_ProviderNotConsolidated(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, providerRepo)

	ctx := context.Background()
	workspaceID := int32(1)
	providerID := int32(1)

	// Setup provider with per_item mode (not consolidated)
	providerRepo.Providers[providerID] = &domain.LoanProvider{
		ID:          providerID,
		WorkspaceID: workspaceID,
		Name:        "Test Provider",
		PaymentMode: domain.PaymentModePerItem, // Not consolidated
	}

	result, err := svc.PayMonth(ctx, workspaceID, providerID, "2026-01", []int32{1, 2, 3})
	assert.Error(t, err)
	assert.Equal(t, domain.ErrProviderNotConsolidated, err)
	assert.Nil(t, result)
}

func TestPayMonth_InvalidMonthFormat(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, providerRepo)

	ctx := context.Background()
	workspaceID := int32(1)
	providerID := int32(1)

	// Setup provider with consolidated mode
	providerRepo.Providers[providerID] = &domain.LoanProvider{
		ID:          providerID,
		WorkspaceID: workspaceID,
		Name:        "Test Provider",
		PaymentMode: domain.PaymentModeConsolidatedMonthly,
	}

	result, err := svc.PayMonth(ctx, workspaceID, providerID, "invalid-month", []int32{1, 2, 3})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid month format")
	assert.Nil(t, result)
}

func TestPayMonth_NoUnpaidMonths(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, providerRepo)

	ctx := context.Background()
	workspaceID := int32(1)
	providerID := int32(1)

	// Setup provider with consolidated mode
	providerRepo.Providers[providerID] = &domain.LoanProvider{
		ID:          providerID,
		WorkspaceID: workspaceID,
		Name:        "Test Provider",
		PaymentMode: domain.PaymentModeConsolidatedMonthly,
	}

	// No unpaid months (default mock returns nil)
	result, err := svc.PayMonth(ctx, workspaceID, providerID, "2026-01", []int32{1, 2, 3})
	assert.Error(t, err)
	assert.Equal(t, domain.ErrNoUnpaidMonths, err)
	assert.Nil(t, result)
}

func TestPayMonth_SequentialEnforcementViolation(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, providerRepo)

	ctx := context.Background()
	workspaceID := int32(1)
	providerID := int32(1)

	// Setup provider with consolidated mode
	providerRepo.Providers[providerID] = &domain.LoanProvider{
		ID:          providerID,
		WorkspaceID: workspaceID,
		Name:        "Test Provider",
		PaymentMode: domain.PaymentModeConsolidatedMonthly,
	}

	// Setup earliest unpaid month as February
	paymentRepo.GetEarliestUnpaidMonthFn = func(wID int32, pID int32) (*domain.EarliestUnpaidMonth, error) {
		return &domain.EarliestUnpaidMonth{Year: 2026, Month: 2}, nil
	}

	// Try to pay March (should fail - must pay February first)
	result, err := svc.PayMonth(ctx, workspaceID, providerID, "2026-03", []int32{1, 2, 3})
	assert.Error(t, err)

	// Check it's the right error type
	seqErr, ok := err.(domain.ErrMustPayEarlierMonth)
	assert.True(t, ok, "Expected ErrMustPayEarlierMonth error type")
	assert.Equal(t, "2026-02", seqErr.Expected)
	assert.Equal(t, "2026-03", seqErr.Requested)
	assert.Nil(t, result)
}

func TestPayMonth_EmptyPaymentIDs(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, providerRepo)

	ctx := context.Background()
	workspaceID := int32(1)
	providerID := int32(1)

	// Setup provider with consolidated mode
	providerRepo.Providers[providerID] = &domain.LoanProvider{
		ID:          providerID,
		WorkspaceID: workspaceID,
		Name:        "Test Provider",
		PaymentMode: domain.PaymentModeConsolidatedMonthly,
	}

	// Setup earliest unpaid month
	paymentRepo.GetEarliestUnpaidMonthFn = func(wID int32, pID int32) (*domain.EarliestUnpaidMonth, error) {
		return &domain.EarliestUnpaidMonth{Year: 2026, Month: 1}, nil
	}

	// Try to pay with empty payment IDs
	result, err := svc.PayMonth(ctx, workspaceID, providerID, "2026-01", []int32{})
	assert.Error(t, err)
	assert.Equal(t, domain.ErrPaymentIDsInvalid, err)
	assert.Nil(t, result)
}

func TestPayMonth_InvalidPaymentIDs(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, providerRepo)

	ctx := context.Background()
	workspaceID := int32(1)
	providerID := int32(1)

	// Setup provider with consolidated mode
	providerRepo.Providers[providerID] = &domain.LoanProvider{
		ID:          providerID,
		WorkspaceID: workspaceID,
		Name:        "Test Provider",
		PaymentMode: domain.PaymentModeConsolidatedMonthly,
	}

	// Setup earliest unpaid month
	paymentRepo.GetEarliestUnpaidMonthFn = func(wID int32, pID int32) (*domain.EarliestUnpaidMonth, error) {
		return &domain.EarliestUnpaidMonth{Year: 2026, Month: 1}, nil
	}

	// Setup expected payments for January (only IDs 1 and 2)
	paymentRepo.GetUnpaidPaymentsByProviderMonthFn = func(wID int32, pID int32, year int32, month int32) ([]*domain.LoanPayment, error) {
		return []*domain.LoanPayment{
			{ID: 1, LoanID: 10, Amount: decimal.NewFromInt(100)},
			{ID: 2, LoanID: 11, Amount: decimal.NewFromInt(150)},
		}, nil
	}

	// Try to pay with invalid payment ID (3 doesn't exist)
	result, err := svc.PayMonth(ctx, workspaceID, providerID, "2026-01", []int32{1, 2, 3})
	assert.Error(t, err)
	assert.Equal(t, domain.ErrPaymentIDsInvalid, err)
	assert.Nil(t, result)
}

func TestValidatePayMonth_Success(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, providerRepo)

	ctx := context.Background()
	workspaceID := int32(1)
	providerID := int32(1)

	// Setup provider with consolidated mode
	providerRepo.Providers[providerID] = &domain.LoanProvider{
		ID:          providerID,
		WorkspaceID: workspaceID,
		Name:        "Test Provider",
		PaymentMode: domain.PaymentModeConsolidatedMonthly,
	}

	// Setup earliest unpaid month
	paymentRepo.GetEarliestUnpaidMonthFn = func(wID int32, pID int32) (*domain.EarliestUnpaidMonth, error) {
		return &domain.EarliestUnpaidMonth{Year: 2026, Month: 1}, nil
	}

	// Setup expected payments for January
	paymentRepo.GetUnpaidPaymentsByProviderMonthFn = func(wID int32, pID int32, year int32, month int32) ([]*domain.LoanPayment, error) {
		return []*domain.LoanPayment{
			{ID: 1, LoanID: 10, Amount: decimal.NewFromInt(100)},
			{ID: 2, LoanID: 11, Amount: decimal.NewFromInt(150)},
		}, nil
	}

	// Validate with correct payment IDs
	err := svc.ValidatePayMonth(ctx, workspaceID, providerID, "2026-01", []int32{1, 2})
	assert.NoError(t, err)
}

func TestValidatePayMonth_SequentialEnforcementViolation(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, providerRepo)

	ctx := context.Background()
	workspaceID := int32(1)
	providerID := int32(1)

	// Setup provider with consolidated mode
	providerRepo.Providers[providerID] = &domain.LoanProvider{
		ID:          providerID,
		WorkspaceID: workspaceID,
		Name:        "Test Provider",
		PaymentMode: domain.PaymentModeConsolidatedMonthly,
	}

	// Setup earliest unpaid month as February
	paymentRepo.GetEarliestUnpaidMonthFn = func(wID int32, pID int32) (*domain.EarliestUnpaidMonth, error) {
		return &domain.EarliestUnpaidMonth{Year: 2026, Month: 2}, nil
	}

	// Try to validate March (should fail - must pay February first)
	err := svc.ValidatePayMonth(ctx, workspaceID, providerID, "2026-03", []int32{1, 2, 3})
	assert.Error(t, err)

	// Check it's the right error type
	seqErr, ok := err.(domain.ErrMustPayEarlierMonth)
	assert.True(t, ok, "Expected ErrMustPayEarlierMonth error type")
	assert.Equal(t, "2026-02", seqErr.Expected)
	assert.Equal(t, "2026-03", seqErr.Requested)
}

func TestParseMonth_ValidFormats(t *testing.T) {
	tests := []struct {
		input         string
		expectedYear  int
		expectedMonth int
	}{
		{"2026-01", 2026, 1},
		{"2026-12", 2026, 12},
		{"2025-06", 2025, 6},
	}

	for _, tc := range tests {
		year, month, err := parseMonth(tc.input)
		assert.NoError(t, err)
		assert.Equal(t, tc.expectedYear, year)
		assert.Equal(t, tc.expectedMonth, month)
	}
}

func TestParseMonth_InvalidFormats(t *testing.T) {
	tests := []string{
		"invalid",
		"2026",
		"2026-1",  // Single digit month
		"2026-13", // Invalid month
		"2026-00", // Invalid month
		"01-2026", // Wrong order
		"2026/01", // Wrong separator
	}

	for _, input := range tests {
		_, _, err := parseMonth(input)
		assert.Error(t, err, "Expected error for input: %s", input)
	}
}

// =============================================================================
// PayRange Tests
// =============================================================================

func TestPayRange_ProviderNotFound(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, providerRepo)

	ctx := context.Background()
	workspaceID := int32(1)
	providerID := int32(999) // Non-existent provider

	result, err := svc.PayRange(ctx, workspaceID, providerID, "2026-02", "2026-05", []int32{1, 2, 3})
	assert.Error(t, err)
	assert.Equal(t, domain.ErrLoanProviderNotFound, err)
	assert.Nil(t, result)
}

func TestPayRange_ProviderNotConsolidated(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, providerRepo)

	ctx := context.Background()
	workspaceID := int32(1)
	providerID := int32(1)

	// Setup provider with per_item mode (not consolidated)
	providerRepo.Providers[providerID] = &domain.LoanProvider{
		ID:          providerID,
		WorkspaceID: workspaceID,
		Name:        "Test Provider",
		PaymentMode: domain.PaymentModePerItem, // Not consolidated
	}

	result, err := svc.PayRange(ctx, workspaceID, providerID, "2026-02", "2026-05", []int32{1, 2, 3})
	assert.Error(t, err)
	assert.Equal(t, domain.ErrProviderNotConsolidated, err)
	assert.Nil(t, result)
}

func TestPayRange_InvalidStartMonthFormat(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, providerRepo)

	ctx := context.Background()
	workspaceID := int32(1)
	providerID := int32(1)

	// Setup provider with consolidated mode
	providerRepo.Providers[providerID] = &domain.LoanProvider{
		ID:          providerID,
		WorkspaceID: workspaceID,
		Name:        "Test Provider",
		PaymentMode: domain.PaymentModeConsolidatedMonthly,
	}

	result, err := svc.PayRange(ctx, workspaceID, providerID, "invalid", "2026-05", []int32{1, 2, 3})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid month format")
	assert.Nil(t, result)
}

func TestPayRange_InvalidEndMonthFormat(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, providerRepo)

	ctx := context.Background()
	workspaceID := int32(1)
	providerID := int32(1)

	// Setup provider with consolidated mode
	providerRepo.Providers[providerID] = &domain.LoanProvider{
		ID:          providerID,
		WorkspaceID: workspaceID,
		Name:        "Test Provider",
		PaymentMode: domain.PaymentModeConsolidatedMonthly,
	}

	result, err := svc.PayRange(ctx, workspaceID, providerID, "2026-02", "invalid", []int32{1, 2, 3})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid month format")
	assert.Nil(t, result)
}

func TestPayRange_EndMonthBeforeStart(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, providerRepo)

	ctx := context.Background()
	workspaceID := int32(1)
	providerID := int32(1)

	// Setup provider with consolidated mode
	providerRepo.Providers[providerID] = &domain.LoanProvider{
		ID:          providerID,
		WorkspaceID: workspaceID,
		Name:        "Test Provider",
		PaymentMode: domain.PaymentModeConsolidatedMonthly,
	}

	// End month is before start month
	result, err := svc.PayRange(ctx, workspaceID, providerID, "2026-05", "2026-02", []int32{1, 2, 3})
	assert.Error(t, err)
	assert.Equal(t, domain.ErrEndMonthBeforeStart, err)
	assert.Nil(t, result)
}

func TestPayRange_EndMonthEqualStart(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, providerRepo)

	ctx := context.Background()
	workspaceID := int32(1)
	providerID := int32(1)

	// Setup provider with consolidated mode
	providerRepo.Providers[providerID] = &domain.LoanProvider{
		ID:          providerID,
		WorkspaceID: workspaceID,
		Name:        "Test Provider",
		PaymentMode: domain.PaymentModeConsolidatedMonthly,
	}

	// End month equals start month
	result, err := svc.PayRange(ctx, workspaceID, providerID, "2026-02", "2026-02", []int32{1, 2, 3})
	assert.Error(t, err)
	assert.Equal(t, domain.ErrEndMonthBeforeStart, err)
	assert.Nil(t, result)
}

func TestPayRange_NoUnpaidMonths(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, providerRepo)

	ctx := context.Background()
	workspaceID := int32(1)
	providerID := int32(1)

	// Setup provider with consolidated mode
	providerRepo.Providers[providerID] = &domain.LoanProvider{
		ID:          providerID,
		WorkspaceID: workspaceID,
		Name:        "Test Provider",
		PaymentMode: domain.PaymentModeConsolidatedMonthly,
	}

	// No unpaid months (default mock returns nil)
	result, err := svc.PayRange(ctx, workspaceID, providerID, "2026-02", "2026-05", []int32{1, 2, 3})
	assert.Error(t, err)
	assert.Equal(t, domain.ErrNoUnpaidMonths, err)
	assert.Nil(t, result)
}

func TestPayRange_SequentialEnforcementViolation(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, providerRepo)

	ctx := context.Background()
	workspaceID := int32(1)
	providerID := int32(1)

	// Setup provider with consolidated mode
	providerRepo.Providers[providerID] = &domain.LoanProvider{
		ID:          providerID,
		WorkspaceID: workspaceID,
		Name:        "Test Provider",
		PaymentMode: domain.PaymentModeConsolidatedMonthly,
	}

	// Setup earliest unpaid month as January
	paymentRepo.GetEarliestUnpaidMonthFn = func(wID int32, pID int32) (*domain.EarliestUnpaidMonth, error) {
		return &domain.EarliestUnpaidMonth{Year: 2026, Month: 1}, nil
	}

	// Try to pay February-May (should fail - must start from January)
	result, err := svc.PayRange(ctx, workspaceID, providerID, "2026-02", "2026-05", []int32{1, 2, 3})
	assert.Error(t, err)

	// Check it's the right error type
	seqErr, ok := err.(domain.ErrMustPayEarlierMonth)
	assert.True(t, ok, "Expected ErrMustPayEarlierMonth error type")
	assert.Equal(t, "2026-01", seqErr.Expected)
	assert.Equal(t, "2026-02", seqErr.Requested)
	assert.Nil(t, result)
}

func TestPayRange_EmptyPaymentIDs(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, providerRepo)

	ctx := context.Background()
	workspaceID := int32(1)
	providerID := int32(1)

	// Setup provider with consolidated mode
	providerRepo.Providers[providerID] = &domain.LoanProvider{
		ID:          providerID,
		WorkspaceID: workspaceID,
		Name:        "Test Provider",
		PaymentMode: domain.PaymentModeConsolidatedMonthly,
	}

	// Setup earliest unpaid month
	paymentRepo.GetEarliestUnpaidMonthFn = func(wID int32, pID int32) (*domain.EarliestUnpaidMonth, error) {
		return &domain.EarliestUnpaidMonth{Year: 2026, Month: 2}, nil
	}

	// Try to pay with empty payment IDs
	result, err := svc.PayRange(ctx, workspaceID, providerID, "2026-02", "2026-05", []int32{})
	assert.Error(t, err)
	assert.Equal(t, domain.ErrPaymentIDsInvalid, err)
	assert.Nil(t, result)
}

func TestPayRange_GapInMonths(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, providerRepo)

	ctx := context.Background()
	workspaceID := int32(1)
	providerID := int32(1)

	// Setup provider with consolidated mode
	providerRepo.Providers[providerID] = &domain.LoanProvider{
		ID:          providerID,
		WorkspaceID: workspaceID,
		Name:        "Test Provider",
		PaymentMode: domain.PaymentModeConsolidatedMonthly,
	}

	// Setup earliest unpaid month
	paymentRepo.GetEarliestUnpaidMonthFn = func(wID int32, pID int32) (*domain.EarliestUnpaidMonth, error) {
		return &domain.EarliestUnpaidMonth{Year: 2026, Month: 2}, nil
	}

	// Setup payments - Feb has payments, March has no payments (gap)
	paymentRepo.GetUnpaidPaymentsByProviderMonthFn = func(wID int32, pID int32, year int32, month int32) ([]*domain.LoanPayment, error) {
		if year == 2026 && month == 2 {
			return []*domain.LoanPayment{
				{ID: 1, LoanID: 10, Amount: decimal.NewFromInt(100)},
			}, nil
		}
		// March has no payments - gap
		if year == 2026 && month == 3 {
			return []*domain.LoanPayment{}, nil
		}
		return []*domain.LoanPayment{}, nil
	}

	// Try to pay Feb-May (should fail at March - gap detected)
	result, err := svc.PayRange(ctx, workspaceID, providerID, "2026-02", "2026-05", []int32{1, 2, 3})
	assert.Error(t, err)

	// Check it's the right error type
	skipErr, ok := err.(domain.ErrCannotSkipMonth)
	assert.True(t, ok, "Expected ErrCannotSkipMonth error type")
	assert.Equal(t, "2026-03", skipErr.Skipped)
	assert.Nil(t, result)
}

func TestPayRange_InvalidPaymentIDs(t *testing.T) {
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	svc := NewLoanPaymentService(nil, paymentRepo, loanRepo, providerRepo)

	ctx := context.Background()
	workspaceID := int32(1)
	providerID := int32(1)

	// Setup provider with consolidated mode
	providerRepo.Providers[providerID] = &domain.LoanProvider{
		ID:          providerID,
		WorkspaceID: workspaceID,
		Name:        "Test Provider",
		PaymentMode: domain.PaymentModeConsolidatedMonthly,
	}

	// Setup earliest unpaid month
	paymentRepo.GetEarliestUnpaidMonthFn = func(wID int32, pID int32) (*domain.EarliestUnpaidMonth, error) {
		return &domain.EarliestUnpaidMonth{Year: 2026, Month: 2}, nil
	}

	// Setup payments for Feb, Mar, Apr, May
	paymentRepo.GetUnpaidPaymentsByProviderMonthFn = func(wID int32, pID int32, year int32, month int32) ([]*domain.LoanPayment, error) {
		// Each month has one payment with sequential IDs
		paymentID := int32(month) // Feb=2, Mar=3, Apr=4, May=5
		return []*domain.LoanPayment{
			{ID: paymentID, LoanID: 10, Amount: decimal.NewFromInt(100)},
		}, nil
	}

	// Try to pay with invalid payment ID (99 doesn't exist)
	result, err := svc.PayRange(ctx, workspaceID, providerID, "2026-02", "2026-05", []int32{2, 3, 4, 99})
	assert.Error(t, err)
	assert.Equal(t, domain.ErrPaymentIDsInvalid, err)
	assert.Nil(t, result)
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestFormatMonth(t *testing.T) {
	tests := []struct {
		year     int
		month    int
		expected string
	}{
		{2026, 1, "2026-01"},
		{2026, 12, "2026-12"},
		{2025, 6, "2025-06"},
	}

	for _, tc := range tests {
		result := formatMonth(tc.year, tc.month)
		assert.Equal(t, tc.expected, result)
	}
}

func TestNextMonth(t *testing.T) {
	tests := []struct {
		inYear      int
		inMonth     int
		outYear     int
		outMonth    int
	}{
		{2026, 1, 2026, 2},
		{2026, 11, 2026, 12},
		{2026, 12, 2027, 1}, // Year rollover
		{2025, 6, 2025, 7},
	}

	for _, tc := range tests {
		year, month := nextMonth(tc.inYear, tc.inMonth)
		assert.Equal(t, tc.outYear, year)
		assert.Equal(t, tc.outMonth, month)
	}
}

func TestCompareMonths(t *testing.T) {
	tests := []struct {
		yearA    int
		monthA   int
		yearB    int
		monthB   int
		expected int
	}{
		{2026, 1, 2026, 1, 0},   // Equal
		{2026, 1, 2026, 2, -1},  // A before B (same year)
		{2026, 2, 2026, 1, 1},   // A after B (same year)
		{2025, 12, 2026, 1, -1}, // A before B (different year)
		{2026, 1, 2025, 12, 1},  // A after B (different year)
	}

	for _, tc := range tests {
		result := compareMonths(tc.yearA, tc.monthA, tc.yearB, tc.monthB)
		assert.Equal(t, tc.expected, result)
	}
}

func TestGenerateMonthRange(t *testing.T) {
	tests := []struct {
		name       string
		startYear  int
		startMonth int
		endYear    int
		endMonth   int
		expected   []string
	}{
		{
			name:       "Single month span",
			startYear:  2026, startMonth: 2,
			endYear:    2026, endMonth: 3,
			expected:   []string{"2026-02", "2026-03"},
		},
		{
			name:       "Four month span",
			startYear:  2026, startMonth: 2,
			endYear:    2026, endMonth: 5,
			expected:   []string{"2026-02", "2026-03", "2026-04", "2026-05"},
		},
		{
			name:       "Year boundary crossing",
			startYear:  2025, startMonth: 11,
			endYear:    2026, endMonth: 2,
			expected:   []string{"2025-11", "2025-12", "2026-01", "2026-02"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := generateMonthRange(tc.startYear, tc.startMonth, tc.endYear, tc.endMonth)
			assert.Equal(t, tc.expected, result)
		})
	}
}
