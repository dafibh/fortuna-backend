package service

import (
	"testing"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/shopspring/decimal"
)

func TestCreateTransaction_Success(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	// Add account to mock
	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	input := CreateTransactionInput{
		AccountID: accountID,
		Name:      "Groceries",
		Amount:    decimal.NewFromFloat(150.00),
		Type:      domain.TransactionTypeExpense,
	}

	transaction, err := transactionService.CreateTransaction(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if transaction.Name != "Groceries" {
		t.Errorf("Expected name 'Groceries', got %s", transaction.Name)
	}

	if !transaction.Amount.Equal(decimal.NewFromFloat(150.00)) {
		t.Errorf("Expected amount '150.00', got %s", transaction.Amount.String())
	}

	if transaction.Type != domain.TransactionTypeExpense {
		t.Errorf("Expected type 'expense', got %s", transaction.Type)
	}

	if transaction.WorkspaceID != workspaceID {
		t.Errorf("Expected workspace ID %d, got %d", workspaceID, transaction.WorkspaceID)
	}

	if transaction.AccountID != accountID {
		t.Errorf("Expected account ID %d, got %d", accountID, transaction.AccountID)
	}

	// Default values
	if !transaction.IsPaid {
		t.Error("Expected is_paid to default to true")
	}
}

func TestCreateTransaction_WithCustomDate(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	customDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	input := CreateTransactionInput{
		AccountID:       accountID,
		Name:            "Past Transaction",
		Amount:          decimal.NewFromFloat(100.00),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: &customDate,
	}

	transaction, err := transactionService.CreateTransaction(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !transaction.TransactionDate.Equal(customDate) {
		t.Errorf("Expected date %v, got %v", customDate, transaction.TransactionDate)
	}
}

func TestCreateTransaction_WithIsPaidFalse(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	isPaid := false
	input := CreateTransactionInput{
		AccountID: accountID,
		Name:      "Unpaid Bill",
		Amount:    decimal.NewFromFloat(200.00),
		Type:      domain.TransactionTypeExpense,
		IsPaid:    &isPaid,
	}

	transaction, err := transactionService.CreateTransaction(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if transaction.IsPaid {
		t.Error("Expected is_paid to be false")
	}
}

func TestCreateTransaction_WithNotes(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	notes := "Weekly shopping"
	input := CreateTransactionInput{
		AccountID: accountID,
		Name:      "Groceries",
		Amount:    decimal.NewFromFloat(150.00),
		Type:      domain.TransactionTypeExpense,
		Notes:     &notes,
	}

	transaction, err := transactionService.CreateTransaction(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if transaction.Notes == nil || *transaction.Notes != "Weekly shopping" {
		t.Errorf("Expected notes 'Weekly shopping', got %v", transaction.Notes)
	}
}

func TestCreateTransaction_IncomeType(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	input := CreateTransactionInput{
		AccountID: accountID,
		Name:      "Salary",
		Amount:    decimal.NewFromFloat(5000.00),
		Type:      domain.TransactionTypeIncome,
	}

	transaction, err := transactionService.CreateTransaction(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if transaction.Type != domain.TransactionTypeIncome {
		t.Errorf("Expected type 'income', got %s", transaction.Type)
	}
}

func TestCreateTransaction_EmptyName(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	input := CreateTransactionInput{
		AccountID: accountID,
		Name:      "",
		Amount:    decimal.NewFromFloat(100.00),
		Type:      domain.TransactionTypeExpense,
	}

	_, err := transactionService.CreateTransaction(workspaceID, input)
	if err != domain.ErrNameRequired {
		t.Errorf("Expected ErrNameRequired, got %v", err)
	}
}

func TestCreateTransaction_WhitespaceOnlyName(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	input := CreateTransactionInput{
		AccountID: accountID,
		Name:      "   ",
		Amount:    decimal.NewFromFloat(100.00),
		Type:      domain.TransactionTypeExpense,
	}

	_, err := transactionService.CreateTransaction(workspaceID, input)
	if err != domain.ErrNameRequired {
		t.Errorf("Expected ErrNameRequired, got %v", err)
	}
}

func TestCreateTransaction_NameTooLong(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	longName := ""
	for i := 0; i < 256; i++ {
		longName += "a"
	}

	input := CreateTransactionInput{
		AccountID: accountID,
		Name:      longName,
		Amount:    decimal.NewFromFloat(100.00),
		Type:      domain.TransactionTypeExpense,
	}

	_, err := transactionService.CreateTransaction(workspaceID, input)
	if err != domain.ErrNameTooLong {
		t.Errorf("Expected ErrNameTooLong, got %v", err)
	}
}

func TestCreateTransaction_ZeroAmount(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	input := CreateTransactionInput{
		AccountID: accountID,
		Name:      "Zero Transaction",
		Amount:    decimal.Zero,
		Type:      domain.TransactionTypeExpense,
	}

	_, err := transactionService.CreateTransaction(workspaceID, input)
	if err != domain.ErrInvalidAmount {
		t.Errorf("Expected ErrInvalidAmount, got %v", err)
	}
}

func TestCreateTransaction_NegativeAmount(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	input := CreateTransactionInput{
		AccountID: accountID,
		Name:      "Negative Transaction",
		Amount:    decimal.NewFromFloat(-100.00),
		Type:      domain.TransactionTypeExpense,
	}

	_, err := transactionService.CreateTransaction(workspaceID, input)
	if err != domain.ErrInvalidAmount {
		t.Errorf("Expected ErrInvalidAmount, got %v", err)
	}
}

func TestCreateTransaction_InvalidType(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	input := CreateTransactionInput{
		AccountID: accountID,
		Name:      "Invalid Type Transaction",
		Amount:    decimal.NewFromFloat(100.00),
		Type:      domain.TransactionType("invalid"),
	}

	_, err := transactionService.CreateTransaction(workspaceID, input)
	if err != domain.ErrInvalidTransactionType {
		t.Errorf("Expected ErrInvalidTransactionType, got %v", err)
	}
}

func TestCreateTransaction_AccountNotFound(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)

	input := CreateTransactionInput{
		AccountID: 999, // Non-existent account
		Name:      "Transaction",
		Amount:    decimal.NewFromFloat(100.00),
		Type:      domain.TransactionTypeExpense,
	}

	_, err := transactionService.CreateTransaction(workspaceID, input)
	if err != domain.ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound, got %v", err)
	}
}

func TestCreateTransaction_AccountWrongWorkspace(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	// Account belongs to workspace 1
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test Account",
	})

	// Try to create transaction in workspace 2 with account from workspace 1
	input := CreateTransactionInput{
		AccountID: 1,
		Name:      "Transaction",
		Amount:    decimal.NewFromFloat(100.00),
		Type:      domain.TransactionTypeExpense,
	}

	_, err := transactionService.CreateTransaction(2, input)
	if err != domain.ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound for wrong workspace, got %v", err)
	}
}

