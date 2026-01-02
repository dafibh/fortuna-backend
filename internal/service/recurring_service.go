package service

import (
	"strings"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/shopspring/decimal"
)

// RecurringService handles recurring transaction business logic
type RecurringService struct {
	recurringRepo domain.RecurringRepository
	accountRepo   domain.AccountRepository
	categoryRepo  domain.BudgetCategoryRepository
}

// NewRecurringService creates a new RecurringService
func NewRecurringService(
	recurringRepo domain.RecurringRepository,
	accountRepo domain.AccountRepository,
	categoryRepo domain.BudgetCategoryRepository,
) *RecurringService {
	return &RecurringService{
		recurringRepo: recurringRepo,
		accountRepo:   accountRepo,
		categoryRepo:  categoryRepo,
	}
}

// CreateRecurringInput holds the input for creating a recurring transaction
type CreateRecurringInput struct {
	Name       string
	Amount     decimal.Decimal
	AccountID  int32
	Type       domain.TransactionType
	CategoryID *int32
	Frequency  domain.Frequency
	DueDay     int32
}

// CreateRecurring creates a new recurring transaction template
func (s *RecurringService) CreateRecurring(workspaceID int32, input CreateRecurringInput) (*domain.RecurringTransaction, error) {
	// Validate name
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, domain.ErrNameRequired
	}
	if len(name) > domain.MaxAccountNameLength {
		return nil, domain.ErrNameTooLong
	}

	// Validate amount
	if input.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrInvalidAmount
	}

	// Validate type
	if input.Type != domain.TransactionTypeIncome && input.Type != domain.TransactionTypeExpense {
		return nil, domain.ErrInvalidTransactionType
	}

	// Validate frequency (only monthly for MVP)
	if input.Frequency != domain.FrequencyMonthly {
		return nil, domain.ErrInvalidFrequency
	}

	// Validate due day
	dueDay := input.DueDay
	if dueDay == 0 {
		dueDay = 1 // Default to 1 if not provided
	}
	if dueDay < 1 || dueDay > 31 {
		return nil, domain.ErrInvalidDueDay
	}

	// Validate account exists and belongs to workspace
	_, err := s.accountRepo.GetByID(workspaceID, input.AccountID)
	if err != nil {
		return nil, domain.ErrAccountNotFound
	}

	// Validate category exists if provided
	if input.CategoryID != nil {
		_, err := s.categoryRepo.GetByID(workspaceID, *input.CategoryID)
		if err != nil {
			return nil, domain.ErrBudgetCategoryNotFound
		}
	}

	rt := &domain.RecurringTransaction{
		WorkspaceID: workspaceID,
		Name:        name,
		Amount:      input.Amount,
		AccountID:   input.AccountID,
		Type:        input.Type,
		CategoryID:  input.CategoryID,
		Frequency:   input.Frequency,
		DueDay:      dueDay,
		IsActive:    true,
	}

	return s.recurringRepo.Create(rt)
}

// ListRecurring retrieves all recurring transactions for a workspace
func (s *RecurringService) ListRecurring(workspaceID int32, activeOnly *bool) ([]*domain.RecurringTransaction, error) {
	return s.recurringRepo.ListByWorkspace(workspaceID, activeOnly)
}

// GetRecurringByID retrieves a recurring transaction by ID
func (s *RecurringService) GetRecurringByID(workspaceID int32, id int32) (*domain.RecurringTransaction, error) {
	return s.recurringRepo.GetByID(workspaceID, id)
}

// UpdateRecurringInput holds the input for updating a recurring transaction
type UpdateRecurringInput struct {
	Name       string
	Amount     decimal.Decimal
	AccountID  int32
	Type       domain.TransactionType
	CategoryID *int32
	Frequency  domain.Frequency
	DueDay     int32
	IsActive   bool
}

// UpdateRecurring updates an existing recurring transaction
func (s *RecurringService) UpdateRecurring(workspaceID int32, id int32, input UpdateRecurringInput) (*domain.RecurringTransaction, error) {
	// Validate name
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, domain.ErrNameRequired
	}
	if len(name) > domain.MaxAccountNameLength {
		return nil, domain.ErrNameTooLong
	}

	// Validate amount
	if input.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrInvalidAmount
	}

	// Validate type
	if input.Type != domain.TransactionTypeIncome && input.Type != domain.TransactionTypeExpense {
		return nil, domain.ErrInvalidTransactionType
	}

	// Validate frequency (only monthly for MVP)
	if input.Frequency != domain.FrequencyMonthly {
		return nil, domain.ErrInvalidFrequency
	}

	// Validate due day
	if input.DueDay < 1 || input.DueDay > 31 {
		return nil, domain.ErrInvalidDueDay
	}

	// Validate account exists and belongs to workspace
	_, err := s.accountRepo.GetByID(workspaceID, input.AccountID)
	if err != nil {
		return nil, domain.ErrAccountNotFound
	}

	// Validate category exists if provided
	if input.CategoryID != nil {
		_, err := s.categoryRepo.GetByID(workspaceID, *input.CategoryID)
		if err != nil {
			return nil, domain.ErrBudgetCategoryNotFound
		}
	}

	// Check existing record
	existing, err := s.recurringRepo.GetByID(workspaceID, id)
	if err != nil {
		return nil, err
	}

	// Update fields
	existing.Name = name
	existing.Amount = input.Amount
	existing.AccountID = input.AccountID
	existing.Type = input.Type
	existing.CategoryID = input.CategoryID
	existing.Frequency = input.Frequency
	existing.DueDay = input.DueDay
	existing.IsActive = input.IsActive

	return s.recurringRepo.Update(existing)
}

// DeleteRecurring soft-deletes a recurring transaction
func (s *RecurringService) DeleteRecurring(workspaceID int32, id int32) error {
	return s.recurringRepo.Delete(workspaceID, id)
}
