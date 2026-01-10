package service

import (
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/shopspring/decimal"
)

// RecurringTemplateServiceImpl handles recurring template business logic (v2)
type RecurringTemplateServiceImpl struct {
	templateRepo    domain.RecurringTemplateRepository
	transactionRepo domain.TransactionRepository
	accountRepo     domain.AccountRepository
	categoryRepo    domain.BudgetCategoryRepository
}

// NewRecurringTemplateService creates a new RecurringTemplateService
func NewRecurringTemplateService(
	templateRepo domain.RecurringTemplateRepository,
	transactionRepo domain.TransactionRepository,
	accountRepo domain.AccountRepository,
	categoryRepo domain.BudgetCategoryRepository,
) domain.RecurringTemplateService {
	return &RecurringTemplateServiceImpl{
		templateRepo:    templateRepo,
		transactionRepo: transactionRepo,
		accountRepo:     accountRepo,
		categoryRepo:    categoryRepo,
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

	// Create the template
	template := &domain.RecurringTemplate{
		WorkspaceID: workspaceID,
		Description: input.Description,
		Amount:      input.Amount,
		CategoryID:  input.CategoryID,
		AccountID:   input.AccountID,
		Frequency:   input.Frequency,
		StartDate:   input.StartDate,
		EndDate:     input.EndDate,
	}

	created, err := s.templateRepo.Create(template)
	if err != nil {
		return nil, err
	}

	// If linkTransactionID provided, link the existing transaction to this template
	if input.LinkTransactionID != nil {
		_ = s.linkTransactionToTemplate(workspaceID, *input.LinkTransactionID, created.ID)
	}

	// Generate projections for 12 months (errors logged but don't fail template creation)
	_ = s.generateProjections(workspaceID, created)

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
	_ = s.recalculateProjections(workspaceID, existing, updated)

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
	return s.templateRepo.Delete(workspaceID, id)
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
	return nil
}

// generateProjections creates projected transactions for a template
func (s *RecurringTemplateServiceImpl) generateProjections(workspaceID int32, template *domain.RecurringTemplate) error {
	// Calculate projection range
	startDate := template.StartDate
	now := time.Now()
	if startDate.Before(now) {
		// Start from beginning of current month if start date is in the past
		startDate = time.Date(now.Year(), now.Month(), template.StartDate.Day(), 0, 0, 0, 0, time.UTC)
		if startDate.Before(now) {
			// If we've passed that day this month, start next month
			startDate = startDate.AddDate(0, 1, 0)
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

		transaction := &domain.Transaction{
			WorkspaceID: workspaceID,
			Name:        template.Description,
			Amount:      template.Amount,
			Type:        domain.TransactionTypeExpense, // Default to expense
			CategoryID:  &template.CategoryID,
			AccountID:   template.AccountID,
			TransactionDate: actualDate,
			Source:      "recurring",
			TemplateID:  &template.ID,
			IsProjected: true,
			IsPaid:      false,
		}

		if _, err := s.transactionRepo.Create(transaction); err != nil {
			return err
		}

		// Move to next month
		current = current.AddDate(0, 1, 0)
	}

	return nil
}

// recalculateProjections deletes existing projections and regenerates them
func (s *RecurringTemplateServiceImpl) recalculateProjections(workspaceID int32, oldTemplate, newTemplate *domain.RecurringTemplate) error {
	// Get existing projections to check for user edits
	existingProjections, err := s.transactionRepo.GetProjectionsByTemplate(workspaceID, newTemplate.ID)
	if err != nil {
		return err
	}

	// Find projections that have been user-edited (values differ from old template)
	editedDates := make(map[string]bool)
	for _, proj := range existingProjections {
		if s.isUserEdited(proj, oldTemplate) {
			// Mark this date as user-edited (don't regenerate)
			dateKey := proj.TransactionDate.Format("2006-01-02")
			editedDates[dateKey] = true
		}
	}

	// Delete all projections (we'll regenerate non-edited ones)
	if err := s.transactionRepo.DeleteProjectionsByTemplate(workspaceID, newTemplate.ID); err != nil {
		return err
	}

	// Regenerate projections, skipping user-edited dates
	return s.generateProjectionsWithSkips(workspaceID, newTemplate, editedDates)
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

// generateProjectionsWithSkips creates projections but skips specified dates
func (s *RecurringTemplateServiceImpl) generateProjectionsWithSkips(workspaceID int32, template *domain.RecurringTemplate, skipDates map[string]bool) error {
	// Calculate projection range
	startDate := template.StartDate
	now := time.Now()
	if startDate.Before(now) {
		startDate = time.Date(now.Year(), now.Month(), template.StartDate.Day(), 0, 0, 0, 0, time.UTC)
		if startDate.Before(now) {
			startDate = startDate.AddDate(0, 1, 0)
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
		dateKey := actualDate.Format("2006-01-02")

		// Skip if this date was user-edited
		if skipDates[dateKey] {
			current = current.AddDate(0, 1, 0)
			continue
		}

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
			return err
		}

		current = current.AddDate(0, 1, 0)
	}

	return nil
}

// calculateActualDate returns the actual date for a target day in a given month,
// handling months with fewer days (e.g., day 31 in February returns Feb 28/29)
func (s *RecurringTemplateServiceImpl) calculateActualDate(year int, month time.Month, targetDay int) time.Time {
	// Get last day of month by going to day 0 of next month
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()

	actualDay := targetDay
	if actualDay > lastDay {
		actualDay = lastDay
	}

	return time.Date(year, month, actualDay, 0, 0, 0, 0, time.UTC)
}
