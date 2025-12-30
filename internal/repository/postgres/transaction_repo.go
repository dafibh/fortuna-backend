package postgres

import (
	"context"
	"fmt"

	"github.com/dafibh/fortuna/fortuna-backend/db/sqlc"
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
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
	return transaction
}
