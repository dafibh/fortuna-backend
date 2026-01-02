package service

import (
	"testing"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/shopspring/decimal"
)

func setupRecurringServiceTest() (*RecurringService, *testutil.MockRecurringRepository, *testutil.MockAccountRepository, *testutil.MockBudgetCategoryRepository) {
	recurringRepo := testutil.NewMockRecurringRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewRecurringService(recurringRepo, accountRepo, categoryRepo)
	return service, recurringRepo, accountRepo, categoryRepo
}

// CreateRecurring tests

func TestCreateRecurring_Success(t *testing.T) {
	service, _, accountRepo, _ := setupRecurringServiceTest()

	workspaceID := int32(1)
	accountID := int32(1)

	// Add account to mock
	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Bank Account",
	})

	input := CreateRecurringInput{
		Name:      "Rent",
		Amount:    decimal.NewFromFloat(1200.00),
		AccountID: accountID,
		Type:      domain.TransactionTypeExpense,
		Frequency: domain.FrequencyMonthly,
		DueDay:    1,
	}

	rt, err := service.CreateRecurring(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rt.Name != "Rent" {
		t.Errorf("Expected name 'Rent', got %s", rt.Name)
	}
	if !rt.Amount.Equal(decimal.NewFromFloat(1200.00)) {
		t.Errorf("Expected amount 1200.00, got %s", rt.Amount.String())
	}
	if rt.Type != domain.TransactionTypeExpense {
		t.Errorf("Expected type 'expense', got %s", rt.Type)
	}
	if rt.Frequency != domain.FrequencyMonthly {
		t.Errorf("Expected frequency 'monthly', got %s", rt.Frequency)
	}
	if rt.DueDay != 1 {
		t.Errorf("Expected due day 1, got %d", rt.DueDay)
	}
	if !rt.IsActive {
		t.Error("Expected IsActive to be true")
	}
}

func TestCreateRecurring_WithCategory(t *testing.T) {
	service, _, accountRepo, categoryRepo := setupRecurringServiceTest()

	workspaceID := int32(1)
	accountID := int32(1)
	categoryID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Bank Account",
	})

	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          categoryID,
		WorkspaceID: workspaceID,
		Name:        "Housing",
	})

	input := CreateRecurringInput{
		Name:       "Rent",
		Amount:     decimal.NewFromFloat(1200.00),
		AccountID:  accountID,
		Type:       domain.TransactionTypeExpense,
		CategoryID: &categoryID,
		Frequency:  domain.FrequencyMonthly,
		DueDay:     1,
	}

	rt, err := service.CreateRecurring(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rt.CategoryID == nil {
		t.Error("Expected CategoryID to be set")
	} else if *rt.CategoryID != categoryID {
		t.Errorf("Expected CategoryID %d, got %d", categoryID, *rt.CategoryID)
	}
}

func TestCreateRecurring_DefaultDueDay(t *testing.T) {
	service, _, accountRepo, _ := setupRecurringServiceTest()

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Bank Account",
	})

	input := CreateRecurringInput{
		Name:      "Salary",
		Amount:    decimal.NewFromFloat(5000.00),
		AccountID: accountID,
		Type:      domain.TransactionTypeIncome,
		Frequency: domain.FrequencyMonthly,
		DueDay:    0, // Should default to 1
	}

	rt, err := service.CreateRecurring(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rt.DueDay != 1 {
		t.Errorf("Expected due day to default to 1, got %d", rt.DueDay)
	}
}

func TestCreateRecurring_EmptyName(t *testing.T) {
	service, _, accountRepo, _ := setupRecurringServiceTest()

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
	})

	input := CreateRecurringInput{
		Name:      "",
		Amount:    decimal.NewFromFloat(100.00),
		AccountID: accountID,
		Type:      domain.TransactionTypeExpense,
		Frequency: domain.FrequencyMonthly,
		DueDay:    1,
	}

	_, err := service.CreateRecurring(workspaceID, input)
	if err != domain.ErrNameRequired {
		t.Errorf("Expected ErrNameRequired, got %v", err)
	}
}

func TestCreateRecurring_WhitespaceOnlyName(t *testing.T) {
	service, _, accountRepo, _ := setupRecurringServiceTest()

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
	})

	input := CreateRecurringInput{
		Name:      "   ",
		Amount:    decimal.NewFromFloat(100.00),
		AccountID: accountID,
		Type:      domain.TransactionTypeExpense,
		Frequency: domain.FrequencyMonthly,
		DueDay:    1,
	}

	_, err := service.CreateRecurring(workspaceID, input)
	if err != domain.ErrNameRequired {
		t.Errorf("Expected ErrNameRequired, got %v", err)
	}
}

