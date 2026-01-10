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
	StartDate  time.Time  // V2: When recurring pattern starts
	EndDate    *time.Time // V2: Optional end date (nil = runs forever)
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

	// Validate start date - must not be zero
	startDate := input.StartDate
	if startDate.IsZero() {
		startDate = time.Now() // Default to today
	}

	// Validate end date - must be after start date if provided
	if input.EndDate != nil && !input.EndDate.After(startDate) {
		return nil, domain.ErrInvalidInput
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
		StartDate:   startDate,
		EndDate:     input.EndDate,
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
	StartDate  time.Time  // V2: When recurring pattern starts
	EndDate    *time.Time // V2: Optional end date (nil = runs forever)
	IsActive   bool
}

// UpdateRecurringResult holds the result of updating a recurring template
type UpdateRecurringResult struct {
	Template           *domain.RecurringTransaction `json:"template"`
	ProjectionsDeleted int64                        `json:"projectionsDeleted"`
}

// UpdateRecurring updates an existing recurring transaction
// When updated, existing PROJECTIONS are deleted so they regenerate with new values.
// Actual transactions (edited projections) are preserved.
func (s *RecurringService) UpdateRecurring(workspaceID int32, id int32, input UpdateRecurringInput) (*UpdateRecurringResult, error) {
	result := &UpdateRecurringResult{}

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

	// Validate start date - must not be zero
	startDate := input.StartDate
	if startDate.IsZero() {
		startDate = time.Now() // Default to today
	}

	// Validate end date - must be after start date if provided
	if input.EndDate != nil && !input.EndDate.After(startDate) {
		return nil, domain.ErrInvalidInput
	}

	// Check existing record
	existing, err := s.recurringRepo.GetByID(workspaceID, id)
	if err != nil {
		return nil, err
	}

	// Delete existing projections so they regenerate with new template values
	// This only deletes is_projected=true transactions, preserving edited actuals
	projectionsDeleted, err := s.transactionRepo.DeleteProjectionsByTemplateID(workspaceID, id)
	if err != nil {
		return nil, err
	}
	result.ProjectionsDeleted = projectionsDeleted

	// Update fields
	existing.Name = name
	existing.Amount = input.Amount
	existing.AccountID = input.AccountID
	existing.Type = input.Type
	existing.CategoryID = input.CategoryID
	existing.Frequency = input.Frequency
	existing.DueDay = input.DueDay
	existing.StartDate = startDate
	existing.EndDate = input.EndDate
	existing.IsActive = input.IsActive

	updated, err := s.recurringRepo.Update(existing)
	if err != nil {
		return nil, err
	}
	result.Template = updated

	return result, nil
}

// DeleteRecurringResult holds the result of deleting a recurring template
type DeleteRecurringResult struct {
	ProjectionsDeleted int64 `json:"projectionsDeleted"`
	ActualsOrphaned    int64 `json:"actualsOrphaned"`
}

// DeleteRecurring soft-deletes a recurring transaction and handles related transactions
// - Deletes all future projections linked to this template
// - Orphans actual transactions (removes template_id but keeps the transaction)
func (s *RecurringService) DeleteRecurring(workspaceID int32, id int32) (*DeleteRecurringResult, error) {
	result := &DeleteRecurringResult{}

	// First verify the template exists
	_, err := s.recurringRepo.GetByID(workspaceID, id)
	if err != nil {
		return nil, err
	}

	// Delete all projections for this template
	projectionsDeleted, err := s.transactionRepo.DeleteProjectionsByTemplateID(workspaceID, id)
	if err != nil {
		return nil, err
	}
	result.ProjectionsDeleted = projectionsDeleted

	// Orphan actual transactions (remove template_id reference)
	actualsOrphaned, err := s.transactionRepo.OrphanActualsByTemplateID(workspaceID, id)
	if err != nil {
		return nil, err
	}
	result.ActualsOrphaned = actualsOrphaned

	// Finally soft-delete the template itself
	if err := s.recurringRepo.Delete(workspaceID, id); err != nil {
		return nil, err
	}

	return result, nil
}

// ToggleActive toggles the is_active status of a recurring transaction
func (s *RecurringService) ToggleActive(workspaceID int32, id int32) (*domain.RecurringTransaction, error) {
	existing, err := s.recurringRepo.GetByID(workspaceID, id)
	if err != nil {
		return nil, err
	}

	// Toggle is_active
	existing.IsActive = !existing.IsActive

	updated, err := s.recurringRepo.Update(existing)
	if err != nil {
		return nil, err
	}

	return updated, nil
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
			WorkspaceID:     workspaceID,
			AccountID:       rt.AccountID,
			Name:            rt.Name,
			Amount:          rt.Amount,
			Type:            rt.Type,
			TransactionDate: dueDate,
			IsPaid:          false,
			CategoryID:      rt.CategoryID,
			TemplateID:      &rt.ID,
			Source:          domain.TransactionSourceRecurring,
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