func TestCreateTransaction_TrimsName(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	input := CreateTransactionInput{
		AccountID: accountID,
		Name:      "  Groceries  ",
		Amount:    decimal.NewFromFloat(150.00),
		Type:      domain.TransactionTypeExpense,
	}

	transaction, err := transactionService.CreateTransaction(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if transaction.Name != "Groceries" {
		t.Errorf("Expected trimmed name 'Groceries', got '%s'", transaction.Name)
	}
}

func TestCreateTransaction_TrimsNotes(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	notes := "  Weekly shopping  "
	input := CreateTransactionInput{
		AccountID: accountID,
		Name:      "Groceries",
		Amount:    decimal.NewFromFloat(150.00),
		Type:      domain.TransactionTypeExpense,
		Notes:     &notes,
	}

	transaction, err := transactionService.CreateTransaction(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if transaction.Notes == nil || *transaction.Notes != "Weekly shopping" {
		t.Errorf("Expected trimmed notes 'Weekly shopping', got '%v'", transaction.Notes)
	}
}

func TestCreateTransaction_EmptyNotesBecomesNil(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	notes := "   " // Whitespace only
	input := CreateTransactionInput{
		AccountID: accountID,
		Name:      "Groceries",
		Amount:    decimal.NewFromFloat(150.00),
		Type:      domain.TransactionTypeExpense,
		Notes:     &notes,
	}

	transaction, err := transactionService.CreateTransaction(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if transaction.Notes != nil {
		t.Errorf("Expected nil notes for whitespace-only input, got '%s'", *transaction.Notes)
	}
}

func TestCreateTransaction_NotesTooLong(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	// Create notes longer than 1000 characters
	longNotes := ""
	for i := 0; i < 1001; i++ {
		longNotes += "a"
	}

	input := CreateTransactionInput{
		AccountID: accountID,
		Name:      "Groceries",
		Amount:    decimal.NewFromFloat(150.00),
		Type:      domain.TransactionTypeExpense,
		Notes:     &longNotes,
	}

	_, err := transactionService.CreateTransaction(workspaceID, input)
	if err != domain.ErrNotesTooLong {
		t.Errorf("Expected ErrNotesTooLong, got %v", err)
	}
}

func TestGetTransactions_Success(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)

	// Add transactions
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          1,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Transaction 1",
		Amount:      decimal.NewFromFloat(100.00),
		Type:        domain.TransactionTypeExpense,
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          2,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Transaction 2",
		Amount:      decimal.NewFromFloat(200.00),
		Type:        domain.TransactionTypeIncome,
	})

	result, err := transactionService.GetTransactions(workspaceID, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Data) != 2 {
		t.Errorf("Expected 2 transactions, got %d", len(result.Data))
	}

	if result.TotalItems != 2 {
		t.Errorf("Expected totalItems 2, got %d", result.TotalItems)
	}
}

func TestGetTransactions_EmptyList(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)

	result, err := transactionService.GetTransactions(workspaceID, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Data) != 0 {
		t.Errorf("Expected 0 transactions, got %d", len(result.Data))
	}

	if result.TotalItems != 0 {
		t.Errorf("Expected totalItems 0, got %d", result.TotalItems)
	}
}

func TestGetTransactionByID_Success(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	transactionID := int32(1)

	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          transactionID,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Test Transaction",
		Amount:      decimal.NewFromFloat(100.00),
		Type:        domain.TransactionTypeExpense,
	})

	transaction, err := transactionService.GetTransactionByID(workspaceID, transactionID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if transaction.Name != "Test Transaction" {
		t.Errorf("Expected name 'Test Transaction', got %s", transaction.Name)
	}
}

func TestGetTransactionByID_NotFound(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)

	_, err := transactionService.GetTransactionByID(workspaceID, 999)
	if err != domain.ErrTransactionNotFound {
		t.Errorf("Expected ErrTransactionNotFound, got %v", err)
	}
}

func TestGetTransactionByID_WrongWorkspace(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	// Transaction belongs to workspace 1
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          1,
		WorkspaceID: 1,
		AccountID:   1,
		Name:        "Test Transaction",
		Amount:      decimal.NewFromFloat(100.00),
		Type:        domain.TransactionTypeExpense,
	})

	// Try to get it from workspace 2
	_, err := transactionService.GetTransactionByID(2, 1)
	if err != domain.ErrTransactionNotFound {
		t.Errorf("Expected ErrTransactionNotFound for wrong workspace, got %v", err)
	}
}

func TestTogglePaidStatus_PaidToUnpaid(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	transactionID := int32(1)

	// Add a paid transaction
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          transactionID,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Paid Transaction",
		Amount:      decimal.NewFromFloat(100.00),
		Type:        domain.TransactionTypeExpense,
		IsPaid:      true,
	})

	transaction, err := transactionService.TogglePaidStatus(workspaceID, transactionID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if transaction.IsPaid {
		t.Error("Expected is_paid to be false after toggle")
	}
}

func TestTogglePaidStatus_UnpaidToPaid(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	transactionID := int32(1)

	// Add an unpaid transaction
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          transactionID,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Unpaid Transaction",
		Amount:      decimal.NewFromFloat(100.00),
		Type:        domain.TransactionTypeExpense,
		IsPaid:      false,
	})

	transaction, err := transactionService.TogglePaidStatus(workspaceID, transactionID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !transaction.IsPaid {
		t.Error("Expected is_paid to be true after toggle")
	}
}

func TestTogglePaidStatus_NotFound(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)

	_, err := transactionService.TogglePaidStatus(workspaceID, 999)
	if err != domain.ErrTransactionNotFound {
		t.Errorf("Expected ErrTransactionNotFound, got %v", err)
	}
}

func TestTogglePaidStatus_WrongWorkspace(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	// Transaction belongs to workspace 1
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          1,
		WorkspaceID: 1,
		AccountID:   1,
		Name:        "Test Transaction",
		Amount:      decimal.NewFromFloat(100.00),
		Type:        domain.TransactionTypeExpense,
		IsPaid:      true,
	})

	// Try to toggle from workspace 2
	_, err := transactionService.TogglePaidStatus(2, 1)
	if err != domain.ErrTransactionNotFound {
		t.Errorf("Expected ErrTransactionNotFound for wrong workspace, got %v", err)
	}
}

