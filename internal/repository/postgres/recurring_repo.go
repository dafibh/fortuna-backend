package postgres

import (
	"context"

	"github.com/dafibh/fortuna/fortuna-backend/db/sqlc"
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RecurringRepository implements domain.RecurringRepository using PostgreSQL
type RecurringRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewRecurringRepository creates a new RecurringRepository
func NewRecurringRepository(pool *pgxpool.Pool) *RecurringRepository {
	return &RecurringRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Create creates a new recurring transaction
func (r *RecurringRepository) Create(rt *domain.RecurringTransaction) (*domain.RecurringTransaction, error) {
	ctx := context.Background()

	amount, err := decimalToPgNumeric(rt.Amount)
	if err != nil {
		return nil, err
	}

	var categoryID pgtype.Int4
	if rt.CategoryID != nil {
		categoryID = pgtype.Int4{Int32: *rt.CategoryID, Valid: true}
	}

	created, err := r.queries.CreateRecurringTransaction(ctx, sqlc.CreateRecurringTransactionParams{
		WorkspaceID: rt.WorkspaceID,
		Name:        rt.Name,
		Amount:      amount,
		AccountID:   rt.AccountID,
		Type:        string(rt.Type),
		CategoryID:  categoryID,
		Frequency:   string(rt.Frequency),
		DueDay:      rt.DueDay,
		IsActive:    rt.IsActive,
	})
	if err != nil {
		return nil, err
	}

	return sqlcRecurringToDomain(created), nil
}

// GetByID retrieves a recurring transaction by ID
func (r *RecurringRepository) GetByID(workspaceID int32, id int32) (*domain.RecurringTransaction, error) {
	ctx := context.Background()

	rt, err := r.queries.GetRecurringTransaction(ctx, sqlc.GetRecurringTransactionParams{
		WorkspaceID: workspaceID,
		ID:          id,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrRecurringNotFound
		}
		return nil, err
	}

	return sqlcRecurringToDomain(rt), nil
}

// ListByWorkspace retrieves all recurring transactions for a workspace
func (r *RecurringRepository) ListByWorkspace(workspaceID int32, activeOnly *bool) ([]*domain.RecurringTransaction, error) {
	ctx := context.Background()

	var activeFilter bool
	if activeOnly != nil {
		activeFilter = *activeOnly
	}

	rts, err := r.queries.ListRecurringTransactions(ctx, sqlc.ListRecurringTransactionsParams{
		WorkspaceID: workspaceID,
		Column2:     activeFilter,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*domain.RecurringTransaction, len(rts))
	for i, rt := range rts {
		result[i] = sqlcRecurringToDomain(rt)
	}

	return result, nil
}

// Update updates a recurring transaction
func (r *RecurringRepository) Update(rt *domain.RecurringTransaction) (*domain.RecurringTransaction, error) {
	ctx := context.Background()

	amount, err := decimalToPgNumeric(rt.Amount)
	if err != nil {
		return nil, err
	}

	var categoryID pgtype.Int4
	if rt.CategoryID != nil {
		categoryID = pgtype.Int4{Int32: *rt.CategoryID, Valid: true}
	}

	updated, err := r.queries.UpdateRecurringTransaction(ctx, sqlc.UpdateRecurringTransactionParams{
		WorkspaceID: rt.WorkspaceID,
		ID:          rt.ID,
		Name:        rt.Name,
		Amount:      amount,
		AccountID:   rt.AccountID,
		Type:        string(rt.Type),
		CategoryID:  categoryID,
		Frequency:   string(rt.Frequency),
		DueDay:      rt.DueDay,
		IsActive:    rt.IsActive,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrRecurringNotFound
		}
		return nil, err
	}

	return sqlcRecurringToDomain(updated), nil
}

// Delete soft-deletes a recurring transaction
func (r *RecurringRepository) Delete(workspaceID int32, id int32) error {
	ctx := context.Background()

	rowsAffected, err := r.queries.SoftDeleteRecurringTransaction(ctx, sqlc.SoftDeleteRecurringTransactionParams{
		WorkspaceID: workspaceID,
		ID:          id,
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrRecurringNotFound
	}

	return nil
}

// CheckTransactionExists checks if a transaction already exists for a recurring template in a specific month
func (r *RecurringRepository) CheckTransactionExists(recurringID, workspaceID int32, year, month int) (bool, error) {
	ctx := context.Background()

	count, err := r.queries.CheckRecurringTransactionExists(ctx, sqlc.CheckRecurringTransactionExistsParams{
		RecurringID: recurringID,
		WorkspaceID: workspaceID,
		Year:        int32(year),
		Month:       int32(month),
	})
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// Helper function to convert sqlc model to domain model
func sqlcRecurringToDomain(rt sqlc.RecurringTransaction) *domain.RecurringTransaction {
	recurring := &domain.RecurringTransaction{
		ID:          rt.ID,
		WorkspaceID: rt.WorkspaceID,
		Name:        rt.Name,
		Amount:      pgNumericToDecimal(rt.Amount),
		AccountID:   rt.AccountID,
		Type:        domain.TransactionType(rt.Type),
		Frequency:   domain.Frequency(rt.Frequency),
		DueDay:      rt.DueDay,
		IsActive:    rt.IsActive,
		CreatedAt:   rt.CreatedAt.Time,
		UpdatedAt:   rt.UpdatedAt.Time,
	}

	if rt.CategoryID.Valid {
		recurring.CategoryID = &rt.CategoryID.Int32
	}

	if rt.DeletedAt.Valid {
		recurring.DeletedAt = &rt.DeletedAt.Time
	}

	return recurring
}
