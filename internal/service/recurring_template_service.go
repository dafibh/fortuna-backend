package service

import (
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/util"
	"github.com/dafibh/fortuna/fortuna-backend/internal/websocket"
	"github.com/shopspring/decimal"
)

// RecurringTemplateServiceImpl handles recurring template business logic (v2)
type RecurringTemplateServiceImpl struct {
	templateRepo    domain.RecurringTemplateRepository
	transactionRepo domain.TransactionRepository
	accountRepo     domain.AccountRepository
	categoryRepo    domain.BudgetCategoryRepository
	exclusionRepo   domain.ProjectionExclusionRepository
	eventPublisher  websocket.EventPublisher
}

// NewRecurringTemplateService creates a new RecurringTemplateService
func NewRecurringTemplateService(
	templateRepo domain.RecurringTemplateRepository,
	transactionRepo domain.TransactionRepository,
	accountRepo domain.AccountRepository,
	categoryRepo domain.BudgetCategoryRepository,
) *RecurringTemplateServiceImpl {
	return &RecurringTemplateServiceImpl{
		templateRepo:    templateRepo,
		transactionRepo: transactionRepo,
		accountRepo:     accountRepo,
		categoryRepo:    categoryRepo,
	}
}

// SetExclusionRepository sets the exclusion repository for projection exclusion tracking
func (s *RecurringTemplateServiceImpl) SetExclusionRepository(exclusionRepo domain.ProjectionExclusionRepository) {
	s.exclusionRepo = exclusionRepo
}

// SetEventPublisher sets the event publisher for real-time updates
func (s *RecurringTemplateServiceImpl) SetEventPublisher(publisher websocket.EventPublisher) {
	s.eventPublisher = publisher
}

// publishEvent publishes a WebSocket event if a publisher is configured
func (s *RecurringTemplateServiceImpl) publishEvent(workspaceID int32, event websocket.Event) {
	if s.eventPublisher != nil {
		s.eventPublisher.Publish(workspaceID, event)
	}
}

// CreateTemplate creates a new recurring template and generates projections
func (s *RecurringTemplateServiceImpl) CreateTemplate(workspaceID int32, input domain.CreateRecurringTemplateInput) (*domain.RecurringTemplate, error) {
	// Validate input
	if err := s.validateCreateInput(input); err != nil {
		return nil, err
	}

	// Validate account exists and belongs to workspace
	_, err := s.accountRepo.GetByID(workspaceID, input.AccountID)
	if err != nil {
		return nil, domain.ErrAccountNotFound
	}

	// Validate category exists and belongs to workspace
	_, err = s.categoryRepo.GetByID(workspaceID, input.CategoryID)
	if err != nil {
		return nil, domain.ErrBudgetCategoryNotFound
	}

	// If linkTransactionID provided, validate transaction exists and belongs to workspace BEFORE creating template
	if input.LinkTransactionID != nil {
		_, err := s.transactionRepo.GetByID(workspaceID, *input.LinkTransactionID)
		if err != nil {
			return nil, domain.ErrTransactionNotFound
		}
	}

	// Create the template
	template := &domain.RecurringTemplate{
		WorkspaceID:      workspaceID,
		Description:      input.Description,
		Amount:           input.Amount,
		CategoryID:       input.CategoryID,
		AccountID:        input.AccountID,
		Frequency:        input.Frequency,
		StartDate:        input.StartDate,
		EndDate:          input.EndDate,
		SettlementIntent: input.SettlementIntent,
	}

	created, err := s.templateRepo.Create(template)
	if err != nil {
		return nil, err
	}

	// If linkTransactionID provided, link the existing transaction to this template
	if input.LinkTransactionID != nil {
		if err := s.linkTransactionToTemplate(workspaceID, *input.LinkTransactionID, created.ID); err != nil {
			// Transaction link failed - delete the template to maintain consistency
			_ = s.templateRepo.Delete(workspaceID, created.ID)
			return nil, err
		}
	}

	// Generate projections for 12 months - fail template creation if projections fail
	if err := s.generateProjections(workspaceID, created); err != nil {
		// Projection generation failed - clean up template
		_ = s.templateRepo.Delete(workspaceID, created.ID)
		return nil, err
	}

	// Publish event for real-time updates
	s.publishEvent(workspaceID, websocket.RecurringCreated(created))

	return created, nil
}

