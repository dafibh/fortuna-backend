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

func setupProjectionService() (*ProjectionService, *testutil.MockRecurringRepository, *testutil.MockTransactionRepository) {
	recurringRepo := testutil.NewMockRecurringRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	service := NewProjectionService(recurringRepo, transactionRepo)
	return service, recurringRepo, transactionRepo
}

func TestProjection_GenerateProjections_NoTemplates(t *testing.T) {
	service, _, _ := setupProjectionService()

	result, err := service.GenerateProjections(1, 12)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Generated)
	assert.Equal(t, 0, result.Skipped)
	assert.Empty(t, result.Errors)
}

func TestProjection_GenerateProjections_SingleActiveTemplate(t *testing.T) {
	service, recurringRepo, transactionRepo := setupProjectionService()

	// Add an active template
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	template := &domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Netflix",
		Amount:      decimal.NewFromFloat(15.99),
		AccountID:   1,
		Type:        domain.TransactionTypeExpense,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      15,
		StartDate:   startDate,
		IsActive:    true,
	}
	recurringRepo.AddRecurring(template)

	result, err := service.GenerateProjections(1, 3)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, result.Generated, 1) // At least 1 projection created
	assert.Empty(t, result.Errors)

	// Check that transactions were created with correct attributes
	assert.GreaterOrEqual(t, len(transactionRepo.Transactions), 1)
	for _, tx := range transactionRepo.Transactions {
		assert.Equal(t, "Netflix", tx.Name)
		assert.Equal(t, domain.TransactionSourceRecurring, tx.Source)
		assert.NotNil(t, tx.TemplateID)
		assert.Equal(t, int32(1), *tx.TemplateID)
	}
}

func TestProjection_GenerateProjections_InactiveTemplateSkipped(t *testing.T) {
	service, recurringRepo, _ := setupProjectionService()

	// Add an inactive template
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	template := &domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Inactive Sub",
		Amount:      decimal.NewFromFloat(9.99),
		AccountID:   1,
		Type:        domain.TransactionTypeExpense,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      1,
		StartDate:   startDate,
		IsActive:    false, // Inactive
	}
	recurringRepo.AddRecurring(template)

	result, err := service.GenerateProjections(1, 3)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Generated) // Nothing generated
}

func TestProjection_GenerateProjections_TemplateWithEndDate(t *testing.T) {
	service, recurringRepo, transactionRepo := setupProjectionService()

	// Add a template with an end date in 2 months
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 2, 0) // Ends in 2 months

	template := &domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Limited Sub",
		Amount:      decimal.NewFromFloat(5.00),
		AccountID:   1,
		Type:        domain.TransactionTypeExpense,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      10,
		StartDate:   startDate,
		EndDate:     &endDate,
		IsActive:    true,
	}
	recurringRepo.AddRecurring(template)

	result, err := service.GenerateProjections(1, 12) // Request 12 months
	require.NoError(t, err)

	// Should only generate up to end date (max 3 months: current + 2)
	assert.LessOrEqual(t, result.Generated, 3)
	assert.Empty(t, result.Errors)

	// Verify no transactions beyond end date
	for _, tx := range transactionRepo.Transactions {
		assert.True(t, tx.TransactionDate.Before(endDate) || tx.TransactionDate.Equal(endDate) ||
			tx.TransactionDate.Month() == endDate.Month() && tx.TransactionDate.Year() == endDate.Year())
	}
}

func TestProjection_GenerateProjections_FutureStartDate(t *testing.T) {
	service, recurringRepo, transactionRepo := setupProjectionService()

	// Add a template that starts in 2 months
	now := time.Now()
	futureStart := now.AddDate(0, 2, 0)
	futureStart = time.Date(futureStart.Year(), futureStart.Month(), 1, 0, 0, 0, 0, time.UTC)

	template := &domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Future Sub",
		Amount:      decimal.NewFromFloat(20.00),
		AccountID:   1,
		Type:        domain.TransactionTypeExpense,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      5,
		StartDate:   futureStart,
		IsActive:    true,
	}
	recurringRepo.AddRecurring(template)

	_, err := service.GenerateProjections(1, 6)
	require.NoError(t, err)

	// Should only generate projections starting from the future start date
	for _, tx := range transactionRepo.Transactions {
		txMonth := time.Date(tx.TransactionDate.Year(), tx.TransactionDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		futureStartMonth := time.Date(futureStart.Year(), futureStart.Month(), 1, 0, 0, 0, 0, time.UTC)
		assert.True(t, !txMonth.Before(futureStartMonth), "Transaction should not be before start date")
	}
}

