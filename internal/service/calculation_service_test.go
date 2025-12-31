package service

import (
	"testing"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/shopspring/decimal"
)

func TestCalculateAccountBalances_NoTransactions(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	calculationService := NewCalculationService(accountRepo, transactionRepo)

	workspaceID := int32(1)

	// Add account with initial balance
	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    workspaceID,
		Name:           "Checking Account",
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromFloat(1000.00),
	})

	results, err := calculationService.CalculateAccountBalances(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[1]
	if result == nil {
		t.Fatal("Expected result for account 1")
	}

	// No transactions, so calculated balance should equal initial balance
	if !result.CalculatedBalance.Equal(decimal.NewFromFloat(1000.00)) {
		t.Errorf("Expected calculated balance 1000.00, got %s", result.CalculatedBalance.String())
	}
}

func TestCalculateAccountBalances_WithIncome(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	calculationService := NewCalculationService(accountRepo, transactionRepo)

	workspaceID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    workspaceID,
		Name:           "Checking Account",
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromFloat(1000.00),
	})

	// Add income transactions
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          1,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Salary",
		Amount:      decimal.NewFromFloat(5000.00),
		Type:        domain.TransactionTypeIncome,
		IsPaid:      true,
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          2,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Bonus",
		Amount:      decimal.NewFromFloat(500.00),
		Type:        domain.TransactionTypeIncome,
		IsPaid:      true,
	})

	results, err := calculationService.CalculateAccountBalances(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	result := results[1]
	// initial (1000) + income (5000 + 500) = 6500
	expected := decimal.NewFromFloat(6500.00)
	if !result.CalculatedBalance.Equal(expected) {
		t.Errorf("Expected calculated balance %s, got %s", expected.String(), result.CalculatedBalance.String())
	}
}

func TestCalculateAccountBalances_WithExpenses(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	calculationService := NewCalculationService(accountRepo, transactionRepo)

	workspaceID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    workspaceID,
		Name:           "Checking Account",
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromFloat(1000.00),
	})

	// Add expense transactions
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          1,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Groceries",
		Amount:      decimal.NewFromFloat(150.00),
		Type:        domain.TransactionTypeExpense,
		IsPaid:      true,
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          2,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Utilities",
		Amount:      decimal.NewFromFloat(100.00),
		Type:        domain.TransactionTypeExpense,
		IsPaid:      true,
	})

	results, err := calculationService.CalculateAccountBalances(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	result := results[1]
	// initial (1000) - expenses (150 + 100) = 750
	expected := decimal.NewFromFloat(750.00)
	if !result.CalculatedBalance.Equal(expected) {
		t.Errorf("Expected calculated balance %s, got %s", expected.String(), result.CalculatedBalance.String())
	}
}

func TestCalculateAccountBalances_MixedTransactions(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	calculationService := NewCalculationService(accountRepo, transactionRepo)

	workspaceID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    workspaceID,
		Name:           "Checking Account",
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromFloat(1000.00),
	})

	// Add mixed transactions
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          1,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Salary",
		Amount:      decimal.NewFromFloat(5000.00),
		Type:        domain.TransactionTypeIncome,
		IsPaid:      true,
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          2,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Rent",
		Amount:      decimal.NewFromFloat(1500.00),
		Type:        domain.TransactionTypeExpense,
		IsPaid:      true,
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          3,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Groceries",
		Amount:      decimal.NewFromFloat(300.00),
		Type:        domain.TransactionTypeExpense,
		IsPaid:      true,
	})

	results, err := calculationService.CalculateAccountBalances(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	result := results[1]
	// initial (1000) + income (5000) - expenses (1500 + 300) = 4200
	expected := decimal.NewFromFloat(4200.00)
	if !result.CalculatedBalance.Equal(expected) {
		t.Errorf("Expected calculated balance %s, got %s", expected.String(), result.CalculatedBalance.String())
	}
}

