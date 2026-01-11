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
func int32Ptr(v int32) *int32 {
	return &v
}

func TestCreateTemplate_ValidInput(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()

	// Setup test data
	workspaceID := int32(1)
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Checking",
	})
	_, _ = categoryRepo.Create(&domain.BudgetCategory{
		WorkspaceID: workspaceID,
		Name:        "Utilities",
	})

	service := NewRecurringTemplateService(templateRepo, transactionRepo, accountRepo, categoryRepo)

	input := domain.CreateRecurringTemplateInput{
		WorkspaceID: workspaceID,
		Description: "Monthly Rent",
		Amount:      decimal.NewFromInt(1500),
		CategoryID:  int32Ptr(1),
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   time.Now().AddDate(0, 0, 1),
	}

	template, err := service.CreateTemplate(workspaceID, input)

	require.NoError(t, err)
	assert.NotNil(t, template)
	assert.Equal(t, "Monthly Rent", template.Description)
	assert.True(t, template.Amount.Equal(decimal.NewFromInt(1500)))
	assert.NotNil(t, template.CategoryID)
	assert.Equal(t, int32(1), *template.CategoryID)
	assert.Equal(t, int32(1), template.AccountID)
	assert.Equal(t, "monthly", template.Frequency)
}

func TestCreateTemplate_InvalidInput_EmptyDescription(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()

	service := NewRecurringTemplateService(templateRepo, transactionRepo, accountRepo, categoryRepo)

	input := domain.CreateRecurringTemplateInput{
		Description: "",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  int32Ptr(1),
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   time.Now(),
	}

	_, err := service.CreateTemplate(1, input)

	assert.Error(t, err)
	assert.Equal(t, domain.ErrNameRequired, err)
}

func TestCreateTemplate_InvalidInput_NegativeAmount(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()

	service := NewRecurringTemplateService(templateRepo, transactionRepo, accountRepo, categoryRepo)

	input := domain.CreateRecurringTemplateInput{
		Description: "Test",
		Amount:      decimal.NewFromInt(-100),
		CategoryID:  int32Ptr(1),
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   time.Now(),
	}

	_, err := service.CreateTemplate(1, input)

	assert.Error(t, err)
	assert.Equal(t, domain.ErrInvalidAmount, err)
}

func TestCreateTemplate_InvalidInput_InvalidFrequency(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()

	service := NewRecurringTemplateService(templateRepo, transactionRepo, accountRepo, categoryRepo)

	input := domain.CreateRecurringTemplateInput{
		Description: "Test",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  int32Ptr(1),
		AccountID:   1,
		Frequency:   "weekly", // Not supported in MVP
		StartDate:   time.Now(),
	}

	_, err := service.CreateTemplate(1, input)

	assert.Error(t, err)
	assert.Equal(t, domain.ErrInvalidFrequency, err)
}

func TestCreateTemplate_AccountNotFound(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()

	// No accounts added

	service := NewRecurringTemplateService(templateRepo, transactionRepo, accountRepo, categoryRepo)

	input := domain.CreateRecurringTemplateInput{
		Description: "Test",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  int32Ptr(1),
		AccountID:   999, // Non-existent
		Frequency:   "monthly",
		StartDate:   time.Now(),
	}

	_, err := service.CreateTemplate(1, input)

	assert.Error(t, err)
	assert.Equal(t, domain.ErrAccountNotFound, err)
}

func TestDeleteTemplate_CascadesBehavior(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()

	workspaceID := int32(1)
	templateID := int32(1)

	// Add template
	templateRepo.AddTemplate(&domain.RecurringTemplate{
		ID:          templateID,
		WorkspaceID: workspaceID,
		Description: "Monthly Rent",
		Amount:      decimal.NewFromInt(1500),
	})

	// Add projected transactions
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          1,
		WorkspaceID: workspaceID,
		TemplateID:  &templateID,
		IsProjected: true,
		Name:        "Monthly Rent",
		Amount:      decimal.NewFromInt(1500),
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          2,
		WorkspaceID: workspaceID,
		TemplateID:  &templateID,
		IsProjected: true,
		Name:        "Monthly Rent",
		Amount:      decimal.NewFromInt(1500),
	})

	// Add actual transaction (should be orphaned, not deleted)
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          3,
		WorkspaceID: workspaceID,
		TemplateID:  &templateID,
		IsProjected: false,
		Name:        "Monthly Rent",
		Amount:      decimal.NewFromInt(1500),
	})

	service := NewRecurringTemplateService(templateRepo, transactionRepo, accountRepo, categoryRepo)

	err := service.DeleteTemplate(workspaceID, templateID)

	require.NoError(t, err)

	// Verify template is deleted
	_, err = templateRepo.GetByID(workspaceID, templateID)
	assert.Error(t, err)

	// Verify projected transactions are deleted
	projections, _ := transactionRepo.GetProjectionsByTemplate(workspaceID, templateID)
	assert.Empty(t, projections)

	// Verify actual transaction exists but is orphaned (template_id = nil)
	actual, err := transactionRepo.GetByID(workspaceID, 3)
	require.NoError(t, err)
	assert.Nil(t, actual.TemplateID)
}

