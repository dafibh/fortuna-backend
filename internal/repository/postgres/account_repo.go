package postgres

import (
	"context"
	"fmt"

	"github.com/dafibh/fortuna/fortuna-backend/db/sqlc"
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// AccountRepository implements domain.AccountRepository using PostgreSQL
type AccountRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewAccountRepository creates a new AccountRepository
func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Create creates a new account
func (r *AccountRepository) Create(account *domain.Account) (*domain.Account, error) {
	ctx := context.Background()
	initialBalance, err := decimalToPgNumeric(account.InitialBalance)
	if err != nil {
		return nil, fmt.Errorf("invalid initial balance: %w", err)
	}

	created, err := r.queries.CreateAccount(ctx, sqlc.CreateAccountParams{
		WorkspaceID:    account.WorkspaceID,
		Name:           account.Name,
		AccountType:    string(account.AccountType),
		Template:       string(account.Template),
		InitialBalance: initialBalance,
	})
	if err != nil {
		return nil, err
	}
	return sqlcAccountToDomain(created), nil
}

// GetByID retrieves an account by its ID within a workspace
func (r *AccountRepository) GetByID(workspaceID int32, id int32) (*domain.Account, error) {
	ctx := context.Background()
	account, err := r.queries.GetAccountByID(ctx, sqlc.GetAccountByIDParams{
		WorkspaceID: workspaceID,
		ID:          id,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrAccountNotFound
		}
		return nil, err
	}
	return sqlcAccountToDomain(account), nil
}

// GetAllByWorkspace retrieves all accounts for a workspace
func (r *AccountRepository) GetAllByWorkspace(workspaceID int32) ([]*domain.Account, error) {
	ctx := context.Background()
	accounts, err := r.queries.GetAccountsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Account, len(accounts))
	for i, a := range accounts {
		result[i] = sqlcAccountToDomain(a)
	}
	return result, nil
}

// Helper functions

func sqlcAccountToDomain(a sqlc.Account) *domain.Account {
	return &domain.Account{
		ID:             a.ID,
		WorkspaceID:    a.WorkspaceID,
		Name:           a.Name,
		AccountType:    domain.AccountType(a.AccountType),
		Template:       domain.AccountTemplate(a.Template),
		InitialBalance: pgNumericToDecimal(a.InitialBalance),
		CreatedAt:      a.CreatedAt.Time,
		UpdatedAt:      a.UpdatedAt.Time,
	}
}

func decimalToPgNumeric(d decimal.Decimal) (pgtype.Numeric, error) {
	var num pgtype.Numeric
	if err := num.Scan(d.String()); err != nil {
		return pgtype.Numeric{}, err
	}
	return num, nil
}

func pgNumericToDecimal(n pgtype.Numeric) decimal.Decimal {
	if !n.Valid {
		return decimal.Zero
	}
	if n.Int == nil {
		return decimal.Zero
	}
	return decimal.NewFromBigInt(n.Int, n.Exp)
}
