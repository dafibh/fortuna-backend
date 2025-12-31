package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/db/sqlc"
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// TransactionRepository implements domain.TransactionRepository using PostgreSQL
type TransactionRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewTransactionRepository creates a new TransactionRepository
func NewTransactionRepository(pool *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Create creates a new transaction
func (r *TransactionRepository) Create(transaction *domain.Transaction) (*domain.Transaction, error) {
	ctx := context.Background()

	amount, err := decimalToPgNumeric(transaction.Amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	var transactionDate pgtype.Date
	transactionDate.Time = transaction.TransactionDate
	transactionDate.Valid = true

	var ccSettlementIntent pgtype.Text
	if transaction.CCSettlementIntent != nil {
		ccSettlementIntent.String = string(*transaction.CCSettlementIntent)
		ccSettlementIntent.Valid = true
	}

	var notes pgtype.Text
	if transaction.Notes != nil {
		notes.String = *transaction.Notes
		notes.Valid = true
	}

	var transferPairID pgtype.UUID
	if transaction.TransferPairID != nil {
		transferPairID.Bytes = *transaction.TransferPairID
		transferPairID.Valid = true
	}

	created, err := r.queries.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		WorkspaceID:        transaction.WorkspaceID,
		AccountID:          transaction.AccountID,
		Name:               transaction.Name,
		Amount:             amount,
		Type:               string(transaction.Type),
		TransactionDate:    transactionDate,
		IsPaid:             transaction.IsPaid,
		CcSettlementIntent: ccSettlementIntent,
		Notes:              notes,
		TransferPairID:     transferPairID,
	})
	if err != nil {
		return nil, err
	}
	return sqlcTransactionToDomain(created), nil
}

// GetByID retrieves a transaction by its ID within a workspace
func (r *TransactionRepository) GetByID(workspaceID int32, id int32) (*domain.Transaction, error) {
	ctx := context.Background()
	transaction, err := r.queries.GetTransactionByID(ctx, sqlc.GetTransactionByIDParams{
		WorkspaceID: workspaceID,
		ID:          id,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrTransactionNotFound
		}
		return nil, err
	}
	return sqlcTransactionToDomain(transaction), nil
}

// GetByWorkspace retrieves transactions for a workspace with optional filters and pagination
func (r *TransactionRepository) GetByWorkspace(workspaceID int32, filters *domain.TransactionFilters) (*domain.PaginatedTransactions, error) {
	ctx := context.Background()

	// Set default pagination values
	page := int32(1)
	pageSize := int32(domain.DefaultPageSize)

	if filters != nil {
		if filters.Page > 0 {
			page = filters.Page
		}
		if filters.PageSize > 0 {
			pageSize = filters.PageSize
			if pageSize > domain.MaxPageSize {
				pageSize = domain.MaxPageSize
			}
		}
	}

	offset := (page - 1) * pageSize

	// Build query params
	params := sqlc.GetTransactionsByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       pageSize,
		Offset:      offset,
	}

	countParams := sqlc.CountTransactionsByWorkspaceParams{
		WorkspaceID: workspaceID,
	}

	if filters != nil {
		if filters.AccountID != nil {
			params.Column2 = *filters.AccountID
			countParams.Column2 = *filters.AccountID
		}
		if filters.StartDate != nil {
			params.Column3.Time = *filters.StartDate
			params.Column3.Valid = true
			countParams.Column3.Time = *filters.StartDate
			countParams.Column3.Valid = true
		}
		if filters.EndDate != nil {
			params.Column4.Time = *filters.EndDate
			params.Column4.Valid = true
			countParams.Column4.Time = *filters.EndDate
			countParams.Column4.Valid = true
		}
		if filters.Type != nil {
			params.Column5 = string(*filters.Type)
			countParams.Column5 = string(*filters.Type)
		}
	}

	// Get total count
	totalItems, err := r.queries.CountTransactionsByWorkspace(ctx, countParams)
	if err != nil {
		return nil, err
	}

	// Get transactions
	transactions, err := r.queries.GetTransactionsByWorkspace(ctx, params)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Transaction, len(transactions))
	for i, t := range transactions {
		result[i] = sqlcTransactionToDomain(t)
	}

	// Calculate total pages
	totalPages := int32(totalItems / int64(pageSize))
	if totalItems%int64(pageSize) > 0 {
		totalPages++
	}

	return &domain.PaginatedTransactions{
		Data:       result,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}, nil
}

// TogglePaid toggles the paid status of a transaction
func (r *TransactionRepository) TogglePaid(workspaceID int32, id int32) (*domain.Transaction, error) {
	ctx := context.Background()
	transaction, err := r.queries.ToggleTransactionPaidStatus(ctx, sqlc.ToggleTransactionPaidStatusParams{
		WorkspaceID: workspaceID,
		ID:          id,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrTransactionNotFound
		}
		return nil, err
	}
	return sqlcTransactionToDomain(transaction), nil
}

