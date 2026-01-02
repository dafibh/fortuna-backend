package postgres

import (
	"context"

	"github.com/dafibh/fortuna/fortuna-backend/db/sqlc"
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LoanProviderRepository implements domain.LoanProviderRepository using PostgreSQL
type LoanProviderRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewLoanProviderRepository creates a new LoanProviderRepository
func NewLoanProviderRepository(pool *pgxpool.Pool) *LoanProviderRepository {
	return &LoanProviderRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Create creates a new loan provider
func (r *LoanProviderRepository) Create(provider *domain.LoanProvider) (*domain.LoanProvider, error) {
	ctx := context.Background()
	interestRate, err := decimalToPgNumeric(provider.DefaultInterestRate)
	if err != nil {
		return nil, err
	}
	created, err := r.queries.CreateLoanProvider(ctx, sqlc.CreateLoanProviderParams{
		WorkspaceID:         provider.WorkspaceID,
		Name:                provider.Name,
		CutoffDay:           provider.CutoffDay,
		DefaultInterestRate: interestRate,
	})
	if err != nil {
		if isPgUniqueViolation(err) {
			return nil, domain.ErrLoanProviderNameExists
		}
		return nil, err
	}
	return sqlcLoanProviderToDomain(created), nil
}

// GetByID retrieves a loan provider by its ID within a workspace
func (r *LoanProviderRepository) GetByID(workspaceID int32, id int32) (*domain.LoanProvider, error) {
	ctx := context.Background()
	provider, err := r.queries.GetLoanProviderByID(ctx, sqlc.GetLoanProviderByIDParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrLoanProviderNotFound
		}
		return nil, err
	}
	return sqlcLoanProviderToDomain(provider), nil
}

// GetAllByWorkspace retrieves all loan providers for a workspace
func (r *LoanProviderRepository) GetAllByWorkspace(workspaceID int32) ([]*domain.LoanProvider, error) {
	ctx := context.Background()
	providers, err := r.queries.ListLoanProviders(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.LoanProvider, len(providers))
	for i, p := range providers {
		result[i] = sqlcLoanProviderToDomain(p)
	}
	return result, nil
}

// Update updates a loan provider
func (r *LoanProviderRepository) Update(provider *domain.LoanProvider) (*domain.LoanProvider, error) {
	ctx := context.Background()
	interestRate, err := decimalToPgNumeric(provider.DefaultInterestRate)
	if err != nil {
		return nil, err
	}
	updated, err := r.queries.UpdateLoanProvider(ctx, sqlc.UpdateLoanProviderParams{
		ID:                  provider.ID,
		WorkspaceID:         provider.WorkspaceID,
		Name:                provider.Name,
		CutoffDay:           provider.CutoffDay,
		DefaultInterestRate: interestRate,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrLoanProviderNotFound
		}
		if isPgUniqueViolation(err) {
			return nil, domain.ErrLoanProviderNameExists
		}
		return nil, err
	}
	return sqlcLoanProviderToDomain(updated), nil
}

// SoftDelete marks a loan provider as deleted
func (r *LoanProviderRepository) SoftDelete(workspaceID int32, id int32) error {
	ctx := context.Background()
	return r.queries.DeleteLoanProvider(ctx, sqlc.DeleteLoanProviderParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
}

// Helper functions

func sqlcLoanProviderToDomain(p sqlc.LoanProvider) *domain.LoanProvider {
	provider := &domain.LoanProvider{
		ID:                  p.ID,
		WorkspaceID:         p.WorkspaceID,
		Name:                p.Name,
		CutoffDay:           p.CutoffDay,
		DefaultInterestRate: pgNumericToDecimal(p.DefaultInterestRate),
		CreatedAt:           p.CreatedAt.Time,
		UpdatedAt:           p.UpdatedAt.Time,
	}
	if p.DeletedAt.Valid {
		provider.DeletedAt = &p.DeletedAt.Time
	}
	return provider
}