// linkTransactionToTemplate updates an existing transaction to link it to a template
func (s *RecurringTemplateServiceImpl) linkTransactionToTemplate(workspaceID int32, transactionID int32, templateID int32) error {
	// Get the existing transaction to verify it exists and belongs to workspace
	existingTx, err := s.transactionRepo.GetByID(workspaceID, transactionID)
	if err != nil {
		return err
	}

	// Update the transaction with template link
	updateData := &domain.UpdateTransactionData{
		Name:            existingTx.Name,
		Amount:          existingTx.Amount,
		Type:            existingTx.Type,
		TransactionDate: existingTx.TransactionDate,
		AccountID:       existingTx.AccountID,
		Notes:           existingTx.Notes,
		CategoryID:      existingTx.CategoryID,
		// Link to template
		Source:      "recurring",
		TemplateID:  &templateID,
		IsProjected: false, // This is an actual transaction, not a projection
	}

	_, err = s.transactionRepo.Update(workspaceID, transactionID, updateData)
	return err
}

// UpdateTemplate updates a recurring template and recalculates projections
func (s *RecurringTemplateServiceImpl) UpdateTemplate(workspaceID int32, id int32, input domain.UpdateRecurringTemplateInput) (*domain.RecurringTemplate, error) {
	// Validate input
	if err := s.validateUpdateInput(input); err != nil {
		return nil, err
	}

	// Verify template exists
	existing, err := s.templateRepo.GetByID(workspaceID, id)
	if err != nil {
		return nil, err
	}

	// Validate account exists and belongs to workspace
	_, err = s.accountRepo.GetByID(workspaceID, input.AccountID)
	if err != nil {
		return nil, domain.ErrAccountNotFound
	}

	// Validate category exists and belongs to workspace
	_, err = s.categoryRepo.GetByID(workspaceID, input.CategoryID)
	if err != nil {
		return nil, domain.ErrBudgetCategoryNotFound
	}

	// Update the template
	updated, err := s.templateRepo.Update(workspaceID, id, &input)
	if err != nil {
		return nil, err
	}

	// Recalculate projections: delete existing and regenerate
	// Note: We preserve user-edited instances by checking if values differ from template
	if err := s.recalculateProjections(workspaceID, existing, updated); err != nil {
		return nil, err
	}

	// Publish event for real-time updates
	s.publishEvent(workspaceID, websocket.RecurringUpdated(updated))

	return updated, nil
}

// DeleteTemplate deletes a template with cascade logic
func (s *RecurringTemplateServiceImpl) DeleteTemplate(workspaceID int32, id int32) error {
	// Verify template exists
	_, err := s.templateRepo.GetByID(workspaceID, id)
	if err != nil {
		return err
	}

	// 1. Delete all projected transactions (is_projected=true)
	if err := s.transactionRepo.DeleteProjectionsByTemplate(workspaceID, id); err != nil {
		return err
	}

	// 2. Orphan actual transactions (is_projected=false) by setting template_id=NULL
	if err := s.transactionRepo.OrphanActualsByTemplate(workspaceID, id); err != nil {
		return err
	}

	// 3. Delete the template
	if err := s.templateRepo.Delete(workspaceID, id); err != nil {
		return err
	}

	// Publish event for real-time updates
	s.publishEvent(workspaceID, websocket.RecurringDeleted(map[string]any{"id": id}))

	return nil
}

// GetTemplate retrieves a single template by ID
func (s *RecurringTemplateServiceImpl) GetTemplate(workspaceID int32, id int32) (*domain.RecurringTemplate, error) {
	return s.templateRepo.GetByID(workspaceID, id)
}

// ListTemplates retrieves all templates for a workspace
func (s *RecurringTemplateServiceImpl) ListTemplates(workspaceID int32) ([]*domain.RecurringTemplate, error) {
	return s.templateRepo.ListByWorkspace(workspaceID)
}

