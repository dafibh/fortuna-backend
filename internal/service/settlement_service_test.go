package service

import (
	"testing"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/shopspring/decimal"
)

func TestSettlementService_Settle_Success(t *testing.T) {
	// Setup
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Create source bank account
	bankAccount := &domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Bank Account",
		AccountType: domain.AccountTypeAsset,
		Template:    domain.TemplateBank,
	}
	accountRepo.AddAccount(bankAccount)

	// Create target CC account
	ccAccount := &domain.Account{
		ID:          2,
		WorkspaceID: 1,
		Name:        "Credit Card",
		AccountType: domain.AccountTypeLiability,
		Template:    domain.TemplateCreditCard,
	}
	accountRepo.AddAccount(ccAccount)

	// Create billed CC transactions with deferred settlement intent
	billedState := domain.CCStateBilled
	deferredIntent := domain.SettlementIntentDeferred
	tx1 := &domain.Transaction{
		ID:               1,
		WorkspaceID:      1,
		AccountID:        2,
		Name:             "Grocery",
		Amount:           decimal.NewFromFloat(50.00),
		Type:             domain.TransactionTypeExpense,
		CCState:          &billedState,
		SettlementIntent: &deferredIntent,
	}
	tx2 := &domain.Transaction{
		ID:               2,
		WorkspaceID:      1,
		AccountID:        2,
		Name:             "Restaurant",
		Amount:           decimal.NewFromFloat(30.00),
		Type:             domain.TransactionTypeExpense,
		CCState:          &billedState,
		SettlementIntent: &deferredIntent,
	}
	transactionRepo.AddTransaction(tx1)
	transactionRepo.AddTransaction(tx2)

	service := NewSettlementService(transactionRepo, accountRepo)

	// Execute
	input := domain.SettlementInput{
		TransactionIDs:    []int32{1, 2},
		SourceAccountID:   1,
		TargetCCAccountID: 2,
	}
	result, err := service.Settle(1, input)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.SettledCount != 2 {
		t.Errorf("expected settled count 2, got %d", result.SettledCount)
	}
	expectedAmount := decimal.NewFromFloat(80.00)
	if !result.TotalAmount.Equal(expectedAmount) {
		t.Errorf("expected total amount %s, got %s", expectedAmount, result.TotalAmount)
	}
	if result.TransferID == 0 {
		t.Error("expected non-zero transfer ID")
	}
	if result.SettledAt.IsZero() {
		t.Error("expected non-zero settled at time")
	}
}

func TestSettlementService_Settle_EmptyTransactionIDs(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	service := NewSettlementService(transactionRepo, accountRepo)

	input := domain.SettlementInput{
		TransactionIDs:    []int32{},
		SourceAccountID:   1,
		TargetCCAccountID: 2,
	}
	_, err := service.Settle(1, input)

	if err != domain.ErrEmptySettlement {
		t.Errorf("expected ErrEmptySettlement, got %v", err)
	}
}

func TestSettlementService_Settle_TransactionNotFound(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Create accounts but no transactions
	bankAccount := &domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Bank Account",
		Template:    domain.TemplateBank,
	}
	accountRepo.AddAccount(bankAccount)

	ccAccount := &domain.Account{
		ID:          2,
		WorkspaceID: 1,
		Name:        "Credit Card",
		Template:    domain.TemplateCreditCard,
	}
	accountRepo.AddAccount(ccAccount)

	service := NewSettlementService(transactionRepo, accountRepo)

	input := domain.SettlementInput{
		TransactionIDs:    []int32{999}, // Non-existent
		SourceAccountID:   1,
		TargetCCAccountID: 2,
	}
	_, err := service.Settle(1, input)

	if err != domain.ErrTransactionsNotFound {
		t.Errorf("expected ErrTransactionsNotFound, got %v", err)
	}
}

func TestSettlementService_Settle_TransactionNotBilled(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Create accounts
	bankAccount := &domain.Account{ID: 1, WorkspaceID: 1, Template: domain.TemplateBank}
	accountRepo.AddAccount(bankAccount)
	ccAccount := &domain.Account{ID: 2, WorkspaceID: 1, Template: domain.TemplateCreditCard}
	accountRepo.AddAccount(ccAccount)

	// Create transaction in pending state (not billed)
	pendingState := domain.CCStatePending
	tx := &domain.Transaction{
		ID:          1,
		WorkspaceID: 1,
		AccountID:   2,
		Amount:      decimal.NewFromFloat(50.00),
		CCState:     &pendingState,
	}
	transactionRepo.AddTransaction(tx)

	service := NewSettlementService(transactionRepo, accountRepo)

	input := domain.SettlementInput{
		TransactionIDs:    []int32{1},
		SourceAccountID:   1,
		TargetCCAccountID: 2,
	}
	_, err := service.Settle(1, input)

	if err != domain.ErrTransactionNotBilled {
		t.Errorf("expected ErrTransactionNotBilled, got %v", err)
	}
}