func TestCalculateAccountBalances_CreditCardOutstanding(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	calculationService := NewCalculationService(accountRepo, transactionRepo)

	workspaceID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    workspaceID,
		Name:           "Credit Card",
		Template:       domain.TemplateCreditCard,
		InitialBalance: decimal.Zero,
	})

	// Add CC expenses - some paid, some unpaid
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          1,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Online Shopping",
		Amount:      decimal.NewFromFloat(200.00),
		Type:        domain.TransactionTypeExpense,
		IsPaid:      false, // Unpaid
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          2,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Restaurant",
		Amount:      decimal.NewFromFloat(50.00),
		Type:        domain.TransactionTypeExpense,
		IsPaid:      true, // Paid
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          3,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Gas",
		Amount:      decimal.NewFromFloat(100.00),
		Type:        domain.TransactionTypeExpense,
		IsPaid:      false, // Unpaid
	})

	results, err := calculationService.CalculateAccountBalances(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	result := results[1]

	// Total balance = 0 - (200 + 50 + 100) = -350
	expectedBalance := decimal.NewFromFloat(-350.00)
	if !result.CalculatedBalance.Equal(expectedBalance) {
		t.Errorf("Expected calculated balance %s, got %s", expectedBalance.String(), result.CalculatedBalance.String())
	}

	// Outstanding = unpaid expenses = 200 + 100 = 300
	expectedOutstanding := decimal.NewFromFloat(300.00)
	if !result.CCOutstanding.Equal(expectedOutstanding) {
		t.Errorf("Expected CC outstanding %s, got %s", expectedOutstanding.String(), result.CCOutstanding.String())
	}
}

func TestCalculateAccountBalances_MultipleAccounts(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	calculationService := NewCalculationService(accountRepo, transactionRepo)

	workspaceID := int32(1)

	// Add multiple accounts
	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    workspaceID,
		Name:           "Checking",
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromFloat(1000.00),
	})
	accountRepo.AddAccount(&domain.Account{
		ID:             2,
		WorkspaceID:    workspaceID,
		Name:           "Savings",
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromFloat(5000.00),
	})

	// Add transactions for each account
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          1,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Expense 1",
		Amount:      decimal.NewFromFloat(100.00),
		Type:        domain.TransactionTypeExpense,
		IsPaid:      true,
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          2,
		WorkspaceID: workspaceID,
		AccountID:   2,
		Name:        "Interest",
		Amount:      decimal.NewFromFloat(50.00),
		Type:        domain.TransactionTypeIncome,
		IsPaid:      true,
	})

	results, err := calculationService.CalculateAccountBalances(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// Checking: 1000 - 100 = 900
	if !results[1].CalculatedBalance.Equal(decimal.NewFromFloat(900.00)) {
		t.Errorf("Checking: Expected 900.00, got %s", results[1].CalculatedBalance.String())
	}

	// Savings: 5000 + 50 = 5050
	if !results[2].CalculatedBalance.Equal(decimal.NewFromFloat(5050.00)) {
		t.Errorf("Savings: Expected 5050.00, got %s", results[2].CalculatedBalance.String())
	}
}

func TestCalculateAccountBalance_SingleAccount(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	calculationService := NewCalculationService(accountRepo, transactionRepo)

	workspaceID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    workspaceID,
		Name:           "Checking Account",
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromFloat(1000.00),
	})

	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          1,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Salary",
		Amount:      decimal.NewFromFloat(2000.00),
		Type:        domain.TransactionTypeIncome,
		IsPaid:      true,
	})

	result, err := calculationService.CalculateAccountBalance(workspaceID, 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 1000 + 2000 = 3000
	expected := decimal.NewFromFloat(3000.00)
	if !result.CalculatedBalance.Equal(expected) {
		t.Errorf("Expected %s, got %s", expected.String(), result.CalculatedBalance.String())
	}
}

func TestCalculateAccountBalances_WorkspaceIsolation(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	calculationService := NewCalculationService(accountRepo, transactionRepo)

	// Account in workspace 1
	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    1,
		Name:           "Workspace 1 Account",
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromFloat(1000.00),
	})

	// Account in workspace 2
	accountRepo.AddAccount(&domain.Account{
		ID:             2,
		WorkspaceID:    2,
		Name:           "Workspace 2 Account",
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromFloat(2000.00),
	})

	// Get balances for workspace 1 only
	results, err := calculationService.CalculateAccountBalances(1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result for workspace 1, got %d", len(results))
	}

	if results[1] == nil {
		t.Error("Expected result for account 1")
	}

	if results[2] != nil {
		t.Error("Should not have result for account 2 (different workspace)")
	}
}

func TestCalculateAccountBalances_NonCCDoesNotHaveOutstanding(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	calculationService := NewCalculationService(accountRepo, transactionRepo)

	workspaceID := int32(1)

	// Non-CC account
	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    workspaceID,
		Name:           "Bank Account",
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromFloat(1000.00),
	})

	// Add unpaid expense
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          1,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Pending Bill",
		Amount:      decimal.NewFromFloat(100.00),
		Type:        domain.TransactionTypeExpense,
		IsPaid:      false,
	})

	results, err := calculationService.CalculateAccountBalances(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	result := results[1]

	// Non-CC accounts should not have CCOutstanding set
	if !result.CCOutstanding.IsZero() {
		t.Errorf("Expected CCOutstanding to be zero for non-CC account, got %s", result.CCOutstanding.String())
	}
}