func TestCreateRecurring_ZeroAmount(t *testing.T) {
	service, _, accountRepo, _ := setupRecurringServiceTest()

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
	})

	input := CreateRecurringInput{
		Name:      "Test",
		Amount:    decimal.Zero,
		AccountID: accountID,
		Type:      domain.TransactionTypeExpense,
		Frequency: domain.FrequencyMonthly,
		DueDay:    1,
	}

	_, err := service.CreateRecurring(workspaceID, input)
	if err != domain.ErrInvalidAmount {
		t.Errorf("Expected ErrInvalidAmount, got %v", err)
	}
}

func TestCreateRecurring_NegativeAmount(t *testing.T) {
	service, _, accountRepo, _ := setupRecurringServiceTest()

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
	})

	input := CreateRecurringInput{
		Name:      "Test",
		Amount:    decimal.NewFromFloat(-100.00),
		AccountID: accountID,
		Type:      domain.TransactionTypeExpense,
		Frequency: domain.FrequencyMonthly,
		DueDay:    1,
	}

	_, err := service.CreateRecurring(workspaceID, input)
	if err != domain.ErrInvalidAmount {
		t.Errorf("Expected ErrInvalidAmount, got %v", err)
	}
}

func TestCreateRecurring_InvalidType(t *testing.T) {
	service, _, accountRepo, _ := setupRecurringServiceTest()

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
	})

	input := CreateRecurringInput{
		Name:      "Test",
		Amount:    decimal.NewFromFloat(100.00),
		AccountID: accountID,
		Type:      domain.TransactionType("invalid"),
		Frequency: domain.FrequencyMonthly,
		DueDay:    1,
	}

	_, err := service.CreateRecurring(workspaceID, input)
	if err != domain.ErrInvalidTransactionType {
		t.Errorf("Expected ErrInvalidTransactionType, got %v", err)
	}
}

func TestCreateRecurring_InvalidFrequency(t *testing.T) {
	service, _, accountRepo, _ := setupRecurringServiceTest()

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
	})

	input := CreateRecurringInput{
		Name:      "Test",
		Amount:    decimal.NewFromFloat(100.00),
		AccountID: accountID,
		Type:      domain.TransactionTypeExpense,
		Frequency: domain.Frequency("weekly"),
		DueDay:    1,
	}

	_, err := service.CreateRecurring(workspaceID, input)
	if err != domain.ErrInvalidFrequency {
		t.Errorf("Expected ErrInvalidFrequency, got %v", err)
	}
}

func TestCreateRecurring_InvalidDueDay_TooLow(t *testing.T) {
	service, _, accountRepo, _ := setupRecurringServiceTest()

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
	})

	input := CreateRecurringInput{
		Name:      "Test",
		Amount:    decimal.NewFromFloat(100.00),
		AccountID: accountID,
		Type:      domain.TransactionTypeExpense,
		Frequency: domain.FrequencyMonthly,
		DueDay:    -1,
	}

	_, err := service.CreateRecurring(workspaceID, input)
	if err != domain.ErrInvalidDueDay {
		t.Errorf("Expected ErrInvalidDueDay, got %v", err)
	}
}

func TestCreateRecurring_InvalidDueDay_TooHigh(t *testing.T) {
	service, _, accountRepo, _ := setupRecurringServiceTest()

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
	})

	input := CreateRecurringInput{
		Name:      "Test",
		Amount:    decimal.NewFromFloat(100.00),
		AccountID: accountID,
		Type:      domain.TransactionTypeExpense,
		Frequency: domain.FrequencyMonthly,
		DueDay:    32,
	}

	_, err := service.CreateRecurring(workspaceID, input)
	if err != domain.ErrInvalidDueDay {
		t.Errorf("Expected ErrInvalidDueDay, got %v", err)
	}
}

func TestCreateRecurring_AccountNotFound(t *testing.T) {
	service, _, _, _ := setupRecurringServiceTest()

	workspaceID := int32(1)

	input := CreateRecurringInput{
		Name:      "Test",
		Amount:    decimal.NewFromFloat(100.00),
		AccountID: 999, // Non-existent
		Type:      domain.TransactionTypeExpense,
		Frequency: domain.FrequencyMonthly,
		DueDay:    1,
	}

	_, err := service.CreateRecurring(workspaceID, input)
	if err != domain.ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound, got %v", err)
	}
}

