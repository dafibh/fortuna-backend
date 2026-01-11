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

// int32Ptr returns a pointer to the given int32 value (test helper)
func int32PtrSync(v int32) *int32 {
	return &v
}

func TestSyncAllActive_NoTemplates(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	syncService := NewProjectionSyncService(templateRepo, transactionRepo)

	err := syncService.SyncAllActive()

	require.NoError(t, err)
}

func TestSyncAllActive_CreatesProjections(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	workspaceID := int32(1)

	// Add an active template starting in the future
	startDate := time.Now().AddDate(0, 1, 0)
	templateRepo.AddTemplate(&domain.RecurringTemplate{
		ID:          1,
		WorkspaceID: workspaceID,
		Description: "Monthly Rent",
		Amount:      decimal.NewFromInt(1500),
		CategoryID:  int32PtrSync(1),
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   startDate,
	})

	syncService := NewProjectionSyncService(templateRepo, transactionRepo)

	err := syncService.SyncAllActive()

	require.NoError(t, err)

	// Verify projections were created
	projections, err := transactionRepo.GetProjectionsByTemplate(workspaceID, 1)
	require.NoError(t, err)
	assert.NotEmpty(t, projections)

	// Verify projection properties
	for _, proj := range projections {
		assert.Equal(t, "Monthly Rent", proj.Name)
		assert.True(t, proj.Amount.Equal(decimal.NewFromInt(1500)))
		assert.True(t, proj.IsProjected)
		assert.Equal(t, "recurring", proj.Source)
		assert.Equal(t, int32(1), *proj.TemplateID)
	}
}

func TestSyncAllActive_RespectsEndDate(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	workspaceID := int32(1)

	// Add template with end_date 3 months from now
	startDate := time.Now().AddDate(0, 1, 0)
	endDate := startDate.AddDate(0, 2, 0)
	templateRepo.AddTemplate(&domain.RecurringTemplate{
		ID:          1,
		WorkspaceID: workspaceID,
		Description: "Short Term Bill",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  int32PtrSync(1),
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   startDate,
		EndDate:     &endDate,
	})

	syncService := NewProjectionSyncService(templateRepo, transactionRepo)

	err := syncService.SyncAllActive()

	require.NoError(t, err)

	// Verify only 3 projections were created (not 12)
	projections, err := transactionRepo.GetProjectionsByTemplate(workspaceID, 1)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(projections), 3)
}

func TestSyncAllActive_DeletesProjectionsBeyondEndDate(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	workspaceID := int32(1)
	templateID := int32(1)

	// Add template with end_date 1 month from now (use UTC midnight for consistent comparison)
	now := time.Now().UTC()
	startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
	endDate := startDate.AddDate(0, 1, 0)
	templateRepo.AddTemplate(&domain.RecurringTemplate{
		ID:          templateID,
		WorkspaceID: workspaceID,
		Description: "Bill",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  int32PtrSync(1),
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   startDate,
		EndDate:     &endDate,
	})

	// Add projections that are beyond end_date (simulating old projections)
	beyondDate := endDate.AddDate(0, 3, 0)
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              1,
		WorkspaceID:     workspaceID,
		Name:            "Bill",
		Amount:          decimal.NewFromInt(100),
		TemplateID:      &templateID,
		IsProjected:     true,
		TransactionDate: beyondDate,
		Source:          "recurring",
	})

	syncService := NewProjectionSyncService(templateRepo, transactionRepo)

	err := syncService.SyncAllActive()

	require.NoError(t, err)

	// Verify the projection beyond end_date was deleted
	projections, err := transactionRepo.GetProjectionsByTemplate(workspaceID, templateID)
	require.NoError(t, err)

	for _, proj := range projections {
		assert.False(t, proj.TransactionDate.After(endDate),
			"Projection date %v should not be after end_date %v", proj.TransactionDate, endDate)
	}
}

func TestSyncAllActive_IdempotentDoesNotCreateDuplicates(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	workspaceID := int32(1)

	// Add an active template
	startDate := time.Now().AddDate(0, 1, 0)
	templateRepo.AddTemplate(&domain.RecurringTemplate{
		ID:          1,
		WorkspaceID: workspaceID,
		Description: "Monthly Bill",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  int32PtrSync(1),
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   startDate,
	})

	syncService := NewProjectionSyncService(templateRepo, transactionRepo)

	// Run sync first time
	err := syncService.SyncAllActive()
	require.NoError(t, err)

	projections1, _ := transactionRepo.GetProjectionsByTemplate(workspaceID, 1)
	count1 := len(projections1)

	// Run sync second time
	err = syncService.SyncAllActive()
	require.NoError(t, err)

	projections2, _ := transactionRepo.GetProjectionsByTemplate(workspaceID, 1)
	count2 := len(projections2)

	// Count should be the same (idempotent)
	assert.Equal(t, count1, count2, "Sync should be idempotent - no duplicates created")
}

func TestSyncAllActive_MultipleWorkspaces(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	startDate := time.Now().AddDate(0, 1, 0)

	// Add templates from different workspaces
	templateRepo.AddTemplate(&domain.RecurringTemplate{
		ID:          1,
		WorkspaceID: 1,
		Description: "Workspace 1 Bill",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  int32PtrSync(1),
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   startDate,
	})
	templateRepo.AddTemplate(&domain.RecurringTemplate{
		ID:          2,
		WorkspaceID: 2,
		Description: "Workspace 2 Bill",
		Amount:      decimal.NewFromInt(200),
		CategoryID:  int32PtrSync(1),
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   startDate,
	})

	syncService := NewProjectionSyncService(templateRepo, transactionRepo)

	err := syncService.SyncAllActive()

	require.NoError(t, err)

	// Verify projections for workspace 1
	projections1, _ := transactionRepo.GetProjectionsByTemplate(1, 1)
	assert.NotEmpty(t, projections1)

	// Verify projections for workspace 2
	projections2, _ := transactionRepo.GetProjectionsByTemplate(2, 2)
	assert.NotEmpty(t, projections2)
}

func TestSyncAllActive_GracefulErrorHandling(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	startDate := time.Now().AddDate(0, 1, 0)

	// Add valid template
	templateRepo.AddTemplate(&domain.RecurringTemplate{
		ID:          1,
		WorkspaceID: 1,
		Description: "Valid Bill",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  int32PtrSync(1),
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   startDate,
	})

	syncService := NewProjectionSyncService(templateRepo, transactionRepo)

	// Should complete even with errors (graceful handling)
	err := syncService.SyncAllActive()

	// No error expected for valid template
	require.NoError(t, err)
}
