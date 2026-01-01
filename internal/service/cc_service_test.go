package service

import (
	"strings"
	"testing"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/shopspring/decimal"
)

// ============================================================================
// CreateCCPayment Tests
// ============================================================================

func TestCreateCCPayment_Success_WithoutSourceAccount(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	ccService := NewCCService(transactionRepo, accountRepo)

	workspaceID := int32(1)

	// Add CC account
	ccAccount := &domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "My CC",
		Template:    domain.TemplateCreditCard,
	}
	accountRepo.AddAccount(ccAccount)

	req := &domain.CreateCCPaymentRequest{
		CCAccountID:     1,
		Amount:          decimal.NewFromFloat(500.00),
		TransactionDate: time.Now(),
		Notes:           "Monthly payment",
	}

	result, err := ccService.CreateCCPayment(workspaceID, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.CCTransaction == nil {
		t.Fatal("Expected CC transaction, got nil")
	}
	if result.SourceTransaction != nil {
		t.Fatal("Expected no source transaction, got one")
	}
	if result.CCTransaction.Type != domain.TransactionTypeIncome {
		t.Errorf("Expected income type, got %s", result.CCTransaction.Type)
	}
	if !result.CCTransaction.IsCCPayment {
		t.Error("Expected IsCCPayment=true")
	}
	if !result.CCTransaction.IsPaid {
		t.Error("Expected IsPaid=true")
	}
}

func TestCreateCCPayment_Success_WithSourceAccount(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	ccService := NewCCService(transactionRepo, accountRepo)

	workspaceID := int32(1)

	// Add CC account
	ccAccount := &domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "My CC",
		Template:    domain.TemplateCreditCard,
	}
	accountRepo.AddAccount(ccAccount)

	// Add bank account
	bankAccount := &domain.Account{
		ID:          2,
		WorkspaceID: workspaceID,
		Name:        "My Bank",
		Template:    domain.TemplateBank,
	}
	accountRepo.AddAccount(bankAccount)

	sourceID := int32(2)
	req := &domain.CreateCCPaymentRequest{
		CCAccountID:     1,
		SourceAccountID: &sourceID,
		Amount:          decimal.NewFromFloat(500.00),
		TransactionDate: time.Now(),
		Notes:           "Monthly payment",
	}

	result, err := ccService.CreateCCPayment(workspaceID, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.CCTransaction == nil {
		t.Fatal("Expected CC transaction, got nil")
	}
	if result.SourceTransaction == nil {
		t.Fatal("Expected source transaction, got nil")
	}
	if result.CCTransaction.Type != domain.TransactionTypeIncome {
		t.Errorf("Expected CC transaction to be income, got %s", result.CCTransaction.Type)
	}
	if result.SourceTransaction.Type != domain.TransactionTypeExpense {
		t.Errorf("Expected source transaction to be expense, got %s", result.SourceTransaction.Type)
	}
	if result.CCTransaction.TransferPairID == nil {
		t.Error("Expected CC transaction to have transfer pair ID")
	}
	if result.SourceTransaction.TransferPairID == nil {
		t.Error("Expected source transaction to have transfer pair ID")
	}
	if *result.CCTransaction.TransferPairID != *result.SourceTransaction.TransferPairID {
		t.Error("Expected both transactions to have same transfer pair ID")
	}
}

func TestCreateCCPayment_InvalidCCAccount(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	ccService := NewCCService(transactionRepo, accountRepo)

	workspaceID := int32(1)

	// Add bank account (not CC)
	bankAccount := &domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "My Bank",
		Template:    domain.TemplateBank,
	}
	accountRepo.AddAccount(bankAccount)

	req := &domain.CreateCCPaymentRequest{
		CCAccountID:     1,
		Amount:          decimal.NewFromFloat(500.00),
		TransactionDate: time.Now(),
	}

	_, err := ccService.CreateCCPayment(workspaceID, req)
	if err != domain.ErrInvalidAccountType {
		t.Errorf("Expected ErrInvalidAccountType, got %v", err)
	}
}

