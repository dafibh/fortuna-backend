package postgres

import (
	"context"
	"fmt"

	"github.com/dafibh/fortuna/fortuna-backend/db/sqlc"
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// APITokenRepository implements domain.APITokenRepository using PostgreSQL
type APITokenRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewAPITokenRepository creates a new APITokenRepository
func NewAPITokenRepository(pool *pgxpool.Pool) *APITokenRepository {
	return &APITokenRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Create creates a new API token
func (r *APITokenRepository) Create(ctx context.Context, token *domain.APIToken) error {
	created, err := r.queries.CreateAPIToken(ctx, sqlc.CreateAPITokenParams{
		UserID:      pgtype.UUID{Bytes: token.UserID, Valid: true},
		WorkspaceID: token.WorkspaceID,
		Description: token.Description,
		TokenHash:   token.TokenHash,
		TokenPrefix: token.TokenPrefix,
	})
	if err != nil {
		return err
	}
	// Update the token with the generated ID
	id, err := uuid.FromBytes(created.ID.Bytes[:])
	if err != nil {
		return fmt.Errorf("failed to parse token ID: %w", err)
	}
	token.ID = id
	token.CreatedAt = created.CreatedAt.Time
	return nil
}

// GetByWorkspace retrieves all active API tokens for a workspace
func (r *APITokenRepository) GetByWorkspace(ctx context.Context, workspaceID int32) ([]*domain.APIToken, error) {
	tokens, err := r.queries.GetAPITokensByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.APIToken, len(tokens))
	for i, t := range tokens {
		result[i] = sqlcAPITokenToDomain(t)
	}
	return result, nil
}

// GetByID retrieves an API token by ID within a workspace
func (r *APITokenRepository) GetByID(ctx context.Context, workspaceID int32, id uuid.UUID) (*domain.APIToken, error) {
	token, err := r.queries.GetAPITokenByID(ctx, sqlc.GetAPITokenByIDParams{
		WorkspaceID: workspaceID,
		ID:          pgtype.UUID{Bytes: id, Valid: true},
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrAPITokenNotFound
		}
		return nil, err
	}
	return sqlcAPITokenToDomain(token), nil
}

// GetByHash retrieves an active API token by its hash (for authentication)
func (r *APITokenRepository) GetByHash(ctx context.Context, hash string) (*domain.APIToken, error) {
	token, err := r.queries.GetAPITokenByHash(ctx, hash)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrAPITokenNotFound
		}
		return nil, err
	}
	return sqlcAPITokenToDomain(token), nil
}

// Revoke marks an API token as revoked
func (r *APITokenRepository) Revoke(ctx context.Context, workspaceID int32, id uuid.UUID) error {
	rowsAffected, err := r.queries.RevokeAPIToken(ctx, sqlc.RevokeAPITokenParams{
		WorkspaceID: workspaceID,
		ID:          pgtype.UUID{Bytes: id, Valid: true},
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrAPITokenNotFound
	}
	return nil
}

// UpdateLastUsed updates the last_used_at timestamp for a token
func (r *APITokenRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	return r.queries.UpdateAPITokenLastUsed(ctx, pgtype.UUID{Bytes: id, Valid: true})
}

// Helper functions

func sqlcAPITokenToDomain(t sqlc.ApiToken) *domain.APIToken {
	id, _ := uuid.FromBytes(t.ID.Bytes[:])
	userID, _ := uuid.FromBytes(t.UserID.Bytes[:])

	token := &domain.APIToken{
		ID:          id,
		UserID:      userID,
		WorkspaceID: t.WorkspaceID,
		Description: t.Description,
		TokenHash:   t.TokenHash,
		TokenPrefix: t.TokenPrefix,
		CreatedAt:   t.CreatedAt.Time,
	}

	if t.LastUsedAt.Valid {
		token.LastUsedAt = &t.LastUsedAt.Time
	}
	if t.RevokedAt.Valid {
		token.RevokedAt = &t.RevokedAt.Time
	}

	return token
}
