package postgres

import (
	"context"
	"errors"

	"github.com/dafibh/fortuna/fortuna-backend/db/sqlc"
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BudgetCategoryRepository implements domain.BudgetCategoryRepository using PostgreSQL
type BudgetCategoryRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewBudgetCategoryRepository creates a new BudgetCategoryRepository
func NewBudgetCategoryRepository(pool *pgxpool.Pool) *BudgetCategoryRepository {
	return &BudgetCategoryRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Create creates a new budget category
func (r *BudgetCategoryRepository) Create(category *domain.BudgetCategory) (*domain.BudgetCategory, error) {
	ctx := context.Background()
	created, err := r.queries.CreateBudgetCategory(ctx, sqlc.CreateBudgetCategoryParams{
		WorkspaceID: category.WorkspaceID,
		Name:        category.Name,
	})
	if err != nil {
		// Check for unique constraint violation
		if isPgUniqueViolation(err) {
			return nil, domain.ErrBudgetCategoryAlreadyExists
		}
		return nil, err
	}
	return sqlcBudgetCategoryToDomain(created), nil
}

// GetByID retrieves a budget category by its ID within a workspace
func (r *BudgetCategoryRepository) GetByID(workspaceID int32, id int32) (*domain.BudgetCategory, error) {
	ctx := context.Background()
	category, err := r.queries.GetBudgetCategoryByID(ctx, sqlc.GetBudgetCategoryByIDParams{
		WorkspaceID: workspaceID,
		ID:          id,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrBudgetCategoryNotFound
		}
		return nil, err
	}
	return sqlcBudgetCategoryToDomain(category), nil
}

// GetByName retrieves a budget category by its name within a workspace
func (r *BudgetCategoryRepository) GetByName(workspaceID int32, name string) (*domain.BudgetCategory, error) {
	ctx := context.Background()
	category, err := r.queries.GetBudgetCategoryByName(ctx, sqlc.GetBudgetCategoryByNameParams{
		WorkspaceID: workspaceID,
		Name:        name,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrBudgetCategoryNotFound
		}
		return nil, err
	}
	return sqlcBudgetCategoryToDomain(category), nil
}

// GetAllByWorkspace retrieves all budget categories for a workspace
func (r *BudgetCategoryRepository) GetAllByWorkspace(workspaceID int32) ([]*domain.BudgetCategory, error) {
	ctx := context.Background()
	categories, err := r.queries.GetAllBudgetCategories(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.BudgetCategory, len(categories))
	for i, c := range categories {
		result[i] = sqlcBudgetCategoryToDomain(c)
	}
	return result, nil
}

// Update updates a budget category's name
func (r *BudgetCategoryRepository) Update(workspaceID int32, id int32, name string) (*domain.BudgetCategory, error) {
	ctx := context.Background()
	category, err := r.queries.UpdateBudgetCategory(ctx, sqlc.UpdateBudgetCategoryParams{
		WorkspaceID: workspaceID,
		ID:          id,
		Name:        name,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrBudgetCategoryNotFound
		}
		// Check for unique constraint violation
		if isPgUniqueViolation(err) {
			return nil, domain.ErrBudgetCategoryAlreadyExists
		}
		return nil, err
	}
	return sqlcBudgetCategoryToDomain(category), nil
}

// SoftDelete marks a budget category as deleted
func (r *BudgetCategoryRepository) SoftDelete(workspaceID int32, id int32) error {
	ctx := context.Background()
	return r.queries.SoftDeleteBudgetCategory(ctx, sqlc.SoftDeleteBudgetCategoryParams{
		WorkspaceID: workspaceID,
		ID:          id,
	})
}

// HasTransactions checks if a budget category has any transactions assigned
// NOTE: This will always return false until Story 4.2 adds category_id to transactions
// TODO(Story 4.2): Update CountTransactionsByCategory query to accept workspaceID and categoryID parameters
func (r *BudgetCategoryRepository) HasTransactions(workspaceID int32, id int32) (bool, error) {
	ctx := context.Background()
	// Parameters intentionally unused until Story 4.2 adds category_id to transactions table
	_ = workspaceID
	_ = id
	count, err := r.queries.CountTransactionsByCategory(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Helper functions

func sqlcBudgetCategoryToDomain(c sqlc.BudgetCategory) *domain.BudgetCategory {
	category := &domain.BudgetCategory{
		ID:          c.ID,
		WorkspaceID: c.WorkspaceID,
		Name:        c.Name,
		CreatedAt:   c.CreatedAt.Time,
		UpdatedAt:   c.UpdatedAt.Time,
	}
	if c.DeletedAt.Valid {
		category.DeletedAt = &c.DeletedAt.Time
	}
	return category
}

// isPgUniqueViolation checks if an error is a PostgreSQL unique constraint violation
func isPgUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// PostgreSQL unique violation error code is 23505
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
