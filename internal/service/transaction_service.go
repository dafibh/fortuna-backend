package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/util"
	"github.com/dafibh/fortuna/fortuna-backend/internal/websocket"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

// TransactionService handles transaction-related business logic
type TransactionService struct {
	transactionRepo domain.TransactionRepository
	accountRepo     domain.AccountRepository
	categoryRepo    domain.BudgetCategoryRepository
	templateRepo    domain.RecurringTemplateRepository
	exclusionRepo   domain.ProjectionExclusionRepository
	eventPublisher  websocket.EventPublisher
}

// NewTransactionService creates a new TransactionService
func NewTransactionService(transactionRepo domain.TransactionRepository, accountRepo domain.AccountRepository, categoryRepo domain.BudgetCategoryRepository) *TransactionService {
	return &TransactionService{
		transactionRepo: transactionRepo,
		accountRepo:     accountRepo,
		categoryRepo:    categoryRepo,
	}
}

// SetRecurringTemplateRepository sets the template repository for on-access projection generation
func (s *TransactionService) SetRecurringTemplateRepository(templateRepo domain.RecurringTemplateRepository) {
	s.templateRepo = templateRepo
}

// SetExclusionRepository sets the exclusion repository for projection deletion tracking
func (s *TransactionService) SetExclusionRepository(exclusionRepo domain.ProjectionExclusionRepository) {
	s.exclusionRepo = exclusionRepo
}

// SetEventPublisher sets the event publisher for real-time updates
func (s *TransactionService) SetEventPublisher(publisher websocket.EventPublisher) {
	s.eventPublisher = publisher
}

// publishEvent publishes a WebSocket event if a publisher is configured
func (s *TransactionService) publishEvent(workspaceID int32, event websocket.Event) {
	if s.eventPublisher != nil {
		s.eventPublisher.Publish(workspaceID, event)
	}
}

// CreateTransactionInput holds the input for creating a transaction
type CreateTransactionInput struct {
	AccountID        int32
	Name             string
	Amount           decimal.Decimal
	Type             domain.TransactionType
	TransactionDate  *time.Time
	IsPaid           *bool
	Notes            *string
	CategoryID       *int32
	SettlementIntent *domain.SettlementIntent
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

	// Handle CC lifecycle fields
	// v2 simplified: CCState is computed from isPaid and billedAt
	// - pending: billedAt IS NULL AND isPaid = false (default for new CC transactions)
	// - billed: billedAt IS NOT NULL AND isPaid = false
	// - settled: isPaid = true
	var v2SettlementIntent *domain.SettlementIntent
	if account.Template == domain.TemplateCreditCard {
		// Determine settlement intent: use provided or default to deferred
		// Settlement intent is just a plan for when to pay (this month vs next month)
		// It does NOT affect the actual CC state - all CC transactions start as pending
		intent := domain.SettlementIntentDeferred
		if input.SettlementIntent != nil {
			intent = *input.SettlementIntent
		}
		v2SettlementIntent = &intent
		// All CC transactions start as pending (billedAt = nil, isPaid = false)
		// Override isPaid to false unless explicitly set by user
		if input.IsPaid == nil {
			isPaid = false
		}
	}
	// For non-CC accounts, all CC lifecycle fields remain nil

	// Validate category exists and belongs to workspace if provided
	if input.CategoryID != nil {
		_, err := s.categoryRepo.GetByID(workspaceID, *input.CategoryID)
		if err != nil {
			return nil, domain.ErrBudgetCategoryNotFound
		}
	}

	transaction := &domain.Transaction{
		WorkspaceID:      workspaceID,
		AccountID:        input.AccountID,
		Name:             name,
		Amount:           input.Amount,
		Type:             input.Type,
		TransactionDate:  transactionDate,
		IsPaid:           isPaid,
		Notes:            notes,
		CategoryID:       input.CategoryID,
		SettlementIntent: v2SettlementIntent,
		// CCState is computed from billedAt and isPaid (nil billedAt + false isPaid = pending)
	}

	created, err := s.transactionRepo.Create(transaction)
	if err != nil {
		return nil, err
	}

	// Publish event for real-time updates
	s.publishEvent(workspaceID, websocket.TransactionCreated(created))

	return created, nil
}