func TestCreateRecurring_AccountWrongWorkspace(t *testing.T) {
	service, _, accountRepo, _ := setupRecurringServiceTest()

	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: 1, // Belongs to workspace 1
	})

	input := CreateRecurringInput{
		Name:      "Test",
		Amount:    decimal.NewFromFloat(100.00),
		AccountID: 1,
		Type:      domain.TransactionTypeExpense,
		Frequency: domain.FrequencyMonthly,
		DueDay:    1,
	}

	// Try to create in workspace 2
	_, err := service.CreateRecurring(2, input)
	if err != domain.ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound, got %v", err)
	}
}

func TestCreateRecurring_CategoryNotFound(t *testing.T) {
	service, _, accountRepo, _ := setupRecurringServiceTest()

	workspaceID := int32(1)
	accountID := int32(1)
	categoryID := int32(999) // Non-existent

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
	})

	input := CreateRecurringInput{
		Name:       "Test",
		Amount:     decimal.NewFromFloat(100.00),
		AccountID:  accountID,
		Type:       domain.TransactionTypeExpense,
		CategoryID: &categoryID,
		Frequency:  domain.FrequencyMonthly,
		DueDay:     1,
	}

	_, err := service.CreateRecurring(workspaceID, input)
	if err != domain.ErrBudgetCategoryNotFound {
		t.Errorf("Expected ErrBudgetCategoryNotFound, got %v", err)
	}
}

// ListRecurring tests

func TestListRecurring_Success(t *testing.T) {
	service, recurringRepo, _, _ := setupRecurringServiceTest()

	workspaceID := int32(1)

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Rent",
		IsActive:    true,
	})
	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          2,
		WorkspaceID: workspaceID,
		Name:        "Salary",
		IsActive:    true,
	})

	rts, err := service.ListRecurring(workspaceID, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(rts) != 2 {
		t.Errorf("Expected 2 recurring transactions, got %d", len(rts))
	}
}

func TestListRecurring_FilterActiveOnly(t *testing.T) {
	service, recurringRepo, _, _ := setupRecurringServiceTest()

	workspaceID := int32(1)

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Active",
		IsActive:    true,
	})
	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          2,
		WorkspaceID: workspaceID,
		Name:        "Inactive",
		IsActive:    false,
	})

	activeOnly := true
	rts, err := service.ListRecurring(workspaceID, &activeOnly)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(rts) != 1 {
		t.Errorf("Expected 1 recurring transaction, got %d", len(rts))
	}
	if rts[0].Name != "Active" {
		t.Errorf("Expected 'Active', got %s", rts[0].Name)
	}
}

func TestListRecurring_EmptyList(t *testing.T) {
	service, _, _, _ := setupRecurringServiceTest()

	workspaceID := int32(1)

	rts, err := service.ListRecurring(workspaceID, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(rts) != 0 {
		t.Errorf("Expected 0 recurring transactions, got %d", len(rts))
	}
}

func TestListRecurring_WorkspaceIsolation(t *testing.T) {
	service, recurringRepo, _, _ := setupRecurringServiceTest()

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Workspace 1",
		IsActive:    true,
	})
	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          2,
		WorkspaceID: 2,
		Name:        "Workspace 2",
		IsActive:    true,
	})

	rts, err := service.ListRecurring(1, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(rts) != 1 {
		t.Errorf("Expected 1 recurring transaction for workspace 1, got %d", len(rts))
	}
	if rts[0].Name != "Workspace 1" {
		t.Errorf("Expected 'Workspace 1', got %s", rts[0].Name)
	}
}

// GetRecurringByID tests

func TestGetRecurringByID_Success(t *testing.T) {
	service, recurringRepo, _, _ := setupRecurringServiceTest()

	workspaceID := int32(1)

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Rent",
	})

	rt, err := service.GetRecurringByID(workspaceID, 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rt.Name != "Rent" {
		t.Errorf("Expected name 'Rent', got %s", rt.Name)
	}
}

func TestGetRecurringByID_NotFound(t *testing.T) {
	service, _, _, _ := setupRecurringServiceTest()

	_, err := service.GetRecurringByID(1, 999)
	if err != domain.ErrRecurringNotFound {
		t.Errorf("Expected ErrRecurringNotFound, got %v", err)
	}
}

func TestGetRecurringByID_WrongWorkspace(t *testing.T) {
	service, recurringRepo, _, _ := setupRecurringServiceTest()

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Rent",
	})

	_, err := service.GetRecurringByID(2, 1)
	if err != domain.ErrRecurringNotFound {
		t.Errorf("Expected ErrRecurringNotFound, got %v", err)
	}
}

