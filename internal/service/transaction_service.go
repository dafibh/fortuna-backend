package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TransactionService handles transaction-related business logic
type TransactionService struct {
	transactionRepo domain.TransactionRepository
	accountRepo     domain.AccountRepository
	categoryRepo    domain.BudgetCategoryRepository
}

// NewTransactionService creates a new TransactionService
func NewTransactionService(transactionRepo domain.TransactionRepository, accountRepo domain.AccountRepository, categoryRepo domain.BudgetCategoryRepository) *TransactionService {
	return &TransactionService{
		transactionRepo: transactionRepo,
		accountRepo:     accountRepo,
		categoryRepo:    categoryRepo,
	}
}

// CreateTransactionInput holds the input for creating a transaction
type CreateTransactionInput struct {
	AccountID          int32
	Name               string
	Amount             decimal.Decimal
	Type               domain.TransactionType
	TransactionDate    *time.Time
	IsPaid             *bool
	CCSettlementIntent *domain.CCSettlementIntent
	Notes              *string
	CategoryID         *int32
}

// CreateTransaction creates a new transaction with validation
func (s *TransactionService) CreateTransaction(workspaceID int32, input CreateTransactionInput) (*domain.Transaction, error) {
	// Validate name
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, domain.ErrNameRequired
	}
	if len(name) > domain.MaxTransactionNameLength {
		return nil, domain.ErrNameTooLong
	}

	// Validate amount (must be positive)
	if input.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrInvalidAmount
	}

	// Validate transaction type
	if input.Type != domain.TransactionTypeIncome && input.Type != domain.TransactionTypeExpense {
		return nil, domain.ErrInvalidTransactionType
	}

	// Validate account exists and belongs to workspace
	account, err := s.accountRepo.GetByID(workspaceID, input.AccountID)
	if err != nil {
		return nil, domain.ErrAccountNotFound
	}

	// Default transaction_date to today if not provided
	transactionDate := time.Now().UTC().Truncate(24 * time.Hour)
	if input.TransactionDate != nil {
		transactionDate = *input.TransactionDate
	}

	// Default is_paid to true if not provided
	isPaid := true
	if input.IsPaid != nil {
		isPaid = *input.IsPaid
	}

	// Trim and validate notes if provided
	var notes *string
	if input.Notes != nil {
		trimmed := strings.TrimSpace(*input.Notes)
		if trimmed != "" {
			if len(trimmed) > domain.MaxTransactionNotesLength {
				return nil, domain.ErrNotesTooLong
			}
			notes = &trimmed
		}
	}

	// Handle CC settlement intent based on account type
	var settlementIntent *domain.CCSettlementIntent
	if account.Template == domain.TemplateCreditCard {
		// For CC accounts: use provided intent or default to 'this_month'
		if input.CCSettlementIntent != nil {
			// Validate the provided intent
			if *input.CCSettlementIntent != domain.CCSettlementThisMonth && *input.CCSettlementIntent != domain.CCSettlementNextMonth {
				return nil, domain.ErrInvalidSettlementIntent
			}
			settlementIntent = input.CCSettlementIntent
		} else {
			// Default to 'this_month'
			defaultIntent := domain.CCSettlementThisMonth
			settlementIntent = &defaultIntent
		}
	}
	// For non-CC accounts, settlementIntent remains nil (ignores any provided value)

	// Validate category exists and belongs to workspace if provided
	if input.CategoryID != nil {
		_, err := s.categoryRepo.GetByID(workspaceID, *input.CategoryID)
		if err != nil {
			return nil, domain.ErrBudgetCategoryNotFound
		}
	}

	transaction := &domain.Transaction{
		WorkspaceID:        workspaceID,
		AccountID:          input.AccountID,
		Name:               name,
		Amount:             input.Amount,
		Type:               input.Type,
		TransactionDate:    transactionDate,
		IsPaid:             isPaid,
		CCSettlementIntent: settlementIntent,
		Notes:              notes,
		CategoryID:         input.CategoryID,
	}

	return s.transactionRepo.Create(transaction)
}