func TestProjection_GenerateProjectionsForMonth_Success(t *testing.T) {
	service, recurringRepo, transactionRepo := setupProjectionService()

	// Add an active template
	now := time.Now()
	startDate := time.Date(now.Year()-1, 1, 1, 0, 0, 0, 0, time.UTC) // Started last year
	template := &domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Monthly Bill",
		Amount:      decimal.NewFromFloat(50.00),
		AccountID:   1,
		Type:        domain.TransactionTypeExpense,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      20,
		StartDate:   startDate,
		IsActive:    true,
	}
	recurringRepo.AddRecurring(template)

	// Generate for a specific future month
	targetDate := now.AddDate(0, 1, 0) // Next month
	targetYear := targetDate.Year()
	targetMonth := targetDate.Month()

	result, err := service.GenerateProjectionsForMonth(1, targetYear, targetMonth)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Generated)
	assert.Empty(t, result.Errors)

	// Verify the transaction
	assert.Len(t, transactionRepo.Transactions, 1)
	for _, tx := range transactionRepo.Transactions {
		assert.Equal(t, "Monthly Bill", tx.Name)
		assert.Equal(t, targetMonth, tx.TransactionDate.Month())
		assert.Equal(t, targetYear, tx.TransactionDate.Year())
	}
}

func TestProjection_RegenerateProjectionsForTemplate_Success(t *testing.T) {
	service, recurringRepo, transactionRepo := setupProjectionService()

	// Add an active template
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	template := &domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Regenerate Test",
		Amount:      decimal.NewFromFloat(25.00),
		AccountID:   1,
		Type:        domain.TransactionTypeExpense,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      15,
		StartDate:   startDate,
		IsActive:    true,
	}
	recurringRepo.AddRecurring(template)

	// First, generate some projections
	_, err := service.GenerateProjections(1, 3)
	require.NoError(t, err)
	initialCount := len(transactionRepo.Transactions)
	assert.Greater(t, initialCount, 0)

	// Clear transactions to simulate delete (mock doesn't actually delete)
	transactionRepo.Transactions = make(map[int32]*domain.Transaction)

	// Regenerate projections
	result, err := service.RegenerateProjectionsForTemplate(1, 1, 3)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, result.Generated, 1)
}

func TestProjection_RegenerateProjectionsForTemplate_InactiveTemplate(t *testing.T) {
	service, recurringRepo, _ := setupProjectionService()

	// Add an inactive template
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	template := &domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Inactive Test",
		Amount:      decimal.NewFromFloat(10.00),
		AccountID:   1,
		Type:        domain.TransactionTypeExpense,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      1,
		StartDate:   startDate,
		IsActive:    false,
	}
	recurringRepo.AddRecurring(template)

	// Regenerate should return 0 for inactive template
	result, err := service.RegenerateProjectionsForTemplate(1, 1, 3)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Generated)
	assert.Equal(t, 0, result.Skipped)
}

func TestProjection_CalculateActualDueDate_NormalDay(t *testing.T) {
	// Test normal day in a 31-day month
	result := CalculateActualDueDate(15, 2026, time.January)
	assert.Equal(t, 15, result.Day())
	assert.Equal(t, time.January, result.Month())
	assert.Equal(t, 2026, result.Year())
}

func TestProjection_CalculateActualDueDate_Day31InShortMonth(t *testing.T) {
	// Test day 31 in February (should clamp to last day)
	result := CalculateActualDueDate(31, 2026, time.February)
	assert.Equal(t, 28, result.Day()) // 2026 is not a leap year
	assert.Equal(t, time.February, result.Month())
}

func TestProjection_CalculateActualDueDate_Day31InApril(t *testing.T) {
	// Test day 31 in April (30-day month)
	result := CalculateActualDueDate(31, 2026, time.April)
	assert.Equal(t, 30, result.Day())
	assert.Equal(t, time.April, result.Month())
}

func TestProjection_CalculateActualDueDate_InvalidDayClampedTo1(t *testing.T) {
	// Test invalid day (0 or negative) should clamp to 1
	result := CalculateActualDueDate(0, 2026, time.March)
	assert.Equal(t, 1, result.Day())

	result2 := CalculateActualDueDate(-5, 2026, time.March)
	assert.Equal(t, 1, result2.Day())
}

func TestProjection_DefaultProjectionMonths(t *testing.T) {
	// Verify the default constant
	assert.Equal(t, 12, DefaultProjectionMonths)
}

func TestProjection_GenerateProjections_DefaultMonthsWhenZero(t *testing.T) {
	service, _, _ := setupProjectionService()

	// Pass 0 or negative months should use default
	result, err := service.GenerateProjections(1, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = service.GenerateProjections(1, -1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestProjection_CleanupProjectionsBeyondEndDate(t *testing.T) {
	service, _, _ := setupProjectionService()

	// Test cleanup function - mock just returns 0
	endDate := time.Now().AddDate(0, 1, 0)
	deleted, err := service.CleanupProjectionsBeyondEndDate(1, 1, endDate)
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted) // Mock returns 0
}