// validateCreateInput validates input for creating a template
func (s *RecurringTemplateServiceImpl) validateCreateInput(input domain.CreateRecurringTemplateInput) error {
	if input.Description == "" {
		return domain.ErrNameRequired
	}
	if len(input.Description) > domain.MaxTransactionNameLength {
		return domain.ErrNameTooLong
	}
	if input.Amount.LessThanOrEqual(decimal.Zero) {
		return domain.ErrInvalidAmount
	}
	if input.CategoryID <= 0 {
		return domain.ErrBudgetCategoryNotFound
	}
	if input.AccountID <= 0 {
		return domain.ErrAccountNotFound
	}
	if input.Frequency != "monthly" {
		return domain.ErrInvalidFrequency
	}
	// Validate end date is after start date if provided
	if input.EndDate != nil && input.EndDate.Before(input.StartDate) {
		return domain.ErrInvalidDateRange
	}
	return nil
}

// validateUpdateInput validates input for updating a template
func (s *RecurringTemplateServiceImpl) validateUpdateInput(input domain.UpdateRecurringTemplateInput) error {
	if input.Description == "" {
		return domain.ErrNameRequired
	}
	if len(input.Description) > domain.MaxTransactionNameLength {
		return domain.ErrNameTooLong
	}
	if input.Amount.LessThanOrEqual(decimal.Zero) {
		return domain.ErrInvalidAmount
	}
	if input.CategoryID <= 0 {
		return domain.ErrBudgetCategoryNotFound
	}
	if input.AccountID <= 0 {
		return domain.ErrAccountNotFound
	}
	if input.Frequency != "monthly" {
		return domain.ErrInvalidFrequency
	}
	// Validate end date is after start date if provided
	if input.EndDate != nil && input.EndDate.Before(input.StartDate) {
		return domain.ErrInvalidDateRange
	}
	return nil
}

// getSettlementIntentForTemplate returns the settlement intent for CC templates
// CCState is now computed from isPaid and billedAt, so we only need settlement intent
func (s *RecurringTemplateServiceImpl) getSettlementIntentForTemplate(workspaceID int32, template *domain.RecurringTemplate) *domain.SettlementIntent {
	account, err := s.accountRepo.GetByID(workspaceID, template.AccountID)
	if err != nil {
		return nil
	}
	if account.Template != domain.TemplateCreditCard {
		return nil
	}
	// Use template's settlement intent if set, otherwise default to deferred
	var settlementIntent domain.SettlementIntent
	if template.SettlementIntent != nil {
		settlementIntent = *template.SettlementIntent
	} else {
		settlementIntent = domain.SettlementIntentDeferred
	}
	return &settlementIntent
}

