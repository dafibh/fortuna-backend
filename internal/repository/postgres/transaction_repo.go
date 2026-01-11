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

	var categoryID pgtype.Int4
	if transaction.CategoryID != nil {
		categoryID.Int32 = *transaction.CategoryID
		categoryID.Valid = true
	}

	// CC Lifecycle (v2 simplified)
	var billedAt pgtype.Timestamptz
	if transaction.BilledAt != nil {
		billedAt.Time = *transaction.BilledAt
		billedAt.Valid = true
	}

	var settlementIntent pgtype.Text
	if transaction.SettlementIntent != nil {
		settlementIntent.String = string(*transaction.SettlementIntent)
		settlementIntent.Valid = true
	}

	// Recurring/Projection (v2)
	var source pgtype.Text
	if transaction.Source != "" {
		source.String = transaction.Source
		source.Valid = true
	} else {
		source.String = "manual"
		source.Valid = true
	}

	var templateID pgtype.Int4
	if transaction.TemplateID != nil {
		templateID.Int32 = *transaction.TemplateID
		templateID.Valid = true
	}

	var isProjected pgtype.Bool
	isProjected.Bool = transaction.IsProjected
	isProjected.Valid = true

	created, err := r.queries.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		WorkspaceID:      transaction.WorkspaceID,
		AccountID:        transaction.AccountID,
		Name:             transaction.Name,
		Amount:           amount,
		Type:             string(transaction.Type),
		TransactionDate:  transactionDate,
		IsPaid:           transaction.IsPaid,
		Notes:            notes,
		TransferPairID:   transferPairID,
		CategoryID:       categoryID,
		IsCcPayment:      transaction.IsCCPayment,
		BilledAt:         billedAt,
		SettlementIntent: settlementIntent,
		Source:           source,
		TemplateID:       templateID,
		IsProjected:      isProjected,
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

	// Build query params - using GetTransactionsWithCategory for category info
	params := sqlc.GetTransactionsWithCategoryParams{
		WorkspaceID: workspaceID,
		PageSize:    pageSize,
		PageOffset:  offset,
	}

	countParams := sqlc.CountTransactionsByWorkspaceParams{
		WorkspaceID: workspaceID,
	}

	if filters != nil {
		if filters.AccountID != nil {
			params.AccountID = pgtype.Int4{Int32: *filters.AccountID, Valid: true}
			countParams.AccountID = pgtype.Int4{Int32: *filters.AccountID, Valid: true}
		}
		if filters.StartDate != nil {
			params.StartDate = pgtype.Date{Time: *filters.StartDate, Valid: true}
			countParams.StartDate = pgtype.Date{Time: *filters.StartDate, Valid: true}
		}
		if filters.EndDate != nil {
			params.EndDate = pgtype.Date{Time: *filters.EndDate, Valid: true}
			countParams.EndDate = pgtype.Date{Time: *filters.EndDate, Valid: true}
		}
		if filters.Type != nil {
			params.Type = pgtype.Text{String: string(*filters.Type), Valid: true}
			countParams.Type = pgtype.Text{String: string(*filters.Type), Valid: true}
		}
		// Note: CCStatus filtering now happens via computed ccState from isPaid/billedAt
		// The SQL query no longer has cc_status filter - filtering is done client-side if needed
	}

	// Get total count
	totalItems, err := r.queries.CountTransactionsByWorkspace(ctx, countParams)
	if err != nil {
		return nil, err
	}

	// Get transactions with category info
	transactions, err := r.queries.GetTransactionsWithCategory(ctx, params)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Transaction, len(transactions))
	for i, t := range transactions {
		result[i] = sqlcTransactionWithCategoryToDomain(t)
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

	var notes pgtype.Text
	if data.Notes != nil {
		notes.String = *data.Notes
		notes.Valid = true
	}

	var categoryID pgtype.Int4
	if data.CategoryID != nil {
		categoryID.Int32 = *data.CategoryID
		categoryID.Valid = true
	}

	// CC Lifecycle (v2 simplified)
	var billedAt pgtype.Timestamptz
	if data.BilledAt != nil {
		billedAt.Time = *data.BilledAt
		billedAt.Valid = true
	}

	var settlementIntent pgtype.Text
	if data.SettlementIntent != nil {
		settlementIntent.String = string(*data.SettlementIntent)
		settlementIntent.Valid = true
	}

	// Recurring/Projection (v2)
	var source pgtype.Text
	if data.Source != "" {
		source.String = data.Source
		source.Valid = true
	} else {
		source.String = "manual"
		source.Valid = true
	}

	var templateID pgtype.Int4
	if data.TemplateID != nil {
		templateID.Int32 = *data.TemplateID
		templateID.Valid = true
	}

	var isProjected pgtype.Bool
	isProjected.Bool = data.IsProjected
	isProjected.Valid = true

	transaction, err := r.queries.UpdateTransaction(ctx, sqlc.UpdateTransactionParams{
		WorkspaceID:      workspaceID,
		ID:               id,
		Name:             data.Name,
		Amount:           amount,
		Type:             string(data.Type),
		TransactionDate:  transactionDate,
		AccountID:        data.AccountID,
		Notes:            notes,
		CategoryID:       categoryID,
		IsPaid:           data.IsPaid,
		BilledAt:         billedAt,
		SettlementIntent: settlementIntent,
		Source:           source,
		TemplateID:       templateID,
		IsProjected:      isProjected,
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

	var categoryID pgtype.Int4
	if transaction.CategoryID != nil {
		categoryID.Int32 = *transaction.CategoryID
		categoryID.Valid = true
	}

	// CC Lifecycle (v2 simplified)
	var billedAt pgtype.Timestamptz
	if transaction.BilledAt != nil {
		billedAt.Time = *transaction.BilledAt
		billedAt.Valid = true
	}

	var settlementIntent pgtype.Text
	if transaction.SettlementIntent != nil {
		settlementIntent.String = string(*transaction.SettlementIntent)
		settlementIntent.Valid = true
	}

	// Recurring/Projection (v2)
	var source pgtype.Text
	if transaction.Source != "" {
		source.String = transaction.Source
		source.Valid = true
	} else {
		source.String = "manual"
		source.Valid = true
	}

	var templateID pgtype.Int4
	if transaction.TemplateID != nil {
		templateID.Int32 = *transaction.TemplateID
		templateID.Valid = true
	}

	var isProjected pgtype.Bool
	isProjected.Bool = transaction.IsProjected
	isProjected.Valid = true

	created, err := qtx.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		WorkspaceID:      transaction.WorkspaceID,
		AccountID:        transaction.AccountID,
		Name:             transaction.Name,
		Amount:           amount,
		Type:             string(transaction.Type),
		TransactionDate:  transactionDate,
		IsPaid:           transaction.IsPaid,
		Notes:            notes,
		TransferPairID:   transferPairID,
		CategoryID:       categoryID,
		IsCcPayment:      transaction.IsCCPayment,
		BilledAt:         billedAt,
		SettlementIntent: settlementIntent,
		Source:           source,
		TemplateID:       templateID,
		IsProjected:      isProjected,
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
			SumAllExpenses:    interfaceToDecimal(row.SumAllExpenses),
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

// SumPaidExpensesByDateRange sums paid expenses within a date range
func (r *TransactionRepository) SumPaidExpensesByDateRange(workspaceID int32, startDate, endDate time.Time) (decimal.Decimal, error) {
	ctx := context.Background()

	total, err := r.queries.SumPaidExpensesByDateRange(ctx, sqlc.SumPaidExpensesByDateRangeParams{
		WorkspaceID:       workspaceID,
		TransactionDate:   pgtype.Date{Time: startDate, Valid: true},
		TransactionDate_2: pgtype.Date{Time: endDate, Valid: true},
	})
	if err != nil {
		return decimal.Zero, err
	}
	return pgNumericToDecimal(total), nil
}

// SumUnpaidExpensesByDateRange sums unpaid expenses within a date range
func (r *TransactionRepository) SumUnpaidExpensesByDateRange(workspaceID int32, startDate, endDate time.Time) (decimal.Decimal, error) {
	ctx := context.Background()

	total, err := r.queries.SumUnpaidExpensesByDateRange(ctx, sqlc.SumUnpaidExpensesByDateRangeParams{
		WorkspaceID:       workspaceID,
		TransactionDate:   pgtype.Date{Time: startDate, Valid: true},
		TransactionDate_2: pgtype.Date{Time: endDate, Valid: true},
	})
	if err != nil {
		return decimal.Zero, err
	}
	return pgNumericToDecimal(total), nil
}

// GetRecentlyUsedCategories returns recently used categories for suggestions
func (r *TransactionRepository) GetRecentlyUsedCategories(workspaceID int32) ([]*domain.RecentCategory, error) {
	ctx := context.Background()

	rows, err := r.queries.GetRecentlyUsedCategories(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.RecentCategory, len(rows))
	for i, row := range rows {
		// LastUsed is an interface{} that could be a time.Time or pgtype.Timestamptz
		var lastUsed time.Time
		switch v := row.LastUsed.(type) {
		case time.Time:
			lastUsed = v
		case pgtype.Timestamptz:
			if v.Valid {
				lastUsed = v.Time
			}
		}
		result[i] = &domain.RecentCategory{
			ID:       row.ID,
			Name:     row.Name,
			LastUsed: lastUsed,
		}
	}
	return result, nil
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
		IsCCPayment:     t.IsCcPayment,
		CreatedAt:       t.CreatedAt.Time,
		UpdatedAt:       t.UpdatedAt.Time,
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
	if t.CategoryID.Valid {
		transaction.CategoryID = &t.CategoryID.Int32
	}
	// CC Lifecycle (v2 simplified - compute CCState from isPaid and billedAt)
	if t.BilledAt.Valid {
		transaction.BilledAt = &t.BilledAt.Time
	}
	if t.SettlementIntent.Valid {
		intent := domain.SettlementIntent(t.SettlementIntent.String)
		transaction.SettlementIntent = &intent
	}
	// Compute CCState from isPaid and billedAt (only for CC transactions - those with settlement intent)
	if t.SettlementIntent.Valid {
		transaction.CCState = domain.ComputeCCState(t.IsPaid, transaction.BilledAt)
	}
	// Recurring/Projection (v2)
	if t.Source.Valid {
		transaction.Source = t.Source.String
	} else {
		transaction.Source = "manual" // default
	}
	if t.TemplateID.Valid {
		transaction.TemplateID = &t.TemplateID.Int32
	}
	transaction.IsProjected = t.IsProjected.Bool
	return transaction
}

func sqlcTransactionWithCategoryToDomain(t sqlc.GetTransactionsWithCategoryRow) *domain.Transaction {
	transaction := &domain.Transaction{
		ID:              t.ID,
		WorkspaceID:     t.WorkspaceID,
		AccountID:       t.AccountID,
		Name:            t.Name,
		Amount:          pgNumericToDecimal(t.Amount),
		Type:            domain.TransactionType(t.Type),
		TransactionDate: t.TransactionDate.Time,
		IsPaid:          t.IsPaid,
		IsCCPayment:     t.IsCcPayment,
		CreatedAt:       t.CreatedAt.Time,
		UpdatedAt:       t.UpdatedAt.Time,
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
	if t.CategoryID.Valid {
		transaction.CategoryID = &t.CategoryID.Int32
	}
	if t.CategoryName.Valid {
		transaction.CategoryName = &t.CategoryName.String
	}
	// CC Lifecycle (v2 simplified - compute CCState from isPaid and billedAt)
	if t.BilledAt.Valid {
		transaction.BilledAt = &t.BilledAt.Time
	}
	if t.SettlementIntent.Valid {
		intent := domain.SettlementIntent(t.SettlementIntent.String)
		transaction.SettlementIntent = &intent
	}
	// Compute CCState from isPaid and billedAt (only for CC transactions - those with settlement intent)
	if t.SettlementIntent.Valid {
		transaction.CCState = domain.ComputeCCState(t.IsPaid, transaction.BilledAt)
	}
	// Recurring/Projection (v2)
	if t.Source.Valid {
		transaction.Source = t.Source.String
	} else {
		transaction.Source = "manual" // default
	}
	if t.TemplateID.Valid {
		transaction.TemplateID = &t.TemplateID.Int32
	}
	transaction.IsProjected = t.IsProjected.Bool
	return transaction
}

// GetProjectionsByTemplate retrieves all projected transactions for a specific template
func (r *TransactionRepository) GetProjectionsByTemplate(workspaceID int32, templateID int32) ([]*domain.Transaction, error) {
	ctx := context.Background()

	rows, err := r.queries.GetProjectionsByTemplate(ctx, sqlc.GetProjectionsByTemplateParams{
		WorkspaceID: workspaceID,
		TemplateID:  pgtype.Int4{Int32: templateID, Valid: true},
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

// DeleteProjectionsByTemplate deletes all projected transactions for a template
func (r *TransactionRepository) DeleteProjectionsByTemplate(workspaceID int32, templateID int32) error {
	ctx := context.Background()

	return r.queries.DeleteProjectionsByTemplate(ctx, sqlc.DeleteProjectionsByTemplateParams{
		WorkspaceID: workspaceID,
		TemplateID:  pgtype.Int4{Int32: templateID, Valid: true},
	})
}

// OrphanActualsByTemplate unlinks actual transactions from a template (keeps them, clears template_id)
func (r *TransactionRepository) OrphanActualsByTemplate(workspaceID int32, templateID int32) error {
	ctx := context.Background()

	return r.queries.OrphanActualsByTemplate(ctx, sqlc.OrphanActualsByTemplateParams{
		WorkspaceID: workspaceID,
		TemplateID:  pgtype.Int4{Int32: templateID, Valid: true},
	})
}

// DeleteProjectionsBeyondDate deletes projections beyond a specific date (used when template end_date changes)
func (r *TransactionRepository) DeleteProjectionsBeyondDate(workspaceID int32, templateID int32, date time.Time) error {
	ctx := context.Background()

	return r.queries.DeleteProjectionsBeyondDate(ctx, sqlc.DeleteProjectionsBeyondDateParams{
		WorkspaceID:     workspaceID,
		TemplateID:      pgtype.Int4{Int32: templateID, Valid: true},
		TransactionDate: pgtype.Date{Time: date, Valid: true},
	})
}

// GetCCMetrics returns CC metrics (pending, outstanding, purchases) for a date range
func (r *TransactionRepository) GetCCMetrics(workspaceID int32, startDate, endDate time.Time) (*domain.CCMetrics, error) {
	ctx := context.Background()

	row, err := r.queries.GetCCMetrics(ctx, sqlc.GetCCMetricsParams{
		WorkspaceID:       workspaceID,
		TransactionDate:   pgtype.Date{Time: startDate, Valid: true},
		TransactionDate_2: pgtype.Date{Time: endDate, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	return &domain.CCMetrics{
		Pending:     pgNumericToDecimal(row.PendingTotal),
		Outstanding: pgNumericToDecimal(row.OutstandingTotal),
		Purchases:   pgNumericToDecimal(row.PurchasesTotal),
	}, nil
}

// BatchToggleToBilled toggles multiple pending transactions to billed state
func (r *TransactionRepository) BatchToggleToBilled(workspaceID int32, ids []int32) ([]*domain.Transaction, error) {
	ctx := context.Background()

	rows, err := r.queries.BatchToggleToBilled(ctx, sqlc.BatchToggleToBilledParams{
		Column1:     ids,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, err
	}

	transactions := make([]*domain.Transaction, len(rows))
	for i, row := range rows {
		transactions[i] = sqlcTransactionToDomain(row)
	}
	return transactions, nil
}

// GetByIDs retrieves multiple transactions by their IDs
func (r *TransactionRepository) GetByIDs(workspaceID int32, ids []int32) ([]*domain.Transaction, error) {
	ctx := context.Background()

	rows, err := r.queries.GetTransactionsByIDs(ctx, sqlc.GetTransactionsByIDsParams{
		WorkspaceID: workspaceID,
		Column2:     ids,
	})
	if err != nil {
		return nil, err
	}

	transactions := make([]*domain.Transaction, len(rows))
	for i, row := range rows {
		transactions[i] = sqlcTransactionToDomain(row)
	}
	return transactions, nil
}

// BulkSettle updates multiple transactions to settled state
func (r *TransactionRepository) BulkSettle(workspaceID int32, ids []int32) ([]*domain.Transaction, error) {
	ctx := context.Background()

	rows, err := r.queries.BulkSettleTransactions(ctx, sqlc.BulkSettleTransactionsParams{
		Column1:     ids,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, err
	}

	transactions := make([]*domain.Transaction, len(rows))
	for i, row := range rows {
		transactions[i] = sqlcTransactionToDomain(row)
	}
	return transactions, nil
}

// GetDeferredForSettlement retrieves all billed+deferred transactions that need settlement
func (r *TransactionRepository) GetDeferredForSettlement(workspaceID int32) ([]*domain.Transaction, error) {
	ctx := context.Background()

	rows, err := r.queries.GetDeferredForSettlement(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	transactions := make([]*domain.Transaction, len(rows))
	for i, row := range rows {
		transactions[i] = sqlcDeferredRowToDomain(row)
	}
	return transactions, nil
}

// GetImmediateForSettlement retrieves billed transactions with immediate intent for the current month
func (r *TransactionRepository) GetImmediateForSettlement(workspaceID int32, startDate, endDate time.Time) ([]*domain.Transaction, error) {
	ctx := context.Background()

	rows, err := r.queries.GetImmediateForSettlement(ctx, sqlc.GetImmediateForSettlementParams{
		WorkspaceID:       workspaceID,
		TransactionDate:   pgtype.Date{Time: startDate, Valid: true},
		TransactionDate_2: pgtype.Date{Time: endDate, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	transactions := make([]*domain.Transaction, len(rows))
	for i, row := range rows {
		transactions[i] = sqlcImmediateRowToDomain(row)
	}
	return transactions, nil
}

// GetOverdueCC retrieves overdue CC transactions (billed + deferred for 2+ months)
func (r *TransactionRepository) GetOverdueCC(workspaceID int32) ([]*domain.Transaction, error) {
	ctx := context.Background()

	rows, err := r.queries.GetOverdueCC(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	transactions := make([]*domain.Transaction, len(rows))
	for i, row := range rows {
		transactions[i] = sqlcOverdueRowToDomain(row)
	}
	return transactions, nil
}

// AtomicSettle creates a transfer transaction and settles CC transactions atomically
// within a single database transaction. If any operation fails, all changes are rolled back.
func (r *TransactionRepository) AtomicSettle(transferTx *domain.Transaction, settleIDs []int32) (*domain.Transaction, int, error) {
	ctx := context.Background()

	// Start a transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// 1. Create transfer transaction
	createdTransfer, err := r.createTransactionWithTx(ctx, qtx, transferTx)
	if err != nil {
		return nil, 0, err
	}

	// 2. Bulk settle all CC transactions
	rows, err := qtx.BulkSettleTransactions(ctx, sqlc.BulkSettleTransactionsParams{
		Column1:     settleIDs,
		WorkspaceID: transferTx.WorkspaceID,
	})
	if err != nil {
		return nil, 0, err
	}

	// 3. Verify all requested transactions were settled
	if len(rows) != len(settleIDs) {
		return nil, 0, domain.ErrTransactionsNotFound
	}

	// 4. Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, 0, err
	}

	return createdTransfer, len(rows), nil
}

// GetByDateRangeForAggregation retrieves all transactions in a date range for aggregation
// This method returns all transactions without pagination, intended for dashboard calculations
func (r *TransactionRepository) GetByDateRangeForAggregation(workspaceID int32, startDate, endDate time.Time) ([]*domain.Transaction, error) {
	ctx := context.Background()

	rows, err := r.queries.GetTransactionsForAggregation(ctx, sqlc.GetTransactionsForAggregationParams{
		WorkspaceID: workspaceID,
		StartDate:   pgtype.Date{Time: startDate, Valid: true},
		EndDate:     pgtype.Date{Time: endDate, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	transactions := make([]*domain.Transaction, len(rows))
	for i, row := range rows {
		transactions[i] = sqlcAggregationRowToDomain(row)
	}
	return transactions, nil
}

// sqlcAggregationRowToDomain converts a GetTransactionsForAggregationRow to domain.Transaction
func sqlcAggregationRowToDomain(row sqlc.GetTransactionsForAggregationRow) *domain.Transaction {
	transaction := &domain.Transaction{
		ID:              row.ID,
		WorkspaceID:     row.WorkspaceID,
		AccountID:       row.AccountID,
		Name:            row.Name,
		Amount:          pgNumericToDecimal(row.Amount),
		Type:            domain.TransactionType(row.Type),
		TransactionDate: row.TransactionDate.Time,
		IsPaid:          row.IsPaid,
		IsCCPayment:     row.IsCcPayment,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}

	if row.Notes.Valid {
		transaction.Notes = &row.Notes.String
	}
	if row.DeletedAt.Valid {
		transaction.DeletedAt = &row.DeletedAt.Time
	}
	if row.TransferPairID.Valid {
		pairID := uuid.UUID(row.TransferPairID.Bytes)
		transaction.TransferPairID = &pairID
	}
	if row.CategoryID.Valid {
		transaction.CategoryID = &row.CategoryID.Int32
	}
	if row.CategoryName.Valid {
		transaction.CategoryName = &row.CategoryName.String
	}
	// CC Lifecycle (v2 simplified - compute CCState from isPaid and billedAt)
	if row.BilledAt.Valid {
		transaction.BilledAt = &row.BilledAt.Time
	}
	if row.SettlementIntent.Valid {
		intent := domain.SettlementIntent(row.SettlementIntent.String)
		transaction.SettlementIntent = &intent
	}
	// Compute CCState from isPaid and billedAt (only for CC transactions - those with settlement intent)
	if row.SettlementIntent.Valid {
		transaction.CCState = domain.ComputeCCState(row.IsPaid, transaction.BilledAt)
	}
	// Projection fields (v2)
	if row.Source.Valid {
		transaction.Source = row.Source.String
	}
	if row.TemplateID.Valid {
		transaction.TemplateID = &row.TemplateID.Int32
	}
	if row.IsProjected.Valid {
		transaction.IsProjected = row.IsProjected.Bool
	}

	return transaction
}

// sqlcDeferredRowToDomain converts a GetDeferredForSettlementRow to domain.Transaction
func sqlcDeferredRowToDomain(row sqlc.GetDeferredForSettlementRow) *domain.Transaction {
	transaction := &domain.Transaction{
		ID:              row.ID,
		WorkspaceID:     row.WorkspaceID,
		AccountID:       row.AccountID,
		Name:            row.Name,
		Amount:          pgNumericToDecimal(row.Amount),
		Type:            domain.TransactionType(row.Type),
		TransactionDate: row.TransactionDate.Time,
		IsPaid:          row.IsPaid,
		IsCCPayment:     row.IsCcPayment,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}

	if row.Notes.Valid {
		transaction.Notes = &row.Notes.String
	}
	if row.DeletedAt.Valid {
		transaction.DeletedAt = &row.DeletedAt.Time
	}
	if row.TransferPairID.Valid {
		pairID := uuid.UUID(row.TransferPairID.Bytes)
		transaction.TransferPairID = &pairID
	}
	if row.CategoryID.Valid {
		transaction.CategoryID = &row.CategoryID.Int32
	}
	// CC Lifecycle (v2 simplified)
	if row.BilledAt.Valid {
		transaction.BilledAt = &row.BilledAt.Time
	}
	if row.SettlementIntent.Valid {
		intent := domain.SettlementIntent(row.SettlementIntent.String)
		transaction.SettlementIntent = &intent
	}
	// Compute CCState from isPaid and billedAt
	if row.SettlementIntent.Valid {
		transaction.CCState = domain.ComputeCCState(row.IsPaid, transaction.BilledAt)
	}
	// Projection fields
	if row.Source.Valid {
		transaction.Source = row.Source.String
	}
	if row.TemplateID.Valid {
		transaction.TemplateID = &row.TemplateID.Int32
	}
	if row.IsProjected.Valid {
		transaction.IsProjected = row.IsProjected.Bool
	}

	return transaction
}

// sqlcImmediateRowToDomain converts a GetImmediateForSettlementRow to domain.Transaction
func sqlcImmediateRowToDomain(row sqlc.GetImmediateForSettlementRow) *domain.Transaction {
	transaction := &domain.Transaction{
		ID:              row.ID,
		WorkspaceID:     row.WorkspaceID,
		AccountID:       row.AccountID,
		Name:            row.Name,
		Amount:          pgNumericToDecimal(row.Amount),
		Type:            domain.TransactionType(row.Type),
		TransactionDate: row.TransactionDate.Time,
		IsPaid:          row.IsPaid,
		IsCCPayment:     row.IsCcPayment,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}

	if row.Notes.Valid {
		transaction.Notes = &row.Notes.String
	}
	if row.DeletedAt.Valid {
		transaction.DeletedAt = &row.DeletedAt.Time
	}
	if row.TransferPairID.Valid {
		pairID := uuid.UUID(row.TransferPairID.Bytes)
		transaction.TransferPairID = &pairID
	}
	if row.CategoryID.Valid {
		transaction.CategoryID = &row.CategoryID.Int32
	}
	// CC Lifecycle (v2 simplified)
	if row.BilledAt.Valid {
		transaction.BilledAt = &row.BilledAt.Time
	}
	if row.SettlementIntent.Valid {
		intent := domain.SettlementIntent(row.SettlementIntent.String)
		transaction.SettlementIntent = &intent
	}
	// Compute CCState from isPaid and billedAt
	if row.SettlementIntent.Valid {
		transaction.CCState = domain.ComputeCCState(row.IsPaid, transaction.BilledAt)
	}
	// Projection fields
	if row.Source.Valid {
		transaction.Source = row.Source.String
	}
	if row.TemplateID.Valid {
		transaction.TemplateID = &row.TemplateID.Int32
	}
	if row.IsProjected.Valid {
		transaction.IsProjected = row.IsProjected.Bool
	}

	return transaction
}

// sqlcOverdueRowToDomain converts a GetOverdueCCRow to domain.Transaction
func sqlcOverdueRowToDomain(row sqlc.GetOverdueCCRow) *domain.Transaction {
	transaction := &domain.Transaction{
		ID:              row.ID,
		WorkspaceID:     row.WorkspaceID,
		AccountID:       row.AccountID,
		Name:            row.Name,
		Amount:          pgNumericToDecimal(row.Amount),
		Type:            domain.TransactionType(row.Type),
		TransactionDate: row.TransactionDate.Time,
		IsPaid:          row.IsPaid,
		IsCCPayment:     row.IsCcPayment,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}

	if row.Notes.Valid {
		transaction.Notes = &row.Notes.String
	}
	if row.DeletedAt.Valid {
		transaction.DeletedAt = &row.DeletedAt.Time
	}
	if row.TransferPairID.Valid {
		pairID := uuid.UUID(row.TransferPairID.Bytes)
		transaction.TransferPairID = &pairID
	}
	if row.CategoryID.Valid {
		transaction.CategoryID = &row.CategoryID.Int32
	}
	// CC Lifecycle (v2 simplified)
	if row.BilledAt.Valid {
		transaction.BilledAt = &row.BilledAt.Time
	}
	if row.SettlementIntent.Valid {
		intent := domain.SettlementIntent(row.SettlementIntent.String)
		transaction.SettlementIntent = &intent
	}
	// Compute CCState from isPaid and billedAt
	if row.SettlementIntent.Valid {
		transaction.CCState = domain.ComputeCCState(row.IsPaid, transaction.BilledAt)
	}
	// Projection fields
	if row.Source.Valid {
		transaction.Source = row.Source.String
	}
	if row.TemplateID.Valid {
		transaction.TemplateID = &row.TemplateID.Int32
	}
	if row.IsProjected.Valid {
		transaction.IsProjected = row.IsProjected.Bool
	}

	return transaction
}
