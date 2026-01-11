package service

import (
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/websocket"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// SettlementService handles credit card settlement operations
type SettlementService struct {
	transactionRepo domain.TransactionRepository
	accountRepo     domain.AccountRepository
	eventPublisher  websocket.EventPublisher
}

// NewSettlementService creates a new SettlementService
func NewSettlementService(transactionRepo domain.TransactionRepository, accountRepo domain.AccountRepository) *SettlementService {
	return &SettlementService{
		transactionRepo: transactionRepo,
		accountRepo:     accountRepo,
	}
}

// SetEventPublisher sets the event publisher for real-time updates
func (s *SettlementService) SetEventPublisher(publisher websocket.EventPublisher) {
	s.eventPublisher = publisher
}

// publishEvent publishes a WebSocket event if a publisher is configured
func (s *SettlementService) publishEvent(workspaceID int32, event websocket.Event) {
	if s.eventPublisher != nil {
		s.eventPublisher.Publish(workspaceID, event)
	}
}

// Settle atomically settles CC transactions and creates a transfer transaction.
// All operations happen within a single database transaction for atomicity.
// If any operation fails, all changes are rolled back.
func (s *SettlementService) Settle(workspaceID int32, input domain.SettlementInput) (*domain.SettlementResult, error) {
	// 1. Validate input
	if len(input.TransactionIDs) == 0 {
		return nil, domain.ErrEmptySettlement
	}

	// 2. Validate source account (must exist and NOT be CC)
	sourceAccount, err := s.accountRepo.GetByID(workspaceID, input.SourceAccountID)
	if err != nil {
		return nil, domain.ErrAccountNotFound
	}
	if sourceAccount.Template == domain.TemplateCreditCard {
		return nil, domain.ErrInvalidSourceAccount
	}

	// 3. Validate target account (must exist and BE CC)
	targetAccount, err := s.accountRepo.GetByID(workspaceID, input.TargetCCAccountID)
	if err != nil {
		return nil, domain.ErrAccountNotFound
	}
	if targetAccount.Template != domain.TemplateCreditCard {
		return nil, domain.ErrInvalidTargetAccount
	}

	// 4. Fetch and validate all transactions
	transactions, err := s.transactionRepo.GetByIDs(workspaceID, input.TransactionIDs)
	if err != nil {
		return nil, err
	}

	// Check all requested transactions were found
	if len(transactions) != len(input.TransactionIDs) {
		return nil, domain.ErrTransactionsNotFound
	}

	// Validate each transaction is eligible for settlement
	totalAmount := decimal.Zero
	for _, tx := range transactions {
		// Must be billed
		if tx.CCState == nil || *tx.CCState != domain.CCStateBilled {
			return nil, domain.ErrTransactionNotBilled
		}
		// Must have deferred settlement intent
		if tx.SettlementIntent == nil || *tx.SettlementIntent != domain.SettlementIntentDeferred {
			return nil, domain.ErrTransactionNotDeferred
		}
		totalAmount = totalAmount.Add(tx.Amount)
	}

	now := time.Now()

	// 5. Prepare transfer transaction (Bank → CC expense)
	// This represents the payment from bank to CC
	transferPairID := uuid.New()
	transferTx := &domain.Transaction{
		WorkspaceID:     workspaceID,
		AccountID:       input.SourceAccountID,
		Name:            "CC Settlement",
		Amount:          totalAmount,
		Type:            domain.TransactionTypeExpense,
		TransactionDate: now,
		IsPaid:          true,
		TransferPairID:  &transferPairID,
		IsCCPayment:     true,
		Source:          "manual",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// 6. Atomically create transfer and settle CC transactions
	// This ensures all-or-nothing behavior: if any operation fails, all changes are rolled back
	createdTransfer, settledCount, err := s.transactionRepo.AtomicSettle(transferTx, input.TransactionIDs)
	if err != nil {
		return nil, err
	}

	// 7. Verify all transactions were settled (count check for extra safety)
	if settledCount != len(input.TransactionIDs) {
		return nil, domain.ErrTransactionsNotFound
	}

	// 8. Build settlement result
	result := &domain.SettlementResult{
		TransferID:   createdTransfer.ID,
		SettledCount: settledCount,
		TotalAmount:  totalAmount,
		SettledAt:    now,
	}

	// 9. Publish event for real-time updates
	s.publishEvent(workspaceID, websocket.SettlementCreated(result))

	return result, nil
}