func TestUpdateSettlementIntent_Success(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)
	transactionID := int32(1)

	// Add credit card account
	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Credit Card",
		Template:    domain.TemplateCreditCard,
	})

	// Add unpaid CC transaction with initial intent
	thisMonth := domain.CCSettlementThisMonth
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:                 transactionID,
		WorkspaceID:        workspaceID,
		AccountID:          accountID,
		Name:               "Online Purchase",
		Amount:             decimal.NewFromFloat(250.00),
		Type:               domain.TransactionTypeExpense,
		IsPaid:             false,
		CCSettlementIntent: &thisMonth,
	})

	// Update to next_month
	transaction, err := transactionService.UpdateSettlementIntent(workspaceID, transactionID, domain.CCSettlementNextMonth)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if transaction.CCSettlementIntent == nil || *transaction.CCSettlementIntent != domain.CCSettlementNextMonth {
		t.Errorf("Expected settlement intent 'next_month', got %v", transaction.CCSettlementIntent)
	}
}

func TestUpdateSettlementIntent_InvalidIntent(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)

	_, err := transactionService.UpdateSettlementIntent(workspaceID, 1, domain.CCSettlementIntent("invalid"))
	if err != domain.ErrInvalidSettlementIntent {
		t.Errorf("Expected ErrInvalidSettlementIntent, got %v", err)
	}
}

func TestUpdateSettlementIntent_TransactionNotFound(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)

	_, err := transactionService.UpdateSettlementIntent(workspaceID, 999, domain.CCSettlementNextMonth)
	if err != domain.ErrTransactionNotFound {
		t.Errorf("Expected ErrTransactionNotFound, got %v", err)
	}
}

func TestUpdateSettlementIntent_TransactionAlreadyPaid(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)
	transactionID := int32(1)

	// Add credit card account
	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Credit Card",
		Template:    domain.TemplateCreditCard,
	})

	// Add PAID CC transaction
	thisMonth := domain.CCSettlementThisMonth
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:                 transactionID,
		WorkspaceID:        workspaceID,
		AccountID:          accountID,
		Name:               "Paid Purchase",
		Amount:             decimal.NewFromFloat(100.00),
		Type:               domain.TransactionTypeExpense,
		IsPaid:             true, // Already paid
		CCSettlementIntent: &thisMonth,
	})

	_, err := transactionService.UpdateSettlementIntent(workspaceID, transactionID, domain.CCSettlementNextMonth)
	if err != domain.ErrTransactionAlreadyPaid {
		t.Errorf("Expected ErrTransactionAlreadyPaid, got %v", err)
	}
}

func TestUpdateSettlementIntent_NotCreditCardAccount(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)
	transactionID := int32(1)

	// Add bank account (not credit card)
	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Checking Account",
		Template:    domain.TemplateBank,
	})

	// Add transaction for bank account
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          transactionID,
		WorkspaceID: workspaceID,
		AccountID:   accountID,
		Name:        "Bank Transaction",
		Amount:      decimal.NewFromFloat(100.00),
		Type:        domain.TransactionTypeExpense,
		IsPaid:      false,
	})

	_, err := transactionService.UpdateSettlementIntent(workspaceID, transactionID, domain.CCSettlementNextMonth)
	if err != domain.ErrSettlementIntentNotApplicable {
		t.Errorf("Expected ErrSettlementIntentNotApplicable, got %v", err)
	}
}

func TestUpdateSettlementIntent_WrongWorkspace(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	// Transaction belongs to workspace 1
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Credit Card",
		Template:    domain.TemplateCreditCard,
	})

	thisMonth := domain.CCSettlementThisMonth
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:                 1,
		WorkspaceID:        1,
		AccountID:          1,
		Name:               "Test Transaction",
		Amount:             decimal.NewFromFloat(100.00),
		Type:               domain.TransactionTypeExpense,
		IsPaid:             false,
		CCSettlementIntent: &thisMonth,
	})

	// Try to update from workspace 2
	_, err := transactionService.UpdateSettlementIntent(2, 1, domain.CCSettlementNextMonth)
	if err != domain.ErrTransactionNotFound {
		t.Errorf("Expected ErrTransactionNotFound for wrong workspace, got %v", err)
	}
}

func TestCreateTransaction_CCAccountDefaultsToThisMonth(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	// Add credit card account
	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Credit Card",
		Template:    domain.TemplateCreditCard,
	})

	input := CreateTransactionInput{
		AccountID: accountID,
		Name:      "Online Purchase",
		Amount:    decimal.NewFromFloat(250.00),
		Type:      domain.TransactionTypeExpense,
		// CCSettlementIntent not provided - should default to 'this_month'
	}

	transaction, err := transactionService.CreateTransaction(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if transaction.CCSettlementIntent == nil {
		t.Fatal("Expected CCSettlementIntent to be set for CC account")
	}

	if *transaction.CCSettlementIntent != domain.CCSettlementThisMonth {
		t.Errorf("Expected settlement intent 'this_month', got %s", *transaction.CCSettlementIntent)
	}
}

func TestCreateTransaction_CCAccountWithExplicitIntent(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	// Add credit card account
	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Credit Card",
		Template:    domain.TemplateCreditCard,
	})

	nextMonth := domain.CCSettlementNextMonth
	input := CreateTransactionInput{
		AccountID:          accountID,
		Name:               "Online Purchase",
		Amount:             decimal.NewFromFloat(250.00),
		Type:               domain.TransactionTypeExpense,
		CCSettlementIntent: &nextMonth,
	}

	transaction, err := transactionService.CreateTransaction(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if transaction.CCSettlementIntent == nil {
		t.Fatal("Expected CCSettlementIntent to be set for CC account")
	}

	if *transaction.CCSettlementIntent != domain.CCSettlementNextMonth {
		t.Errorf("Expected settlement intent 'next_month', got %s", *transaction.CCSettlementIntent)
	}
}

func TestCreateTransaction_NonCCAccountHasNilSettlementIntent(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	// Add bank account (non-CC)
	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Checking Account",
		Template:    domain.TemplateBank,
	})

	input := CreateTransactionInput{
		AccountID: accountID,
		Name:      "Bank Transaction",
		Amount:    decimal.NewFromFloat(100.00),
		Type:      domain.TransactionTypeExpense,
	}

	transaction, err := transactionService.CreateTransaction(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if transaction.CCSettlementIntent != nil {
		t.Errorf("Expected nil CCSettlementIntent for non-CC account, got %s", *transaction.CCSettlementIntent)
	}
}

func TestCreateTransaction_NonCCAccountIgnoresProvidedIntent(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	// Add bank account (non-CC)
	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Checking Account",
		Template:    domain.TemplateBank,
	})

	// Try to provide settlement intent for non-CC account - should be ignored
	thisMonth := domain.CCSettlementThisMonth
	input := CreateTransactionInput{
		AccountID:          accountID,
		Name:               "Bank Transaction",
		Amount:             decimal.NewFromFloat(100.00),
		Type:               domain.TransactionTypeExpense,
		CCSettlementIntent: &thisMonth,
	}

	transaction, err := transactionService.CreateTransaction(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if transaction.CCSettlementIntent != nil {
		t.Errorf("Expected nil CCSettlementIntent for non-CC account even when provided, got %s", *transaction.CCSettlementIntent)
	}
}

