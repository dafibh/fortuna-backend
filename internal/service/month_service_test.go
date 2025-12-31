package service

import (
	"testing"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMonthService_GetOrCreateMonth_CreatesNewMonth(t *testing.T) {
	monthRepo := testutil.NewMockMonthRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	calcService := NewCalculationService(accountRepo, transactionRepo)
	svc := NewMonthService(monthRepo, transactionRepo, calcService)

	// Add an account with initial balance
	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    1,
		Name:           "Bank Account",
		AccountType:    domain.AccountTypeAsset,
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromInt(1000),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})

	result, err := svc.GetOrCreateMonth(1, 2025, 1)

	require.NoError(t, err)
	assert.Equal(t, 2025, result.Month.Year)
	assert.Equal(t, 1, result.Month.Month)
	assert.Equal(t, "1000.00", result.StartingBalance.StringFixed(2))
	assert.Equal(t, "0.00", result.TotalIncome.StringFixed(2))
	assert.Equal(t, "0.00", result.TotalExpenses.StringFixed(2))
	assert.Equal(t, "1000.00", result.ClosingBalance.StringFixed(2))
}

func TestMonthService_GetOrCreateMonth_ReturnsExistingMonth(t *testing.T) {
	monthRepo := testutil.NewMockMonthRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	calcService := NewCalculationService(accountRepo, transactionRepo)
	svc := NewMonthService(monthRepo, transactionRepo, calcService)

	// Pre-create a month
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	monthRepo.AddMonth(&domain.Month{
		ID:              1,
		WorkspaceID:     1,
		Year:            2025,
		Month:           1,
		StartDate:       startDate,
		EndDate:         endDate,
		StartingBalance: decimal.NewFromInt(5000),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})

	result, err := svc.GetOrCreateMonth(1, 2025, 1)

	require.NoError(t, err)
	assert.Equal(t, int32(1), result.ID)
	assert.Equal(t, "5000.00", result.StartingBalance.StringFixed(2))
}

func TestMonthService_GetOrCreateMonth_UsesPreviousMonthClosingBalance(t *testing.T) {
	monthRepo := testutil.NewMockMonthRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	calcService := NewCalculationService(accountRepo, transactionRepo)
	svc := NewMonthService(monthRepo, transactionRepo, calcService)

	// Create December 2024 with starting balance
	decStart := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	decEnd := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	monthRepo.AddMonth(&domain.Month{
		ID:              1,
		WorkspaceID:     1,
		Year:            2024,
		Month:           12,
		StartDate:       decStart,
		EndDate:         decEnd,
		StartingBalance: decimal.NewFromInt(1000),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})

	// Add transactions for December
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              1,
		WorkspaceID:     1,
		AccountID:       1,
		Name:            "Salary",
		Amount:          decimal.NewFromInt(5000),
		Type:            domain.TransactionTypeIncome,
		TransactionDate: time.Date(2024, 12, 15, 0, 0, 0, 0, time.UTC),
		IsPaid:          true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              2,
		WorkspaceID:     1,
		AccountID:       1,
		Name:            "Rent",
		Amount:          decimal.NewFromInt(1500),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: time.Date(2024, 12, 20, 0, 0, 0, 0, time.UTC),
		IsPaid:          true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})

	// Create January 2025 - should use December's closing balance (1000 + 5000 - 1500 = 4500)
	result, err := svc.GetOrCreateMonth(1, 2025, 1)

	require.NoError(t, err)
	assert.Equal(t, 2025, result.Month.Year)
	assert.Equal(t, 1, result.Month.Month)
	assert.Equal(t, "4500.00", result.StartingBalance.StringFixed(2))
}

func TestMonthService_GetOrCreateMonth_CalculatesTransactions(t *testing.T) {
	monthRepo := testutil.NewMockMonthRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	calcService := NewCalculationService(accountRepo, transactionRepo)
	svc := NewMonthService(monthRepo, transactionRepo, calcService)

	// Create a month
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	monthRepo.AddMonth(&domain.Month{
		ID:              1,
		WorkspaceID:     1,
		Year:            2025,
		Month:           1,
		StartDate:       startDate,
		EndDate:         endDate,
		StartingBalance: decimal.NewFromInt(1000),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})

	// Add transactions for January
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              1,
		WorkspaceID:     1,
		AccountID:       1,
		Name:            "Salary",
		Amount:          decimal.NewFromInt(5000),
		Type:            domain.TransactionTypeIncome,
		TransactionDate: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		IsPaid:          true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              2,
		WorkspaceID:     1,
		AccountID:       1,
		Name:            "Groceries",
		Amount:          decimal.NewFromInt(300),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC),
		IsPaid:          true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})

	result, err := svc.GetOrCreateMonth(1, 2025, 1)

	require.NoError(t, err)
	assert.Equal(t, "1000.00", result.StartingBalance.StringFixed(2))
	assert.Equal(t, "5000.00", result.TotalIncome.StringFixed(2))
	assert.Equal(t, "300.00", result.TotalExpenses.StringFixed(2))
	assert.Equal(t, "5700.00", result.ClosingBalance.StringFixed(2)) // 1000 + 5000 - 300
}

func TestMonthService_GetOrCreateMonth_InvalidMonth(t *testing.T) {
	monthRepo := testutil.NewMockMonthRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	calcService := NewCalculationService(accountRepo, transactionRepo)
	svc := NewMonthService(monthRepo, transactionRepo, calcService)

	_, err := svc.GetOrCreateMonth(1, 2025, 0)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)

	_, err = svc.GetOrCreateMonth(1, 2025, 13)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestMonthService_GetOrCreateMonth_InvalidYear(t *testing.T) {
	monthRepo := testutil.NewMockMonthRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	calcService := NewCalculationService(accountRepo, transactionRepo)
	svc := NewMonthService(monthRepo, transactionRepo, calcService)

	_, err := svc.GetOrCreateMonth(1, 1999, 1)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)

	_, err = svc.GetOrCreateMonth(1, 2101, 1)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestGetMonthBoundaries(t *testing.T) {
	tests := []struct {
		name          string
		year          int
		month         int
		expectedStart time.Time
		expectedEnd   time.Time
	}{
		{
			name:          "January",
			year:          2025,
			month:         1,
			expectedStart: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedEnd:   time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "February non-leap",
			year:          2025,
			month:         2,
			expectedStart: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			expectedEnd:   time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "February leap year",
			year:          2024,
			month:         2,
			expectedStart: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			expectedEnd:   time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "December",
			year:          2024,
			month:         12,
			expectedStart: time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
			expectedEnd:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := getMonthBoundaries(tt.year, tt.month)
			assert.Equal(t, tt.expectedStart, start)
			assert.Equal(t, tt.expectedEnd, end)
		})
	}
}

func TestGetPreviousMonth(t *testing.T) {
	tests := []struct {
		name          string
		year          int
		month         int
		expectedYear  int
		expectedMonth int
	}{
		{"January -> December", 2025, 1, 2024, 12},
		{"February -> January", 2025, 2, 2025, 1},
		{"December -> November", 2025, 12, 2025, 11},
		{"July -> June", 2025, 7, 2025, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			year, month := getPreviousMonth(tt.year, tt.month)
			assert.Equal(t, tt.expectedYear, year)
			assert.Equal(t, tt.expectedMonth, month)
		})
	}
}
