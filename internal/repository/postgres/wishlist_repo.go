package postgres

import (
	"context"

	"github.com/dafibh/fortuna/fortuna-backend/db/sqlc"
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WishlistRepository implements domain.WishlistRepository using PostgreSQL
type WishlistRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewWishlistRepository creates a new WishlistRepository
func NewWishlistRepository(pool *pgxpool.Pool) *WishlistRepository {
	return &WishlistRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Create creates a new wishlist
func (r *WishlistRepository) Create(wishlist *domain.Wishlist) (*domain.Wishlist, error) {
	ctx := context.Background()
	created, err := r.queries.CreateWishlist(ctx, sqlc.CreateWishlistParams{
		WorkspaceID: wishlist.WorkspaceID,
		Name:        wishlist.Name,
	})
	if err != nil {
		if isPgUniqueViolation(err) {
			return nil, domain.ErrWishlistNameExists
		}
		return nil, err
	}
	return sqlcWishlistToDomain(created), nil
}

// GetByID retrieves a wishlist by its ID within a workspace
func (r *WishlistRepository) GetByID(workspaceID int32, id int32) (*domain.Wishlist, error) {
	ctx := context.Background()
	wishlist, err := r.queries.GetWishlistByID(ctx, sqlc.GetWishlistByIDParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrWishlistNotFound
		}
		return nil, err
	}
	return sqlcWishlistToDomain(wishlist), nil
}

// GetByName retrieves a wishlist by name within a workspace
func (r *WishlistRepository) GetByName(workspaceID int32, name string) (*domain.Wishlist, error) {
	ctx := context.Background()
	wishlist, err := r.queries.GetWishlistByName(ctx, sqlc.GetWishlistByNameParams{
		WorkspaceID: workspaceID,
		Name:        name,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrWishlistNotFound
		}
		return nil, err
	}
	return sqlcWishlistToDomain(wishlist), nil
}

// GetAllByWorkspace retrieves all wishlists for a workspace
func (r *WishlistRepository) GetAllByWorkspace(workspaceID int32) ([]*domain.Wishlist, error) {
	ctx := context.Background()
	wishlists, err := r.queries.ListWishlists(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Wishlist, len(wishlists))
	for i, w := range wishlists {
		result[i] = sqlcWishlistToDomain(w)
	}
	return result, nil
}

// Update updates a wishlist
func (r *WishlistRepository) Update(wishlist *domain.Wishlist) (*domain.Wishlist, error) {
	ctx := context.Background()
	updated, err := r.queries.UpdateWishlist(ctx, sqlc.UpdateWishlistParams{
		ID:          wishlist.ID,
		WorkspaceID: wishlist.WorkspaceID,
		Name:        wishlist.Name,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrWishlistNotFound
		}
		if isPgUniqueViolation(err) {
			return nil, domain.ErrWishlistNameExists
		}
		return nil, err
	}
	return sqlcWishlistToDomain(updated), nil
}

// SoftDelete marks a wishlist as deleted
func (r *WishlistRepository) SoftDelete(workspaceID int32, id int32) error {
	ctx := context.Background()
	return r.queries.DeleteWishlist(ctx, sqlc.DeleteWishlistParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
}

// Helper function to convert sqlc type to domain type
func sqlcWishlistToDomain(w sqlc.Wishlist) *domain.Wishlist {
	wishlist := &domain.Wishlist{
		ID:          w.ID,
		WorkspaceID: w.WorkspaceID,
		Name:        w.Name,
		CreatedAt:   w.CreatedAt.Time,
		UpdatedAt:   w.UpdatedAt.Time,
	}
	if w.DeletedAt.Valid {
		wishlist.DeletedAt = &w.DeletedAt.Time
	}
	return wishlist
}