func TestGetTemplate_Found(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()

	workspaceID := int32(1)

	templateRepo.AddTemplate(&domain.RecurringTemplate{
		ID:          1,
		WorkspaceID: workspaceID,
		Description: "Monthly Rent",
		Amount:      decimal.NewFromInt(1500),
	})

	service := NewRecurringTemplateService(templateRepo, transactionRepo, accountRepo, categoryRepo)

	template, err := service.GetTemplate(workspaceID, 1)

	require.NoError(t, err)
	assert.NotNil(t, template)
	assert.Equal(t, "Monthly Rent", template.Description)
}

func TestGetTemplate_NotFound(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()

	service := NewRecurringTemplateService(templateRepo, transactionRepo, accountRepo, categoryRepo)

	_, err := service.GetTemplate(1, 999)

	assert.Error(t, err)
	assert.Equal(t, domain.ErrRecurringTemplateNotFound, err)
}

func TestListTemplates(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()

	workspaceID := int32(1)

	templateRepo.AddTemplate(&domain.RecurringTemplate{
		ID:          1,
		WorkspaceID: workspaceID,
		Description: "Monthly Rent",
	})
	templateRepo.AddTemplate(&domain.RecurringTemplate{
		ID:          2,
		WorkspaceID: workspaceID,
		Description: "Utilities",
	})

	service := NewRecurringTemplateService(templateRepo, transactionRepo, accountRepo, categoryRepo)

	templates, err := service.ListTemplates(workspaceID)

	require.NoError(t, err)
	assert.Len(t, templates, 2)
}

func TestListTemplates_WorkspaceIsolation(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()

	// Add templates to different workspaces
	templateRepo.AddTemplate(&domain.RecurringTemplate{
		ID:          1,
		WorkspaceID: 1,
		Description: "Workspace 1 Template",
	})
	templateRepo.AddTemplate(&domain.RecurringTemplate{
		ID:          2,
		WorkspaceID: 2,
		Description: "Workspace 2 Template",
	})

	service := NewRecurringTemplateService(templateRepo, transactionRepo, accountRepo, categoryRepo)

	// List templates for workspace 1
	templates, err := service.ListTemplates(1)

	require.NoError(t, err)
	assert.Len(t, templates, 1)
	assert.Equal(t, "Workspace 1 Template", templates[0].Description)
}

func TestUpdateTemplate_ValidInput(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()

	workspaceID := int32(1)

	// Setup
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
	})
	_, _ = categoryRepo.Create(&domain.BudgetCategory{
		WorkspaceID: workspaceID,
		Name:        "Category1",
	})
	templateRepo.AddTemplate(&domain.RecurringTemplate{
		ID:          1,
		WorkspaceID: workspaceID,
		Description: "Original",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  int32Ptr(1),
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   time.Now(),
	})

	service := NewRecurringTemplateService(templateRepo, transactionRepo, accountRepo, categoryRepo)

	input := domain.UpdateRecurringTemplateInput{
		Description: "Updated",
		Amount:      decimal.NewFromInt(200),
		CategoryID:  int32Ptr(1),
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   time.Now(),
	}

	updated, err := service.UpdateTemplate(workspaceID, 1, input)

	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Description)
	assert.True(t, updated.Amount.Equal(decimal.NewFromInt(200)))
}

func TestUpdateTemplate_NotFound(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()

	service := NewRecurringTemplateService(templateRepo, transactionRepo, accountRepo, categoryRepo)

	input := domain.UpdateRecurringTemplateInput{
		Description: "Test",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  int32Ptr(1),
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   time.Now(),
	}

	_, err := service.UpdateTemplate(1, 999, input)

	assert.Error(t, err)
	assert.Equal(t, domain.ErrRecurringTemplateNotFound, err)
}