func TestCreateTransaction_CCAccountWithInvalidIntent(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	// Add credit card account
	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Credit Card",
		Template:    domain.TemplateCreditCard,
	})

	invalidIntent := domain.CCSettlementIntent("invalid")
	input := CreateTransactionInput{
		AccountID:          accountID,
		Name:               "Online Purchase",
		Amount:             decimal.NewFromFloat(250.00),
		Type:               domain.TransactionTypeExpense,
		CCSettlementIntent: &invalidIntent,
	}

	_, err := transactionService.CreateTransaction(workspaceID, input)
	if err != domain.ErrInvalidSettlementIntent {
		t.Errorf("Expected ErrInvalidSettlementIntent, got %v", err)
	}
}

// ============================================
// Transfer Tests
// ============================================

func TestCreateTransfer_Success(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)

	// Add source and destination accounts
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Checking Account",
		Template:    domain.TemplateBank,
	})
	accountRepo.AddAccount(&domain.Account{
		ID:          2,
		WorkspaceID: workspaceID,
		Name:        "Savings Account",
		Template:    domain.TemplateBank,
	})

	date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	notes := "Monthly savings transfer"
	input := CreateTransferInput{
		FromAccountID: 1,
		ToAccountID:   2,
		Amount:        decimal.NewFromFloat(500.00),
		Date:          date,
		Notes:         &notes,
	}

	result, err := transactionService.CreateTransfer(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Validate from transaction (expense)
	if result.FromTransaction.Type != domain.TransactionTypeExpense {
		t.Errorf("Expected from transaction type 'expense', got %s", result.FromTransaction.Type)
	}
	if result.FromTransaction.AccountID != 1 {
		t.Errorf("Expected from account ID 1, got %d", result.FromTransaction.AccountID)
	}
	if !result.FromTransaction.Amount.Equal(decimal.NewFromFloat(500.00)) {
		t.Errorf("Expected amount 500.00, got %s", result.FromTransaction.Amount.String())
	}
	if result.FromTransaction.Name != "Transfer to Savings Account" {
		t.Errorf("Expected name 'Transfer to Savings Account', got %s", result.FromTransaction.Name)
	}

	// Validate to transaction (income)
	if result.ToTransaction.Type != domain.TransactionTypeIncome {
		t.Errorf("Expected to transaction type 'income', got %s", result.ToTransaction.Type)
	}
	if result.ToTransaction.AccountID != 2 {
		t.Errorf("Expected to account ID 2, got %d", result.ToTransaction.AccountID)
	}
	if result.ToTransaction.Name != "Transfer from Checking Account" {
		t.Errorf("Expected name 'Transfer from Checking Account', got %s", result.ToTransaction.Name)
	}

	// Both should have same transfer pair ID
	if result.FromTransaction.TransferPairID == nil || result.ToTransaction.TransferPairID == nil {
		t.Fatal("Expected both transactions to have transfer pair ID")
	}
	if *result.FromTransaction.TransferPairID != *result.ToTransaction.TransferPairID {
		t.Error("Expected both transactions to have the same transfer pair ID")
	}

	// Both should be marked as paid
	if !result.FromTransaction.IsPaid || !result.ToTransaction.IsPaid {
		t.Error("Expected both transactions to be marked as paid")
	}
}

func TestCreateTransfer_SameAccountError(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Checking Account",
		Template:    domain.TemplateBank,
	})

	input := CreateTransferInput{
		FromAccountID: 1,
		ToAccountID:   1, // Same account
		Amount:        decimal.NewFromFloat(500.00),
	}

	_, err := transactionService.CreateTransfer(workspaceID, input)
	if err != domain.ErrSameAccountTransfer {
		t.Errorf("Expected ErrSameAccountTransfer, got %v", err)
	}
}

func TestCreateTransfer_SourceAccountNotFound(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)

	// Only add destination account
	accountRepo.AddAccount(&domain.Account{
		ID:          2,
		WorkspaceID: workspaceID,
		Name:        "Savings Account",
		Template:    domain.TemplateBank,
	})

	input := CreateTransferInput{
		FromAccountID: 1, // Does not exist
		ToAccountID:   2,
		Amount:        decimal.NewFromFloat(500.00),
	}

	_, err := transactionService.CreateTransfer(workspaceID, input)
	if err != domain.ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound, got %v", err)
	}
}

func TestCreateTransfer_DestinationAccountNotFound(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)

	// Only add source account
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Checking Account",
		Template:    domain.TemplateBank,
	})

	input := CreateTransferInput{
		FromAccountID: 1,
		ToAccountID:   2, // Does not exist
		Amount:        decimal.NewFromFloat(500.00),
	}

	_, err := transactionService.CreateTransfer(workspaceID, input)
	if err != domain.ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound, got %v", err)
	}
}

func TestCreateTransfer_ZeroAmount(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Checking Account",
		Template:    domain.TemplateBank,
	})
	accountRepo.AddAccount(&domain.Account{
		ID:          2,
		WorkspaceID: workspaceID,
		Name:        "Savings Account",
		Template:    domain.TemplateBank,
	})

	input := CreateTransferInput{
		FromAccountID: 1,
		ToAccountID:   2,
		Amount:        decimal.Zero,
	}

	_, err := transactionService.CreateTransfer(workspaceID, input)
	if err != domain.ErrInvalidAmount {
		t.Errorf("Expected ErrInvalidAmount, got %v", err)
	}
}

func TestCreateTransfer_NegativeAmount(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Checking Account",
		Template:    domain.TemplateBank,
	})
	accountRepo.AddAccount(&domain.Account{
		ID:          2,
		WorkspaceID: workspaceID,
		Name:        "Savings Account",
		Template:    domain.TemplateBank,
	})

	input := CreateTransferInput{
		FromAccountID: 1,
		ToAccountID:   2,
		Amount:        decimal.NewFromFloat(-100.00),
	}

	_, err := transactionService.CreateTransfer(workspaceID, input)
	if err != domain.ErrInvalidAmount {
		t.Errorf("Expected ErrInvalidAmount, got %v", err)
	}
}

func TestCreateTransfer_UsesProvidedDate(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Checking Account",
		Template:    domain.TemplateBank,
	})
	accountRepo.AddAccount(&domain.Account{
		ID:          2,
		WorkspaceID: workspaceID,
		Name:        "Savings Account",
		Template:    domain.TemplateBank,
	})

	customDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	input := CreateTransferInput{
		FromAccountID: 1,
		ToAccountID:   2,
		Amount:        decimal.NewFromFloat(500.00),
		Date:          customDate,
	}

	result, err := transactionService.CreateTransfer(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !result.FromTransaction.TransactionDate.Equal(customDate) {
		t.Errorf("Expected date %v, got %v", customDate, result.FromTransaction.TransactionDate)
	}
	if !result.ToTransaction.TransactionDate.Equal(customDate) {
		t.Errorf("Expected date %v, got %v", customDate, result.ToTransaction.TransactionDate)
	}
}

