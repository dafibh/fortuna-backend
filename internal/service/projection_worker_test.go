package service

import (
	"context"
	"testing"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupProjectionWorker() (*ProjectionWorker, *testutil.MockRecurringRepository, *testutil.MockTransactionRepository, *testutil.MockWorkspaceRepository) {
	recurringRepo := testutil.NewMockRecurringRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()

	projectionService := NewProjectionService(recurringRepo, transactionRepo)

	logger := zerolog.Nop() // Silent logger for tests

	config := ProjectionWorkerConfig{
		Interval:    100 * time.Millisecond, // Fast interval for testing
		MonthsAhead: 3,
	}

	worker := NewProjectionWorker(projectionService, workspaceRepo, logger, config)
	return worker, recurringRepo, transactionRepo, workspaceRepo
}

func TestProjectionWorker_NewProjectionWorker(t *testing.T) {
	worker, _, _, _ := setupProjectionWorker()

	assert.NotNil(t, worker)
	assert.Equal(t, 100*time.Millisecond, worker.interval)
	assert.Equal(t, 3, worker.monthsAhead)
	assert.False(t, worker.IsRunning())
}

func TestProjectionWorker_DefaultConfig(t *testing.T) {
	config := DefaultProjectionWorkerConfig()

	assert.Equal(t, 1*time.Hour, config.Interval)
	assert.Equal(t, 12, config.MonthsAhead)
}

func TestProjectionWorker_StartStop(t *testing.T) {
	worker, _, _, _ := setupProjectionWorker()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the worker
	worker.Start(ctx)
	time.Sleep(50 * time.Millisecond) // Give it time to start

	assert.True(t, worker.IsRunning())

	// Stop the worker
	worker.Stop()

	assert.False(t, worker.IsRunning())
}

func TestProjectionWorker_StartTwice(t *testing.T) {
	worker, _, _, _ := setupProjectionWorker()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the worker twice (should be idempotent)
	worker.Start(ctx)
	worker.Start(ctx)

	time.Sleep(50 * time.Millisecond)
	assert.True(t, worker.IsRunning())

	worker.Stop()
	assert.False(t, worker.IsRunning())
}

func TestProjectionWorker_StopWithoutStart(t *testing.T) {
	worker, _, _, _ := setupProjectionWorker()

	// Stop without starting should not panic
	worker.Stop()
	assert.False(t, worker.IsRunning())
}

func TestProjectionWorker_SyncWorkspace(t *testing.T) {
	worker, recurringRepo, _, workspaceRepo := setupProjectionWorker()

	// Add a workspace
	workspace := &domain.Workspace{
		ID:     1,
		UserID: uuid.New(),
		Name:   "Test Workspace",
	}
	workspaceRepo.AddWorkspace(workspace, "auth0|test")

	// Add a recurring template
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	template := &domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test Recurring",
		Amount:      decimal.NewFromFloat(100.00),
		AccountID:   1,
		Type:        domain.TransactionTypeExpense,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      15,
		StartDate:   startDate,
		IsActive:    true,
	}
	recurringRepo.AddRecurring(template)

	// Manually sync the workspace
	result, err := worker.SyncWorkspace(1)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, result.Generated, 1)
}

func TestProjectionWorker_ContextCancellation(t *testing.T) {
	worker, _, _, _ := setupProjectionWorker()

	ctx, cancel := context.WithCancel(context.Background())

	worker.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	assert.True(t, worker.IsRunning())

	// Cancel the context
	cancel()
	time.Sleep(50 * time.Millisecond)

	// Worker should stop
	assert.False(t, worker.IsRunning())
}

func TestProjectionWorker_DefaultsForInvalidConfig(t *testing.T) {
	recurringRepo := testutil.NewMockRecurringRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()

	projectionService := NewProjectionService(recurringRepo, transactionRepo)
	logger := zerolog.Nop()

	// Config with invalid values
	config := ProjectionWorkerConfig{
		Interval:    0,  // Invalid
		MonthsAhead: -1, // Invalid
	}

	worker := NewProjectionWorker(projectionService, workspaceRepo, logger, config)

	// Should use defaults
	assert.Equal(t, 1*time.Hour, worker.interval)
	assert.Equal(t, DefaultProjectionMonths, worker.monthsAhead)
}