func TestSettlementService_Settle_TransactionNotDeferred(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Create accounts
	bankAccount := &domain.Account{ID: 1, WorkspaceID: 1, Template: domain.TemplateBank}
	accountRepo.AddAccount(bankAccount)
	ccAccount := &domain.Account{ID: 2, WorkspaceID: 1, Template: domain.TemplateCreditCard}
	accountRepo.AddAccount(ccAccount)

	// Create billed transaction but with immediate intent (not deferred)
	billedState := domain.CCStateBilled
	immediateIntent := domain.SettlementIntentImmediate
	tx := &domain.Transaction{
		ID:               1,
		WorkspaceID:      1,
		AccountID:        2,
		Amount:           decimal.NewFromFloat(50.00),
		CCState:          &billedState,
		SettlementIntent: &immediateIntent,
	}
	transactionRepo.AddTransaction(tx)

	service := NewSettlementService(transactionRepo, accountRepo)

	input := domain.SettlementInput{
		TransactionIDs:    []int32{1},
		SourceAccountID:   1,
		TargetCCAccountID: 2,
	}
	_, err := service.Settle(1, input)

	if err != domain.ErrTransactionNotDeferred {
		t.Errorf("expected ErrTransactionNotDeferred, got %v", err)
	}
}

func TestSettlementService_Settle_InvalidSourceAccount(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Create source as CC (invalid)
	sourceCC := &domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Template:    domain.TemplateCreditCard,
	}
	accountRepo.AddAccount(sourceCC)

	// Create target CC
	targetCC := &domain.Account{
		ID:          2,
		WorkspaceID: 1,
		Template:    domain.TemplateCreditCard,
	}
	accountRepo.AddAccount(targetCC)

	// Create valid billed transaction
	billedState := domain.CCStateBilled
	deferredIntent := domain.SettlementIntentDeferred
	tx := &domain.Transaction{
		ID:               1,
		WorkspaceID:      1,
		AccountID:        2,
		Amount:           decimal.NewFromFloat(50.00),
		CCState:          &billedState,
		SettlementIntent: &deferredIntent,
	}
	transactionRepo.AddTransaction(tx)

	service := NewSettlementService(transactionRepo, accountRepo)

	input := domain.SettlementInput{
		TransactionIDs:    []int32{1},
		SourceAccountID:   1, // CC account (invalid as source)
		TargetCCAccountID: 2,
	}
	_, err := service.Settle(1, input)

	if err != domain.ErrInvalidSourceAccount {
		t.Errorf("expected ErrInvalidSourceAccount, got %v", err)
	}
}

func TestSettlementService_Settle_InvalidTargetAccount(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Create valid source bank account
	bankAccount := &domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Template:    domain.TemplateBank,
	}
	accountRepo.AddAccount(bankAccount)

	// Create invalid target (not CC)
	anotherBank := &domain.Account{
		ID:          2,
		WorkspaceID: 1,
		Template:    domain.TemplateBank,
	}
	accountRepo.AddAccount(anotherBank)

	// Create billed transaction
	billedState := domain.CCStateBilled
	deferredIntent := domain.SettlementIntentDeferred
	tx := &domain.Transaction{
		ID:               1,
		WorkspaceID:      1,
		AccountID:        2,
		Amount:           decimal.NewFromFloat(50.00),
		CCState:          &billedState,
		SettlementIntent: &deferredIntent,
	}
	transactionRepo.AddTransaction(tx)

	service := NewSettlementService(transactionRepo, accountRepo)

	input := domain.SettlementInput{
		TransactionIDs:    []int32{1},
		SourceAccountID:   1,
		TargetCCAccountID: 2, // Bank account (not CC)
	}
	_, err := service.Settle(1, input)

	if err != domain.ErrInvalidTargetAccount {
		t.Errorf("expected ErrInvalidTargetAccount, got %v", err)
	}
}

