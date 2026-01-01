package postgres

import (
	"context"

	"github.com/dafibh/fortuna/fortuna-backend/db/sqlc"
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BudgetAllocationRepository implements domain.BudgetAllocationRepository using PostgreSQL
type BudgetAllocationRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewBudgetAllocationRepository creates a new BudgetAllocationRepository
func NewBudgetAllocationRepository(pool *pgxpool.Pool) *BudgetAllocationRepository {
	return &BudgetAllocationRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Upsert creates or updates a budget allocation
func (r *BudgetAllocationRepository) Upsert(allocation *domain.BudgetAllocation) (*domain.BudgetAllocation, error) {
	ctx := context.Background()

	amount, err := decimalToPgNumeric(allocation.Amount)
	if err != nil {
		return nil, err
	}

	result, err := r.queries.UpsertBudgetAllocation(ctx, sqlc.UpsertBudgetAllocationParams{
		WorkspaceID: allocation.WorkspaceID,
		CategoryID:  allocation.CategoryID,
		Year:        int32(allocation.Year),
		Month:       int32(allocation.Month),
		Amount:      amount,
	})
	if err != nil {
		return nil, err
	}

	return sqlcBudgetAllocationToDomain(result), nil
}

// UpsertBatch creates or updates multiple budget allocations atomically
func (r *BudgetAllocationRepository) UpsertBatch(allocations []*domain.BudgetAllocation) error {
	ctx := context.Background()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	for _, allocation := range allocations {
		amount, err := decimalToPgNumeric(allocation.Amount)
		if err != nil {
			return err
		}

		_, err = qtx.UpsertBudgetAllocation(ctx, sqlc.UpsertBudgetAllocationParams{
			WorkspaceID: allocation.WorkspaceID,
			CategoryID:  allocation.CategoryID,
			Year:        int32(allocation.Year),
			Month:       int32(allocation.Month),
			Amount:      amount,
		})
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// GetByMonth retrieves all budget allocations for a specific month
func (r *BudgetAllocationRepository) GetByMonth(workspaceID int32, year, month int) ([]*domain.BudgetAllocation, error) {
	ctx := context.Background()

	allocations, err := r.queries.GetBudgetAllocationsByMonth(ctx, sqlc.GetBudgetAllocationsByMonthParams{
		WorkspaceID: workspaceID,
		Year:        int32(year),
		Month:       int32(month),
	})
	if err != nil {
		return nil, err
	}

	result := make([]*domain.BudgetAllocation, len(allocations))
	for i, a := range allocations {
		result[i] = &domain.BudgetAllocation{
			ID:          a.ID,
			WorkspaceID: a.WorkspaceID,
			CategoryID:  a.CategoryID,
			Year:        int(a.Year),
			Month:       int(a.Month),
			Amount:      pgNumericToDecimal(a.Amount),
			CreatedAt:   a.CreatedAt.Time,
			UpdatedAt:   a.UpdatedAt.Time,
		}
	}
	return result, nil
}

// GetByCategory retrieves a budget allocation for a specific category and month
func (r *BudgetAllocationRepository) GetByCategory(workspaceID int32, categoryID int32, year, month int) (*domain.BudgetAllocation, error) {
	ctx := context.Background()

	allocation, err := r.queries.GetBudgetAllocationByCategory(ctx, sqlc.GetBudgetAllocationByCategoryParams{
		WorkspaceID: workspaceID,
		CategoryID:  categoryID,
		Year:        int32(year),
		Month:       int32(month),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrBudgetAllocationNotFound
		}
		return nil, err
	}

	return sqlcBudgetAllocationToDomain(allocation), nil
}

// Delete removes a budget allocation
func (r *BudgetAllocationRepository) Delete(workspaceID int32, categoryID int32, year, month int) error {
	ctx := context.Background()
	return r.queries.DeleteBudgetAllocation(ctx, sqlc.DeleteBudgetAllocationParams{
		WorkspaceID: workspaceID,
		CategoryID:  categoryID,
		Year:        int32(year),
		Month:       int32(month),
	})
}

// GetCategoriesWithAllocations retrieves all categories with their allocations for a month
func (r *BudgetAllocationRepository) GetCategoriesWithAllocations(workspaceID int32, year, month int) ([]*domain.BudgetCategoryWithAllocation, error) {
	ctx := context.Background()

	rows, err := r.queries.GetCategoriesWithAllocations(ctx, sqlc.GetCategoriesWithAllocationsParams{
		WorkspaceID: workspaceID,
		Year:        int32(year),
		Month:       int32(month),
	})
	if err != nil {
		return nil, err
	}

	result := make([]*domain.BudgetCategoryWithAllocation, len(rows))
	for i, row := range rows {
		result[i] = &domain.BudgetCategoryWithAllocation{
			CategoryID:   row.CategoryID,
			CategoryName: row.CategoryName,
			Allocated:    pgNumericToDecimal(row.Allocated),
		}
	}
	return result, nil
}

// Helper function to convert sqlc BudgetAllocation to domain
func sqlcBudgetAllocationToDomain(a sqlc.BudgetAllocation) *domain.BudgetAllocation {
	return &domain.BudgetAllocation{
		ID:          a.ID,
		WorkspaceID: a.WorkspaceID,
		CategoryID:  a.CategoryID,
		Year:        int(a.Year),
		Month:       int(a.Month),
		Amount:      pgNumericToDecimal(a.Amount),
		CreatedAt:   a.CreatedAt.Time,
		UpdatedAt:   a.UpdatedAt.Time,
	}
}
