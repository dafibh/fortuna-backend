package service

import (
	"testing"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/shopspring/decimal"
)

func setupRecurringServiceTest() (*RecurringService, *testutil.MockRecurringRepository, *testutil.MockTransactionRepository, *testutil.MockAccountRepository, *testutil.MockBudgetCategoryRepository) {
	recurringRepo := testutil.NewMockRecurringRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewRecurringService(recurringRepo, transactionRepo, accountRepo, categoryRepo)
	return service, recurringRepo, transactionRepo, accountRepo, categoryRepo
}

// CreateRecurring tests

func TestCreateRecurring_Success(t *testing.T) {
	service, _, _, accountRepo, _ := setupRecurringServiceTest()

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
	service, _, _, accountRepo, categoryRepo := setupRecurringServiceTest()

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
	service, _, _, accountRepo, _ := setupRecurringServiceTest()

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
	service, _, _, accountRepo, _ := setupRecurringServiceTest()

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
	service, _, _, accountRepo, _ := setupRecurringServiceTest()

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
	service, _, _, accountRepo, _ := setupRecurringServiceTest()

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
	service, _, _, accountRepo, _ := setupRecurringServiceTest()

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
	service, _, _, accountRepo, _ := setupRecurringServiceTest()

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
	service, _, _, accountRepo, _ := setupRecurringServiceTest()

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
	service, _, _, accountRepo, _ := setupRecurringServiceTest()

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
	service, _, _, accountRepo, _ := setupRecurringServiceTest()

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
	service, _, _, _, _ := setupRecurringServiceTest()

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
	service, _, _, accountRepo, _ := setupRecurringServiceTest()

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
	service, _, _, accountRepo, _ := setupRecurringServiceTest()

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
	service, recurringRepo, _, _, _ := setupRecurringServiceTest()

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
	service, recurringRepo, _, _, _ := setupRecurringServiceTest()

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
	service, _, _, _, _ := setupRecurringServiceTest()

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
	service, recurringRepo, _, _, _ := setupRecurringServiceTest()

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
	service, recurringRepo, _, _, _ := setupRecurringServiceTest()

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
	service, _, _, _, _ := setupRecurringServiceTest()

	_, err := service.GetRecurringByID(1, 999)
	if err != domain.ErrRecurringNotFound {
		t.Errorf("Expected ErrRecurringNotFound, got %v", err)
	}
}

func TestGetRecurringByID_WrongWorkspace(t *testing.T) {
	service, recurringRepo, _, _, _ := setupRecurringServiceTest()

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
	service, recurringRepo, _, accountRepo, _ := setupRecurringServiceTest()

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
	service, _, _, accountRepo, _ := setupRecurringServiceTest()

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
	service, recurringRepo, _, accountRepo, _ := setupRecurringServiceTest()

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
	service, recurringRepo, _, _, _ := setupRecurringServiceTest()

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
	service, _, _, _, _ := setupRecurringServiceTest()

	err := service.DeleteRecurring(1, 999)
	if err != domain.ErrRecurringNotFound {
		t.Errorf("Expected ErrRecurringNotFound, got %v", err)
	}
}

