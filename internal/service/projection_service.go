package service

import (
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
)

const (
	// DefaultProjectionMonths is the default number of months to project ahead
	DefaultProjectionMonths = 12
)

// ProjectionService handles projection generation for recurring templates
type ProjectionService struct {
	recurringRepo   domain.RecurringRepository
	transactionRepo domain.TransactionRepository
}

// NewProjectionService creates a new ProjectionService
func NewProjectionService(
	recurringRepo domain.RecurringRepository,
	transactionRepo domain.TransactionRepository,
) *ProjectionService {
	return &ProjectionService{
		recurringRepo:   recurringRepo,
		transactionRepo: transactionRepo,
	}
}

// ProjectionResult holds the result of projection generation
type ProjectionResult struct {
	Generated int      `json:"generated"`
	Skipped   int      `json:"skipped"`
	Errors    []string `json:"errors,omitempty"`
}

// GenerateProjections generates projected transactions for all active recurring templates
// for the specified number of months ahead from today.
// Returns the number of projections created, skipped, and any errors encountered.
func (s *ProjectionService) GenerateProjections(workspaceID int32, monthsAhead int) (*ProjectionResult, error) {
	if monthsAhead <= 0 {
		monthsAhead = DefaultProjectionMonths
	}

	result := &ProjectionResult{
		Generated: 0,
		Skipped:   0,
		Errors:    make([]string, 0),
	}

	// Get all active recurring templates
	activeOnly := true
	templates, err := s.recurringRepo.ListByWorkspace(workspaceID, &activeOnly)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	currentYear, currentMonth, _ := now.Date()

	// Calculate the end projection date (N months from now)
	endProjectionDate := now.AddDate(0, monthsAhead, 0)

	for _, template := range templates {
		templateResult := s.generateProjectionsForTemplate(workspaceID, template, currentYear, currentMonth, endProjectionDate)
		result.Generated += templateResult.Generated
		result.Skipped += templateResult.Skipped
		result.Errors = append(result.Errors, templateResult.Errors...)
	}

	return result, nil
}

// GenerateProjectionsForMonth generates projections for a specific month
// This is used for on-access projection generation (Story 3.3)
func (s *ProjectionService) GenerateProjectionsForMonth(workspaceID int32, year int, month time.Month) (*ProjectionResult, error) {
	result := &ProjectionResult{
		Generated: 0,
		Skipped:   0,
		Errors:    make([]string, 0),
	}

	// Get all active recurring templates
	activeOnly := true
	templates, err := s.recurringRepo.ListByWorkspace(workspaceID, &activeOnly)
	if err != nil {
		return nil, err
	}

	targetDate := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)

	for _, template := range templates {
		generated, skipped, projErr := s.generateSingleProjection(workspaceID, template, year, month, targetDate)
		result.Generated += generated
		result.Skipped += skipped
		if projErr != "" {
			result.Errors = append(result.Errors, projErr)
		}
	}

	return result, nil
}

// generateProjectionsForTemplate generates projections for a single template
// across the date range from start to end projection dates
func (s *ProjectionService) generateProjectionsForTemplate(
	workspaceID int32,
	template *domain.RecurringTransaction,
	currentYear int,
	currentMonth time.Month,
	endProjectionDate time.Time,
) *ProjectionResult {
	result := &ProjectionResult{
		Generated: 0,
		Skipped:   0,
		Errors:    make([]string, 0),
	}

	// Determine start month for projections
	// Use the later of: template start date or current month
	startDate := template.StartDate
	currentMonthStart := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.UTC)
	if startDate.Before(currentMonthStart) {
		startDate = currentMonthStart
	}

	// Determine end date for projections
	// Use the earlier of: template end date or projection lookahead limit
	effectiveEndDate := endProjectionDate
	if template.EndDate != nil && template.EndDate.Before(endProjectionDate) {
		effectiveEndDate = *template.EndDate
	}

	// Iterate month by month from start to end
	iterDate := time.Date(startDate.Year(), startDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	for !iterDate.After(effectiveEndDate) {
		year := iterDate.Year()
		month := iterDate.Month()

		generated, skipped, projErr := s.generateSingleProjection(workspaceID, template, year, month, iterDate)
		result.Generated += generated
		result.Skipped += skipped
		if projErr != "" {
			result.Errors = append(result.Errors, projErr)
		}

		// Move to next month
		iterDate = iterDate.AddDate(0, 1, 0)
	}

	return result
}