func TestCreateCCPayment_CCAccountNotFound(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	ccService := NewCCService(transactionRepo, accountRepo)

	workspaceID := int32(1)

	req := &domain.CreateCCPaymentRequest{
		CCAccountID:     999, // Doesn't exist
		Amount:          decimal.NewFromFloat(500.00),
		TransactionDate: time.Now(),
	}

	_, err := ccService.CreateCCPayment(workspaceID, req)
	if err != domain.ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound, got %v", err)
	}
}

func TestCreateCCPayment_InvalidSourceAccount_IsCreditCard(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	ccService := NewCCService(transactionRepo, accountRepo)

	workspaceID := int32(1)

	// Add two CC accounts
	cc1 := &domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "CC 1",
		Template:    domain.TemplateCreditCard,
	}
	cc2 := &domain.Account{
		ID:          2,
		WorkspaceID: workspaceID,
		Name:        "CC 2",
		Template:    domain.TemplateCreditCard,
	}
	accountRepo.AddAccount(cc1)
	accountRepo.AddAccount(cc2)

	sourceID := int32(2)
	req := &domain.CreateCCPaymentRequest{
		CCAccountID:     1,
		SourceAccountID: &sourceID, // Another CC
		Amount:          decimal.NewFromFloat(500.00),
		TransactionDate: time.Now(),
	}

	_, err := ccService.CreateCCPayment(workspaceID, req)
	if err != domain.ErrInvalidSourceAccount {
		t.Errorf("Expected ErrInvalidSourceAccount, got %v", err)
	}
}

func TestCreateCCPayment_InvalidAmount_Zero(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	ccService := NewCCService(transactionRepo, accountRepo)

	workspaceID := int32(1)

	// Add CC account
	ccAccount := &domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "My CC",
		Template:    domain.TemplateCreditCard,
	}
	accountRepo.AddAccount(ccAccount)

	req := &domain.CreateCCPaymentRequest{
		CCAccountID:     1,
		Amount:          decimal.Zero,
		TransactionDate: time.Now(),
	}

	_, err := ccService.CreateCCPayment(workspaceID, req)
	if err != domain.ErrInvalidAmount {
		t.Errorf("Expected ErrInvalidAmount, got %v", err)
	}
}

func TestCreateCCPayment_InvalidAmount_Negative(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	ccService := NewCCService(transactionRepo, accountRepo)

	workspaceID := int32(1)

	// Add CC account
	ccAccount := &domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "My CC",
		Template:    domain.TemplateCreditCard,
	}
	accountRepo.AddAccount(ccAccount)

	req := &domain.CreateCCPaymentRequest{
		CCAccountID:     1,
		Amount:          decimal.NewFromFloat(-100.00),
		TransactionDate: time.Now(),
	}

	_, err := ccService.CreateCCPayment(workspaceID, req)
	if err != domain.ErrInvalidAmount {
		t.Errorf("Expected ErrInvalidAmount, got %v", err)
	}
}

func TestCreateCCPayment_NotesTooLong(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	ccService := NewCCService(transactionRepo, accountRepo)

	workspaceID := int32(1)

	// Add CC account
	ccAccount := &domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "My CC",
		Template:    domain.TemplateCreditCard,
	}
	accountRepo.AddAccount(ccAccount)

	// Create notes longer than MaxTransactionNotesLength (1000)
	longNotes := strings.Repeat("a", domain.MaxTransactionNotesLength+1)

	req := &domain.CreateCCPaymentRequest{
		CCAccountID:     1,
		Amount:          decimal.NewFromFloat(500.00),
		TransactionDate: time.Now(),
		Notes:           longNotes,
	}

	_, err := ccService.CreateCCPayment(workspaceID, req)
	if err != domain.ErrNotesTooLong {
		t.Errorf("Expected ErrNotesTooLong, got %v", err)
	}
}