func TestSettlementService_Settle_SourceAccountNotFound(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Only create target CC, not source
	ccAccount := &domain.Account{
		ID:          2,
		WorkspaceID: 1,
		Template:    domain.TemplateCreditCard,
	}
	accountRepo.AddAccount(ccAccount)

	// Create billed transaction
	billedState := domain.CCStateBilled
	deferredIntent := domain.SettlementIntentDeferred
	tx := &domain.Transaction{
		ID:               1,
		WorkspaceID:      1,
		AccountID:        2,
		Amount:           decimal.NewFromFloat(50.00),
		CCState:          &billedState,
		SettlementIntent: &deferredIntent,
	}
	transactionRepo.AddTransaction(tx)

	service := NewSettlementService(transactionRepo, accountRepo)

	input := domain.SettlementInput{
		TransactionIDs:    []int32{1},
		SourceAccountID:   999, // Non-existent
		TargetCCAccountID: 2,
	}
	_, err := service.Settle(1, input)

	if err != domain.ErrAccountNotFound {
		t.Errorf("expected ErrAccountNotFound, got %v", err)
	}
}

func TestSettlementService_Settle_TargetAccountNotFound(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Only create source bank, not target CC
	bankAccount := &domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Template:    domain.TemplateBank,
	}
	accountRepo.AddAccount(bankAccount)

	// Create billed transaction
	billedState := domain.CCStateBilled
	deferredIntent := domain.SettlementIntentDeferred
	tx := &domain.Transaction{
		ID:               1,
		WorkspaceID:      1,
		AccountID:        2,
		Amount:           decimal.NewFromFloat(50.00),
		CCState:          &billedState,
		SettlementIntent: &deferredIntent,
	}
	transactionRepo.AddTransaction(tx)

	service := NewSettlementService(transactionRepo, accountRepo)

	input := domain.SettlementInput{
		TransactionIDs:    []int32{1},
		SourceAccountID:   1,
		TargetCCAccountID: 999, // Non-existent
	}
	_, err := service.Settle(1, input)

	if err != domain.ErrAccountNotFound {
		t.Errorf("expected ErrAccountNotFound, got %v", err)
	}
}

func TestSettlementService_Settle_PartialTransactionsNotFound(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Create accounts
	bankAccount := &domain.Account{ID: 1, WorkspaceID: 1, Template: domain.TemplateBank}
	accountRepo.AddAccount(bankAccount)
	ccAccount := &domain.Account{ID: 2, WorkspaceID: 1, Template: domain.TemplateCreditCard}
	accountRepo.AddAccount(ccAccount)

	// Create only one of the requested transactions
	billedState := domain.CCStateBilled
	deferredIntent := domain.SettlementIntentDeferred
	tx := &domain.Transaction{
		ID:               1,
		WorkspaceID:      1,
		AccountID:        2,
		Amount:           decimal.NewFromFloat(50.00),
		CCState:          &billedState,
		SettlementIntent: &deferredIntent,
	}
	transactionRepo.AddTransaction(tx)

	service := NewSettlementService(transactionRepo, accountRepo)

	input := domain.SettlementInput{
		TransactionIDs:    []int32{1, 999}, // One exists, one doesn't
		SourceAccountID:   1,
		TargetCCAccountID: 2,
	}
	_, err := service.Settle(1, input)

	if err != domain.ErrTransactionsNotFound {
		t.Errorf("expected ErrTransactionsNotFound, got %v", err)
	}
}