// ==================== PROJECTION EDGE CASE TESTS ====================

func TestCreateTemplate_GeneratesProjections(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()

	workspaceID := int32(1)
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Checking",
	})
	_, _ = categoryRepo.Create(&domain.BudgetCategory{
		WorkspaceID: workspaceID,
		Name:        "Utilities",
	})

	service := NewRecurringTemplateService(templateRepo, transactionRepo, accountRepo, categoryRepo)

	// Create template starting in the future
	startDate := time.Now().AddDate(0, 1, 0) // Next month
	input := domain.CreateRecurringTemplateInput{
		WorkspaceID: workspaceID,
		Description: "Monthly Bill",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  int32Ptr(1),
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   startDate,
	}

	template, err := service.CreateTemplate(workspaceID, input)
	require.NoError(t, err)

	// Verify projections were created
	projections, err := transactionRepo.GetProjectionsByTemplate(workspaceID, template.ID)
	require.NoError(t, err)

	// Should have 12 months of projections
	assert.GreaterOrEqual(t, len(projections), 1)
	assert.LessOrEqual(t, len(projections), 13) // At most 13 (12 months + 1)

	// Verify projections have correct data
	for _, proj := range projections {
		assert.Equal(t, "Monthly Bill", proj.Name)
		assert.True(t, proj.Amount.Equal(decimal.NewFromInt(100)))
		assert.True(t, proj.IsProjected)
		assert.Equal(t, "recurring", proj.Source)
		assert.Equal(t, template.ID, *proj.TemplateID)
	}
}

func TestCreateTemplate_MonthEndEdgeCase(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()

	workspaceID := int32(1)
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
	})
	_, _ = categoryRepo.Create(&domain.BudgetCategory{
		WorkspaceID: workspaceID,
		Name:        "Test",
	})

	service := NewRecurringTemplateService(templateRepo, transactionRepo, accountRepo, categoryRepo)

	// Create template with start date on 31st
	startDate := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	input := domain.CreateRecurringTemplateInput{
		WorkspaceID: workspaceID,
		Description: "End of Month Bill",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  int32Ptr(1),
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   startDate,
	}

	template, err := service.CreateTemplate(workspaceID, input)
	require.NoError(t, err)

	projections, err := transactionRepo.GetProjectionsByTemplate(workspaceID, template.ID)
	require.NoError(t, err)
	require.NotEmpty(t, projections)

	// Check that February projection is on 28th or 29th (not March 3rd!)
	for _, proj := range projections {
		if proj.TransactionDate.Month() == time.February {
			day := proj.TransactionDate.Day()
			assert.True(t, day == 28 || day == 29, "February projection should be on 28th or 29th, got %d", day)
		}
	}
}

func TestCreateTemplate_EndDateLimitsProjections(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()

	workspaceID := int32(1)
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
	})
	_, _ = categoryRepo.Create(&domain.BudgetCategory{
		WorkspaceID: workspaceID,
		Name:        "Test",
	})

	service := NewRecurringTemplateService(templateRepo, transactionRepo, accountRepo, categoryRepo)

	// Create template with end date 2 months from start
	startDate := time.Now().AddDate(0, 1, 0)
	endDate := startDate.AddDate(0, 2, 0)
	input := domain.CreateRecurringTemplateInput{
		WorkspaceID: workspaceID,
		Description: "Short-term Bill",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  int32Ptr(1),
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   startDate,
		EndDate:     &endDate,
	}

	template, err := service.CreateTemplate(workspaceID, input)
	require.NoError(t, err)

	projections, err := transactionRepo.GetProjectionsByTemplate(workspaceID, template.ID)
	require.NoError(t, err)

	// Should have only 3 projections (months 0, 1, 2 from start)
	assert.LessOrEqual(t, len(projections), 3)
}

func TestCreateTemplate_EndDateBeforeStartDate_Fails(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()

	service := NewRecurringTemplateService(templateRepo, transactionRepo, accountRepo, categoryRepo)

	startDate := time.Now().AddDate(0, 1, 0)
	endDate := startDate.AddDate(0, -1, 0) // Before start date

	input := domain.CreateRecurringTemplateInput{
		Description: "Invalid Template",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  int32Ptr(1),
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   startDate,
		EndDate:     &endDate,
	}

	_, err := service.CreateTemplate(1, input)

	assert.Error(t, err)
	assert.Equal(t, domain.ErrInvalidDateRange, err)
}

