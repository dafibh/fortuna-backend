package service

import (
	"fmt"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/rs/zerolog/log"
)

// ProjectionSyncService handles daily projection synchronization across all workspaces
type ProjectionSyncService struct {
	templateRepo    domain.RecurringTemplateRepository
	transactionRepo domain.TransactionRepository
}

// NewProjectionSyncService creates a new ProjectionSyncService
func NewProjectionSyncService(
	templateRepo domain.RecurringTemplateRepository,
	transactionRepo domain.TransactionRepository,
) *ProjectionSyncService {
	return &ProjectionSyncService{
		templateRepo:    templateRepo,
		transactionRepo: transactionRepo,
	}
}

// SyncAllActive synchronizes projections for all active templates across all workspaces
// It ensures projections exist up to current_date + 12 months and removes any beyond end_date
func (s *ProjectionSyncService) SyncAllActive() error {
	start := time.Now()
	log.Info().Msg("Starting projection sync for all active templates")

	templates, err := s.templateRepo.GetAllActive()
	if err != nil {
		return fmt.Errorf("failed to get active templates: %w", err)
	}

	var syncErrors []error
	processed := 0

	for _, template := range templates {
		if err := s.syncTemplate(template); err != nil {
			log.Error().
				Err(err).
				Int32("templateID", template.ID).
				Int32("workspaceID", template.WorkspaceID).
				Str("description", template.Description).
				Msg("Failed to sync template projections")
			syncErrors = append(syncErrors, fmt.Errorf("template %d: %w", template.ID, err))
			continue
		}
		processed++
	}

	duration := time.Since(start)
	log.Info().
		Int("templatesProcessed", processed).
		Int("errorsCount", len(syncErrors)).
		Dur("duration", duration).
		Msg("Projection sync completed")

	if len(syncErrors) > 0 {
		return fmt.Errorf("sync completed with %d errors out of %d templates", len(syncErrors), len(templates))
	}

	return nil
}

// syncTemplate ensures a single template has projections up to now + 12 months
func (s *ProjectionSyncService) syncTemplate(template *domain.RecurringTemplate) error {
	now := time.Now()
	targetEnd := now.AddDate(0, 12, 0)

	// If template has end_date before target, use end_date and delete beyond
	if template.EndDate != nil && template.EndDate.Before(targetEnd) {
		targetEnd = *template.EndDate

		// Delete any projections beyond end_date (AC #4)
		if err := s.transactionRepo.DeleteProjectionsBeyondDate(template.WorkspaceID, template.ID, *template.EndDate); err != nil {
			return fmt.Errorf("failed to delete projections beyond end_date: %w", err)
		}
	}

	// Generate missing projections up to targetEnd
	return s.generateUpToMonth(template, targetEnd)
}

// generateUpToMonth creates any missing projections up to the target month
func (s *ProjectionSyncService) generateUpToMonth(template *domain.RecurringTemplate, targetEnd time.Time) error {
	// Get existing projections to check for duplicates
	existingProjections, err := s.transactionRepo.GetProjectionsByTemplate(template.WorkspaceID, template.ID)
	if err != nil {
		return fmt.Errorf("failed to get existing projections: %w", err)
	}

	// Build set of existing projection dates
	existingDates := make(map[string]bool)
	for _, proj := range existingProjections {
		dateKey := proj.TransactionDate.Format("2006-01")
		existingDates[dateKey] = true
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

	// Generate projections month by month
	current := startDate
	created := 0

	for !current.After(targetEnd) {
		actualDate := s.calculateActualDate(current.Year(), current.Month(), targetDay)
		dateKey := actualDate.Format("2006-01")

		// Skip if projection already exists
		if existingDates[dateKey] {
			current = current.AddDate(0, 1, 0)
			continue
		}

		// Create projection transaction
		transaction := &domain.Transaction{
			WorkspaceID:     template.WorkspaceID,
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
			return fmt.Errorf("failed to create projection for %s: %w", dateKey, err)
		}

		created++
		current = current.AddDate(0, 1, 0)
	}

	if created > 0 {
		log.Debug().
			Int32("templateID", template.ID).
			Int("projectionsCreated", created).
			Msg("Created new projections for template")
	}

	return nil
}

// calculateActualDate returns the actual date for a target day in a given month
func (s *ProjectionSyncService) calculateActualDate(year int, month time.Month, targetDay int) time.Time {
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	actualDay := targetDay
	if actualDay > lastDay {
		actualDay = lastDay
	}
	return time.Date(year, month, actualDay, 0, 0, 0, 0, time.UTC)
}