func TestCreateTransfer_WorkspaceIsolation(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	// Accounts in workspace 1
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Checking Account",
		Template:    domain.TemplateBank,
	})
	// Account in workspace 2
	accountRepo.AddAccount(&domain.Account{
		ID:          2,
		WorkspaceID: 2,
		Name:        "Savings Account",
		Template:    domain.TemplateBank,
	})

	// Try to transfer from workspace 1's account to workspace 2's account
	// Should fail because we're in workspace 1
	input := CreateTransferInput{
		FromAccountID: 1,
		ToAccountID:   2,
		Amount:        decimal.NewFromFloat(500.00),
	}

	_, err := transactionService.CreateTransfer(1, input)
	if err != domain.ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound for cross-workspace transfer, got %v", err)
	}
}

// ============================================================
// Category Assignment Tests (Story 4.2)
// ============================================================

func TestCreateTransaction_WithCategory(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)
	categoryID := int32(5)

	// Add account to mock
	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
		Template:    domain.TemplateBank,
	})

	// Add category to mock
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          categoryID,
		WorkspaceID: workspaceID,
		Name:        "Food & Dining",
	})

	input := CreateTransactionInput{
		AccountID:  accountID,
		Name:       "Lunch",
		Amount:     decimal.NewFromFloat(15.00),
		Type:       domain.TransactionTypeExpense,
		CategoryID: &categoryID,
	}

	transaction, err := transactionService.CreateTransaction(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if transaction.CategoryID == nil {
		t.Fatal("Expected CategoryID to be set")
	}
	if *transaction.CategoryID != categoryID {
		t.Errorf("Expected CategoryID %d, got %d", categoryID, *transaction.CategoryID)
	}
}

func TestCreateTransaction_WithoutCategory(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
		Template:    domain.TemplateBank,
	})

	input := CreateTransactionInput{
		AccountID:  accountID,
		Name:       "Uncategorized Expense",
		Amount:     decimal.NewFromFloat(50.00),
		Type:       domain.TransactionTypeExpense,
		CategoryID: nil, // No category
	}

	transaction, err := transactionService.CreateTransaction(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if transaction.CategoryID != nil {
		t.Errorf("Expected CategoryID to be nil, got %d", *transaction.CategoryID)
	}
}

func TestCreateTransaction_WithInvalidCategory(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)
	invalidCategoryID := int32(999) // Non-existent category

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
		Template:    domain.TemplateBank,
	})

	// Don't add category - it should not exist

	input := CreateTransactionInput{
		AccountID:  accountID,
		Name:       "Test",
		Amount:     decimal.NewFromFloat(10.00),
		Type:       domain.TransactionTypeExpense,
		CategoryID: &invalidCategoryID,
	}

	_, err := transactionService.CreateTransaction(workspaceID, input)
	if err != domain.ErrBudgetCategoryNotFound {
		t.Errorf("Expected ErrBudgetCategoryNotFound, got %v", err)
	}
}

func TestCreateTransaction_CategoryFromDifferentWorkspace(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	otherWorkspaceID := int32(2)
	accountID := int32(1)
	categoryID := int32(5)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
		Template:    domain.TemplateBank,
	})

	// Category belongs to a DIFFERENT workspace
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          categoryID,
		WorkspaceID: otherWorkspaceID, // Different workspace!
		Name:        "Other Workspace Category",
	})

	input := CreateTransactionInput{
		AccountID:  accountID,
		Name:       "Test",
		Amount:     decimal.NewFromFloat(10.00),
		Type:       domain.TransactionTypeExpense,
		CategoryID: &categoryID,
	}

	_, err := transactionService.CreateTransaction(workspaceID, input)
	if err != domain.ErrBudgetCategoryNotFound {
		t.Errorf("Expected ErrBudgetCategoryNotFound for category from different workspace, got %v", err)
	}
}

func TestUpdateTransaction_AddCategory(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)
	transactionID := int32(10)
	categoryID := int32(5)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
		Template:    domain.TemplateBank,
	})

	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          categoryID,
		WorkspaceID: workspaceID,
		Name:        "Food & Dining",
	})

	// Add transaction without category
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              transactionID,
		WorkspaceID:     workspaceID,
		AccountID:       accountID,
		Name:            "Lunch",
		Amount:          decimal.NewFromFloat(15.00),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: time.Now(),
		IsPaid:          true,
		CategoryID:      nil,
	})

	input := UpdateTransactionInput{
		AccountID:       accountID,
		Name:            "Lunch",
		Amount:          decimal.NewFromFloat(15.00),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: time.Now(),
		CategoryID:      &categoryID, // Adding category
	}

	updated, err := transactionService.UpdateTransaction(workspaceID, transactionID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if updated.CategoryID == nil {
		t.Fatal("Expected CategoryID to be set")
	}
	if *updated.CategoryID != categoryID {
		t.Errorf("Expected CategoryID %d, got %d", categoryID, *updated.CategoryID)
	}
}

func TestUpdateTransaction_RemoveCategory(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)
	transactionID := int32(10)
	categoryID := int32(5)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
		Template:    domain.TemplateBank,
	})

	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          categoryID,
		WorkspaceID: workspaceID,
		Name:        "Food & Dining",
	})

	// Add transaction WITH category
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              transactionID,
		WorkspaceID:     workspaceID,
		AccountID:       accountID,
		Name:            "Lunch",
		Amount:          decimal.NewFromFloat(15.00),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: time.Now(),
		IsPaid:          true,
		CategoryID:      &categoryID,
	})

	input := UpdateTransactionInput{
		AccountID:       accountID,
		Name:            "Lunch",
		Amount:          decimal.NewFromFloat(15.00),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: time.Now(),
		CategoryID:      nil, // Removing category
	}

	updated, err := transactionService.UpdateTransaction(workspaceID, transactionID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if updated.CategoryID != nil {
		t.Errorf("Expected CategoryID to be nil after removal, got %d", *updated.CategoryID)
	}
}

