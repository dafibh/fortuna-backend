package postgres

import (
	"context"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/db/sqlc"
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ExclusionRepository implements domain.ProjectionExclusionRepository
type ExclusionRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewExclusionRepository creates a new ExclusionRepository
func NewExclusionRepository(pool *pgxpool.Pool) *ExclusionRepository {
	return &ExclusionRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Create creates a new exclusion record (idempotent via ON CONFLICT DO NOTHING)
func (r *ExclusionRepository) Create(workspaceID int32, templateID int32, excludedMonth time.Time) error {
	return r.queries.CreateProjectionExclusion(context.Background(), sqlc.CreateProjectionExclusionParams{
		WorkspaceID:   workspaceID,
		TemplateID:    templateID,
		ExcludedMonth: pgtype.Date{Time: excludedMonth, Valid: true},
	})
}

// IsExcluded checks if a specific month is excluded for a template
func (r *ExclusionRepository) IsExcluded(workspaceID int32, templateID int32, excludedMonth time.Time) (bool, error) {
	return r.queries.IsMonthExcluded(context.Background(), sqlc.IsMonthExcludedParams{
		WorkspaceID:   workspaceID,
		TemplateID:    templateID,
		ExcludedMonth: pgtype.Date{Time: excludedMonth, Valid: true},
	})
}

// DeleteByTemplate removes all exclusions for a template
func (r *ExclusionRepository) DeleteByTemplate(templateID int32) error {
	return r.queries.DeleteExclusionsByTemplate(context.Background(), templateID)
}

// GetByTemplate gets all exclusions for a template
func (r *ExclusionRepository) GetByTemplate(workspaceID int32, templateID int32) ([]*domain.ProjectionExclusion, error) {
	rows, err := r.queries.GetExclusionsByTemplate(context.Background(), sqlc.GetExclusionsByTemplateParams{
		WorkspaceID: workspaceID,
		TemplateID:  templateID,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*domain.ProjectionExclusion, len(rows))
	for i, row := range rows {
		result[i] = &domain.ProjectionExclusion{
			ID:            row.ID,
			WorkspaceID:   row.WorkspaceID,
			TemplateID:    row.TemplateID,
			ExcludedMonth: row.ExcludedMonth.Time,
			CreatedAt:     row.CreatedAt.Time,
		}
	}
	return result, nil
}
