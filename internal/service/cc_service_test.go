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