func TestUpdateTransaction_ChangeCategory(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)
	transactionID := int32(10)
	oldCategoryID := int32(5)
	newCategoryID := int32(6)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
		Template:    domain.TemplateBank,
	})

	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          oldCategoryID,
		WorkspaceID: workspaceID,
		Name:        "Food & Dining",
	})
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          newCategoryID,
		WorkspaceID: workspaceID,
		Name:        "Transportation",
	})

	// Add transaction with old category
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              transactionID,
		WorkspaceID:     workspaceID,
		AccountID:       accountID,
		Name:            "Expense",
		Amount:          decimal.NewFromFloat(25.00),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: time.Now(),
		IsPaid:          true,
		CategoryID:      &oldCategoryID,
	})

	input := UpdateTransactionInput{
		AccountID:       accountID,
		Name:            "Expense",
		Amount:          decimal.NewFromFloat(25.00),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: time.Now(),
		CategoryID:      &newCategoryID, // Changing to new category
	}

	updated, err := transactionService.UpdateTransaction(workspaceID, transactionID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if updated.CategoryID == nil {
		t.Fatal("Expected CategoryID to be set")
	}
	if *updated.CategoryID != newCategoryID {
		t.Errorf("Expected CategoryID %d, got %d", newCategoryID, *updated.CategoryID)
	}
}

func TestUpdateTransaction_InvalidCategory(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)
	transactionID := int32(10)
	invalidCategoryID := int32(999)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
		Template:    domain.TemplateBank,
	})

	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              transactionID,
		WorkspaceID:     workspaceID,
		AccountID:       accountID,
		Name:            "Test",
		Amount:          decimal.NewFromFloat(10.00),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: time.Now(),
		IsPaid:          true,
	})

	input := UpdateTransactionInput{
		AccountID:       accountID,
		Name:            "Test",
		Amount:          decimal.NewFromFloat(10.00),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: time.Now(),
		CategoryID:      &invalidCategoryID,
	}

	_, err := transactionService.UpdateTransaction(workspaceID, transactionID, input)
	if err != domain.ErrBudgetCategoryNotFound {
		t.Errorf("Expected ErrBudgetCategoryNotFound, got %v", err)
	}
}

func TestGetRecentlyUsedCategories(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)

	// Set up mock to return recent categories
	expectedCategories := []*domain.RecentCategory{
		{ID: 1, Name: "Food", LastUsed: time.Now()},
		{ID: 2, Name: "Transport", LastUsed: time.Now().Add(-1 * time.Hour)},
	}
	transactionRepo.GetRecentlyUsedCategoriesFn = func(wsID int32) ([]*domain.RecentCategory, error) {
		if wsID != workspaceID {
			return nil, nil
		}
		return expectedCategories, nil
	}

	categories, err := transactionService.GetRecentlyUsedCategories(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(categories))
	}
	if categories[0].Name != "Food" {
		t.Errorf("Expected first category 'Food', got '%s'", categories[0].Name)
	}
	if categories[1].Name != "Transport" {
		t.Errorf("Expected second category 'Transport', got '%s'", categories[1].Name)
	}
}

// ==================== ON-ACCESS PROJECTION GENERATION TESTS ====================

func TestGetTransactions_OnAccessProjectionGeneration(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	templateRepo := testutil.NewMockRecurringTemplateRepository()

	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	transactionService.SetRecurringTemplateRepository(templateRepo)

	workspaceID := int32(1)

	// Add an active template
	startDate := time.Now().AddDate(0, 1, 0) // Next month
	templateRepo.AddTemplate(&domain.RecurringTemplate{
		ID:          1,
		WorkspaceID: workspaceID,
		Description: "Monthly Bill",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  1,
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   startDate,
	})

	// Request transactions for a future month (beyond the 12-month default generation)
	futureDate := time.Now().AddDate(0, 18, 0) // 18 months in the future
	filters := &domain.TransactionFilters{
		EndDate: &futureDate,
	}

	// This should trigger on-access projection generation
	_, err := transactionService.GetTransactions(workspaceID, filters)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify projections were created
	projections, err := transactionRepo.GetProjectionsByTemplate(workspaceID, 1)
	if err != nil {
		t.Fatalf("Failed to get projections: %v", err)
	}

	// Should have projections created
	if len(projections) == 0 {
		t.Errorf("Expected projections to be created on-access, got 0")
	}
}

func TestGetTransactions_NoProjectionsForPastDates(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	templateRepo := testutil.NewMockRecurringTemplateRepository()

	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	transactionService.SetRecurringTemplateRepository(templateRepo)

	workspaceID := int32(1)

	// Add an active template
	startDate := time.Now().AddDate(0, 1, 0)
	templateRepo.AddTemplate(&domain.RecurringTemplate{
		ID:          1,
		WorkspaceID: workspaceID,
		Description: "Monthly Bill",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  1,
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   startDate,
	})

	// Request transactions for a past month
	pastDate := time.Now().AddDate(0, -2, 0)
	filters := &domain.TransactionFilters{
		EndDate: &pastDate,
	}

	// This should NOT trigger projection generation (past dates)
	_, err := transactionService.GetTransactions(workspaceID, filters)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify no projections were created
	projections, err := transactionRepo.GetProjectionsByTemplate(workspaceID, 1)
	if err != nil {
		t.Fatalf("Failed to get projections: %v", err)
	}

	if len(projections) != 0 {
		t.Errorf("Expected no projections for past dates, got %d", len(projections))
	}
}

func TestGetTransactions_RespectsTemplateEndDate(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	templateRepo := testutil.NewMockRecurringTemplateRepository()

	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	transactionService.SetRecurringTemplateRepository(templateRepo)

	workspaceID := int32(1)

	// Add a template with end_date 3 months from now
	startDate := time.Now().AddDate(0, 1, 0)
	endDate := startDate.AddDate(0, 2, 0)
	templateRepo.AddTemplate(&domain.RecurringTemplate{
		ID:          1,
		WorkspaceID: workspaceID,
		Description: "Short Term Bill",
		Amount:      decimal.NewFromInt(100),
		CategoryID:  1,
		AccountID:   1,
		Frequency:   "monthly",
		StartDate:   startDate,
		EndDate:     &endDate,
	})

	// Request transactions for a date beyond template end_date
	futureDate := time.Now().AddDate(0, 12, 0)
	filters := &domain.TransactionFilters{
		EndDate: &futureDate,
	}

	_, err := transactionService.GetTransactions(workspaceID, filters)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify projections don't go beyond end_date
	projections, err := transactionRepo.GetProjectionsByTemplate(workspaceID, 1)
	if err != nil {
		t.Fatalf("Failed to get projections: %v", err)
	}

	for _, proj := range projections {
		if proj.TransactionDate.After(endDate) {
			t.Errorf("Projection date %v should not be after template end_date %v",
				proj.TransactionDate, endDate)
		}
	}
}

func TestGetTransactions_WithoutTemplateRepo_NoError(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()

	// Don't set template repo
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)

	// Request transactions for a future month
	futureDate := time.Now().AddDate(0, 6, 0)
	filters := &domain.TransactionFilters{
		EndDate: &futureDate,
	}

	// Should not error even without template repo
	_, err := transactionService.GetTransactions(workspaceID, filters)
	if err != nil {
		t.Errorf("Expected no error without template repo, got %v", err)
	}
}