// GetTransactions retrieves transactions for a workspace with optional filters and pagination
// If requesting future dates, ensures projections exist (on-access projection generation)
func (s *TransactionService) GetTransactions(workspaceID int32, filters *domain.TransactionFilters) (*domain.PaginatedTransactions, error) {
	// Check if requesting future dates and ensure projections exist
	if filters != nil && filters.EndDate != nil && s.templateRepo != nil {
		now := time.Now()
		if filters.EndDate.After(now) {
			// Ensure projections exist for the requested date range
			s.ensureProjectionsForDateRange(workspaceID, *filters.EndDate)
		}
	}

	return s.transactionRepo.GetByWorkspace(workspaceID, filters)
}

// GetTransactionByID retrieves a transaction by ID within a workspace
func (s *TransactionService) GetTransactionByID(workspaceID int32, id int32) (*domain.Transaction, error) {
	return s.transactionRepo.GetByID(workspaceID, id)
}

// TogglePaidStatus toggles the paid status of a transaction
func (s *TransactionService) TogglePaidStatus(workspaceID int32, id int32) (*domain.Transaction, error) {
	updated, err := s.transactionRepo.TogglePaid(workspaceID, id)
	if err != nil {
		return nil, err
	}

	// Publish event for real-time updates
	s.publishEvent(workspaceID, websocket.TransactionUpdated(updated))

	return updated, nil
}

// ToggleBilled toggles the billed state of a CC transaction between pending and billed
func (s *TransactionService) ToggleBilled(workspaceID int32, id int32) (*domain.Transaction, error) {
	txn, err := s.transactionRepo.GetByID(workspaceID, id)
	if err != nil {
		return nil, err
	}

	// Verify this is a CC transaction (has settlement intent)
	if txn.SettlementIntent == nil {
		return nil, domain.ErrNotCCTransaction
	}

	// Compute current CC state from billedAt and isPaid
	// - pending: billedAt IS NULL AND isPaid = false
	// - billed: billedAt IS NOT NULL AND isPaid = false
	// - settled: isPaid = true
	if txn.IsPaid {
		// Cannot toggle settled transactions
		return nil, domain.ErrInvalidCCStateTransition
	}

	now := time.Now()
	var newBilledAt *time.Time

	if txn.BilledAt == nil {
		// Currently pending -> toggle to billed
		newBilledAt = &now
	} else {
		// Currently billed -> toggle back to pending
		newBilledAt = nil
	}

	// Update the transaction with new billedAt
	updated, err := s.transactionRepo.Update(workspaceID, id, &domain.UpdateTransactionData{
		Name:             txn.Name,
		Amount:           txn.Amount,
		Type:             txn.Type,
		TransactionDate:  txn.TransactionDate,
		AccountID:        txn.AccountID,
		Notes:            txn.Notes,
		CategoryID:       txn.CategoryID,
		IsPaid:           txn.IsPaid, // Preserve isPaid (should be false here)
		BilledAt:         newBilledAt,
		SettlementIntent: txn.SettlementIntent,
	})
	if err != nil {
		return nil, err
	}

	// Publish event for real-time updates
	s.publishEvent(workspaceID, websocket.TransactionBilled(updated))

	return updated, nil
}

// UpdateTransactionInput holds the input for updating a transaction
type UpdateTransactionInput struct {
	Name             string
	Amount           decimal.Decimal
	Type             domain.TransactionType
	TransactionDate  time.Time
	AccountID        int32
	Notes            *string
	CategoryID       *int32
	SettlementIntent *domain.SettlementIntent // Only for CC transactions
}

