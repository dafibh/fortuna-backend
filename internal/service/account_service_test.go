package service

import (
	"testing"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/shopspring/decimal"
)

func TestCreateAccount_Success_BankAccount(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)
	input := CreateAccountInput{
		Name:           "My Savings",
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromFloat(1000.50),
	}

	account, err := accountService.CreateAccount(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if account.Name != "My Savings" {
		t.Errorf("Expected name 'My Savings', got %s", account.Name)
	}

	if account.Template != domain.TemplateBank {
		t.Errorf("Expected template 'bank', got %s", account.Template)
	}

	if account.AccountType != domain.AccountTypeAsset {
		t.Errorf("Expected account type 'asset', got %s", account.AccountType)
	}

	if !account.InitialBalance.Equal(decimal.NewFromFloat(1000.50)) {
		t.Errorf("Expected initial balance '1000.50', got %s", account.InitialBalance.String())
	}

	if account.WorkspaceID != workspaceID {
		t.Errorf("Expected workspace ID %d, got %d", workspaceID, account.WorkspaceID)
	}
}

func TestCreateAccount_Success_CashAccount(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)
	input := CreateAccountInput{
		Name:           "Wallet",
		Template:       domain.TemplateCash,
		InitialBalance: decimal.Zero,
	}

	account, err := accountService.CreateAccount(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if account.AccountType != domain.AccountTypeAsset {
		t.Errorf("Expected account type 'asset', got %s", account.AccountType)
	}
}

func TestCreateAccount_Success_EwalletAccount(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)
	input := CreateAccountInput{
		Name:           "GoPay",
		Template:       domain.TemplateEwallet,
		InitialBalance: decimal.Zero,
	}

	account, err := accountService.CreateAccount(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if account.AccountType != domain.AccountTypeAsset {
		t.Errorf("Expected account type 'asset', got %s", account.AccountType)
	}
}

func TestCreateAccount_Success_CreditCardAccount(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)
	input := CreateAccountInput{
		Name:           "Visa Card",
		Template:       domain.TemplateCreditCard,
		InitialBalance: decimal.Zero,
	}

	account, err := accountService.CreateAccount(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if account.AccountType != domain.AccountTypeLiability {
		t.Errorf("Expected account type 'liability', got %s", account.AccountType)
	}
}

func TestCreateAccount_EmptyName(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)
	input := CreateAccountInput{
		Name:           "",
		Template:       domain.TemplateBank,
		InitialBalance: decimal.Zero,
	}

	_, err := accountService.CreateAccount(workspaceID, input)
	if err == nil {
		t.Fatal("Expected error for empty name, got nil")
	}

	if err != domain.ErrNameRequired {
		t.Errorf("Expected ErrNameRequired, got %v", err)
	}
}

func TestCreateAccount_WhitespaceOnlyName(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)
	input := CreateAccountInput{
		Name:           "   ",
		Template:       domain.TemplateBank,
		InitialBalance: decimal.Zero,
	}

	_, err := accountService.CreateAccount(workspaceID, input)
	if err == nil {
		t.Fatal("Expected error for whitespace-only name, got nil")
	}

	if err != domain.ErrNameRequired {
		t.Errorf("Expected ErrNameRequired, got %v", err)
	}
}

func TestCreateAccount_InvalidTemplate(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)
	input := CreateAccountInput{
		Name:           "Invalid",
		Template:       domain.AccountTemplate("invalid"),
		InitialBalance: decimal.Zero,
	}

	_, err := accountService.CreateAccount(workspaceID, input)
	if err == nil {
		t.Fatal("Expected error for invalid template, got nil")
	}

	if err != domain.ErrInvalidTemplate {
		t.Errorf("Expected ErrInvalidTemplate, got %v", err)
	}
}

func TestCreateAccount_TrimsName(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)
	input := CreateAccountInput{
		Name:           "  My Account  ",
		Template:       domain.TemplateBank,
		InitialBalance: decimal.Zero,
	}

	account, err := accountService.CreateAccount(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if account.Name != "My Account" {
		t.Errorf("Expected trimmed name 'My Account', got '%s'", account.Name)
	}
}

func TestCreateAccount_DefaultsInitialBalanceToZero(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)
	input := CreateAccountInput{
		Name:     "Zero Balance",
		Template: domain.TemplateBank,
		// InitialBalance not set, should default to zero value
	}

	account, err := accountService.CreateAccount(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !account.InitialBalance.IsZero() {
		t.Errorf("Expected initial balance to be zero, got %s", account.InitialBalance.String())
	}
}