// generateSingleProjection generates a single projection for a template in a specific month
// Returns (generated count, skipped count, error message)
func (s *ProjectionService) generateSingleProjection(
	workspaceID int32,
	template *domain.RecurringTransaction,
	year int,
	month time.Month,
	targetMonthStart time.Time,
) (int, int, string) {
	// Check if template applies to this month based on start_date
	if template.StartDate.After(targetMonthStart.AddDate(0, 1, -1)) {
		// Template hasn't started yet for this month
		return 0, 1, ""
	}

	// Check if template has ended before this month
	if template.EndDate != nil {
		endMonthStart := time.Date(template.EndDate.Year(), template.EndDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		if targetMonthStart.After(endMonthStart) {
			// Template has already ended
			return 0, 1, ""
		}
	}

	// Idempotency check: skip if projection already exists
	exists, err := s.transactionRepo.CheckProjectionExists(workspaceID, template.ID, year, int(month))
	if err != nil {
		return 0, 0, "Failed to check projection exists for " + template.Name + ": " + err.Error()
	}
	if exists {
		return 0, 1, ""
	}

	// Calculate the actual due date for this month
	dueDate := CalculateActualDueDate(template.DueDay, year, month)

	// Only generate projections for future dates
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	isProjected := dueDate.After(today) || dueDate.Equal(today)

	// Create the projected transaction
	tx := &domain.Transaction{
		WorkspaceID:     workspaceID,
		AccountID:       template.AccountID,
		Name:            template.Name,
		Amount:          template.Amount,
		Type:            template.Type,
		TransactionDate: dueDate,
		IsPaid:          false,
		CategoryID:      template.CategoryID,
		TemplateID:      &template.ID,
		Source:          domain.TransactionSourceRecurring,
		IsProjected:     isProjected,
	}

	_, err = s.transactionRepo.Create(tx)
	if err != nil {
		return 0, 0, "Failed to create projection for " + template.Name + ": " + err.Error()
	}

	return 1, 0, ""
}

// RegenerateProjectionsForTemplate deletes existing projections and regenerates them
// This is called when a template is updated and projections need to be refreshed
func (s *ProjectionService) RegenerateProjectionsForTemplate(workspaceID int32, templateID int32, monthsAhead int) (*ProjectionResult, error) {
	if monthsAhead <= 0 {
		monthsAhead = DefaultProjectionMonths
	}

	// Delete existing projections for this template
	_, err := s.transactionRepo.DeleteProjectionsByTemplateID(workspaceID, templateID)
	if err != nil {
		return nil, err
	}

	// Get the template
	template, err := s.recurringRepo.GetByID(workspaceID, templateID)
	if err != nil {
		return nil, err
	}

	// Only regenerate if template is active
	if !template.IsActive {
		return &ProjectionResult{Generated: 0, Skipped: 0}, nil
	}

	now := time.Now()
	currentYear, currentMonth, _ := now.Date()
	endProjectionDate := now.AddDate(0, monthsAhead, 0)

	return s.generateProjectionsForTemplate(workspaceID, template, currentYear, currentMonth, endProjectionDate), nil
}

// CleanupProjectionsBeyondEndDate removes projections that are beyond a template's end date
// This is called when a template's end_date is set or updated
func (s *ProjectionService) CleanupProjectionsBeyondEndDate(workspaceID int32, templateID int32, endDate time.Time) (int64, error) {
	return s.transactionRepo.DeleteProjectionsBeyondDate(workspaceID, templateID, endDate)
}
