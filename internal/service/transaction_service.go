package service

import (
	"strings"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/shopspring/decimal"
)

// TransactionService handles transaction-related business logic
type TransactionService struct {
	transactionRepo domain.TransactionRepository
	accountRepo     domain.AccountRepository
}

// NewTransactionService creates a new TransactionService
func NewTransactionService(transactionRepo domain.TransactionRepository, accountRepo domain.AccountRepository) *TransactionService {
	return &TransactionService{
		transactionRepo: transactionRepo,
		accountRepo:     accountRepo,
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
	_, err := s.accountRepo.GetByID(workspaceID, input.AccountID)
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

	transaction := &domain.Transaction{
		WorkspaceID:        workspaceID,
		AccountID:          input.AccountID,
		Name:               name,
		Amount:             input.Amount,
		Type:               input.Type,
		TransactionDate:    transactionDate,
		IsPaid:             isPaid,
		CCSettlementIntent: input.CCSettlementIntent,
		Notes:              notes,
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