func TestGetPayableBreakdown_Success_MixedIntents(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	ccService := NewCCService(transactionRepo, accountRepo)

	workspaceID := int32(1)

	// Configure mock with mixed settlement intents
	transactionRepo.GetCCPayableBreakdownFn = func(wsID int32) ([]*domain.CCPayableTransaction, error) {
		return []*domain.CCPayableTransaction{
			{
				ID:               1,
				Name:             "Groceries",
				Amount:           decimal.NewFromFloat(100.00),
				TransactionDate:  time.Now(),
				SettlementIntent: domain.CCSettlementThisMonth,
				AccountID:        1,
				AccountName:      "Maybank CC",
			},
			{
				ID:               2,
				Name:             "Shopping",
				Amount:           decimal.NewFromFloat(200.00),
				TransactionDate:  time.Now(),
				SettlementIntent: domain.CCSettlementThisMonth,
				AccountID:        1,
				AccountName:      "Maybank CC",
			},
			{
				ID:               3,
				Name:             "Dining",
				Amount:           decimal.NewFromFloat(50.00),
				TransactionDate:  time.Now(),
				SettlementIntent: domain.CCSettlementNextMonth,
				AccountID:        2,
				AccountName:      "CIMB CC",
			},
		}, nil
	}

	result, err := ccService.GetPayableBreakdown(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify this month totals
	if !result.ThisMonthTotal.Equal(decimal.NewFromFloat(300.00)) {
		t.Errorf("Expected this month total 300.00, got %s", result.ThisMonthTotal.String())
	}

	// Verify next month totals
	if !result.NextMonthTotal.Equal(decimal.NewFromFloat(50.00)) {
		t.Errorf("Expected next month total 50.00, got %s", result.NextMonthTotal.String())
	}

	// Verify grand total
	if !result.GrandTotal.Equal(decimal.NewFromFloat(350.00)) {
		t.Errorf("Expected grand total 350.00, got %s", result.GrandTotal.String())
	}

	// Verify account grouping for this month
	if len(result.ThisMonth) != 1 {
		t.Fatalf("Expected 1 account in this month, got %d", len(result.ThisMonth))
	}
	if result.ThisMonth[0].AccountName != "Maybank CC" {
		t.Errorf("Expected Maybank CC, got %s", result.ThisMonth[0].AccountName)
	}
	if len(result.ThisMonth[0].Transactions) != 2 {
		t.Errorf("Expected 2 transactions in Maybank CC, got %d", len(result.ThisMonth[0].Transactions))
	}

	// Verify account grouping for next month
	if len(result.NextMonth) != 1 {
		t.Fatalf("Expected 1 account in next month, got %d", len(result.NextMonth))
	}
	if result.NextMonth[0].AccountName != "CIMB CC" {
		t.Errorf("Expected CIMB CC, got %s", result.NextMonth[0].AccountName)
	}
}

func TestGetPayableBreakdown_NoTransactions(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	ccService := NewCCService(transactionRepo, accountRepo)

	workspaceID := int32(1)

	// Configure mock with no transactions
	transactionRepo.GetCCPayableBreakdownFn = func(wsID int32) ([]*domain.CCPayableTransaction, error) {
		return []*domain.CCPayableTransaction{}, nil
	}

	result, err := ccService.GetPayableBreakdown(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !result.ThisMonthTotal.IsZero() {
		t.Errorf("Expected zero this month total, got %s", result.ThisMonthTotal.String())
	}
	if !result.NextMonthTotal.IsZero() {
		t.Errorf("Expected zero next month total, got %s", result.NextMonthTotal.String())
	}
	if !result.GrandTotal.IsZero() {
		t.Errorf("Expected zero grand total, got %s", result.GrandTotal.String())
	}
	if len(result.ThisMonth) != 0 {
		t.Errorf("Expected empty this month, got %d accounts", len(result.ThisMonth))
	}
	if len(result.NextMonth) != 0 {
		t.Errorf("Expected empty next month, got %d accounts", len(result.NextMonth))
	}
}

func TestGetPayableBreakdown_OnlyThisMonth(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	ccService := NewCCService(transactionRepo, accountRepo)

	workspaceID := int32(1)

	transactionRepo.GetCCPayableBreakdownFn = func(wsID int32) ([]*domain.CCPayableTransaction, error) {
		return []*domain.CCPayableTransaction{
			{
				ID:               1,
				Name:             "Transaction 1",
				Amount:           decimal.NewFromFloat(500.00),
				TransactionDate:  time.Now(),
				SettlementIntent: domain.CCSettlementThisMonth,
				AccountID:        1,
				AccountName:      "Card A",
			},
		}, nil
	}

	result, err := ccService.GetPayableBreakdown(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !result.ThisMonthTotal.Equal(decimal.NewFromFloat(500.00)) {
		t.Errorf("Expected this month total 500.00, got %s", result.ThisMonthTotal.String())
	}
	if !result.NextMonthTotal.IsZero() {
		t.Errorf("Expected zero next month total, got %s", result.NextMonthTotal.String())
	}
	if len(result.ThisMonth) != 1 {
		t.Errorf("Expected 1 account in this month, got %d", len(result.ThisMonth))
	}
	if len(result.NextMonth) != 0 {
		t.Errorf("Expected 0 accounts in next month, got %d", len(result.NextMonth))
	}
}

func TestGetPayableBreakdown_OnlyNextMonth(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	ccService := NewCCService(transactionRepo, accountRepo)

	workspaceID := int32(1)

	transactionRepo.GetCCPayableBreakdownFn = func(wsID int32) ([]*domain.CCPayableTransaction, error) {
		return []*domain.CCPayableTransaction{
			{
				ID:               1,
				Name:             "Transaction 1",
				Amount:           decimal.NewFromFloat(250.00),
				TransactionDate:  time.Now(),
				SettlementIntent: domain.CCSettlementNextMonth,
				AccountID:        1,
				AccountName:      "Card B",
			},
		}, nil
	}

	result, err := ccService.GetPayableBreakdown(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !result.ThisMonthTotal.IsZero() {
		t.Errorf("Expected zero this month total, got %s", result.ThisMonthTotal.String())
	}
	if !result.NextMonthTotal.Equal(decimal.NewFromFloat(250.00)) {
		t.Errorf("Expected next month total 250.00, got %s", result.NextMonthTotal.String())
	}
	if len(result.ThisMonth) != 0 {
		t.Errorf("Expected 0 accounts in this month, got %d", len(result.ThisMonth))
	}
	if len(result.NextMonth) != 1 {
		t.Errorf("Expected 1 account in next month, got %d", len(result.NextMonth))
	}
}

func TestGetPayableBreakdown_MultipleAccounts(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	ccService := NewCCService(transactionRepo, accountRepo)

	workspaceID := int32(1)

	transactionRepo.GetCCPayableBreakdownFn = func(wsID int32) ([]*domain.CCPayableTransaction, error) {
		return []*domain.CCPayableTransaction{
			{
				ID:               1,
				Name:             "T1",
				Amount:           decimal.NewFromFloat(100.00),
				TransactionDate:  time.Now(),
				SettlementIntent: domain.CCSettlementThisMonth,
				AccountID:        1,
				AccountName:      "Card A",
			},
			{
				ID:               2,
				Name:             "T2",
				Amount:           decimal.NewFromFloat(200.00),
				TransactionDate:  time.Now(),
				SettlementIntent: domain.CCSettlementThisMonth,
				AccountID:        2,
				AccountName:      "Card B",
			},
			{
				ID:               3,
				Name:             "T3",
				Amount:           decimal.NewFromFloat(150.00),
				TransactionDate:  time.Now(),
				SettlementIntent: domain.CCSettlementThisMonth,
				AccountID:        3,
				AccountName:      "Card C",
			},
		}, nil
	}

	result, err := ccService.GetPayableBreakdown(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.ThisMonth) != 3 {
		t.Errorf("Expected 3 accounts in this month, got %d", len(result.ThisMonth))
	}

	// Verify sorted by account name
	if result.ThisMonth[0].AccountName != "Card A" {
		t.Errorf("Expected first account to be Card A, got %s", result.ThisMonth[0].AccountName)
	}
	if result.ThisMonth[1].AccountName != "Card B" {
		t.Errorf("Expected second account to be Card B, got %s", result.ThisMonth[1].AccountName)
	}
	if result.ThisMonth[2].AccountName != "Card C" {
		t.Errorf("Expected third account to be Card C, got %s", result.ThisMonth[2].AccountName)
	}
}

func TestGetPayableBreakdown_RepoError(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	ccService := NewCCService(transactionRepo, accountRepo)

	workspaceID := int32(1)

	transactionRepo.GetCCPayableBreakdownFn = func(wsID int32) ([]*domain.CCPayableTransaction, error) {
		return nil, domain.ErrTransactionNotFound
	}

	_, err := ccService.GetPayableBreakdown(workspaceID)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if err != domain.ErrTransactionNotFound {
		t.Errorf("Expected ErrTransactionNotFound, got %v", err)
	}
}