// generateProjections creates projected transactions for a template
func (s *RecurringTemplateServiceImpl) generateProjections(workspaceID int32, template *domain.RecurringTemplate) error {
	// Check for existing projections (idempotency check)
	existingProjections, err := s.transactionRepo.GetProjectionsByTemplate(workspaceID, template.ID)
	if err != nil {
		return err
	}

	// Build a set of existing projection months to avoid duplicates (month precision)
	existingMonths := make(map[string]bool)
	for _, proj := range existingProjections {
		monthKey := proj.TransactionDate.Format("2006-01")
		existingMonths[monthKey] = true
	}

	// Calculate projection range
	now := time.Now()
	targetDay := template.StartDate.Day()

	// Calculate start date for projections
	var startDate time.Time
	if template.StartDate.After(now) {
		// Template starts in the future - use template start date
		startDate = template.StartDate
	} else {
		// Template starts in the past - find the next valid projection date
		// Use calculateActualDate to properly handle month-end edge cases (e.g., 31st in Feb -> Feb 28/29)
		startDate = s.calculateActualDate(now.Year(), now.Month(), targetDay)
		if startDate.Before(now) || startDate.Equal(now) {
			// If we've passed that day this month (or it's today), start next month
			nextMonth := now.AddDate(0, 1, 0)
			startDate = s.calculateActualDate(nextMonth.Year(), nextMonth.Month(), targetDay)
		}
	}

	// End date: MIN(template.EndDate, startDate + 12 months)
	endDate := startDate.AddDate(0, 12, 0)
	if template.EndDate != nil && template.EndDate.Before(endDate) {
		endDate = *template.EndDate
	}

	// Generate one transaction per month
	current := startDate
	for !current.After(endDate) {
		// Calculate the actual day for this month (handle months with fewer days)
		targetDay := template.StartDate.Day()
		actualDate := s.calculateActualDate(current.Year(), current.Month(), targetDay)
		monthKey := actualDate.Format("2006-01")

		// Skip if projection already exists for this month (idempotency)
		if existingMonths[monthKey] {
			current = current.AddDate(0, 1, 0)
			continue
		}

		// Check if this month is excluded (user explicitly deleted a projection)
		if s.exclusionRepo != nil {
			monthStart := time.Date(current.Year(), current.Month(), 1, 0, 0, 0, 0, time.UTC)
			excluded, err := s.exclusionRepo.IsExcluded(workspaceID, template.ID, monthStart)
			if err == nil && excluded {
				current = current.AddDate(0, 1, 0)
				continue
			}
		}

		// Get settlement intent if this is a CC account
		settlementIntent := s.getSettlementIntentForTemplate(workspaceID, template)

		transaction := &domain.Transaction{
			WorkspaceID:      workspaceID,
			Name:             template.Description,
			Amount:           template.Amount,
			Type:             domain.TransactionTypeExpense, // Default to expense
			CategoryID:       &template.CategoryID,
			AccountID:        template.AccountID,
			TransactionDate:  actualDate,
			Source:           "recurring",
			TemplateID:       &template.ID,
			IsProjected:      true,
			IsPaid:           false, // CCState computed from isPaid and billedAt (both nil = pending)
			SettlementIntent: settlementIntent,
		}

		if _, err := s.transactionRepo.Create(transaction); err != nil {
			return err
		}

		// Move to next month
		current = current.AddDate(0, 1, 0)
	}

	return nil
}

// recalculateProjections updates existing projections when template changes
// User-edited projections are PRESERVED, unedited ones are updated with new template values
func (s *RecurringTemplateServiceImpl) recalculateProjections(workspaceID int32, oldTemplate, newTemplate *domain.RecurringTemplate) error {
	// Get existing projections to check for user edits
	existingProjections, err := s.transactionRepo.GetProjectionsByTemplate(workspaceID, newTemplate.ID)
	if err != nil {
		return err
	}

	// Build map of existing projections by month (month precision for consistency)
	existingByMonth := make(map[string]*domain.Transaction)
	for _, proj := range existingProjections {
		monthKey := proj.TransactionDate.Format("2006-01")
		existingByMonth[monthKey] = proj
	}

	// Get settlement intent based on new template
	settlementIntent := s.getSettlementIntentForTemplate(workspaceID, newTemplate)

	// Process each existing projection
	for _, proj := range existingProjections {
		if s.isUserEdited(proj, oldTemplate) {
			// PRESERVE user-edited projection - don't modify it
			continue
		}

		// Update unedited projection with new template values
		updateData := &domain.UpdateTransactionData{
			Name:             newTemplate.Description,
			Amount:           newTemplate.Amount,
			Type:             domain.TransactionTypeExpense,
			TransactionDate:  proj.TransactionDate,
			AccountID:        newTemplate.AccountID,
			CategoryID:       &newTemplate.CategoryID,
			Source:           "recurring",
			TemplateID:       &newTemplate.ID,
			IsProjected:      true,
			IsPaid:           proj.IsPaid, // Preserve current isPaid status
			BilledAt:         proj.BilledAt, // Preserve current billedAt
			SettlementIntent: settlementIntent,
		}
		if _, err := s.transactionRepo.Update(workspaceID, proj.ID, updateData); err != nil {
			return err
		}
	}

	// Generate any new projections for months that don't exist yet
	existingMonths := make(map[string]bool)
	for monthKey := range existingByMonth {
		existingMonths[monthKey] = true // Skip all existing months (both edited and just-updated)
	}
	return s.generateProjectionsWithSkips(workspaceID, newTemplate, existingMonths)
}

