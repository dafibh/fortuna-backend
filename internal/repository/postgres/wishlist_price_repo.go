package postgres

import (
	"context"

	"github.com/dafibh/fortuna/fortuna-backend/db/sqlc"
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WishlistPriceRepository implements domain.WishlistPriceRepository using PostgreSQL
type WishlistPriceRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewWishlistPriceRepository creates a new WishlistPriceRepository
func NewWishlistPriceRepository(pool *pgxpool.Pool) *WishlistPriceRepository {
	return &WishlistPriceRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Create creates a new price entry
func (r *WishlistPriceRepository) Create(price *domain.WishlistItemPrice) (*domain.WishlistItemPrice, error) {
	ctx := context.Background()

	priceNum, err := decimalToPgNumeric(price.Price)
	if err != nil {
		return nil, err
	}

	created, err := r.queries.CreateWishlistItemPrice(ctx, sqlc.CreateWishlistItemPriceParams{
		ItemID:       price.ItemID,
		PlatformName: price.PlatformName,
		Price:        priceNum,
		PriceDate:    timeToPgDate(price.PriceDate),
		ImageUrl:     stringPtrToPgText(price.ImageURL),
	})
	if err != nil {
		return nil, err
	}
	return sqlcPriceToDomain(created), nil
}

// GetByID retrieves a price entry by its ID within a workspace
func (r *WishlistPriceRepository) GetByID(workspaceID int32, id int32) (*domain.WishlistItemPrice, error) {
	ctx := context.Background()
	price, err := r.queries.GetWishlistItemPriceByID(ctx, sqlc.GetWishlistItemPriceByIDParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrPriceEntryNotFound
		}
		return nil, err
	}
	return sqlcPriceToDomain(price), nil
}

// ListByItem retrieves all prices for an item, ordered by platform and date
func (r *WishlistPriceRepository) ListByItem(workspaceID int32, itemID int32) ([]*domain.WishlistItemPrice, error) {
	ctx := context.Background()
	prices, err := r.queries.ListPricesByItem(ctx, sqlc.ListPricesByItemParams{
		ItemID:      itemID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.WishlistItemPrice, len(prices))
	for i, price := range prices {
		result[i] = sqlcPriceToDomain(price)
	}
	return result, nil
}

// GetCurrentPricesByItem retrieves the most recent price per platform for an item
func (r *WishlistPriceRepository) GetCurrentPricesByItem(workspaceID int32, itemID int32) ([]*domain.WishlistItemPrice, error) {
	ctx := context.Background()
	prices, err := r.queries.GetCurrentPricesByItem(ctx, sqlc.GetCurrentPricesByItemParams{
		ItemID:      itemID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.WishlistItemPrice, len(prices))
	for i, price := range prices {
		result[i] = sqlcPriceToDomain(price)
	}
	return result, nil
}

// GetBestPriceForItem retrieves the lowest current price among all platforms for an item
func (r *WishlistPriceRepository) GetBestPriceForItem(workspaceID int32, itemID int32) (*string, error) {
	ctx := context.Background()
	bestPrice, err := r.queries.GetBestPriceForItem(ctx, sqlc.GetBestPriceForItemParams{
		ItemID:      itemID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if bestPrice == "" {
		return nil, nil
	}
	return &bestPrice, nil
}

// Delete hard-deletes a price entry (for error correction)
func (r *WishlistPriceRepository) Delete(workspaceID int32, id int32) error {
	ctx := context.Background()
	return r.queries.DeleteWishlistItemPrice(ctx, sqlc.DeleteWishlistItemPriceParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
}

// Helper function to convert sqlc type to domain type
func sqlcPriceToDomain(price sqlc.WishlistItemPrice) *domain.WishlistItemPrice {
	result := &domain.WishlistItemPrice{
		ID:           price.ID,
		ItemID:       price.ItemID,
		PlatformName: price.PlatformName,
		Price:        pgNumericToDecimal(price.Price),
		PriceDate:    pgDateToTime(price.PriceDate),
		CreatedAt:    price.CreatedAt.Time,
	}
	if price.ImageUrl.Valid {
		result.ImageURL = &price.ImageUrl.String
	}
	return result
}