// UpdateTransaction updates an existing transaction with validation
func (s *TransactionService) UpdateTransaction(workspaceID int32, id int32, input UpdateTransactionInput) (*domain.Transaction, error) {
	// Fetch existing transaction to preserve CC fields
	existing, err := s.transactionRepo.GetByID(workspaceID, id)
	if err != nil {
		return nil, err
	}

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
	_, err = s.accountRepo.GetByID(workspaceID, input.AccountID)
	if err != nil {
		return nil, domain.ErrAccountNotFound
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

	// Validate category exists and belongs to workspace if provided
	if input.CategoryID != nil {
		_, err := s.categoryRepo.GetByID(workspaceID, *input.CategoryID)
		if err != nil {
			return nil, domain.ErrBudgetCategoryNotFound
		}
	}

	// Preserve existing CC fields, only update settlementIntent if provided
	settlementIntent := existing.SettlementIntent
	if input.SettlementIntent != nil {
		settlementIntent = input.SettlementIntent
	}

	updated, err := s.transactionRepo.Update(workspaceID, id, &domain.UpdateTransactionData{
		Name:             name,
		Amount:           input.Amount,
		Type:             input.Type,
		TransactionDate:  input.TransactionDate,
		AccountID:        input.AccountID,
		Notes:            notes,
		CategoryID:       input.CategoryID,
		// Preserve CC lifecycle fields (v2 simplified)
		IsPaid:           existing.IsPaid,
		BilledAt:         existing.BilledAt,
		SettlementIntent: settlementIntent,
		// Preserve recurring/projection fields
		Source:      existing.Source,
		TemplateID:  existing.TemplateID,
		IsProjected: existing.IsProjected,
	})
	if err != nil {
		return nil, err
	}

	// Publish event for real-time updates
	s.publishEvent(workspaceID, websocket.TransactionUpdated(updated))

	return updated, nil
}

// DeleteTransaction soft deletes a transaction (or both sides of a transfer)
// For projected transactions, it also creates an exclusion record to prevent re-creation
func (s *TransactionService) DeleteTransaction(workspaceID int32, id int32) error {
	// Get transaction first to check if it's a transfer or projected
	tx, err := s.transactionRepo.GetByID(workspaceID, id)
	if err != nil {
		return err
	}

	// If it's a transfer, delete both linked transactions
	if tx.TransferPairID != nil {
		err := s.transactionRepo.SoftDeleteTransferPair(workspaceID, *tx.TransferPairID)
		if err != nil {
			return err
		}
		// Publish delete events for both transactions
		s.publishEvent(workspaceID, websocket.TransactionDeleted(map[string]any{"id": id, "transferPairId": tx.TransferPairID.String()}))
		return nil
	}

	// If it's a projected transaction from a template, create an exclusion
	if tx.IsProjected && tx.TemplateID != nil && s.exclusionRepo != nil {
		monthStart := time.Date(tx.TransactionDate.Year(), tx.TransactionDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		// Ignore error - idempotent operation, exclusion might already exist
		_ = s.exclusionRepo.Create(workspaceID, *tx.TemplateID, monthStart)
	}

	// Regular delete
	err = s.transactionRepo.SoftDelete(workspaceID, id)
	if err != nil {
		return err
	}

	// Publish event for real-time updates
	s.publishEvent(workspaceID, websocket.TransactionDeleted(map[string]any{"id": id}))

	return nil
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

	result, err := s.transactionRepo.CreateTransferPair(fromTx, toTx)
	if err != nil {
		return nil, err
	}

	// Publish events for both created transactions
	s.publishEvent(workspaceID, websocket.TransactionCreated(result.FromTransaction))
	s.publishEvent(workspaceID, websocket.TransactionCreated(result.ToTransaction))

	return result, nil
}

// GetRecentlyUsedCategories returns recently used categories for suggestions dropdown
func (s *TransactionService) GetRecentlyUsedCategories(workspaceID int32) ([]*domain.RecentCategory, error) {
	return s.transactionRepo.GetRecentlyUsedCategories(workspaceID)
}

// ensureProjectionsForDateRange ensures projections exist up to the target date (on-access generation)
// This is transparent to the user - projections are generated within the same API call
func (s *TransactionService) ensureProjectionsForDateRange(workspaceID int32, targetDate time.Time) {
	if s.templateRepo == nil {
		return
	}

	// Get all active templates for this workspace
	templates, err := s.templateRepo.GetActive(workspaceID)
	if err != nil {
		log.Error().
			Err(err).
			Int32("workspaceID", workspaceID).
			Msg("Failed to get active templates for on-access projection generation")
		return
	}

	for _, template := range templates {
		// Skip if template has ended before target date
		if template.EndDate != nil && template.EndDate.Before(targetDate) {
			continue
		}

		// Generate projections up to target date
		s.generateProjectionsUpTo(workspaceID, template, targetDate)
	}
}

// generateProjectionsUpTo creates missing projections for a template up to the target date
func (s *TransactionService) generateProjectionsUpTo(workspaceID int32, template *domain.RecurringTemplate, targetDate time.Time) {
	// Get existing projections to check what's missing
	existingProjections, err := s.transactionRepo.GetProjectionsByTemplate(workspaceID, template.ID)
	if err != nil {
		log.Error().
			Err(err).
			Int32("workspaceID", workspaceID).
			Int32("templateID", template.ID).
			Msg("Failed to get existing projections for on-access generation")
		return
	}

	// Build set of existing projection months
	existingMonths := make(map[string]bool)
	for _, proj := range existingProjections {
		monthKey := proj.TransactionDate.Format("2006-01")
		existingMonths[monthKey] = true
	}

	// Calculate start date for new projections
	now := time.Now()
	targetDay := template.StartDate.Day()

	var startDate time.Time
	if template.StartDate.After(now) {
		startDate = template.StartDate
	} else {
		startDate = s.calculateActualDate(now.Year(), now.Month(), targetDay)
		if startDate.Before(now) || startDate.Equal(now) {
			nextMonth := now.AddDate(0, 1, 0)
			startDate = s.calculateActualDate(nextMonth.Year(), nextMonth.Month(), targetDay)
		}
	}

	// Calculate end of target month
	endOfTargetMonth := time.Date(targetDate.Year(), targetDate.Month()+1, 0, 0, 0, 0, 0, time.UTC)

	// Use template end_date if it's earlier
	if template.EndDate != nil && template.EndDate.Before(endOfTargetMonth) {
		endOfTargetMonth = *template.EndDate
	}

	// Generate projections month by month
	current := startDate
	for !current.After(endOfTargetMonth) {
		actualDate := s.calculateActualDate(current.Year(), current.Month(), targetDay)
		monthKey := actualDate.Format("2006-01")

		// Skip if projection already exists
		if existingMonths[monthKey] {
			current = current.AddDate(0, 1, 0)
			continue
		}

		// Check if this month is excluded (user explicitly deleted)
		if s.exclusionRepo != nil {
			monthStart := time.Date(current.Year(), current.Month(), 1, 0, 0, 0, 0, time.UTC)
			excluded, err := s.exclusionRepo.IsExcluded(workspaceID, template.ID, monthStart)
			if err == nil && excluded {
				current = current.AddDate(0, 1, 0)
				continue
			}
		}

		// Create projection transaction
		transaction := &domain.Transaction{
			WorkspaceID:     workspaceID,
			Name:            template.Description,
			Amount:          template.Amount,
			Type:            domain.TransactionTypeExpense,
			CategoryID:      &template.CategoryID,
			AccountID:       template.AccountID,
			TransactionDate: actualDate,
			Source:          "recurring",
			TemplateID:      &template.ID,
			IsProjected:     true,
			IsPaid:          false,
		}

		if _, err := s.transactionRepo.Create(transaction); err != nil {
			log.Error().
				Err(err).
				Int32("workspaceID", workspaceID).
				Int32("templateID", template.ID).
				Str("month", monthKey).
				Msg("Failed to create projection during on-access generation")
		}
		current = current.AddDate(0, 1, 0)
	}
}

// calculateActualDate returns the actual date for a target day in a given month
// Handles months with fewer days (e.g., 31st in February -> 28th/29th)
func (s *TransactionService) calculateActualDate(year int, month time.Month, targetDay int) time.Time {
	return util.CalculateActualDate(year, month, targetDay)
}

// EnrichWithModificationStatus checks if projected transactions differ from their templates
// and sets the IsModified flag accordingly
func (s *TransactionService) EnrichWithModificationStatus(workspaceID int32, transactions []*domain.Transaction) {
	if s.templateRepo == nil {
		return
	}

	// Collect unique template IDs from projected transactions
	templateIDs := make(map[int32]bool)
	for _, tx := range transactions {
		if tx.IsProjected && tx.TemplateID != nil {
			templateIDs[*tx.TemplateID] = true
		}
	}

	if len(templateIDs) == 0 {
		return
	}

	// Fetch templates and build a lookup map
	templateMap := make(map[int32]*domain.RecurringTemplate)
	for templateID := range templateIDs {
		template, err := s.templateRepo.GetByID(workspaceID, templateID)
		if err != nil {
			continue
		}
		templateMap[templateID] = template
	}

	// Enrich transactions with modification status
	for _, tx := range transactions {
		if !tx.IsProjected || tx.TemplateID == nil {
			continue
		}

		template, exists := templateMap[*tx.TemplateID]
		if !exists {
			continue
		}

		// Check if transaction differs from template
		tx.IsModified = !tx.Amount.Equal(template.Amount) ||
			(tx.CategoryID != nil && *tx.CategoryID != template.CategoryID) ||
			(tx.CategoryID == nil && template.CategoryID != 0) ||
			tx.Name != template.Description
	}
}

// GetCCMetrics returns CC metrics (pending, billed, month total) for a workspace and month
func (s *TransactionService) GetCCMetrics(workspaceID int32, month time.Time) (*domain.CCMetrics, error) {
	// Calculate start and end of month
	startOfMonth := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	return s.transactionRepo.GetCCMetrics(workspaceID, startOfMonth, endOfMonth)
}

// BatchToggleToBilled toggles multiple pending transactions to billed state
func (s *TransactionService) BatchToggleToBilled(workspaceID int32, ids []int32) ([]*domain.Transaction, error) {
	if len(ids) == 0 {
		return []*domain.Transaction{}, nil
	}

	transactions, err := s.transactionRepo.BatchToggleToBilled(workspaceID, ids)
	if err != nil {
		return nil, err
	}

	// Publish events for each updated transaction
	for _, tx := range transactions {
		s.publishEvent(workspaceID, websocket.TransactionBilled(tx))
	}

	return transactions, nil
}

// GetDeferredForSettlement returns all billed+deferred transactions that need settlement
func (s *TransactionService) GetDeferredForSettlement(workspaceID int32) ([]*domain.Transaction, error) {
	return s.transactionRepo.GetDeferredForSettlement(workspaceID)
}

// GetImmediateForSettlement returns billed transactions with immediate intent for the current month
func (s *TransactionService) GetImmediateForSettlement(workspaceID int32, month time.Time) ([]*domain.Transaction, error) {
	startOfMonth := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)
	return s.transactionRepo.GetImmediateForSettlement(workspaceID, startOfMonth, endOfMonth)
}

// UpdateAmount updates only the amount field of a transaction
// This is used for overdue items where only amount adjustment (for interest/fees) is allowed
func (s *TransactionService) UpdateAmount(workspaceID int32, id int32, amount decimal.Decimal) (*domain.Transaction, error) {
	// Validate amount is positive
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrInvalidAmount
	}

	// Get existing transaction
	existing, err := s.transactionRepo.GetByID(workspaceID, id)
	if err != nil {
		return nil, err
	}

	// Only update the amount, preserve everything else
	updateData := &domain.UpdateTransactionData{
		Name:             existing.Name,
		Amount:           amount,
		Type:             existing.Type,
		TransactionDate:  existing.TransactionDate,
		AccountID:        existing.AccountID,
		Notes:            existing.Notes,
		CategoryID:       existing.CategoryID,
		IsPaid:           existing.IsPaid,
		BilledAt:         existing.BilledAt,
		SettlementIntent: existing.SettlementIntent,
	}

	return s.transactionRepo.Update(workspaceID, id, updateData)
}

