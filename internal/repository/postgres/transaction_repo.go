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

	var categoryID pgtype.Int4
	if transaction.CategoryID != nil {
		categoryID.Int32 = *transaction.CategoryID
		categoryID.Valid = true
	}

	var templateID pgtype.Int4
	if transaction.TemplateID != nil {
		templateID.Int32 = *transaction.TemplateID
		templateID.Valid = true
	}

	var ccState pgtype.Text
	if transaction.CCState != nil {
		ccState.String = string(*transaction.CCState)
		ccState.Valid = true
	}

	// Default source to "manual" if not set
	source := string(transaction.Source)
	if source == "" {
		source = string(domain.TransactionSourceManual)
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
		CategoryID:         categoryID,
		IsCcPayment:        transaction.IsCCPayment,
		TemplateID:         templateID,
		CcState:            ccState,
		Source:             source,
		IsProjected:        transaction.IsProjected,
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

	var categoryID pgtype.Int4
	if data.CategoryID != nil {
		categoryID.Int32 = *data.CategoryID
		categoryID.Valid = true
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
		CategoryID:         categoryID,
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

	var categoryID pgtype.Int4
	if transaction.CategoryID != nil {
		categoryID.Int32 = *transaction.CategoryID
		categoryID.Valid = true
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
		CategoryID:         categoryID,
		IsCcPayment:        transaction.IsCCPayment,
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

// GetCCPayableSummary returns unpaid CC transaction totals grouped by settlement intent
func (r *TransactionRepository) GetCCPayableSummary(workspaceID int32) ([]*domain.CCPayableSummaryRow, error) {
	ctx := context.Background()

	rows, err := r.queries.GetCCPayableSummary(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.CCPayableSummaryRow, len(rows))
	for i, row := range rows {
		result[i] = &domain.CCPayableSummaryRow{
			SettlementIntent: domain.CCSettlementIntent(row.CcSettlementIntent.String),
			Total:            pgNumericToDecimal(row.Total),
		}
	}
	return result, nil
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

// GetCCPayableBreakdown returns all unpaid CC transactions for payable breakdown
func (r *TransactionRepository) GetCCPayableBreakdown(workspaceID int32) ([]*domain.CCPayableTransaction, error) {
	ctx := context.Background()

	rows, err := r.queries.GetCCPayableBreakdown(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.CCPayableTransaction, len(rows))
	for i, row := range rows {
		// Default to this_month if no settlement intent (shouldn't happen with CC transactions)
		settlementIntent := domain.CCSettlementThisMonth
		if row.CcSettlementIntent.Valid {
			settlementIntent = domain.CCSettlementIntent(row.CcSettlementIntent.String)
		}

		result[i] = &domain.CCPayableTransaction{
			ID:               row.ID,
			Name:             row.Name,
			Amount:           pgNumericToDecimal(row.Amount),
			TransactionDate:  row.TransactionDate.Time,
			SettlementIntent: settlementIntent,
			AccountID:        row.AccountID,
			AccountName:      row.AccountName,
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
	if t.CategoryID.Valid {
		transaction.CategoryID = &t.CategoryID.Int32
	}
	if t.TemplateID.Valid {
		transaction.TemplateID = &t.TemplateID.Int32
	}
	// V2 fields
	transaction.Source = domain.TransactionSource(t.Source)
	transaction.IsProjected = t.IsProjected
	if t.CcState.Valid {
		state := domain.CCState(t.CcState.String)
		transaction.CCState = &state
	}
	if t.BilledAt.Valid {
		transaction.BilledAt = &t.BilledAt.Time
	}
	if t.SettledAt.Valid {
		transaction.SettledAt = &t.SettledAt.Time
	}
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
	if t.CategoryID.Valid {
		transaction.CategoryID = &t.CategoryID.Int32
	}
	if t.CategoryName.Valid {
		transaction.CategoryName = &t.CategoryName.String
	}
	if t.TemplateID.Valid {
		transaction.TemplateID = &t.TemplateID.Int32
	}
	// V2 fields
	transaction.Source = domain.TransactionSource(t.Source)
	transaction.IsProjected = t.IsProjected
	if t.CcState.Valid {
		state := domain.CCState(t.CcState.String)
		transaction.CCState = &state
	}
	if t.BilledAt.Valid {
		transaction.BilledAt = &t.BilledAt.Time
	}
	if t.SettledAt.Valid {
		transaction.SettledAt = &t.SettledAt.Time
	}
	return transaction
}

// =====================================================
// V2 Projection Methods
// =====================================================

// DeleteProjectionsByTemplateID soft-deletes all projected transactions for a template
func (r *TransactionRepository) DeleteProjectionsByTemplateID(workspaceID int32, templateID int32) (int64, error) {
	ctx := context.Background()

	var tid pgtype.Int4
	tid.Int32 = templateID
	tid.Valid = true

	return r.queries.DeleteProjectionsByTemplateID(ctx, sqlc.DeleteProjectionsByTemplateIDParams{
		WorkspaceID: workspaceID,
		TemplateID:  tid,
	})
}

// OrphanActualsByTemplateID removes template_id from actual transactions (not projections)
func (r *TransactionRepository) OrphanActualsByTemplateID(workspaceID int32, templateID int32) (int64, error) {
	ctx := context.Background()

	var tid pgtype.Int4
	tid.Int32 = templateID
	tid.Valid = true

	return r.queries.OrphanActualsByTemplateID(ctx, sqlc.OrphanActualsByTemplateIDParams{
		WorkspaceID: workspaceID,
		TemplateID:  tid,
	})
}

// CheckProjectionExists checks if a projection already exists for a template in a specific month
func (r *TransactionRepository) CheckProjectionExists(workspaceID int32, templateID int32, year int, month int) (bool, error) {
	ctx := context.Background()

	tid := pgtype.Int4{Int32: templateID, Valid: true}

	count, err := r.queries.CheckProjectionExists(ctx, sqlc.CheckProjectionExistsParams{
		TemplateID:  tid,
		WorkspaceID: workspaceID,
		Year:        int32(year),
		Month:       int32(month),
	})
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// DeleteProjectionsBeyondDate soft-deletes projections beyond a specific date
func (r *TransactionRepository) DeleteProjectionsBeyondDate(workspaceID int32, templateID int32, endDate time.Time) (int64, error) {
	ctx := context.Background()

	var tid pgtype.Int4
	tid.Int32 = templateID
	tid.Valid = true

	var ed pgtype.Date
	ed.Time = endDate
	ed.Valid = true

	return r.queries.DeleteProjectionsBeyondDate(ctx, sqlc.DeleteProjectionsBeyondDateParams{
		WorkspaceID: workspaceID,
		TemplateID:  tid,
		EndDate:     ed,
	})
}

// GetProjectionsByTemplateID retrieves all projected transactions for a template
func (r *TransactionRepository) GetProjectionsByTemplateID(workspaceID int32, templateID int32) ([]*domain.Transaction, error) {
	ctx := context.Background()

	var tid pgtype.Int4
	tid.Int32 = templateID
	tid.Valid = true

	rows, err := r.queries.GetProjectionsByTemplateID(ctx, sqlc.GetProjectionsByTemplateIDParams{
		WorkspaceID: workspaceID,
		TemplateID:  tid,
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

// =====================================================
// V2 CC LIFECYCLE METHODS
// =====================================================

// ToggleCCBilled toggles a CC transaction between pending and billed states
func (r *TransactionRepository) ToggleCCBilled(workspaceID int32, id int32) (*domain.Transaction, error) {
	ctx := context.Background()

	row, err := r.queries.ToggleCCBilled(ctx, sqlc.ToggleCCBilledParams{
		WorkspaceID: workspaceID,
		ID:          id,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrTransactionNotFound
		}
		return nil, err
	}

	return sqlcTransactionToDomain(row), nil
}

// UpdateCCState updates the CC state of a transaction
func (r *TransactionRepository) UpdateCCState(workspaceID int32, id int32, state domain.CCState) (*domain.Transaction, error) {
	ctx := context.Background()

	row, err := r.queries.UpdateCCState(ctx, sqlc.UpdateCCStateParams{
		WorkspaceID: workspaceID,
		ID:          id,
		CcState:     pgtype.Text{String: string(state), Valid: true},
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrTransactionNotFound
		}
		return nil, err
	}

	return sqlcTransactionToDomain(row), nil
}

// GetPendingCCByMonth retrieves pending CC transactions for a specific month
func (r *TransactionRepository) GetPendingCCByMonth(workspaceID int32, year int, month int) ([]*domain.CCTransactionWithAccount, error) {
	ctx := context.Background()

	rows, err := r.queries.GetPendingCCByMonth(ctx, sqlc.GetPendingCCByMonthParams{
		WorkspaceID: workspaceID,
		Year:        int32(year),
		Month:       int32(month),
	})
	if err != nil {
		return nil, err
	}

	result := make([]*domain.CCTransactionWithAccount, len(rows))
	for i, row := range rows {
		result[i] = sqlcPendingCCRowToDomain(row)
	}
	return result, nil
}

// GetBilledCCByMonth retrieves billed (deferred) CC transactions for a specific month
func (r *TransactionRepository) GetBilledCCByMonth(workspaceID int32, year int, month int) ([]*domain.CCTransactionWithAccount, error) {
	ctx := context.Background()

	rows, err := r.queries.GetBilledCCByMonth(ctx, sqlc.GetBilledCCByMonthParams{
		WorkspaceID: workspaceID,
		Year:        int32(year),
		Month:       int32(month),
	})
	if err != nil {
		return nil, err
	}

	result := make([]*domain.CCTransactionWithAccount, len(rows))
	for i, row := range rows {
		result[i] = sqlcBilledCCRowToDomain(row)
	}
	return result, nil
}

// GetCCMetricsByMonth retrieves CC metrics for a specific month
func (r *TransactionRepository) GetCCMetricsByMonth(workspaceID int32, year int, month int) (*domain.CCMetrics, error) {
	ctx := context.Background()

	row, err := r.queries.GetCCMetricsByMonth(ctx, sqlc.GetCCMetricsByMonthParams{
		WorkspaceID: workspaceID,
		Year:        int32(year),
		Month:       int32(month),
	})
	if err != nil {
		return nil, err
	}

	totalPurchases := pgNumericToDecimal(row.TotalPurchases)
	outstanding := pgNumericToDecimal(row.Outstanding)
	pending := pgNumericToDecimal(row.Pending)

	return &domain.CCMetrics{
		TotalPurchases: totalPurchases,
		Outstanding:    outstanding,
		Pending:        pending,
	}, nil
}

// BulkSettleTransactions settles multiple CC transactions at once
func (r *TransactionRepository) BulkSettleTransactions(workspaceID int32, ids []int32) (int64, error) {
	ctx := context.Background()

	return r.queries.BulkSettleTransactions(ctx, sqlc.BulkSettleTransactionsParams{
		WorkspaceID: workspaceID,
		Ids:         ids,
	})
}

// sqlcPendingCCRowToDomain converts sqlc GetPendingCCByMonthRow to domain type
func sqlcPendingCCRowToDomain(row sqlc.GetPendingCCByMonthRow) *domain.CCTransactionWithAccount {
	tx := &domain.CCTransactionWithAccount{}
	tx.ID = row.ID
	tx.WorkspaceID = row.WorkspaceID
	tx.AccountID = row.AccountID
	tx.Name = row.Name
	tx.Amount = pgNumericToDecimal(row.Amount)
	tx.Type = domain.TransactionType(row.Type)
	tx.TransactionDate = row.TransactionDate.Time
	tx.IsPaid = row.IsPaid
	tx.CreatedAt = row.CreatedAt.Time
	tx.UpdatedAt = row.UpdatedAt.Time

	if row.CcSettlementIntent.Valid {
		intent := domain.CCSettlementIntent(row.CcSettlementIntent.String)
		tx.CCSettlementIntent = &intent
	}
	if row.Notes.Valid {
		tx.Notes = &row.Notes.String
	}
	if row.TransferPairID.Valid {
		pairID := row.TransferPairID.Bytes
		uid := uuid.UUID(pairID)
		tx.TransferPairID = &uid
	}
	if row.CategoryID.Valid {
		tx.CategoryID = &row.CategoryID.Int32
	}
	if row.TemplateID.Valid {
		tx.TemplateID = &row.TemplateID.Int32
	}
	if row.CcState.Valid {
		state := domain.CCState(row.CcState.String)
		tx.CCState = &state
	}
	if row.BilledAt.Valid {
		tx.BilledAt = &row.BilledAt.Time
	}
	if row.SettledAt.Valid {
		tx.SettledAt = &row.SettledAt.Time
	}
	if row.Source != "" {
		tx.Source = domain.TransactionSource(row.Source)
	}
	tx.IsProjected = row.IsProjected
	if row.AccountName != "" {
		tx.AccountName = &row.AccountName
	}

	return tx
}

// sqlcBilledCCRowToDomain converts sqlc GetBilledCCByMonthRow to domain type
func sqlcBilledCCRowToDomain(row sqlc.GetBilledCCByMonthRow) *domain.CCTransactionWithAccount {
	tx := &domain.CCTransactionWithAccount{}
	tx.ID = row.ID
	tx.WorkspaceID = row.WorkspaceID
	tx.AccountID = row.AccountID
	tx.Name = row.Name
	tx.Amount = pgNumericToDecimal(row.Amount)
	tx.Type = domain.TransactionType(row.Type)
	tx.TransactionDate = row.TransactionDate.Time
	tx.IsPaid = row.IsPaid
	tx.CreatedAt = row.CreatedAt.Time
	tx.UpdatedAt = row.UpdatedAt.Time

	if row.CcSettlementIntent.Valid {
		intent := domain.CCSettlementIntent(row.CcSettlementIntent.String)
		tx.CCSettlementIntent = &intent
	}
	if row.Notes.Valid {
		tx.Notes = &row.Notes.String
	}
	if row.TransferPairID.Valid {
		pairID := row.TransferPairID.Bytes
		uid := uuid.UUID(pairID)
		tx.TransferPairID = &uid
	}
	if row.CategoryID.Valid {
		tx.CategoryID = &row.CategoryID.Int32
	}
	if row.TemplateID.Valid {
		tx.TemplateID = &row.TemplateID.Int32
	}
	if row.CcState.Valid {
		state := domain.CCState(row.CcState.String)
		tx.CCState = &state
	}
	if row.BilledAt.Valid {
		tx.BilledAt = &row.BilledAt.Time
	}
	if row.SettledAt.Valid {
		tx.SettledAt = &row.SettledAt.Time
	}
	if row.Source != "" {
		tx.Source = domain.TransactionSource(row.Source)
	}
	tx.IsProjected = row.IsProjected
	if row.AccountName != "" {
		tx.AccountName = &row.AccountName
	}

	return tx
}

// GetTransactionsByIDs retrieves multiple transactions by their IDs
func (r *TransactionRepository) GetTransactionsByIDs(workspaceID int32, ids []int32) ([]*domain.Transaction, error) {
	ctx := context.Background()

	rows, err := r.queries.GetTransactionsByIDs(ctx, sqlc.GetTransactionsByIDsParams{
		WorkspaceID: workspaceID,
		Ids:         ids,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Transaction, len(rows))
	for i, row := range rows {
		result[i] = sqlcTransactionByIDsRowToDomain(row)
	}
	return result, nil
}

// GetDeferredCCByMonth retrieves deferred CC transactions (billed, not yet settled)
func (r *TransactionRepository) GetDeferredCCByMonth(workspaceID int32) ([]*domain.CCTransactionWithAccount, error) {
	ctx := context.Background()

	rows, err := r.queries.GetDeferredCCByMonth(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.CCTransactionWithAccount, len(rows))
	for i, row := range rows {
		result[i] = sqlcDeferredCCRowToDomain(row)
	}
	return result, nil
}

// GetOverdueCC retrieves CC transactions that are overdue (billed 2+ months ago, not yet settled)
func (r *TransactionRepository) GetOverdueCC(workspaceID int32) ([]*domain.CCTransactionWithAccount, error) {
	ctx := context.Background()

	rows, err := r.queries.GetOverdueCC(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.CCTransactionWithAccount, len(rows))
	for i, row := range rows {
		result[i] = sqlcOverdueCCRowToDomain(row)
	}
	return result, nil
}

// sqlcTransactionByIDsRowToDomain converts sqlc GetTransactionsByIDsRow to domain type
func sqlcTransactionByIDsRowToDomain(row sqlc.GetTransactionsByIDsRow) *domain.Transaction {
	tx := &domain.Transaction{
		ID:              row.ID,
		WorkspaceID:     row.WorkspaceID,
		AccountID:       row.AccountID,
		Name:            row.Name,
		Amount:          pgNumericToDecimal(row.Amount),
		Type:            domain.TransactionType(row.Type),
		TransactionDate: row.TransactionDate.Time,
		IsPaid:          row.IsPaid,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}

	if row.CcSettlementIntent.Valid {
		intent := domain.CCSettlementIntent(row.CcSettlementIntent.String)
		tx.CCSettlementIntent = &intent
	}
	if row.Notes.Valid {
		tx.Notes = &row.Notes.String
	}
	if row.TransferPairID.Valid {
		pairID := row.TransferPairID.Bytes
		uid := uuid.UUID(pairID)
		tx.TransferPairID = &uid
	}
	if row.CategoryID.Valid {
		tx.CategoryID = &row.CategoryID.Int32
	}
	if row.TemplateID.Valid {
		tx.TemplateID = &row.TemplateID.Int32
	}
	if row.CcState.Valid {
		state := domain.CCState(row.CcState.String)
		tx.CCState = &state
	}
	if row.BilledAt.Valid {
		tx.BilledAt = &row.BilledAt.Time
	}
	if row.SettledAt.Valid {
		tx.SettledAt = &row.SettledAt.Time
	}
	if row.Source != "" {
		tx.Source = domain.TransactionSource(row.Source)
	}
	tx.IsProjected = row.IsProjected
	tx.IsCCPayment = row.IsCcPayment
	if row.DeletedAt.Valid {
		tx.DeletedAt = &row.DeletedAt.Time
	}

	return tx
}

// sqlcDeferredCCRowToDomain converts sqlc GetDeferredCCByMonthRow to domain type
func sqlcDeferredCCRowToDomain(row sqlc.GetDeferredCCByMonthRow) *domain.CCTransactionWithAccount {
	tx := &domain.CCTransactionWithAccount{}
	tx.ID = row.ID
	tx.WorkspaceID = row.WorkspaceID
	tx.AccountID = row.AccountID
	tx.Name = row.Name
	tx.Amount = pgNumericToDecimal(row.Amount)
	tx.Type = domain.TransactionType(row.Type)
	tx.TransactionDate = row.TransactionDate.Time
	tx.IsPaid = row.IsPaid
	tx.CreatedAt = row.CreatedAt.Time
	tx.UpdatedAt = row.UpdatedAt.Time
	tx.OriginYear = int(row.OriginYear)
	tx.OriginMonth = int(row.OriginMonth)

	if row.CcSettlementIntent.Valid {
		intent := domain.CCSettlementIntent(row.CcSettlementIntent.String)
		tx.CCSettlementIntent = &intent
	}
	if row.Notes.Valid {
		tx.Notes = &row.Notes.String
	}
	if row.TransferPairID.Valid {
		pairID := row.TransferPairID.Bytes
		uid := uuid.UUID(pairID)
		tx.TransferPairID = &uid
	}
	if row.CategoryID.Valid {
		tx.CategoryID = &row.CategoryID.Int32
	}
	if row.TemplateID.Valid {
		tx.TemplateID = &row.TemplateID.Int32
	}
	if row.CcState.Valid {
		state := domain.CCState(row.CcState.String)
		tx.CCState = &state
	}
	if row.BilledAt.Valid {
		tx.BilledAt = &row.BilledAt.Time
	}
	if row.SettledAt.Valid {
		tx.SettledAt = &row.SettledAt.Time
	}
	if row.Source != "" {
		tx.Source = domain.TransactionSource(row.Source)
	}
	tx.IsProjected = row.IsProjected
	tx.IsCCPayment = row.IsCcPayment
	if row.AccountName != "" {
		tx.AccountName = &row.AccountName
	}

	return tx
}

// sqlcOverdueCCRowToDomain converts sqlc GetOverdueCCRow to domain type
func sqlcOverdueCCRowToDomain(row sqlc.GetOverdueCCRow) *domain.CCTransactionWithAccount {
	tx := &domain.CCTransactionWithAccount{}
	tx.ID = row.ID
	tx.WorkspaceID = row.WorkspaceID
	tx.AccountID = row.AccountID
	tx.Name = row.Name
	tx.Amount = pgNumericToDecimal(row.Amount)
	tx.Type = domain.TransactionType(row.Type)
	tx.TransactionDate = row.TransactionDate.Time
	tx.IsPaid = row.IsPaid
	tx.CreatedAt = row.CreatedAt.Time
	tx.UpdatedAt = row.UpdatedAt.Time
	tx.OriginYear = int(row.OriginYear)
	tx.OriginMonth = int(row.OriginMonth)

	if row.CcSettlementIntent.Valid {
		intent := domain.CCSettlementIntent(row.CcSettlementIntent.String)
		tx.CCSettlementIntent = &intent
	}
	if row.Notes.Valid {
		tx.Notes = &row.Notes.String
	}
	if row.TransferPairID.Valid {
		pairID := row.TransferPairID.Bytes
		uid := uuid.UUID(pairID)
		tx.TransferPairID = &uid
	}
	if row.CategoryID.Valid {
		tx.CategoryID = &row.CategoryID.Int32
	}
	if row.TemplateID.Valid {
		tx.TemplateID = &row.TemplateID.Int32
	}
	if row.CcState.Valid {
		state := domain.CCState(row.CcState.String)
		tx.CCState = &state
	}
	if row.BilledAt.Valid {
		tx.BilledAt = &row.BilledAt.Time
	}
	if row.SettledAt.Valid {
		tx.SettledAt = &row.SettledAt.Time
	}
	if row.Source != "" {
		tx.Source = domain.TransactionSource(row.Source)
	}
	tx.IsProjected = row.IsProjected
	tx.IsCCPayment = row.IsCcPayment
	if row.AccountName != "" {
		tx.AccountName = &row.AccountName
	}

	return tx
}

// UpdateAmount updates only the amount of a transaction
func (r *TransactionRepository) UpdateAmount(workspaceID int32, id int32, amount decimal.Decimal) error {
	ctx := context.Background()

	amountPg, err := decimalToPgNumeric(amount)
	if err != nil {
		return err
	}

	err = r.queries.UpdateTransactionAmount(ctx, sqlc.UpdateTransactionAmountParams{
		ID:          id,
		WorkspaceID: workspaceID,
		Amount:      amountPg,
	})

	return err
}

// GetExpensesByDateRange retrieves all expense transactions within a date range
func (r *TransactionRepository) GetExpensesByDateRange(workspaceID int32, startDate, endDate time.Time) ([]*domain.Transaction, error) {
	ctx := context.Background()

	rows, err := r.queries.GetExpensesByDateRange(ctx, sqlc.GetExpensesByDateRangeParams{
		WorkspaceID:     workspaceID,
		TransactionDate: pgtype.Date{Time: startDate, Valid: true},
		TransactionDate_2: pgtype.Date{Time: endDate, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Transaction, len(rows))
	for i, row := range rows {
		result[i] = sqlcExpenseRowToDomain(row)
	}

	return result, nil
}

// sqlcExpenseRowToDomain converts sqlc GetExpensesByDateRangeRow to domain type
func sqlcExpenseRowToDomain(row sqlc.GetExpensesByDateRangeRow) *domain.Transaction {
	tx := &domain.Transaction{
		ID:              row.ID,
		WorkspaceID:    row.WorkspaceID,
		AccountID:      row.AccountID,
		Name:           row.Name,
		Amount:         pgNumericToDecimal(row.Amount),
		Type:           domain.TransactionType(row.Type),
		TransactionDate: row.TransactionDate.Time,
		IsPaid:         row.IsPaid,
		IsCCPayment:    row.IsCcPayment,
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
	}

	if row.CategoryID.Valid {
		tx.CategoryID = &row.CategoryID.Int32
	}
	if row.CcSettlementIntent.Valid {
		intent := domain.CCSettlementIntent(row.CcSettlementIntent.String)
		tx.CCSettlementIntent = &intent
	}
	if row.Notes.Valid {
		tx.Notes = &row.Notes.String
	}
	if row.TransferPairID.Valid {
		pairID := row.TransferPairID.Bytes
		uid := uuid.UUID(pairID)
		tx.TransferPairID = &uid
	}
	if row.TemplateID.Valid {
		tx.TemplateID = &row.TemplateID.Int32
	}
	if row.CcState.Valid {
		state := domain.CCState(row.CcState.String)
		tx.CCState = &state
	}
	if row.BilledAt.Valid {
		tx.BilledAt = &row.BilledAt.Time
	}
	if row.SettledAt.Valid {
		tx.SettledAt = &row.SettledAt.Time
	}
	if row.Source != "" {
		tx.Source = domain.TransactionSource(row.Source)
	}
	tx.IsProjected = row.IsProjected

	return tx
}