// ==================== CC LIFECYCLE TESTS (Story 4.1) ====================

func TestCreateTransaction_CCAccount_DefaultsToPendingDeferred(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	// Add credit card account
	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Credit Card",
		Template:    domain.TemplateCreditCard,
	})

	input := CreateTransactionInput{
		AccountID: accountID,
		Name:      "Online Purchase",
		Amount:    decimal.NewFromFloat(250.00),
		Type:      domain.TransactionTypeExpense,
	}

	transaction, err := transactionService.CreateTransaction(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// CC transaction should default to pending state
	if transaction.CCState == nil {
		t.Fatal("Expected CCState to be set for CC account")
	}
	if *transaction.CCState != domain.CCStatePending {
		t.Errorf("Expected CCState 'pending', got %s", *transaction.CCState)
	}

	// CC transaction should default to deferred settlement intent
	if transaction.SettlementIntent == nil {
		t.Fatal("Expected SettlementIntent to be set for CC account")
	}
	if *transaction.SettlementIntent != domain.SettlementIntentDeferred {
		t.Errorf("Expected SettlementIntent 'deferred', got %s", *transaction.SettlementIntent)
	}

	// BilledAt and SettledAt should be nil for pending transactions
	if transaction.BilledAt != nil {
		t.Errorf("Expected BilledAt to be nil for pending transaction, got %v", transaction.BilledAt)
	}
	if transaction.SettledAt != nil {
		t.Errorf("Expected SettledAt to be nil for pending transaction, got %v", transaction.SettledAt)
	}
}

func TestCreateTransaction_NonCCAccount_NullCCFields(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	// Add bank account (non-CC)
	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Checking Account",
		Template:    domain.TemplateBank,
	})

	input := CreateTransactionInput{
		AccountID: accountID,
		Name:      "Bank Expense",
		Amount:    decimal.NewFromFloat(100.00),
		Type:      domain.TransactionTypeExpense,
	}

	transaction, err := transactionService.CreateTransaction(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Non-CC transaction should have NULL for all CC lifecycle fields
	if transaction.CCState != nil {
		t.Errorf("Expected CCState to be nil for non-CC account, got %s", *transaction.CCState)
	}
	if transaction.SettlementIntent != nil {
		t.Errorf("Expected SettlementIntent to be nil for non-CC account, got %s", *transaction.SettlementIntent)
	}
	if transaction.BilledAt != nil {
		t.Errorf("Expected BilledAt to be nil for non-CC account, got %v", transaction.BilledAt)
	}
	if transaction.SettledAt != nil {
		t.Errorf("Expected SettledAt to be nil for non-CC account, got %v", transaction.SettledAt)
	}
}

func TestCreateTransaction_CCAccount_ImmediateIntent_SettledWithTimestamp(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	// Add credit card account
	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Credit Card",
		Template:    domain.TemplateCreditCard,
	})

	immediateIntent := domain.SettlementIntentImmediate
	input := CreateTransactionInput{
		AccountID:        accountID,
		Name:             "Immediate Settlement Purchase",
		Amount:           decimal.NewFromFloat(50.00),
		Type:             domain.TransactionTypeExpense,
		SettlementIntent: &immediateIntent,
	}

	beforeCreate := time.Now()
	transaction, err := transactionService.CreateTransaction(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	afterCreate := time.Now()

	// CC with immediate intent should be settled
	if transaction.CCState == nil {
		t.Fatal("Expected CCState to be set")
	}
	if *transaction.CCState != domain.CCStateSettled {
		t.Errorf("Expected CCState 'settled' for immediate intent, got %s", *transaction.CCState)
	}

	// SettlementIntent should be immediate
	if transaction.SettlementIntent == nil {
		t.Fatal("Expected SettlementIntent to be set")
	}
	if *transaction.SettlementIntent != domain.SettlementIntentImmediate {
		t.Errorf("Expected SettlementIntent 'immediate', got %s", *transaction.SettlementIntent)
	}

	// SettledAt should be set with current timestamp
	if transaction.SettledAt == nil {
		t.Fatal("Expected SettledAt to be set for immediate settlement")
	}
	if transaction.SettledAt.Before(beforeCreate) || transaction.SettledAt.After(afterCreate) {
		t.Errorf("Expected SettledAt to be between %v and %v, got %v", beforeCreate, afterCreate, transaction.SettledAt)
	}

	// BilledAt should be nil (skipped billed state)
	if transaction.BilledAt != nil {
		t.Errorf("Expected BilledAt to be nil for immediate settlement, got %v", transaction.BilledAt)
	}
}

func TestToggleBilled_PendingToBilled(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	transactionID := int32(1)

	// Add pending CC transaction
	pendingState := domain.CCStatePending
	deferredIntent := domain.SettlementIntentDeferred
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:               transactionID,
		WorkspaceID:      workspaceID,
		AccountID:        1,
		Name:             "CC Purchase",
		Amount:           decimal.NewFromFloat(100.00),
		Type:             domain.TransactionTypeExpense,
		CCState:          &pendingState,
		SettlementIntent: &deferredIntent,
	})

	beforeToggle := time.Now()
	transaction, err := transactionService.ToggleBilled(workspaceID, transactionID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	afterToggle := time.Now()

	// Should be billed now
	if transaction.CCState == nil {
		t.Fatal("Expected CCState to be set")
	}
	if *transaction.CCState != domain.CCStateBilled {
		t.Errorf("Expected CCState 'billed' after toggle, got %s", *transaction.CCState)
	}

	// BilledAt should be set
	if transaction.BilledAt == nil {
		t.Fatal("Expected BilledAt to be set")
	}
	if transaction.BilledAt.Before(beforeToggle) || transaction.BilledAt.After(afterToggle) {
		t.Errorf("Expected BilledAt to be between %v and %v, got %v", beforeToggle, afterToggle, transaction.BilledAt)
	}
}

func TestToggleBilled_BilledToPending(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	transactionID := int32(1)

	// Add billed CC transaction
	billedState := domain.CCStateBilled
	deferredIntent := domain.SettlementIntentDeferred
	billedAt := time.Now().Add(-24 * time.Hour)
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:               transactionID,
		WorkspaceID:      workspaceID,
		AccountID:        1,
		Name:             "CC Purchase",
		Amount:           decimal.NewFromFloat(100.00),
		Type:             domain.TransactionTypeExpense,
		CCState:          &billedState,
		SettlementIntent: &deferredIntent,
		BilledAt:         &billedAt,
	})

	transaction, err := transactionService.ToggleBilled(workspaceID, transactionID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should be pending now
	if transaction.CCState == nil {
		t.Fatal("Expected CCState to be set")
	}
	if *transaction.CCState != domain.CCStatePending {
		t.Errorf("Expected CCState 'pending' after toggle back, got %s", *transaction.CCState)
	}

	// BilledAt should be cleared
	if transaction.BilledAt != nil {
		t.Errorf("Expected BilledAt to be nil after toggling back to pending, got %v", transaction.BilledAt)
	}
}