func TestGetAccounts_Success(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)

	// Add some accounts
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Account 1",
	})
	accountRepo.AddAccount(&domain.Account{
		ID:          2,
		WorkspaceID: workspaceID,
		Name:        "Account 2",
	})

	accounts, err := accountService.GetAccounts(workspaceID, false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(accounts) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(accounts))
	}
}

func TestGetAccounts_EmptyList(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)

	accounts, err := accountService.GetAccounts(workspaceID, false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(accounts) != 0 {
		t.Errorf("Expected 0 accounts, got %d", len(accounts))
	}
}

func TestGetAccountByID_Success(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	account, err := accountService.GetAccountByID(workspaceID, accountID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if account.Name != "Test Account" {
		t.Errorf("Expected name 'Test Account', got %s", account.Name)
	}
}

func TestGetAccountByID_NotFound(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)

	_, err := accountService.GetAccountByID(workspaceID, 999)
	if err != domain.ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound, got %v", err)
	}
}

func TestGetAccountByID_WrongWorkspace(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	// Account belongs to workspace 1
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test Account",
	})

	// Try to get it from workspace 2
	_, err := accountService.GetAccountByID(2, 1)
	if err != domain.ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound for wrong workspace, got %v", err)
	}
}

// UpdateAccount tests

func TestUpdateAccount_Success(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Old Name",
	})

	account, err := accountService.UpdateAccount(workspaceID, 1, "New Name")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if account.Name != "New Name" {
		t.Errorf("Expected name 'New Name', got %s", account.Name)
	}
}

func TestUpdateAccount_TrimsName(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Old Name",
	})

	account, err := accountService.UpdateAccount(workspaceID, 1, "  New Name  ")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if account.Name != "New Name" {
		t.Errorf("Expected trimmed name 'New Name', got '%s'", account.Name)
	}
}

func TestUpdateAccount_EmptyName(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Old Name",
	})

	_, err := accountService.UpdateAccount(workspaceID, 1, "")
	if err != domain.ErrNameRequired {
		t.Errorf("Expected ErrNameRequired, got %v", err)
	}
}

func TestUpdateAccount_WhitespaceOnlyName(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Old Name",
	})

	_, err := accountService.UpdateAccount(workspaceID, 1, "   ")
	if err != domain.ErrNameRequired {
		t.Errorf("Expected ErrNameRequired, got %v", err)
	}
}

func TestUpdateAccount_NameTooLong(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Old Name",
	})

	// Create a name longer than MaxAccountNameLength (255)
	longName := ""
	for i := 0; i < 256; i++ {
		longName += "a"
	}

	_, err := accountService.UpdateAccount(workspaceID, 1, longName)
	if err != domain.ErrNameTooLong {
		t.Errorf("Expected ErrNameTooLong, got %v", err)
	}
}

func TestUpdateAccount_NotFound(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)

	_, err := accountService.UpdateAccount(workspaceID, 999, "New Name")
	if err != domain.ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound, got %v", err)
	}
}

func TestUpdateAccount_WrongWorkspace(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	// Account belongs to workspace 1
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test Account",
	})

	// Try to update it from workspace 2
	_, err := accountService.UpdateAccount(2, 1, "New Name")
	if err != domain.ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound for wrong workspace, got %v", err)
	}
}

// DeleteAccount tests

func TestDeleteAccount_Success(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	err := accountService.DeleteAccount(workspaceID, 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify account is soft-deleted (not found when querying active accounts)
	_, err = accountService.GetAccountByID(workspaceID, 1)
	if err != domain.ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound after soft delete, got %v", err)
	}
}

func TestDeleteAccount_NotFound(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)

	err := accountService.DeleteAccount(workspaceID, 999)
	if err != domain.ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound, got %v", err)
	}
}

func TestDeleteAccount_WrongWorkspace(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	// Account belongs to workspace 1
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test Account",
	})

	// Try to delete it from workspace 2
	err := accountService.DeleteAccount(2, 1)
	if err != domain.ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound for wrong workspace, got %v", err)
	}
}

func TestDeleteAccount_AlreadyDeleted(t *testing.T) {
	accountRepo := testutil.NewMockAccountRepository()
	accountService := NewAccountService(accountRepo)

	workspaceID := int32(1)
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	// First delete should succeed
	err := accountService.DeleteAccount(workspaceID, 1)
	if err != nil {
		t.Fatalf("First delete failed: %v", err)
	}

	// Second delete should fail (already deleted)
	err = accountService.DeleteAccount(workspaceID, 1)
	if err != domain.ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound for already deleted account, got %v", err)
	}
}
