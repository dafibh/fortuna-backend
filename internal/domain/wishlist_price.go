package domain

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

var (
	ErrPriceEntryNotFound   = errors.New("price entry not found")
	ErrPricePlatformEmpty   = errors.New("platform name is required")
	ErrPricePlatformLong    = errors.New("platform name must be 100 characters or less")
	ErrPriceNotPositive     = errors.New("price must be greater than zero")
	ErrPriceDateFuture      = errors.New("price date cannot be in the future")
)

// WishlistItemPrice represents a price entry for a wishlist item
// Price entries are immutable (append-only like a ledger)
type WishlistItemPrice struct {
	ID           int32           `json:"id"`
	ItemID       int32           `json:"itemId"`
	PlatformName string          `json:"platformName"`
	Price        decimal.Decimal `json:"price"`
	PriceDate    time.Time       `json:"priceDate"`
	ImagePath    *string         `json:"imagePath,omitempty"`
	CreatedAt    time.Time       `json:"createdAt"`
}

// PriceChange represents the change between current and previous price
type PriceChange struct {
	Amount    string `json:"amount"`    // e.g., "-50.00" or "+25.00"
	Percent   string `json:"percent"`   // e.g., "-10.5" or "+5.2"
	Direction string `json:"direction"` // "up", "down", or "unchanged"
}

// PriceByPlatform groups price entries by platform for display
type PriceByPlatform struct {
	PlatformName     string               `json:"platformName"`
	CurrentPrice     string               `json:"currentPrice"`
	PreviousPrice    *string              `json:"previousPrice,omitempty"`
	PriceChange      *PriceChange         `json:"priceChange,omitempty"`
	CurrentImagePath *string              `json:"currentImagePath,omitempty"`
	PriceHistory     []*WishlistItemPrice `json:"priceHistory"`
	IsLowestPrice    bool                 `json:"isLowestPrice"`
}

// Validate validates a price entry
func (p *WishlistItemPrice) Validate() error {
	if p.PlatformName == "" {
		return ErrPricePlatformEmpty
	}
	if len(p.PlatformName) > 100 {
		return ErrPricePlatformLong
	}
	if p.Price.LessThanOrEqual(decimal.Zero) {
		return ErrPriceNotPositive
	}
	if p.PriceDate.After(time.Now().AddDate(0, 0, 1)) {
		return ErrPriceDateFuture
	}
	// ImagePath is an S3 object path, not a URL - no URL validation needed
	return nil
}

// WishlistPriceRepository defines the interface for wishlist price data access
type WishlistPriceRepository interface {
	Create(price *WishlistItemPrice) (*WishlistItemPrice, error)
	GetByID(workspaceID int32, id int32) (*WishlistItemPrice, error)
	ListByItem(workspaceID int32, itemID int32) ([]*WishlistItemPrice, error)
	GetCurrentPricesByItem(workspaceID int32, itemID int32) ([]*WishlistItemPrice, error)
	GetBestPriceForItem(workspaceID int32, itemID int32) (*string, error)
	GetPriceHistoryByPlatform(workspaceID int32, itemID int32, platformName string) ([]*WishlistItemPrice, error)
	Delete(workspaceID int32, id int32) error
}