func TestDeleteRecurring_WrongWorkspace(t *testing.T) {
	service, recurringRepo, _, _, _ := setupRecurringServiceTest()

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
	service, recurringRepo, _, _, _ := setupRecurringServiceTest()

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

// CalculateActualDueDate tests

func TestCalculateActualDueDate_January31(t *testing.T) {
	// Due day 31 in January (31 days) → Jan 31
	result := CalculateActualDueDate(31, 2025, time.January)
	expected := time.Date(2025, time.January, 31, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateActualDueDate_February31_NonLeap(t *testing.T) {
	// Due day 31 in February (28 days in non-leap year) → Feb 28
	result := CalculateActualDueDate(31, 2025, time.February)
	expected := time.Date(2025, time.February, 28, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateActualDueDate_February31_LeapYear(t *testing.T) {
	// Due day 31 in February (29 days in leap year) → Feb 29
	result := CalculateActualDueDate(31, 2024, time.February)
	expected := time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateActualDueDate_February29_NonLeap(t *testing.T) {
	// Due day 29 in February (28 days in non-leap year) → Feb 28
	result := CalculateActualDueDate(29, 2025, time.February)
	expected := time.Date(2025, time.February, 28, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateActualDueDate_February29_LeapYear(t *testing.T) {
	// Due day 29 in February (29 days in leap year) → Feb 29
	result := CalculateActualDueDate(29, 2024, time.February)
	expected := time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateActualDueDate_February30(t *testing.T) {
	// Due day 30 in February (28 days) → Feb 28
	result := CalculateActualDueDate(30, 2025, time.February)
	expected := time.Date(2025, time.February, 28, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateActualDueDate_April31(t *testing.T) {
	// Due day 31 in April (30 days) → Apr 30
	result := CalculateActualDueDate(31, 2025, time.April)
	expected := time.Date(2025, time.April, 30, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateActualDueDate_NormalDay(t *testing.T) {
	// Due day 15 in any month → 15th of that month
	result := CalculateActualDueDate(15, 2025, time.March)
	expected := time.Date(2025, time.March, 15, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateActualDueDate_FirstDay(t *testing.T) {
	// Due day 1 in any month → 1st of that month
	result := CalculateActualDueDate(1, 2025, time.July)
	expected := time.Date(2025, time.July, 1, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateActualDueDate_December31(t *testing.T) {
	// Due day 31 in December (31 days) → Dec 31 (tests year boundary in month+1)
	result := CalculateActualDueDate(31, 2025, time.December)
	expected := time.Date(2025, time.December, 31, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateActualDueDate_June30(t *testing.T) {
	// Due day 31 in June (30 days) → Jun 30 (another 30-day month)
	result := CalculateActualDueDate(31, 2025, time.June)
	expected := time.Date(2025, time.June, 30, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateActualDueDate_InvalidDayZero(t *testing.T) {
	// Due day 0 should be clamped to 1 (defensive)
	result := CalculateActualDueDate(0, 2025, time.March)
	expected := time.Date(2025, time.March, 1, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateActualDueDate_InvalidDayNegative(t *testing.T) {
	// Negative due day should be clamped to 1 (defensive)
	result := CalculateActualDueDate(-5, 2025, time.March)
	expected := time.Date(2025, time.March, 1, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

// GenerateRecurringTransactions tests

func TestGenerateRecurringTransactions_Success(t *testing.T) {
	service, recurringRepo, transactionRepo, _, _ := setupRecurringServiceTest()

	workspaceID := int32(1)

	// Add active recurring templates
	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Rent",
		Amount:      decimal.NewFromFloat(1200.00),
		AccountID:   1,
		Type:        domain.TransactionTypeExpense,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      1,
		IsActive:    true,
	})
	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          2,
		WorkspaceID: workspaceID,
		Name:        "Salary",
		Amount:      decimal.NewFromFloat(5000.00),
		AccountID:   1,
		Type:        domain.TransactionTypeIncome,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      25,
		IsActive:    true,
	})

	result, err := service.GenerateRecurringTransactions(workspaceID, 2025, time.January)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Generated) != 2 {
		t.Errorf("Expected 2 generated transactions, got %d", len(result.Generated))
	}
	if result.Skipped != 0 {
		t.Errorf("Expected 0 skipped, got %d", result.Skipped)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}

	// Verify transactions were created in transaction repo
	txs := transactionRepo.ByWorkspace[workspaceID]
	if len(txs) != 2 {
		t.Errorf("Expected 2 transactions in repo, got %d", len(txs))
	}
}

func TestGenerateRecurringTransactions_Idempotency(t *testing.T) {
	service, recurringRepo, transactionRepo, _, _ := setupRecurringServiceTest()

	workspaceID := int32(1)

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Rent",
		Amount:      decimal.NewFromFloat(1200.00),
		AccountID:   1,
		Type:        domain.TransactionTypeExpense,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      1,
		IsActive:    true,
	})

	// First generation
	result1, err := service.GenerateRecurringTransactions(workspaceID, 2025, time.January)
	if err != nil {
		t.Fatalf("First generation failed: %v", err)
	}
	if len(result1.Generated) != 1 {
		t.Errorf("Expected 1 generated on first run, got %d", len(result1.Generated))
	}

	// Mark as existing for second run
	recurringRepo.SetTransactionExists(1, 2025, 1)

	// Second generation should skip
	result2, err := service.GenerateRecurringTransactions(workspaceID, 2025, time.January)
	if err != nil {
		t.Fatalf("Second generation failed: %v", err)
	}
	if len(result2.Generated) != 0 {
		t.Errorf("Expected 0 generated on second run, got %d", len(result2.Generated))
	}
	if result2.Skipped != 1 {
		t.Errorf("Expected 1 skipped on second run, got %d", result2.Skipped)
	}

	// Verify only 1 transaction was created (not 2)
	txs := transactionRepo.ByWorkspace[workspaceID]
	if len(txs) != 1 {
		t.Errorf("Expected 1 transaction total in repo, got %d", len(txs))
	}
}

func TestGenerateRecurringTransactions_SkipInactive(t *testing.T) {
	service, recurringRepo, transactionRepo, _, _ := setupRecurringServiceTest()

	workspaceID := int32(1)

	// Add one active and one inactive
	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Active Rent",
		Amount:      decimal.NewFromFloat(1200.00),
		AccountID:   1,
		Type:        domain.TransactionTypeExpense,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      1,
		IsActive:    true,
	})
	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          2,
		WorkspaceID: workspaceID,
		Name:        "Paused Subscription",
		Amount:      decimal.NewFromFloat(15.00),
		AccountID:   1,
		Type:        domain.TransactionTypeExpense,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      10,
		IsActive:    false, // Inactive
	})

	result, err := service.GenerateRecurringTransactions(workspaceID, 2025, time.January)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Only active template should be generated
	if len(result.Generated) != 1 {
		t.Errorf("Expected 1 generated (only active), got %d", len(result.Generated))
	}
	if result.Generated[0].Name != "Active Rent" {
		t.Errorf("Expected 'Active Rent', got %s", result.Generated[0].Name)
	}

	// Verify only 1 transaction in repo
	txs := transactionRepo.ByWorkspace[workspaceID]
	if len(txs) != 1 {
		t.Errorf("Expected 1 transaction in repo, got %d", len(txs))
	}
}

func TestGenerateRecurringTransactions_NoTemplates(t *testing.T) {
	service, _, _, _, _ := setupRecurringServiceTest()

	workspaceID := int32(1)

	result, err := service.GenerateRecurringTransactions(workspaceID, 2025, time.January)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Generated) != 0 {
		t.Errorf("Expected 0 generated, got %d", len(result.Generated))
	}
	if result.Skipped != 0 {
		t.Errorf("Expected 0 skipped, got %d", result.Skipped)
	}
}

func TestGenerateRecurringTransactions_DueDayAdjustment(t *testing.T) {
	service, recurringRepo, _, _, _ := setupRecurringServiceTest()

	workspaceID := int32(1)

	// Template with due day 31
	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "End of Month",
		Amount:      decimal.NewFromFloat(100.00),
		AccountID:   1,
		Type:        domain.TransactionTypeExpense,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      31, // February doesn't have 31 days
		IsActive:    true,
	})

	// Generate for February 2025 (non-leap year, 28 days)
	result, err := service.GenerateRecurringTransactions(workspaceID, 2025, time.February)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Generated) != 1 {
		t.Fatalf("Expected 1 generated, got %d", len(result.Generated))
	}

	// Transaction date should be Feb 28, not Feb 31
	expectedDate := time.Date(2025, time.February, 28, 0, 0, 0, 0, time.UTC)
	if !result.Generated[0].TransactionDate.Equal(expectedDate) {
		t.Errorf("Expected transaction date %v, got %v", expectedDate, result.Generated[0].TransactionDate)
	}
}

func TestGenerateRecurringTransactions_SetsRecurringTransactionID(t *testing.T) {
	service, recurringRepo, _, _, _ := setupRecurringServiceTest()

	workspaceID := int32(1)
	recurringID := int32(42)

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          recurringID,
		WorkspaceID: workspaceID,
		Name:        "Test",
		Amount:      decimal.NewFromFloat(100.00),
		AccountID:   1,
		Type:        domain.TransactionTypeExpense,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      1,
		IsActive:    true,
	})

	result, err := service.GenerateRecurringTransactions(workspaceID, 2025, time.January)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Generated) != 1 {
		t.Fatalf("Expected 1 generated, got %d", len(result.Generated))
	}

	// Verify recurring_transaction_id is set
	if result.Generated[0].RecurringTransactionID == nil {
		t.Error("Expected RecurringTransactionID to be set")
	} else if *result.Generated[0].RecurringTransactionID != recurringID {
		t.Errorf("Expected RecurringTransactionID %d, got %d", recurringID, *result.Generated[0].RecurringTransactionID)
	}
}

func TestGenerateRecurringTransactions_WorkspaceIsolation(t *testing.T) {
	service, recurringRepo, transactionRepo, _, _ := setupRecurringServiceTest()

	// Add recurring to workspace 1
	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Workspace 1 Rent",
		Amount:      decimal.NewFromFloat(1200.00),
		AccountID:   1,
		Type:        domain.TransactionTypeExpense,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      1,
		IsActive:    true,
	})
	// Add recurring to workspace 2
	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          2,
		WorkspaceID: 2,
		Name:        "Workspace 2 Rent",
		Amount:      decimal.NewFromFloat(800.00),
		AccountID:   2,
		Type:        domain.TransactionTypeExpense,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      1,
		IsActive:    true,
	})

	// Generate for workspace 1 only
	result, err := service.GenerateRecurringTransactions(1, 2025, time.January)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Generated) != 1 {
		t.Errorf("Expected 1 generated for workspace 1, got %d", len(result.Generated))
	}
	if result.Generated[0].Name != "Workspace 1 Rent" {
		t.Errorf("Expected 'Workspace 1 Rent', got %s", result.Generated[0].Name)
	}

	// Workspace 2 should have no transactions
	if len(transactionRepo.ByWorkspace[2]) != 0 {
		t.Errorf("Expected 0 transactions in workspace 2, got %d", len(transactionRepo.ByWorkspace[2]))
	}
}

func TestGenerateRecurringTransactions_CopiesCategory(t *testing.T) {
	service, recurringRepo, _, _, _ := setupRecurringServiceTest()

	workspaceID := int32(1)
	categoryID := int32(5)

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Rent with Category",
		Amount:      decimal.NewFromFloat(1200.00),
		AccountID:   1,
		Type:        domain.TransactionTypeExpense,
		CategoryID:  &categoryID,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      1,
		IsActive:    true,
	})

	result, err := service.GenerateRecurringTransactions(workspaceID, 2025, time.January)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Generated) != 1 {
		t.Fatalf("Expected 1 generated, got %d", len(result.Generated))
	}

	// Verify category is copied
	if result.Generated[0].CategoryID == nil {
		t.Error("Expected CategoryID to be set")
	} else if *result.Generated[0].CategoryID != categoryID {
		t.Errorf("Expected CategoryID %d, got %d", categoryID, *result.Generated[0].CategoryID)
	}
}