func TestSettlementService_Settle_CreatesTransferTransaction(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Track created transactions via AtomicSettle
	var createdTransfer *domain.Transaction
	transactionRepo.AtomicSettleFn = func(transferTx *domain.Transaction, settleIDs []int32) (*domain.Transaction, int, error) {
		transferTx.ID = 100
		transferTx.CreatedAt = time.Now()
		transferTx.UpdatedAt = time.Now()
		createdTransfer = transferTx
		return transferTx, len(settleIDs), nil
	}

	// Create accounts
	bankAccount := &domain.Account{ID: 1, WorkspaceID: 1, Name: "Bank", Template: domain.TemplateBank}
	accountRepo.AddAccount(bankAccount)
	ccAccount := &domain.Account{ID: 2, WorkspaceID: 1, Name: "CC", Template: domain.TemplateCreditCard}
	accountRepo.AddAccount(ccAccount)

	// Create billed transactions
	billedState := domain.CCStateBilled
	deferredIntent := domain.SettlementIntentDeferred
	tx := &domain.Transaction{
		ID:               1,
		WorkspaceID:      1,
		AccountID:        2,
		Name:             "Coffee",
		Amount:           decimal.NewFromFloat(5.00),
		CCState:          &billedState,
		SettlementIntent: &deferredIntent,
	}
	transactionRepo.AddTransaction(tx)

	service := NewSettlementService(transactionRepo, accountRepo)

	input := domain.SettlementInput{
		TransactionIDs:    []int32{1},
		SourceAccountID:   1,
		TargetCCAccountID: 2,
	}
	result, err := service.Settle(1, input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if createdTransfer == nil {
		t.Fatal("expected transfer transaction to be created")
	}
	if createdTransfer.AccountID != 1 {
		t.Errorf("expected transfer from source account 1, got %d", createdTransfer.AccountID)
	}
	if !createdTransfer.Amount.Equal(decimal.NewFromFloat(5.00)) {
		t.Errorf("expected transfer amount 5.00, got %s", createdTransfer.Amount)
	}
	if createdTransfer.Type != domain.TransactionTypeExpense {
		t.Errorf("expected expense type, got %s", createdTransfer.Type)
	}
	if result.TransferID != 100 {
		t.Errorf("expected transfer ID 100, got %d", result.TransferID)
	}
}

func TestSettlementService_Settle_AtomicRollbackOnFailure(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Simulate AtomicSettle failure - this should roll back both create and settle
	atomicSettleCalled := false
	transactionRepo.AtomicSettleFn = func(transferTx *domain.Transaction, settleIDs []int32) (*domain.Transaction, int, error) {
		atomicSettleCalled = true
		// Simulate database error during atomic operation
		return nil, 0, domain.ErrTransactionsNotFound
	}

	// Create accounts
	bankAccount := &domain.Account{ID: 1, WorkspaceID: 1, Template: domain.TemplateBank}
	accountRepo.AddAccount(bankAccount)
	ccAccount := &domain.Account{ID: 2, WorkspaceID: 1, Template: domain.TemplateCreditCard}
	accountRepo.AddAccount(ccAccount)

	// Create billed transactions
	billedState := domain.CCStateBilled
	deferredIntent := domain.SettlementIntentDeferred
	tx := &domain.Transaction{
		ID:               1,
		WorkspaceID:      1,
		AccountID:        2,
		Amount:           decimal.NewFromFloat(50.00),
		CCState:          &billedState,
		SettlementIntent: &deferredIntent,
	}
	transactionRepo.AddTransaction(tx)

	service := NewSettlementService(transactionRepo, accountRepo)

	input := domain.SettlementInput{
		TransactionIDs:    []int32{1},
		SourceAccountID:   1,
		TargetCCAccountID: 2,
	}
	_, err := service.Settle(1, input)

	// Verify atomic operation was called
	if !atomicSettleCalled {
		t.Error("expected AtomicSettle to be called")
	}

	// Verify error is returned
	if err != domain.ErrTransactionsNotFound {
		t.Errorf("expected ErrTransactionsNotFound, got %v", err)
	}
}

func TestSettlementService_Settle_PartialSettleCountMismatch(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Simulate AtomicSettle returning fewer settled transactions than requested
	transactionRepo.AtomicSettleFn = func(transferTx *domain.Transaction, settleIDs []int32) (*domain.Transaction, int, error) {
		transferTx.ID = 100
		// Return fewer settled than requested (simulating partial failure that was caught)
		return transferTx, len(settleIDs) - 1, nil
	}

	// Create accounts
	bankAccount := &domain.Account{ID: 1, WorkspaceID: 1, Template: domain.TemplateBank}
	accountRepo.AddAccount(bankAccount)
	ccAccount := &domain.Account{ID: 2, WorkspaceID: 1, Template: domain.TemplateCreditCard}
	accountRepo.AddAccount(ccAccount)

	// Create billed transactions
	billedState := domain.CCStateBilled
	deferredIntent := domain.SettlementIntentDeferred
	tx1 := &domain.Transaction{
		ID:               1,
		WorkspaceID:      1,
		AccountID:        2,
		Amount:           decimal.NewFromFloat(50.00),
		CCState:          &billedState,
		SettlementIntent: &deferredIntent,
	}
	tx2 := &domain.Transaction{
		ID:               2,
		WorkspaceID:      1,
		AccountID:        2,
		Amount:           decimal.NewFromFloat(30.00),
		CCState:          &billedState,
		SettlementIntent: &deferredIntent,
	}
	transactionRepo.AddTransaction(tx1)
	transactionRepo.AddTransaction(tx2)

	service := NewSettlementService(transactionRepo, accountRepo)

	input := domain.SettlementInput{
		TransactionIDs:    []int32{1, 2},
		SourceAccountID:   1,
		TargetCCAccountID: 2,
	}
	_, err := service.Settle(1, input)

	// Verify count mismatch is detected
	if err != domain.ErrTransactionsNotFound {
		t.Errorf("expected ErrTransactionsNotFound for count mismatch, got %v", err)
	}
}
