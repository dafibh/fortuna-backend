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
	transactionService := NewTransactionService(transactionRepo, accountRepo)

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
	transactionService := NewTransactionService(transactionRepo, accountRepo)

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
	transactionService := NewTransactionService(transactionRepo, accountRepo)

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
	transactionService := NewTransactionService(transactionRepo, accountRepo)

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
	transactionService := NewTransactionService(transactionRepo, accountRepo)

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
	transactionService := NewTransactionService(transactionRepo, accountRepo)

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
	transactionService := NewTransactionService(transactionRepo, accountRepo)

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
	transactionService := NewTransactionService(transactionRepo, accountRepo)

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
	transactionService := NewTransactionService(transactionRepo, accountRepo)

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
	transactionService := NewTransactionService(transactionRepo, accountRepo)

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
	transactionService := NewTransactionService(transactionRepo, accountRepo)

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
	transactionService := NewTransactionService(transactionRepo, accountRepo)

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
	transactionService := NewTransactionService(transactionRepo, accountRepo)

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
	transactionService := NewTransactionService(transactionRepo, accountRepo)

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
	transactionService := NewTransactionService(transactionRepo, accountRepo)

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
	transactionService := NewTransactionService(transactionRepo, accountRepo)

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
	transactionService := NewTransactionService(transactionRepo, accountRepo)

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
	transactionService := NewTransactionService(transactionRepo, accountRepo)

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
	transactionService := NewTransactionService(transactionRepo, accountRepo)

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
	transactionService := NewTransactionService(transactionRepo, accountRepo)

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
	transactionService := NewTransactionService(transactionRepo, accountRepo)

	workspaceID := int32(1)

	_, err := transactionService.GetTransactionByID(workspaceID, 999)
	if err != domain.ErrTransactionNotFound {
		t.Errorf("Expected ErrTransactionNotFound, got %v", err)
	}
}

func TestGetTransactionByID_WrongWorkspace(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	transactionService := NewTransactionService(transactionRepo, accountRepo)

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