func TestUpdateTemplate_EndDateBeforeStartDate_Fails(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()

	workspaceID := int32(1)
	templateRepo.AddTemplate(&domain.RecurringTemplate{
		ID:          1,
		WorkspaceID: workspaceID,
		Description: "Test",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  int32Ptr(1),
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   time.Now(),
	})

	service := NewRecurringTemplateService(templateRepo, transactionRepo, accountRepo, categoryRepo)

	startDate := time.Now().AddDate(0, 1, 0)
	endDate := startDate.AddDate(0, -1, 0) // Before start date

	input := domain.UpdateRecurringTemplateInput{
		Description: "Test",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  int32Ptr(1),
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   startDate,
		EndDate:     &endDate,
	}

	_, err := service.UpdateTemplate(workspaceID, 1, input)

	assert.Error(t, err)
	assert.Equal(t, domain.ErrInvalidDateRange, err)
}

func TestCreateTemplate_LinkTransactionNotFound_Fails(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()

	workspaceID := int32(1)
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
	})
	_, _ = categoryRepo.Create(&domain.BudgetCategory{
		WorkspaceID: workspaceID,
		Name:        "Test",
	})

	service := NewRecurringTemplateService(templateRepo, transactionRepo, accountRepo, categoryRepo)

	nonExistentTxID := int32(999)
	input := domain.CreateRecurringTemplateInput{
		WorkspaceID:       workspaceID,
		Description:       "Test",
		Amount:            decimal.NewFromInt(100),
		CategoryID:        int32Ptr(1),
		AccountID:         1,
		Frequency:         "monthly",
		StartDate:         time.Now().AddDate(0, 1, 0),
		LinkTransactionID: &nonExistentTxID,
	}

	_, err := service.CreateTemplate(workspaceID, input)

	assert.Error(t, err)
	assert.Equal(t, domain.ErrTransactionNotFound, err)
}

func TestCalculateActualDate(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()

	service := NewRecurringTemplateService(templateRepo, transactionRepo, accountRepo, categoryRepo)

	tests := []struct {
		name       string
		year       int
		month      time.Month
		targetDay  int
		expectedDay int
	}{
		{"January 31st", 2026, time.January, 31, 31},
		{"February 31st (non-leap year)", 2026, time.February, 31, 28},
		{"February 31st (leap year)", 2024, time.February, 31, 29},
		{"April 31st", 2026, time.April, 31, 30},
		{"Normal day", 2026, time.March, 15, 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateActualDate(tt.year, tt.month, tt.targetDay)
			assert.Equal(t, tt.expectedDay, result.Day())
			assert.Equal(t, tt.month, result.Month())
			assert.Equal(t, tt.year, result.Year())
		})
	}
}

func TestCreateTemplate_IdempotentProjections(t *testing.T) {
	templateRepo := testutil.NewMockRecurringTemplateRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()

	workspaceID := int32(1)
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
	})
	_, _ = categoryRepo.Create(&domain.BudgetCategory{
		WorkspaceID: workspaceID,
		Name:        "Test",
	})

	service := NewRecurringTemplateService(templateRepo, transactionRepo, accountRepo, categoryRepo)

	startDate := time.Now().AddDate(0, 1, 0)
	input := domain.CreateRecurringTemplateInput{
		WorkspaceID: workspaceID,
		Description: "Test Bill",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  int32Ptr(1),
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   startDate,
	}

	template, err := service.CreateTemplate(workspaceID, input)
	require.NoError(t, err)

	// Get initial projection count
	projections1, err := transactionRepo.GetProjectionsByTemplate(workspaceID, template.ID)
	require.NoError(t, err)
	initialCount := len(projections1)

	// Manually call generateProjections again (simulating duplicate call)
	err = service.generateProjections(workspaceID, template)
	require.NoError(t, err)

	// Verify no duplicate projections were created
	projections2, err := transactionRepo.GetProjectionsByTemplate(workspaceID, template.ID)
	require.NoError(t, err)
	assert.Equal(t, initialCount, len(projections2), "Idempotency check failed - duplicate projections created")
}
