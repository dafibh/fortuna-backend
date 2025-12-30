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
func (r *AccountRepository) GetAllByWorkspace(workspaceID int32, includeArchived bool) ([]*domain.Account, error) {
	ctx := context.Background()

	if includeArchived {
		accounts, err := r.queries.GetAccountsByWorkspaceAll(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		result := make([]*domain.Account, len(accounts))
		for i, a := range accounts {
			result[i] = sqlcAccountToDomain(a)
		}
		return result, nil
	}

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

// Update updates an account's name
func (r *AccountRepository) Update(workspaceID int32, id int32, name string) (*domain.Account, error) {
	ctx := context.Background()
	account, err := r.queries.UpdateAccount(ctx, sqlc.UpdateAccountParams{
		WorkspaceID: workspaceID,
		ID:          id,
		Name:        name,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrAccountNotFound
		}
		return nil, err
	}
	return sqlcAccountToDomain(account), nil
}

// SoftDelete marks an account as deleted (sets deleted_at timestamp)
func (r *AccountRepository) SoftDelete(workspaceID int32, id int32) error {
	ctx := context.Background()
	return r.queries.SoftDeleteAccount(ctx, sqlc.SoftDeleteAccountParams{
		WorkspaceID: workspaceID,
		ID:          id,
	})
}

// HardDelete permanently removes an account from the database
func (r *AccountRepository) HardDelete(workspaceID int32, id int32) error {
	ctx := context.Background()
	return r.queries.HardDeleteAccount(ctx, sqlc.HardDeleteAccountParams{
		WorkspaceID: workspaceID,
		ID:          id,
	})
}

// Helper functions

func sqlcAccountToDomain(a sqlc.Account) *domain.Account {
	account := &domain.Account{
		ID:             a.ID,
		WorkspaceID:    a.WorkspaceID,
		Name:           a.Name,
		AccountType:    domain.AccountType(a.AccountType),
		Template:       domain.AccountTemplate(a.Template),
		InitialBalance: pgNumericToDecimal(a.InitialBalance),
		CreatedAt:      a.CreatedAt.Time,
		UpdatedAt:      a.UpdatedAt.Time,
	}
	if a.DeletedAt.Valid {
		account.DeletedAt = &a.DeletedAt.Time
	}
	return account
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
