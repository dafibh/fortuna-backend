package postgres

import (
	"context"

	"github.com/dafibh/fortuna/fortuna-backend/db/sqlc"
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WorkspaceRepository implements domain.WorkspaceRepository using PostgreSQL
type WorkspaceRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewWorkspaceRepository creates a new WorkspaceRepository
func NewWorkspaceRepository(pool *pgxpool.Pool) *WorkspaceRepository {
	return &WorkspaceRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// GetByID retrieves a workspace by its ID
func (r *WorkspaceRepository) GetByID(id int32) (*domain.Workspace, error) {
	workspace, err := r.queries.GetWorkspaceByID(context.Background(), id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrWorkspaceNotFound
		}
		return nil, err
	}
	return sqlcWorkspaceToDomain(workspace), nil
}

// GetByUserID retrieves a workspace by user ID
func (r *WorkspaceRepository) GetByUserID(userID uuid.UUID) (*domain.Workspace, error) {
	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}
	workspace, err := r.queries.GetWorkspaceByUserID(context.Background(), pgUserID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrWorkspaceNotFound
		}
		return nil, err
	}
	return sqlcWorkspaceToDomain(workspace), nil
}

// GetByUserAuth0ID retrieves a workspace by user's Auth0 ID
func (r *WorkspaceRepository) GetByUserAuth0ID(auth0ID string) (*domain.Workspace, error) {
	workspace, err := r.queries.GetWorkspaceByUserAuth0ID(context.Background(), auth0ID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrWorkspaceNotFound
		}
		return nil, err
	}
	return sqlcWorkspaceToDomain(workspace), nil
}

// Create creates a new workspace
func (r *WorkspaceRepository) Create(workspace *domain.Workspace) (*domain.Workspace, error) {
	pgUserID := pgtype.UUID{Bytes: workspace.UserID, Valid: true}
	created, err := r.queries.CreateWorkspace(context.Background(), sqlc.CreateWorkspaceParams{
		UserID: pgUserID,
		Name:   workspace.Name,
	})
	if err != nil {
		return nil, err
	}
	return sqlcWorkspaceToDomain(created), nil
}

// Update updates an existing workspace
func (r *WorkspaceRepository) Update(workspace *domain.Workspace) (*domain.Workspace, error) {
	updated, err := r.queries.UpdateWorkspace(context.Background(), sqlc.UpdateWorkspaceParams{
		ID:   workspace.ID,
		Name: workspace.Name,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrWorkspaceNotFound
		}
		return nil, err
	}
	return sqlcWorkspaceToDomain(updated), nil
}

// Delete deletes a workspace by its ID
func (r *WorkspaceRepository) Delete(id int32) error {
	return r.queries.DeleteWorkspace(context.Background(), id)
}

// GetAllWorkspaces retrieves all workspaces (for projection sync)
func (r *WorkspaceRepository) GetAllWorkspaces() ([]*domain.Workspace, error) {
	workspaces, err := r.queries.GetAllWorkspaces(context.Background())
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Workspace, len(workspaces))
	for i, w := range workspaces {
		result[i] = sqlcWorkspaceToDomain(w)
	}
	return result, nil
}

// ClearAllData deletes all data for a workspace (but keeps the workspace itself)
// This is a destructive operation that removes all accounts, transactions, budgets, loans, wishlists, etc.
func (r *WorkspaceRepository) ClearAllData(workspaceID int32) error {
	ctx := context.Background()

	// Start a transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Delete in order to respect foreign key constraints
	// Note: Some deletes will cascade automatically, but we're explicit for clarity
	deleteStatements := []string{
		"DELETE FROM wishlist_items WHERE wishlist_id IN (SELECT id FROM wishlists WHERE workspace_id = $1)",
		"DELETE FROM wishlists WHERE workspace_id = $1",
		"DELETE FROM loans WHERE workspace_id = $1",
		"DELETE FROM loan_providers WHERE workspace_id = $1",
		"DELETE FROM transactions WHERE workspace_id = $1",
		"DELETE FROM recurring_transactions WHERE workspace_id = $1",
		"DELETE FROM budget_allocations WHERE workspace_id = $1",
		"DELETE FROM budget_categories WHERE workspace_id = $1",
		"DELETE FROM months WHERE workspace_id = $1",
		"DELETE FROM accounts WHERE workspace_id = $1",
	}

	for _, stmt := range deleteStatements {
		_, err := tx.Exec(ctx, stmt, workspaceID)
		if err != nil {
			return err
		}
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// Helper functions

func sqlcWorkspaceToDomain(w sqlc.Workspace) *domain.Workspace {
	userID, _ := uuid.FromBytes(w.UserID.Bytes[:])
	return &domain.Workspace{
		ID:        w.ID,
		UserID:    userID,
		Name:      w.Name,
		CreatedAt: w.CreatedAt.Time,
		UpdatedAt: w.UpdatedAt.Time,
	}
}
