package postgres

import (
	"context"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/db/sqlc"
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TransactionGroupRepository implements domain.TransactionGroupRepository using PostgreSQL
type TransactionGroupRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewTransactionGroupRepository creates a new TransactionGroupRepository
func NewTransactionGroupRepository(pool *pgxpool.Pool) *TransactionGroupRepository {
	return &TransactionGroupRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Create creates a new transaction group
func (r *TransactionGroupRepository) Create(group *domain.TransactionGroup) (*domain.TransactionGroup, error) {
	ctx := context.Background()

	params := sqlc.CreateGroupParams{
		WorkspaceID:  group.WorkspaceID,
		Name:         group.Name,
		Month:        group.Month,
		AutoDetected: group.AutoDetected,
	}
	if group.LoanProviderID != nil {
		params.LoanProviderID = pgtype.Int4{Int32: *group.LoanProviderID, Valid: true}
	}

	created, err := r.queries.CreateGroup(ctx, params)
	if err != nil {
		return nil, err
	}
	return sqlcGroupToDomain(created), nil
}

// CreateWithAssignment atomically creates a group and assigns transactions to it
func (r *TransactionGroupRepository) CreateWithAssignment(group *domain.TransactionGroup, transactionIDs []int32) (*domain.TransactionGroup, error) {
	ctx := context.Background()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// 1. Create the group
	params := sqlc.CreateGroupParams{
		WorkspaceID:  group.WorkspaceID,
		Name:         group.Name,
		Month:        group.Month,
		AutoDetected: group.AutoDetected,
	}
	if group.LoanProviderID != nil {
		params.LoanProviderID = pgtype.Int4{Int32: *group.LoanProviderID, Valid: true}
	}

	created, err := qtx.CreateGroup(ctx, params)
	if err != nil {
		return nil, err
	}

	// 2. Assign transactions to the group
	err = qtx.AssignGroupToTransactions(ctx, sqlc.AssignGroupToTransactionsParams{
		GroupID:     pgtype.Int4{Int32: created.ID, Valid: true},
		WorkspaceID: group.WorkspaceID,
		Column3:     transactionIDs,
	})
	if err != nil {
		return nil, err
	}

	// 3. Commit
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// 4. Fetch the group with derived totals
	return r.GetByID(group.WorkspaceID, created.ID)
}

// GetByID retrieves a transaction group by ID with derived totals
func (r *TransactionGroupRepository) GetByID(workspaceID int32, id int32) (*domain.TransactionGroup, error) {
	ctx := context.Background()

	row, err := r.queries.GetGroupByID(ctx, sqlc.GetGroupByIDParams{
		WorkspaceID: workspaceID,
		ID:          id,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrGroupNotFound
		}
		return nil, err
	}
	return sqlcGroupByIDRowToDomain(row), nil
}

// GetGroupsByMonth retrieves all groups for a workspace and month with derived totals
func (r *TransactionGroupRepository) GetGroupsByMonth(workspaceID int32, month string) ([]*domain.TransactionGroup, error) {
	ctx := context.Background()

	rows, err := r.queries.GetGroupsByMonth(ctx, sqlc.GetGroupsByMonthParams{
		WorkspaceID: workspaceID,
		Month:       month,
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.TransactionGroup, len(rows))
	for i, row := range rows {
		result[i] = sqlcGroupsByMonthRowToDomain(row)
	}
	return result, nil
}

// UpdateName updates the name of a transaction group
func (r *TransactionGroupRepository) UpdateName(workspaceID int32, id int32, name string) (*domain.TransactionGroup, error) {
	ctx := context.Background()

	updated, err := r.queries.UpdateGroupName(ctx, sqlc.UpdateGroupNameParams{
		WorkspaceID: workspaceID,
		ID:          id,
		Name:        name,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrGroupNotFound
		}
		return nil, err
	}
	return sqlcGroupToDomain(updated), nil
}

// Delete deletes a transaction group (child transactions become ungrouped via ON DELETE SET NULL)
func (r *TransactionGroupRepository) Delete(workspaceID int32, id int32) error {
	ctx := context.Background()
	return r.queries.DeleteGroup(ctx, sqlc.DeleteGroupParams{
		WorkspaceID: workspaceID,
		ID:          id,
	})
}

// AssignGroupToTransactions assigns a group to multiple transactions
func (r *TransactionGroupRepository) AssignGroupToTransactions(workspaceID int32, groupID int32, transactionIDs []int32) error {
	ctx := context.Background()
	return r.queries.AssignGroupToTransactions(ctx, sqlc.AssignGroupToTransactionsParams{
		GroupID:     pgtype.Int4{Int32: groupID, Valid: true},
		WorkspaceID: workspaceID,
		Column3:     transactionIDs,
	})
}

// UnassignGroupFromTransactions removes group assignment from multiple transactions
func (r *TransactionGroupRepository) UnassignGroupFromTransactions(workspaceID int32, transactionIDs []int32) error {
	ctx := context.Background()
	return r.queries.UnassignGroupFromTransactions(ctx, sqlc.UnassignGroupFromTransactionsParams{
		WorkspaceID: workspaceID,
		Column2:     transactionIDs,
	})
}

// UnassignAllFromGroup removes group assignment from all transactions in a group
func (r *TransactionGroupRepository) UnassignAllFromGroup(workspaceID int32, groupID int32) (int64, error) {
	ctx := context.Background()
	count, err := r.queries.UnassignAllFromGroup(ctx, sqlc.UnassignAllFromGroupParams{
		GroupID:     pgtype.Int4{Int32: groupID, Valid: true},
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

// DeleteGroupAndChildren atomically soft-deletes all child transactions and hard-deletes the group
func (r *TransactionGroupRepository) DeleteGroupAndChildren(workspaceID int32, groupID int32) (int32, error) {
	ctx := context.Background()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Soft-delete all child transactions
	count, err := qtx.SoftDeleteTransactionsByGroupID(ctx, sqlc.SoftDeleteTransactionsByGroupIDParams{
		GroupID:     pgtype.Int4{Int32: groupID, Valid: true},
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return 0, err
	}

	// Delete the group record (hard delete)
	err = qtx.DeleteGroup(ctx, sqlc.DeleteGroupParams{
		WorkspaceID: workspaceID,
		ID:          groupID,
	})
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	return int32(count), nil
}

// CountGroupChildren returns the number of active children in a group
func (r *TransactionGroupRepository) CountGroupChildren(workspaceID int32, groupID int32) (int32, error) {
	ctx := context.Background()
	count, err := r.queries.CountGroupChildren(ctx, sqlc.CountGroupChildrenParams{
		GroupID:     pgtype.Int4{Int32: groupID, Valid: true},
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetUngroupedTransactionsByMonth retrieves ungrouped transactions for a date range
func (r *TransactionGroupRepository) GetUngroupedTransactionsByMonth(workspaceID int32, startDate, endDate time.Time) ([]*domain.Transaction, error) {
	ctx := context.Background()

	rows, err := r.queries.GetUngroupedTransactionsByMonth(ctx, sqlc.GetUngroupedTransactionsByMonthParams{
		WorkspaceID:       workspaceID,
		TransactionDate:   pgtype.Date{Time: startDate, Valid: true},
		TransactionDate_2: pgtype.Date{Time: endDate, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Transaction, len(rows))
	for i, row := range rows {
		result[i] = sqlcTransactionToDomain(row)
	}
	return result, nil
}

// GetConsolidatedProvidersByMonth returns consolidated_monthly providers with 2+ ungrouped transactions in a month
func (r *TransactionGroupRepository) GetConsolidatedProvidersByMonth(workspaceID int32, month string) ([]domain.AutoDetectionCandidate, error) {
	ctx := context.Background()

	rows, err := r.queries.GetConsolidatedProvidersByMonth(ctx, sqlc.GetConsolidatedProvidersByMonthParams{
		WorkspaceID: workspaceID,
		Month:       month,
	})
	if err != nil {
		return nil, err
	}
	result := make([]domain.AutoDetectionCandidate, len(rows))
	for i, row := range rows {
		result[i] = domain.AutoDetectionCandidate{
			ProviderID:   row.ProviderID,
			ProviderName: row.ProviderName,
			Count:        row.TxCount,
		}
	}
	return result, nil
}

// GetUngroupedTransactionIDsByProviderMonth returns IDs of ungrouped transactions for a provider in a month
func (r *TransactionGroupRepository) GetUngroupedTransactionIDsByProviderMonth(workspaceID int32, providerID int32, month string) ([]int32, error) {
	ctx := context.Background()

	return r.queries.GetUngroupedTransactionIDsByProviderMonth(ctx, sqlc.GetUngroupedTransactionIDsByProviderMonthParams{
		WorkspaceID: workspaceID,
		ProviderID:  providerID,
		Month:       month,
	})
}

// GetAutoDetectedGroupByProviderMonth returns an existing auto-detected group for a provider+month (idempotency guard)
func (r *TransactionGroupRepository) GetAutoDetectedGroupByProviderMonth(workspaceID int32, providerID int32, month string) (*domain.TransactionGroup, error) {
	ctx := context.Background()

	row, err := r.queries.GetAutoDetectedGroupByProviderMonth(ctx, sqlc.GetAutoDetectedGroupByProviderMonthParams{
		WorkspaceID:    workspaceID,
		LoanProviderID: pgtype.Int4{Int32: providerID, Valid: true},
		Month:          month,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrGroupNotFound
		}
		return nil, err
	}
	return sqlcAutoDetectedGroupToDomain(row), nil
}

// Helper conversion functions

func sqlcGroupToDomain(g sqlc.TransactionGroup) *domain.TransactionGroup {
	group := &domain.TransactionGroup{
		ID:           g.ID,
		WorkspaceID:  g.WorkspaceID,
		Name:         g.Name,
		Month:        g.Month,
		AutoDetected: g.AutoDetected,
		CreatedAt:    g.CreatedAt.Time,
		UpdatedAt:    g.UpdatedAt.Time,
	}
	if g.LoanProviderID.Valid {
		group.LoanProviderID = &g.LoanProviderID.Int32
	}
	return group
}

func sqlcGroupByIDRowToDomain(row sqlc.GetGroupByIDRow) *domain.TransactionGroup {
	group := &domain.TransactionGroup{
		ID:           row.ID,
		WorkspaceID:  row.WorkspaceID,
		Name:         row.Name,
		Month:        row.Month,
		AutoDetected: row.AutoDetected,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
		TotalAmount:  pgNumericToDecimal(row.TotalAmount),
		ChildCount:   row.ChildCount,
	}
	if row.LoanProviderID.Valid {
		group.LoanProviderID = &row.LoanProviderID.Int32
	}
	return group
}

func sqlcGroupsByMonthRowToDomain(row sqlc.GetGroupsByMonthRow) *domain.TransactionGroup {
	group := &domain.TransactionGroup{
		ID:           row.ID,
		WorkspaceID:  row.WorkspaceID,
		Name:         row.Name,
		Month:        row.Month,
		AutoDetected: row.AutoDetected,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
		TotalAmount:  pgNumericToDecimal(row.TotalAmount),
		ChildCount:   row.ChildCount,
	}
	if row.LoanProviderID.Valid {
		group.LoanProviderID = &row.LoanProviderID.Int32
	}
	return group
}

func sqlcAutoDetectedGroupToDomain(row sqlc.GetAutoDetectedGroupByProviderMonthRow) *domain.TransactionGroup {
	group := &domain.TransactionGroup{
		ID:           row.ID,
		WorkspaceID:  row.WorkspaceID,
		Name:         row.Name,
		Month:        row.Month,
		AutoDetected: row.AutoDetected,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
		TotalAmount:  pgNumericToDecimal(row.TotalAmount),
		ChildCount:   row.ChildCount,
	}
	if row.LoanProviderID.Valid {
		group.LoanProviderID = &row.LoanProviderID.Int32
	}
	return group
}
