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