func TestGenerateRecurringTransactions_TransactionIsPaidFalse(t *testing.T) {
	service, recurringRepo, _, _, _ := setupRecurringServiceTest()

	workspaceID := int32(1)

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Rent",
		Amount:      decimal.NewFromFloat(1200.00),
		AccountID:   1,
		Type:        domain.TransactionTypeExpense,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      1,
		IsActive:    true,
	})

	result, err := service.GenerateRecurringTransactions(workspaceID, 2025, time.January)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Generated) != 1 {
		t.Fatalf("Expected 1 generated, got %d", len(result.Generated))
	}

	// Generated transactions should be unpaid
	if result.Generated[0].IsPaid {
		t.Error("Expected IsPaid to be false for generated transaction")
	}
}

// ToggleActive tests

func TestToggleActive_ActiveToInactive(t *testing.T) {
	service, recurringRepo, _, _, _ := setupRecurringServiceTest()

	workspaceID := int32(1)

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Test",
		IsActive:    true, // Start as active
	})

	rt, err := service.ToggleActive(workspaceID, 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rt.IsActive {
		t.Error("Expected IsActive to be false after toggle")
	}
}

func TestToggleActive_InactiveToActive(t *testing.T) {
	service, recurringRepo, _, _, _ := setupRecurringServiceTest()

	workspaceID := int32(1)

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Test",
		IsActive:    false, // Start as inactive
	})

	rt, err := service.ToggleActive(workspaceID, 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !rt.IsActive {
		t.Error("Expected IsActive to be true after toggle")
	}
}

func TestToggleActive_NotFound(t *testing.T) {
	service, _, _, _, _ := setupRecurringServiceTest()

	workspaceID := int32(1)

	_, err := service.ToggleActive(workspaceID, 999)
	if err != domain.ErrRecurringNotFound {
		t.Errorf("Expected ErrRecurringNotFound, got %v", err)
	}
}

func TestToggleActive_WrongWorkspace(t *testing.T) {
	service, recurringRepo, _, _, _ := setupRecurringServiceTest()

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test",
		IsActive:    true,
	})

	// Try to toggle from different workspace
	_, err := service.ToggleActive(2, 1)
	if err != domain.ErrRecurringNotFound {
		t.Errorf("Expected ErrRecurringNotFound for wrong workspace, got %v", err)
	}
}
