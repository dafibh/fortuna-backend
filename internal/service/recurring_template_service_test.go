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
		CategoryID:  1,
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   time.Now().AddDate(0, 0, 1),
	}

	template, err := service.CreateTemplate(workspaceID, input)

	require.NoError(t, err)
	assert.NotNil(t, template)
	assert.Equal(t, "Monthly Rent", template.Description)
	assert.True(t, template.Amount.Equal(decimal.NewFromInt(1500)))
	assert.Equal(t, int32(1), template.CategoryID)
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
		CategoryID:  1,
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
		CategoryID:  1,
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
		CategoryID:  1,
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
		CategoryID:  1,
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
		CategoryID:  1,
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   time.Now(),
	})

	service := NewRecurringTemplateService(templateRepo, transactionRepo, accountRepo, categoryRepo)

	input := domain.UpdateRecurringTemplateInput{
		Description: "Updated",
		Amount:      decimal.NewFromInt(200),
		CategoryID:  1,
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
		CategoryID:  1,
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   time.Now(),
	}

	_, err := service.UpdateTemplate(1, 999, input)

	assert.Error(t, err)
	assert.Equal(t, domain.ErrRecurringTemplateNotFound, err)
}
