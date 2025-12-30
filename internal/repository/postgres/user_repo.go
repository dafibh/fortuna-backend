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

// UserRepository implements domain.UserRepository using PostgreSQL
type UserRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// GetByID retrieves a user by their UUID
func (r *UserRepository) GetByID(id uuid.UUID) (*domain.User, error) {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	user, err := r.queries.GetUserByID(context.Background(), pgID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return sqlcUserToDomain(user), nil
}

// GetByAuth0ID retrieves a user by their Auth0 ID
func (r *UserRepository) GetByAuth0ID(auth0ID string) (*domain.User, error) {
	user, err := r.queries.GetUserByAuth0ID(context.Background(), auth0ID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return sqlcUserToDomain(user), nil
}

// Create creates a new user
func (r *UserRepository) Create(user *domain.User) (*domain.User, error) {
	created, err := r.queries.CreateUser(context.Background(), sqlc.CreateUserParams{
		Auth0ID:    user.Auth0ID,
		Email:      user.Email,
		Name:       stringPtrToPgText(user.Name),
		PictureUrl: stringPtrToPgText(user.PictureURL),
	})
	if err != nil {
		return nil, err
	}
	return sqlcUserToDomain(created), nil
}

// Update updates an existing user
func (r *UserRepository) Update(user *domain.User) (*domain.User, error) {
	pgID := pgtype.UUID{Bytes: user.ID, Valid: true}
	updated, err := r.queries.UpdateUser(context.Background(), sqlc.UpdateUserParams{
		ID:         pgID,
		Email:      user.Email,
		Name:       stringPtrToPgText(user.Name),
		PictureUrl: stringPtrToPgText(user.PictureURL),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return sqlcUserToDomain(updated), nil
}

// UpdateName updates only the user's name by Auth0 ID
func (r *UserRepository) UpdateName(auth0ID string, name string) (*domain.User, error) {
	updated, err := r.queries.UpdateUserName(context.Background(), sqlc.UpdateUserNameParams{
		Auth0ID: auth0ID,
		Name:    pgtype.Text{String: name, Valid: true},
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return sqlcUserToDomain(updated), nil
}

// CreateOrGetByAuth0ID creates a new user or returns existing one (upsert on login)
func (r *UserRepository) CreateOrGetByAuth0ID(auth0ID, email string, name, pictureURL *string) (*domain.User, error) {
	user, err := r.queries.CreateOrGetUserByAuth0ID(context.Background(), sqlc.CreateOrGetUserByAuth0IDParams{
		Auth0ID:    auth0ID,
		Email:      email,
		Name:       stringPtrToPgText(name),
		PictureUrl: stringPtrToPgText(pictureURL),
	})
	if err != nil {
		return nil, err
	}
	return sqlcUserToDomain(user), nil
}

// Helper functions

func sqlcUserToDomain(u sqlc.User) *domain.User {
	id, _ := uuid.FromBytes(u.ID.Bytes[:])
	return &domain.User{
		ID:         id,
		Auth0ID:    u.Auth0ID,
		Email:      u.Email,
		Name:       pgTextToStringPtr(u.Name),
		PictureURL: pgTextToStringPtr(u.PictureUrl),
		CreatedAt:  u.CreatedAt.Time,
		UpdatedAt:  u.UpdatedAt.Time,
	}
}

func stringPtrToPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func pgTextToStringPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}
