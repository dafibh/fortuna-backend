package postgres

import (
	"context"

	"github.com/dafibh/fortuna/fortuna-backend/db/sqlc"
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WishlistItemRepository implements domain.WishlistItemRepository using PostgreSQL
type WishlistItemRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewWishlistItemRepository creates a new WishlistItemRepository
func NewWishlistItemRepository(pool *pgxpool.Pool) *WishlistItemRepository {
	return &WishlistItemRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Create creates a new wishlist item
func (r *WishlistItemRepository) Create(item *domain.WishlistItem) (*domain.WishlistItem, error) {
	ctx := context.Background()
	created, err := r.queries.CreateWishlistItem(ctx, sqlc.CreateWishlistItemParams{
		WishlistID:   item.WishlistID,
		Title:        item.Title,
		Description:  stringPtrToPgText(item.Description),
		ExternalLink: stringPtrToPgText(item.ExternalLink),
		ImageUrl:     stringPtrToPgText(item.ImageURL),
	})
	if err != nil {
		return nil, err
	}
	return sqlcWishlistItemToDomain(created), nil
}

// GetByID retrieves a wishlist item by its ID within a workspace
func (r *WishlistItemRepository) GetByID(workspaceID int32, id int32) (*domain.WishlistItem, error) {
	ctx := context.Background()
	item, err := r.queries.GetWishlistItemByID(ctx, sqlc.GetWishlistItemByIDParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrWishlistItemNotFound
		}
		return nil, err
	}
	return sqlcWishlistItemToDomain(item), nil
}

// GetAllByWishlist retrieves all items in a wishlist
func (r *WishlistItemRepository) GetAllByWishlist(workspaceID int32, wishlistID int32) ([]*domain.WishlistItem, error) {
	ctx := context.Background()
	items, err := r.queries.ListWishlistItems(ctx, sqlc.ListWishlistItemsParams{
		WishlistID:  wishlistID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.WishlistItem, len(items))
	for i, item := range items {
		result[i] = sqlcWishlistItemToDomain(item)
	}
	return result, nil
}

// Update updates a wishlist item
func (r *WishlistItemRepository) Update(workspaceID int32, item *domain.WishlistItem) (*domain.WishlistItem, error) {
	ctx := context.Background()
	updated, err := r.queries.UpdateWishlistItem(ctx, sqlc.UpdateWishlistItemParams{
		ID:           item.ID,
		WorkspaceID:  workspaceID,
		Title:        item.Title,
		Description:  stringPtrToPgText(item.Description),
		ExternalLink: stringPtrToPgText(item.ExternalLink),
		ImageUrl:     stringPtrToPgText(item.ImageURL),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrWishlistItemNotFound
		}
		return nil, err
	}
	return sqlcWishlistItemToDomain(updated), nil
}

// Move moves an item to a different wishlist
func (r *WishlistItemRepository) Move(workspaceID int32, itemID int32, targetWishlistID int32) (*domain.WishlistItem, error) {
	ctx := context.Background()
	updated, err := r.queries.MoveWishlistItem(ctx, sqlc.MoveWishlistItemParams{
		ID:          itemID,
		WorkspaceID: workspaceID,
		WishlistID:  targetWishlistID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrWishlistItemNotFound
		}
		return nil, err
	}
	return sqlcWishlistItemToDomain(updated), nil
}

// SoftDelete marks a wishlist item as deleted
func (r *WishlistItemRepository) SoftDelete(workspaceID int32, id int32) error {
	ctx := context.Background()
	return r.queries.DeleteWishlistItem(ctx, sqlc.DeleteWishlistItemParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
}

// GetFirstItemImage gets the image URL of the first item in a wishlist (for thumbnail)
func (r *WishlistItemRepository) GetFirstItemImage(workspaceID int32, wishlistID int32) (*string, error) {
	ctx := context.Background()
	imageURL, err := r.queries.GetFirstItemImage(ctx, sqlc.GetFirstItemImageParams{
		WishlistID:  wishlistID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if !imageURL.Valid {
		return nil, nil
	}
	return &imageURL.String, nil
}

// CountByWishlist counts items in a wishlist
func (r *WishlistItemRepository) CountByWishlist(workspaceID int32, wishlistID int32) (int64, error) {
	ctx := context.Background()
	return r.queries.CountWishlistItems(ctx, sqlc.CountWishlistItemsParams{
		WishlistID:  wishlistID,
		WorkspaceID: workspaceID,
	})
}

// Helper function to convert sqlc type to domain type
func sqlcWishlistItemToDomain(item sqlc.WishlistItem) *domain.WishlistItem {
	result := &domain.WishlistItem{
		ID:         item.ID,
		WishlistID: item.WishlistID,
		Title:      item.Title,
		CreatedAt:  item.CreatedAt.Time,
		UpdatedAt:  item.UpdatedAt.Time,
	}
	if item.Description.Valid {
		result.Description = &item.Description.String
	}
	if item.ExternalLink.Valid {
		result.ExternalLink = &item.ExternalLink.String
	}
	if item.ImageUrl.Valid {
		result.ImageURL = &item.ImageUrl.String
	}
	if item.DeletedAt.Valid {
		result.DeletedAt = &item.DeletedAt.Time
	}
	return result
}

// Note: stringPtrToPgText is defined in user_repo.go and shared across the package