func TestToggleBilled_NotCCTransaction_Error(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	transactionID := int32(1)

	// Add non-CC transaction (CCState is nil)
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          transactionID,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Bank Transaction",
		Amount:      decimal.NewFromFloat(100.00),
		Type:        domain.TransactionTypeExpense,
		CCState:     nil, // Not a CC transaction
	})

	_, err := transactionService.ToggleBilled(workspaceID, transactionID)
	if err != domain.ErrNotCCTransaction {
		t.Errorf("Expected ErrNotCCTransaction, got %v", err)
	}
}

func TestToggleBilled_SettledTransaction_Error(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	transactionID := int32(1)

	// Add settled CC transaction
	settledState := domain.CCStateSettled
	immediateIntent := domain.SettlementIntentImmediate
	settledAt := time.Now().Add(-24 * time.Hour)
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:               transactionID,
		WorkspaceID:      workspaceID,
		AccountID:        1,
		Name:             "Settled CC Purchase",
		Amount:           decimal.NewFromFloat(100.00),
		Type:             domain.TransactionTypeExpense,
		CCState:          &settledState,
		SettlementIntent: &immediateIntent,
		SettledAt:        &settledAt,
	})

	_, err := transactionService.ToggleBilled(workspaceID, transactionID)
	if err != domain.ErrInvalidCCStateTransition {
		t.Errorf("Expected ErrInvalidCCStateTransition for settled transaction, got %v", err)
	}
}

func TestToggleBilled_TransactionNotFound_Error(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)

	_, err := transactionService.ToggleBilled(workspaceID, 999)
	if err != domain.ErrTransactionNotFound {
		t.Errorf("Expected ErrTransactionNotFound, got %v", err)
	}
}

func TestToggleBilled_WrongWorkspace_Error(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	// Transaction belongs to workspace 1
	pendingState := domain.CCStatePending
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          1,
		WorkspaceID: 1,
		AccountID:   1,
		Name:        "CC Transaction",
		Amount:      decimal.NewFromFloat(100.00),
		Type:        domain.TransactionTypeExpense,
		CCState:     &pendingState,
	})

	// Try to toggle from workspace 2
	_, err := transactionService.ToggleBilled(2, 1)
	if err != domain.ErrTransactionNotFound {
		t.Errorf("Expected ErrTransactionNotFound for wrong workspace, got %v", err)
	}
}

// ========================================
// GetOverdue Tests
// ========================================

func TestGetOverdue_ReturnsEmptyWhenNoOverdue(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)

	groups, err := transactionService.GetOverdue(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups, got %d", len(groups))
	}
}

func TestGetOverdue_GroupsByMonth(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	billedState := domain.CCStateBilled
	deferredIntent := domain.SettlementIntentDeferred

	// Set up overdue transactions in different months (3+ months ago to be safe)
	oct2025 := time.Date(2025, 10, 15, 0, 0, 0, 0, time.UTC)
	nov2025 := time.Date(2025, 11, 10, 0, 0, 0, 0, time.UTC)

	// Use custom mock function to return specific overdue transactions
	transactionRepo.GetOverdueCCFn = func(wsID int32) ([]*domain.Transaction, error) {
		if wsID != workspaceID {
			return []*domain.Transaction{}, nil
		}
		return []*domain.Transaction{
			{
				ID:               1,
				WorkspaceID:      workspaceID,
				Name:             "October Purchase 1",
				Amount:           decimal.NewFromFloat(100.00),
				CCState:          &billedState,
				SettlementIntent: &deferredIntent,
				BilledAt:         &oct2025,
				TransactionDate:  oct2025,
			},
			{
				ID:               2,
				WorkspaceID:      workspaceID,
				Name:             "October Purchase 2",
				Amount:           decimal.NewFromFloat(50.00),
				CCState:          &billedState,
				SettlementIntent: &deferredIntent,
				BilledAt:         &oct2025,
				TransactionDate:  oct2025,
			},
			{
				ID:               3,
				WorkspaceID:      workspaceID,
				Name:             "November Purchase",
				Amount:           decimal.NewFromFloat(75.00),
				CCState:          &billedState,
				SettlementIntent: &deferredIntent,
				BilledAt:         &nov2025,
				TransactionDate:  nov2025,
			},
		}, nil
	}

	groups, err := transactionService.GetOverdue(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(groups) != 2 {
		t.Fatalf("Expected 2 groups (Oct and Nov), got %d", len(groups))
	}

	// First group should be October (oldest first based on order returned by repo)
	oct := groups[0]
	if oct.Month != "2025-10" {
		t.Errorf("Expected first group month '2025-10', got %s", oct.Month)
	}
	if oct.ItemCount != 2 {
		t.Errorf("Expected October group to have 2 items, got %d", oct.ItemCount)
	}
	expectedOctTotal := decimal.NewFromFloat(150.00)
	if !oct.TotalAmount.Equal(expectedOctTotal) {
		t.Errorf("Expected October total '150.00', got %s", oct.TotalAmount.String())
	}

	// Second group should be November
	nov := groups[1]
	if nov.Month != "2025-11" {
		t.Errorf("Expected second group month '2025-11', got %s", nov.Month)
	}
	if nov.ItemCount != 1 {
		t.Errorf("Expected November group to have 1 item, got %d", nov.ItemCount)
	}
}

func TestGetOverdue_CalculatesMonthsOverdue(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo, categoryRepo)

	workspaceID := int32(1)
	billedState := domain.CCStateBilled
	deferredIntent := domain.SettlementIntentDeferred

	// Transaction billed 3 months ago
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)

	transactionRepo.GetOverdueCCFn = func(wsID int32) ([]*domain.Transaction, error) {
		return []*domain.Transaction{
			{
				ID:               1,
				WorkspaceID:      workspaceID,
				Name:             "Old CC Purchase",
				Amount:           decimal.NewFromFloat(200.00),
				CCState:          &billedState,
				SettlementIntent: &deferredIntent,
				BilledAt:         &threeMonthsAgo,
				TransactionDate:  threeMonthsAgo,
			},
		}, nil
	}

	groups, err := transactionService.GetOverdue(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(groups))
	}

	// MonthsOverdue should be approximately 3
	if groups[0].MonthsOverdue < 3 {
		t.Errorf("Expected MonthsOverdue >= 3, got %d", groups[0].MonthsOverdue)
	}
}
