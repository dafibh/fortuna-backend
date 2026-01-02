package service

import (
	"strings"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/shopspring/decimal"
)

// RecurringService handles recurring transaction business logic
type RecurringService struct {
	recurringRepo   domain.RecurringRepository
	transactionRepo domain.TransactionRepository
	accountRepo     domain.AccountRepository
	categoryRepo    domain.BudgetCategoryRepository
}

// NewRecurringService creates a new RecurringService
func NewRecurringService(
	recurringRepo domain.RecurringRepository,
	transactionRepo domain.TransactionRepository,
	accountRepo domain.AccountRepository,
	categoryRepo domain.BudgetCategoryRepository,
) *RecurringService {
	return &RecurringService{
		recurringRepo:   recurringRepo,
		transactionRepo: transactionRepo,
		accountRepo:     accountRepo,
		categoryRepo:    categoryRepo,
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

// CalculateActualDueDate returns the actual due date for a recurring transaction
// given the due day and target month/year. For months with fewer days than the
// due day (e.g., due day 31 in February), returns the last day of that month.
// Invalid due days (<= 0) are clamped to 1.
func CalculateActualDueDate(dueDay int32, year int, month time.Month) time.Time {
	// Clamp invalid due days to 1 (defensive)
	actualDay := int(dueDay)
	if actualDay < 1 {
		actualDay = 1
	}

	// Get last day of month by going to day 0 of next month
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()

	// Clamp to last day of month if needed
	if actualDay > lastDay {
		actualDay = lastDay
	}

	return time.Date(year, month, actualDay, 0, 0, 0, 0, time.UTC)
}

// GenerationResult holds the result of generating recurring transactions
type GenerationResult struct {
	Generated []*domain.Transaction `json:"generated"`
	Skipped   int                   `json:"skipped"`
	Errors    []string              `json:"errors,omitempty"`
}

// GenerateRecurringTransactions generates transactions from active recurring templates
// for the specified month/year. Uses idempotency check to skip already-generated transactions.
func (s *RecurringService) GenerateRecurringTransactions(workspaceID int32, year int, month time.Month) (*GenerationResult, error) {
	result := &GenerationResult{
		Generated: make([]*domain.Transaction, 0),
		Skipped:   0,
		Errors:    make([]string, 0),
	}

	// Get all active recurring transactions
	activeOnly := true
	recurring, err := s.recurringRepo.ListByWorkspace(workspaceID, &activeOnly)
	if err != nil {
		return nil, err
	}

	// Process each recurring template
	for _, rt := range recurring {
		// Check if transaction already exists for this month
		exists, err := s.recurringRepo.CheckTransactionExists(rt.ID, workspaceID, year, int(month))
		if err != nil {
			result.Errors = append(result.Errors, "Failed to check existing for "+rt.Name+": "+err.Error())
			continue
		}

		if exists {
			result.Skipped++
			continue
		}

		// Calculate actual due date for this month
		dueDate := CalculateActualDueDate(rt.DueDay, year, month)

		// Create the transaction
		tx := &domain.Transaction{
			WorkspaceID:            workspaceID,
			AccountID:              rt.AccountID,
			Name:                   rt.Name,
			Amount:                 rt.Amount,
			Type:                   rt.Type,
			TransactionDate:        dueDate,
			IsPaid:                 false,
			CategoryID:             rt.CategoryID,
			RecurringTransactionID: &rt.ID,
		}

		created, err := s.transactionRepo.Create(tx)
		if err != nil {
			result.Errors = append(result.Errors, "Failed to create for "+rt.Name+": "+err.Error())
			continue
		}

		result.Generated = append(result.Generated, created)
	}

	return result, nil
}