// GetTransactions retrieves transactions for a workspace with optional filters and pagination
func (s *TransactionService) GetTransactions(workspaceID int32, filters *domain.TransactionFilters) (*domain.PaginatedTransactions, error) {
	return s.transactionRepo.GetByWorkspace(workspaceID, filters)
}

// GetTransactionByID retrieves a transaction by ID within a workspace
func (s *TransactionService) GetTransactionByID(workspaceID int32, id int32) (*domain.Transaction, error) {
	return s.transactionRepo.GetByID(workspaceID, id)
}

// TogglePaidStatus toggles the paid status of a transaction
func (s *TransactionService) TogglePaidStatus(workspaceID int32, id int32) (*domain.Transaction, error) {
	return s.transactionRepo.TogglePaid(workspaceID, id)
}

// UpdateTransactionInput holds the input for updating a transaction
type UpdateTransactionInput struct {
	Name               string
	Amount             decimal.Decimal
	Type               domain.TransactionType
	TransactionDate    time.Time
	AccountID          int32
	CCSettlementIntent *domain.CCSettlementIntent
	Notes              *string
	CategoryID         *int32
}

// UpdateTransaction updates an existing transaction with validation
func (s *TransactionService) UpdateTransaction(workspaceID int32, id int32, input UpdateTransactionInput) (*domain.Transaction, error) {
	// Validate name
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, domain.ErrNameRequired
	}
	if len(name) > domain.MaxTransactionNameLength {
		return nil, domain.ErrNameTooLong
	}

	// Validate amount (must be positive)
	if input.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrInvalidAmount
	}

	// Validate transaction type
	if input.Type != domain.TransactionTypeIncome && input.Type != domain.TransactionTypeExpense {
		return nil, domain.ErrInvalidTransactionType
	}

	// Validate account exists and belongs to workspace
	account, err := s.accountRepo.GetByID(workspaceID, input.AccountID)
	if err != nil {
		return nil, domain.ErrAccountNotFound
	}

	// Handle CC settlement intent based on account type
	var settlementIntent *domain.CCSettlementIntent
	if account.Template == domain.TemplateCreditCard {
		// For CC accounts: use provided intent or default to 'this_month'
		if input.CCSettlementIntent != nil {
			// Validate the provided intent
			if *input.CCSettlementIntent != domain.CCSettlementThisMonth && *input.CCSettlementIntent != domain.CCSettlementNextMonth {
				return nil, domain.ErrInvalidSettlementIntent
			}
			settlementIntent = input.CCSettlementIntent
		} else {
			// Default to 'this_month'
			defaultIntent := domain.CCSettlementThisMonth
			settlementIntent = &defaultIntent
		}
	}
	// For non-CC accounts, settlementIntent remains nil (clears any existing value)

	// Trim and validate notes if provided
	var notes *string
	if input.Notes != nil {
		trimmed := strings.TrimSpace(*input.Notes)
		if trimmed != "" {
			if len(trimmed) > domain.MaxTransactionNotesLength {
				return nil, domain.ErrNotesTooLong
			}
			notes = &trimmed
		}
	}

	// Validate category exists and belongs to workspace if provided
	if input.CategoryID != nil {
		_, err := s.categoryRepo.GetByID(workspaceID, *input.CategoryID)
		if err != nil {
			return nil, domain.ErrBudgetCategoryNotFound
		}
	}

	return s.transactionRepo.Update(workspaceID, id, &domain.UpdateTransactionData{
		Name:               name,
		Amount:             input.Amount,
		Type:               input.Type,
		TransactionDate:    input.TransactionDate,
		AccountID:          input.AccountID,
		CCSettlementIntent: settlementIntent,
		Notes:              notes,
		CategoryID:         input.CategoryID,
	})
}