// isUserEdited checks if a projection has been modified from the template values
func (s *RecurringTemplateServiceImpl) isUserEdited(projection *domain.Transaction, template *domain.RecurringTemplate) bool {
	// Check if any key fields differ from the template
	if projection.Name != template.Description {
		return true
	}
	if !projection.Amount.Equal(template.Amount) {
		return true
	}
	if projection.CategoryID == nil || *projection.CategoryID != template.CategoryID {
		return true
	}
	if projection.AccountID != template.AccountID {
		return true
	}
	return false
}

// generateProjectionsWithSkips creates projections but skips specified months
func (s *RecurringTemplateServiceImpl) generateProjectionsWithSkips(workspaceID int32, template *domain.RecurringTemplate, skipMonths map[string]bool) error {
	// Get existing projections for idempotency check
	existingProjections, err := s.transactionRepo.GetProjectionsByTemplate(workspaceID, template.ID)
	if err != nil {
		return err
	}

	// Build set of existing projection months
	existingMonths := make(map[string]bool)
	for _, proj := range existingProjections {
		monthKey := proj.TransactionDate.Format("2006-01")
		existingMonths[monthKey] = true
	}

	// Calculate projection range
	now := time.Now()
	targetDay := template.StartDate.Day()

	// Calculate start date for projections
	var startDate time.Time
	if template.StartDate.After(now) {
		// Template starts in the future - use template start date
		startDate = template.StartDate
	} else {
		// Template starts in the past - find the next valid projection date
		// Use calculateActualDate to properly handle month-end edge cases (e.g., 31st in Feb -> Feb 28/29)
		startDate = s.calculateActualDate(now.Year(), now.Month(), targetDay)
		if startDate.Before(now) || startDate.Equal(now) {
			// If we've passed that day this month (or it's today), start next month
			nextMonth := now.AddDate(0, 1, 0)
			startDate = s.calculateActualDate(nextMonth.Year(), nextMonth.Month(), targetDay)
		}
	}

	endDate := startDate.AddDate(0, 12, 0)
	if template.EndDate != nil && template.EndDate.Before(endDate) {
		endDate = *template.EndDate
	}

	current := startDate
	for !current.After(endDate) {
		targetDay := template.StartDate.Day()
		actualDate := s.calculateActualDate(current.Year(), current.Month(), targetDay)
		monthKey := actualDate.Format("2006-01")

		// Skip if this month was passed in (user-edited or existing)
		if skipMonths[monthKey] {
			current = current.AddDate(0, 1, 0)
			continue
		}

		// Idempotency check: skip if projection already exists in database
		if existingMonths[monthKey] {
			current = current.AddDate(0, 1, 0)
			continue
		}

		// Check if this month is excluded (user explicitly deleted a projection)
		if s.exclusionRepo != nil {
			monthStart := time.Date(current.Year(), current.Month(), 1, 0, 0, 0, 0, time.UTC)
			excluded, err := s.exclusionRepo.IsExcluded(workspaceID, template.ID, monthStart)
			if err == nil && excluded {
				current = current.AddDate(0, 1, 0)
				continue
			}
		}

		// Get settlement intent if this is a CC account
		settlementIntent := s.getSettlementIntentForTemplate(workspaceID, template)

		transaction := &domain.Transaction{
			WorkspaceID:      workspaceID,
			Name:             template.Description,
			Amount:           template.Amount,
			Type:             domain.TransactionTypeExpense,
			CategoryID:       &template.CategoryID,
			AccountID:        template.AccountID,
			TransactionDate:  actualDate,
			Source:           "recurring",
			TemplateID:       &template.ID,
			IsProjected:      true,
			IsPaid:           false, // CCState computed from isPaid and billedAt
			SettlementIntent: settlementIntent,
		}

		if _, err := s.transactionRepo.Create(transaction); err != nil {
			return err
		}

		current = current.AddDate(0, 1, 0)
	}

	return nil
}

// calculateActualDate returns the actual date for a target day in a given month,
// handling months with fewer days (e.g., day 31 in February returns Feb 28/29)
func (s *RecurringTemplateServiceImpl) calculateActualDate(year int, month time.Month, targetDay int) time.Time {
	return util.CalculateActualDate(year, month, targetDay)
}