// UpdateSettlementIntent updates the CC settlement intent for an unpaid transaction
func (r *TransactionRepository) UpdateSettlementIntent(workspaceID int32, id int32, intent domain.CCSettlementIntent) (*domain.Transaction, error) {
	ctx := context.Background()

	var ccSettlementIntent pgtype.Text
	ccSettlementIntent.String = string(intent)
	ccSettlementIntent.Valid = true

	transaction, err := r.queries.UpdateTransactionSettlementIntent(ctx, sqlc.UpdateTransactionSettlementIntentParams{
		WorkspaceID:        workspaceID,
		ID:                 id,
		CcSettlementIntent: ccSettlementIntent,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrTransactionNotFound
		}
		return nil, err
	}
	return sqlcTransactionToDomain(transaction), nil
}

// Update updates a transaction's details
func (r *TransactionRepository) Update(workspaceID int32, id int32, data *domain.UpdateTransactionData) (*domain.Transaction, error) {
	ctx := context.Background()

	amount, err := decimalToPgNumeric(data.Amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	var transactionDate pgtype.Date
	transactionDate.Time = data.TransactionDate
	transactionDate.Valid = true

	var ccSettlementIntent pgtype.Text
	if data.CCSettlementIntent != nil {
		ccSettlementIntent.String = string(*data.CCSettlementIntent)
		ccSettlementIntent.Valid = true
	}

	var notes pgtype.Text
	if data.Notes != nil {
		notes.String = *data.Notes
		notes.Valid = true
	}

	transaction, err := r.queries.UpdateTransaction(ctx, sqlc.UpdateTransactionParams{
		WorkspaceID:        workspaceID,
		ID:                 id,
		Name:               data.Name,
		Amount:             amount,
		Type:               string(data.Type),
		TransactionDate:    transactionDate,
		AccountID:          data.AccountID,
		CcSettlementIntent: ccSettlementIntent,
		Notes:              notes,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrTransactionNotFound
		}
		return nil, err
	}
	return sqlcTransactionToDomain(transaction), nil
}

// SoftDelete marks a transaction as deleted
func (r *TransactionRepository) SoftDelete(workspaceID int32, id int32) error {
	ctx := context.Background()
	rowsAffected, err := r.queries.SoftDeleteTransaction(ctx, sqlc.SoftDeleteTransactionParams{
		WorkspaceID: workspaceID,
		ID:          id,
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrTransactionNotFound
	}
	return nil
}

// CreateTransferPair creates two linked transactions atomically
func (r *TransactionRepository) CreateTransferPair(fromTx, toTx *domain.Transaction) (*domain.TransferResult, error) {
	ctx := context.Background()

	// Start a transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Create from transaction
	fromResult, err := r.createTransactionWithTx(ctx, qtx, fromTx)
	if err != nil {
		return nil, err
	}

	// Create to transaction
	toResult, err := r.createTransactionWithTx(ctx, qtx, toTx)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &domain.TransferResult{
		FromTransaction: fromResult,
		ToTransaction:   toResult,
	}, nil
}

// createTransactionWithTx is a helper to create a transaction within a database transaction
func (r *TransactionRepository) createTransactionWithTx(ctx context.Context, qtx *sqlc.Queries, transaction *domain.Transaction) (*domain.Transaction, error) {
	amount, err := decimalToPgNumeric(transaction.Amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	var transactionDate pgtype.Date
	transactionDate.Time = transaction.TransactionDate
	transactionDate.Valid = true

	var ccSettlementIntent pgtype.Text
	if transaction.CCSettlementIntent != nil {
		ccSettlementIntent.String = string(*transaction.CCSettlementIntent)
		ccSettlementIntent.Valid = true
	}

	var notes pgtype.Text
	if transaction.Notes != nil {
		notes.String = *transaction.Notes
		notes.Valid = true
	}

	var transferPairID pgtype.UUID
	if transaction.TransferPairID != nil {
		transferPairID.Bytes = *transaction.TransferPairID
		transferPairID.Valid = true
	}

	created, err := qtx.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		WorkspaceID:        transaction.WorkspaceID,
		AccountID:          transaction.AccountID,
		Name:               transaction.Name,
		Amount:             amount,
		Type:               string(transaction.Type),
		TransactionDate:    transactionDate,
		IsPaid:             transaction.IsPaid,
		CcSettlementIntent: ccSettlementIntent,
		Notes:              notes,
		TransferPairID:     transferPairID,
	})
	if err != nil {
		return nil, err
	}
	return sqlcTransactionToDomain(created), nil
}

// SoftDeleteTransferPair soft deletes both transactions in a transfer pair
func (r *TransactionRepository) SoftDeleteTransferPair(workspaceID int32, pairID uuid.UUID) error {
	ctx := context.Background()
	var pairUUID pgtype.UUID
	pairUUID.Bytes = pairID
	pairUUID.Valid = true

	rowsAffected, err := r.queries.SoftDeleteTransferPair(ctx, sqlc.SoftDeleteTransferPairParams{
		WorkspaceID:    workspaceID,
		TransferPairID: pairUUID,
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrTransactionNotFound
	}
	return nil
}

// GetAccountTransactionSummaries retrieves aggregated transaction data for all accounts in a workspace
func (r *TransactionRepository) GetAccountTransactionSummaries(workspaceID int32) ([]*domain.TransactionSummary, error) {
	ctx := context.Background()
	rows, err := r.queries.GetAccountTransactionSummaries(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	summaries := make([]*domain.TransactionSummary, len(rows))
	for i, row := range rows {
		summaries[i] = &domain.TransactionSummary{
			AccountID:         row.AccountID,
			SumIncome:         interfaceToDecimal(row.SumIncome),
			SumExpenses:       interfaceToDecimal(row.SumExpenses),
			SumUnpaidExpenses: interfaceToDecimal(row.SumUnpaidExpenses),
		}
	}

	return summaries, nil
}

// interfaceToDecimal converts an interface{} value (from aggregated queries) to decimal.Decimal
func interfaceToDecimal(v interface{}) decimal.Decimal {
	if v == nil {
		return decimal.Zero
	}
	switch val := v.(type) {
	case pgtype.Numeric:
		return pgNumericToDecimal(val)
	case string:
		d, _ := decimal.NewFromString(val)
		return d
	case float64:
		return decimal.NewFromFloat(val)
	case int64:
		return decimal.NewFromInt(val)
	default:
		return decimal.Zero
	}
}

// SumByTypeAndDateRange sums transactions by type within a date range
func (r *TransactionRepository) SumByTypeAndDateRange(workspaceID int32, startDate, endDate time.Time, txType domain.TransactionType) (decimal.Decimal, error) {
	ctx := context.Background()

	total, err := r.queries.SumTransactionsByTypeAndDateRange(ctx, sqlc.SumTransactionsByTypeAndDateRangeParams{
		WorkspaceID:       workspaceID,
		TransactionDate:   pgtype.Date{Time: startDate, Valid: true},
		TransactionDate_2: pgtype.Date{Time: endDate, Valid: true},
		Type:              string(txType),
	})
	if err != nil {
		return decimal.Zero, err
	}
	return pgNumericToDecimal(total), nil
}

// GetMonthlyTransactionSummaries returns income/expense totals grouped by year/month
func (r *TransactionRepository) GetMonthlyTransactionSummaries(workspaceID int32) ([]*domain.MonthlyTransactionSummary, error) {
	ctx := context.Background()

	rows, err := r.queries.GetMonthlyTransactionSummaries(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	summaries := make([]*domain.MonthlyTransactionSummary, len(rows))
	for i, row := range rows {
		summaries[i] = &domain.MonthlyTransactionSummary{
			Year:          int(row.Year),
			Month:         int(row.Month),
			TotalIncome:   pgNumericToDecimal(row.TotalIncome),
			TotalExpenses: pgNumericToDecimal(row.TotalExpenses),
		}
	}
	return summaries, nil
}

// Helper functions

func sqlcTransactionToDomain(t sqlc.Transaction) *domain.Transaction {
	transaction := &domain.Transaction{
		ID:              t.ID,
		WorkspaceID:     t.WorkspaceID,
		AccountID:       t.AccountID,
		Name:            t.Name,
		Amount:          pgNumericToDecimal(t.Amount),
		Type:            domain.TransactionType(t.Type),
		TransactionDate: t.TransactionDate.Time,
		IsPaid:          t.IsPaid,
		CreatedAt:       t.CreatedAt.Time,
		UpdatedAt:       t.UpdatedAt.Time,
	}
	if t.CcSettlementIntent.Valid {
		intent := domain.CCSettlementIntent(t.CcSettlementIntent.String)
		transaction.CCSettlementIntent = &intent
	}
	if t.Notes.Valid {
		transaction.Notes = &t.Notes.String
	}
	if t.DeletedAt.Valid {
		transaction.DeletedAt = &t.DeletedAt.Time
	}
	if t.TransferPairID.Valid {
		pairID := uuid.UUID(t.TransferPairID.Bytes)
		transaction.TransferPairID = &pairID
	}
	return transaction
}