// UpdateSettlementIntent updates the CC settlement intent for an unpaid CC transaction
func (s *TransactionService) UpdateSettlementIntent(workspaceID int32, id int32, intent domain.CCSettlementIntent) (*domain.Transaction, error) {
	// Validate intent value
	if intent != domain.CCSettlementThisMonth && intent != domain.CCSettlementNextMonth {
		return nil, domain.ErrInvalidSettlementIntent
	}

	// Get transaction first to verify it exists and get account info
	tx, err := s.transactionRepo.GetByID(workspaceID, id)
	if err != nil {
		return nil, err
	}

	// Verify transaction is unpaid
	if tx.IsPaid {
		return nil, domain.ErrTransactionAlreadyPaid
	}

	// Verify account is Credit Card type
	account, err := s.accountRepo.GetByID(workspaceID, tx.AccountID)
	if err != nil {
		return nil, err
	}
	if account.Template != domain.TemplateCreditCard {
		return nil, domain.ErrSettlementIntentNotApplicable
	}

	return s.transactionRepo.UpdateSettlementIntent(workspaceID, id, intent)
}

// DeleteTransaction soft deletes a transaction (or both sides of a transfer)
func (s *TransactionService) DeleteTransaction(workspaceID int32, id int32) error {
	// Get transaction first to check if it's a transfer
	tx, err := s.transactionRepo.GetByID(workspaceID, id)
	if err != nil {
		return err
	}

	// If it's a transfer, delete both linked transactions
	if tx.TransferPairID != nil {
		return s.transactionRepo.SoftDeleteTransferPair(workspaceID, *tx.TransferPairID)
	}

	// Regular delete
	return s.transactionRepo.SoftDelete(workspaceID, id)
}

// CreateTransferInput holds the input for creating a transfer
type CreateTransferInput struct {
	FromAccountID int32
	ToAccountID   int32
	Amount        decimal.Decimal
	Date          time.Time
	Notes         *string
}

// CreateTransfer creates a transfer between two accounts
func (s *TransactionService) CreateTransfer(workspaceID int32, input CreateTransferInput) (*domain.TransferResult, error) {
	// Validate same account
	if input.FromAccountID == input.ToAccountID {
		return nil, domain.ErrSameAccountTransfer
	}

	// Validate amount
	if input.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrInvalidAmount
	}

	// Validate both accounts exist and belong to workspace
	fromAccount, err := s.accountRepo.GetByID(workspaceID, input.FromAccountID)
	if err != nil {
		return nil, err
	}
	toAccount, err := s.accountRepo.GetByID(workspaceID, input.ToAccountID)
	if err != nil {
		return nil, err
	}

	// Validate notes length if provided
	if input.Notes != nil && len(*input.Notes) > domain.MaxTransactionNotesLength {
		return nil, domain.ErrNotesTooLong
	}

	// Generate transfer pair ID
	pairID := uuid.New()

	// Build transaction names
	fromName := fmt.Sprintf("Transfer to %s", toAccount.Name)
	toName := fmt.Sprintf("Transfer from %s", fromAccount.Name)

	// Create expense transaction (from account)
	fromTx := &domain.Transaction{
		WorkspaceID:     workspaceID,
		AccountID:       input.FromAccountID,
		Name:            fromName,
		Amount:          input.Amount,
		Type:            domain.TransactionTypeExpense,
		TransactionDate: input.Date,
		IsPaid:          true, // Transfers are always considered paid
		TransferPairID:  &pairID,
		Notes:           input.Notes,
	}

	// Create income transaction (to account)
	toTx := &domain.Transaction{
		WorkspaceID:     workspaceID,
		AccountID:       input.ToAccountID,
		Name:            toName,
		Amount:          input.Amount,
		Type:            domain.TransactionTypeIncome,
		TransactionDate: input.Date,
		IsPaid:          true,
		TransferPairID:  &pairID,
		Notes:           input.Notes,
	}

	return s.transactionRepo.CreateTransferPair(fromTx, toTx)
}

// GetRecentlyUsedCategories returns recently used categories for suggestions dropdown
func (s *TransactionService) GetRecentlyUsedCategories(workspaceID int32) ([]*domain.RecentCategory, error) {
	return s.transactionRepo.GetRecentlyUsedCategories(workspaceID)
}