// UpdateRecurring tests

func TestUpdateRecurring_Success(t *testing.T) {
	service, recurringRepo, accountRepo, _ := setupRecurringServiceTest()

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
	})

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Old Name",
		Amount:      decimal.NewFromFloat(100.00),
		AccountID:   accountID,
		Type:        domain.TransactionTypeExpense,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      1,
		IsActive:    true,
	})

	input := UpdateRecurringInput{
		Name:      "New Name",
		Amount:    decimal.NewFromFloat(200.00),
		AccountID: accountID,
		Type:      domain.TransactionTypeExpense,
		Frequency: domain.FrequencyMonthly,
		DueDay:    15,
		IsActive:  false,
	}

	rt, err := service.UpdateRecurring(workspaceID, 1, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rt.Name != "New Name" {
		t.Errorf("Expected name 'New Name', got %s", rt.Name)
	}
	if !rt.Amount.Equal(decimal.NewFromFloat(200.00)) {
		t.Errorf("Expected amount 200.00, got %s", rt.Amount.String())
	}
	if rt.DueDay != 15 {
		t.Errorf("Expected due day 15, got %d", rt.DueDay)
	}
	if rt.IsActive {
		t.Error("Expected IsActive to be false")
	}
}

func TestUpdateRecurring_NotFound(t *testing.T) {
	service, _, accountRepo, _ := setupRecurringServiceTest()

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
	})

	input := UpdateRecurringInput{
		Name:      "Test",
		Amount:    decimal.NewFromFloat(100.00),
		AccountID: accountID,
		Type:      domain.TransactionTypeExpense,
		Frequency: domain.FrequencyMonthly,
		DueDay:    1,
		IsActive:  true,
	}

	_, err := service.UpdateRecurring(workspaceID, 999, input)
	if err != domain.ErrRecurringNotFound {
		t.Errorf("Expected ErrRecurringNotFound, got %v", err)
	}
}

func TestUpdateRecurring_InvalidInput(t *testing.T) {
	service, recurringRepo, accountRepo, _ := setupRecurringServiceTest()

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
	})

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Test",
		AccountID:   accountID,
	})

	// Empty name
	input := UpdateRecurringInput{
		Name:      "",
		Amount:    decimal.NewFromFloat(100.00),
		AccountID: accountID,
		Type:      domain.TransactionTypeExpense,
		Frequency: domain.FrequencyMonthly,
		DueDay:    1,
		IsActive:  true,
	}

	_, err := service.UpdateRecurring(workspaceID, 1, input)
	if err != domain.ErrNameRequired {
		t.Errorf("Expected ErrNameRequired, got %v", err)
	}
}

// DeleteRecurring tests

func TestDeleteRecurring_Success(t *testing.T) {
	service, recurringRepo, _, _ := setupRecurringServiceTest()

	workspaceID := int32(1)

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Test",
	})

	err := service.DeleteRecurring(workspaceID, 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify deleted
	_, err = service.GetRecurringByID(workspaceID, 1)
	if err != domain.ErrRecurringNotFound {
		t.Errorf("Expected ErrRecurringNotFound after delete, got %v", err)
	}
}

func TestDeleteRecurring_NotFound(t *testing.T) {
	service, _, _, _ := setupRecurringServiceTest()

	err := service.DeleteRecurring(1, 999)
	if err != domain.ErrRecurringNotFound {
		t.Errorf("Expected ErrRecurringNotFound, got %v", err)
	}
}

func TestDeleteRecurring_WrongWorkspace(t *testing.T) {
	service, recurringRepo, _, _ := setupRecurringServiceTest()

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test",
	})

	err := service.DeleteRecurring(2, 1)
	if err != domain.ErrRecurringNotFound {
		t.Errorf("Expected ErrRecurringNotFound, got %v", err)
	}
}

func TestDeleteRecurring_AlreadyDeleted(t *testing.T) {
	service, recurringRepo, _, _ := setupRecurringServiceTest()

	workspaceID := int32(1)

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Test",
	})

	// First delete
	err := service.DeleteRecurring(workspaceID, 1)
	if err != nil {
		t.Fatalf("First delete failed: %v", err)
	}

	// Second delete
	err = service.DeleteRecurring(workspaceID, 1)
	if err != domain.ErrRecurringNotFound {
		t.Errorf("Expected ErrRecurringNotFound for already deleted, got %v", err)
	}
}