// GetOverdue returns overdue CC transactions grouped by month
func (s *TransactionService) GetOverdue(workspaceID int32) ([]domain.OverdueGroup, error) {
	transactions, err := s.transactionRepo.GetOverdueCC(workspaceID)
	if err != nil {
		return nil, err
	}

	// Group by month
	groups := make(map[string]*domain.OverdueGroup)
	monthOrder := []string{}

	for _, txn := range transactions {
		var monthKey string
		var monthLabel string

		// Use BilledAt for grouping (overdue is based on when billed)
		if txn.BilledAt != nil {
			monthKey = txn.BilledAt.Format("2006-01")
			monthLabel = txn.BilledAt.Format("January 2006")
		} else {
			// Fallback to transaction date if BilledAt is missing
			monthKey = txn.TransactionDate.Format("2006-01")
			monthLabel = txn.TransactionDate.Format("January 2006")
		}

		if _, exists := groups[monthKey]; !exists {
			// Calculate months overdue
			monthsOverdue := calculateMonthsOverdue(txn.BilledAt)

			groups[monthKey] = &domain.OverdueGroup{
				Month:         monthKey,
				MonthLabel:    monthLabel,
				MonthsOverdue: monthsOverdue,
				TotalAmount:   decimal.Zero,
				ItemCount:     0,
				Transactions:  make([]*domain.Transaction, 0),
			}
			monthOrder = append(monthOrder, monthKey)
		}

		group := groups[monthKey]
		group.TotalAmount = group.TotalAmount.Add(txn.Amount)
		group.ItemCount++
		group.Transactions = append(group.Transactions, txn)
	}

	// Convert to slice and sort by month (oldest first)
	result := make([]domain.OverdueGroup, 0, len(groups))
	for _, monthKey := range monthOrder {
		result = append(result, *groups[monthKey])
	}

	return result, nil
}

// calculateMonthsOverdue calculates how many months a transaction is overdue
func calculateMonthsOverdue(billedAt *time.Time) int {
	if billedAt == nil {
		return 0
	}

	now := time.Now()
	months := (now.Year()-billedAt.Year())*12 + int(now.Month()) - int(billedAt.Month())

	// Account for day of month: if we haven't reached the billed day yet this month,
	// subtract 1 from the count (e.g., billed Jan 31, today Feb 1 = 0 months, not 1)
	if now.Day() < billedAt.Day() {
		months--
	}

	// Ensure non-negative (shouldn't happen for overdue items, but safety check)
	if months < 0 {
		return 0
	}
	return months
}
