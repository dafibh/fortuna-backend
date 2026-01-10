package postgres

import (
	"context"

	"github.com/dafibh/fortuna/fortuna-backend/db/sqlc"
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RecurringTemplateRepository implements domain.RecurringTemplateRepository using PostgreSQL
type RecurringTemplateRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewRecurringTemplateRepository creates a new RecurringTemplateRepository
func NewRecurringTemplateRepository(pool *pgxpool.Pool) *RecurringTemplateRepository {
	return &RecurringTemplateRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Create creates a new recurring template
func (r *RecurringTemplateRepository) Create(template *domain.RecurringTemplate) (*domain.RecurringTemplate, error) {
	ctx := context.Background()

	amount, err := decimalToPgNumeric(template.Amount)
	if err != nil {
		return nil, err
	}

	startDate := pgtype.Date{Time: template.StartDate, Valid: true}

	var endDate pgtype.Date
	if template.EndDate != nil {
		endDate = pgtype.Date{Time: *template.EndDate, Valid: true}
	}

	created, err := r.queries.CreateRecurringTemplate(ctx, sqlc.CreateRecurringTemplateParams{
		WorkspaceID: template.WorkspaceID,
		Description: template.Description,
		Amount:      amount,
		CategoryID:  template.CategoryID,
		AccountID:   template.AccountID,
		Frequency:   template.Frequency,
		StartDate:   startDate,
		EndDate:     endDate,
	})
	if err != nil {
		return nil, err
	}

	return sqlcRecurringTemplateToDomain(created), nil
}

// Update updates a recurring template
func (r *RecurringTemplateRepository) Update(workspaceID int32, id int32, input *domain.UpdateRecurringTemplateInput) (*domain.RecurringTemplate, error) {
	ctx := context.Background()

	amount, err := decimalToPgNumeric(input.Amount)
	if err != nil {
		return nil, err
	}

	startDate := pgtype.Date{Time: input.StartDate, Valid: true}

	var endDate pgtype.Date
	if input.EndDate != nil {
		endDate = pgtype.Date{Time: *input.EndDate, Valid: true}
	}

	updated, err := r.queries.UpdateRecurringTemplate(ctx, sqlc.UpdateRecurringTemplateParams{
		ID:          id,
		WorkspaceID: workspaceID,
		Description: input.Description,
		Amount:      amount,
		CategoryID:  input.CategoryID,
		AccountID:   input.AccountID,
		Frequency:   input.Frequency,
		StartDate:   startDate,
		EndDate:     endDate,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrRecurringTemplateNotFound
		}
		return nil, err
	}

	return sqlcRecurringTemplateToDomain(updated), nil
}

// Delete deletes a recurring template
func (r *RecurringTemplateRepository) Delete(workspaceID int32, id int32) error {
	ctx := context.Background()

	err := r.queries.DeleteRecurringTemplate(ctx, sqlc.DeleteRecurringTemplateParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return err
	}

	return nil
}

// GetByID retrieves a recurring template by ID
func (r *RecurringTemplateRepository) GetByID(workspaceID int32, id int32) (*domain.RecurringTemplate, error) {
	ctx := context.Background()

	template, err := r.queries.GetRecurringTemplateByID(ctx, sqlc.GetRecurringTemplateByIDParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrRecurringTemplateNotFound
		}
		return nil, err
	}

	return sqlcRecurringTemplateToDomain(template), nil
}

// ListByWorkspace retrieves all recurring templates for a workspace
func (r *RecurringTemplateRepository) ListByWorkspace(workspaceID int32) ([]*domain.RecurringTemplate, error) {
	ctx := context.Background()

	templates, err := r.queries.ListRecurringTemplatesByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.RecurringTemplate, len(templates))
	for i, template := range templates {
		result[i] = sqlcRecurringTemplateToDomain(template)
	}

	return result, nil
}

// GetActive retrieves all active recurring templates (no end_date or end_date >= today)
func (r *RecurringTemplateRepository) GetActive(workspaceID int32) ([]*domain.RecurringTemplate, error) {
	ctx := context.Background()

	templates, err := r.queries.GetActiveRecurringTemplates(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.RecurringTemplate, len(templates))
	for i, template := range templates {
		result[i] = sqlcRecurringTemplateToDomain(template)
	}

	return result, nil
}

// GetAllActive retrieves all active recurring templates across all workspaces
// Used by the daily projection sync goroutine
func (r *RecurringTemplateRepository) GetAllActive() ([]*domain.RecurringTemplate, error) {
	ctx := context.Background()

	templates, err := r.queries.GetAllActiveTemplates(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.RecurringTemplate, len(templates))
	for i, template := range templates {
		result[i] = sqlcRecurringTemplateToDomain(template)
	}

	return result, nil
}

// sqlcRecurringTemplateToDomain converts sqlc model to domain model
func sqlcRecurringTemplateToDomain(t sqlc.RecurringTemplate) *domain.RecurringTemplate {
	template := &domain.RecurringTemplate{
		ID:          t.ID,
		WorkspaceID: t.WorkspaceID,
		Description: t.Description,
		Amount:      pgNumericToDecimal(t.Amount),
		CategoryID:  t.CategoryID,
		AccountID:   t.AccountID,
		Frequency:   t.Frequency,
		StartDate:   t.StartDate.Time,
		CreatedAt:   t.CreatedAt.Time,
		UpdatedAt:   t.UpdatedAt.Time,
	}

	if t.EndDate.Valid {
		endDate := t.EndDate.Time
		template.EndDate = &endDate
	}

	return template
}
