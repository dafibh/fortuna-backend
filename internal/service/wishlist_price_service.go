package service

import (
	"net/url"
	"strings"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/shopspring/decimal"
)

// WishlistPriceService handles wishlist item price business logic
type WishlistPriceService struct {
	priceRepo domain.WishlistPriceRepository
	itemRepo  domain.WishlistItemRepository
}

// NewWishlistPriceService creates a new WishlistPriceService
func NewWishlistPriceService(priceRepo domain.WishlistPriceRepository, itemRepo domain.WishlistItemRepository) *WishlistPriceService {
	return &WishlistPriceService{
		priceRepo: priceRepo,
		itemRepo:  itemRepo,
	}
}

// CreatePriceInput contains input for creating a price entry
type CreatePriceInput struct {
	PlatformName string
	Price        decimal.Decimal
	PriceDate    *time.Time
	ImageURL     *string
}

// CreatePrice creates a new price entry for an item
func (s *WishlistPriceService) CreatePrice(workspaceID int32, itemID int32, input CreatePriceInput) (*domain.WishlistItemPrice, error) {
	// Verify item exists and belongs to workspace
	_, err := s.itemRepo.GetByID(workspaceID, itemID)
	if err != nil {
		return nil, err
	}

	// Validate platform name
	platformName := strings.TrimSpace(input.PlatformName)
	if platformName == "" {
		return nil, domain.ErrPricePlatformEmpty
	}
	if len(platformName) > 100 {
		return nil, domain.ErrPricePlatformLong
	}

	// Validate price
	if input.Price.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrPriceNotPositive
	}

	// Validate and set price date (default to today)
	priceDate := time.Now()
	if input.PriceDate != nil {
		priceDate = *input.PriceDate
		// Don't allow future dates (with 1 day tolerance for timezone issues)
		if priceDate.After(time.Now().AddDate(0, 0, 1)) {
			return nil, domain.ErrPriceDateFuture
		}
	}

	// Validate image URL if provided
	if input.ImageURL != nil && *input.ImageURL != "" {
		imageURL := strings.TrimSpace(*input.ImageURL)
		if _, err := url.ParseRequestURI(imageURL); err != nil {
			return nil, domain.ErrPriceInvalidImageURL
		}
		input.ImageURL = &imageURL
	}

	price := &domain.WishlistItemPrice{
		ItemID:       itemID,
		PlatformName: platformName,
		Price:        input.Price,
		PriceDate:    priceDate,
		ImageURL:     input.ImageURL,
	}

	return s.priceRepo.Create(price)
}

// GetPriceByID retrieves a price entry by ID
func (s *WishlistPriceService) GetPriceByID(workspaceID int32, id int32) (*domain.WishlistItemPrice, error) {
	return s.priceRepo.GetByID(workspaceID, id)
}

// ListPricesByItem retrieves all prices for an item (full history, ordered by platform and date)
func (s *WishlistPriceService) ListPricesByItem(workspaceID int32, itemID int32) ([]*domain.WishlistItemPrice, error) {
	// Verify item exists and belongs to workspace
	_, err := s.itemRepo.GetByID(workspaceID, itemID)
	if err != nil {
		return nil, err
	}

	return s.priceRepo.ListByItem(workspaceID, itemID)
}

// GetCurrentPricesByItem retrieves the most recent price per platform for an item
func (s *WishlistPriceService) GetCurrentPricesByItem(workspaceID int32, itemID int32) ([]*domain.WishlistItemPrice, error) {
	// Verify item exists and belongs to workspace
	_, err := s.itemRepo.GetByID(workspaceID, itemID)
	if err != nil {
		return nil, err
	}

	return s.priceRepo.GetCurrentPricesByItem(workspaceID, itemID)
}

// GetPricesGroupedByPlatform retrieves prices grouped by platform with current price and history
func (s *WishlistPriceService) GetPricesGroupedByPlatform(workspaceID int32, itemID int32) ([]*domain.PriceByPlatform, error) {
	// Verify item exists and belongs to workspace
	_, err := s.itemRepo.GetByID(workspaceID, itemID)
	if err != nil {
		return nil, err
	}

	// Get all prices (ordered by platform, date DESC)
	allPrices, err := s.priceRepo.ListByItem(workspaceID, itemID)
	if err != nil {
		return nil, err
	}

	if len(allPrices) == 0 {
		return []*domain.PriceByPlatform{}, nil
	}

	// Group by platform
	platformMap := make(map[string]*domain.PriceByPlatform)
	var platformOrder []string
	var lowestPrice decimal.Decimal
	var lowestPlatform string
	firstPrice := true

	for _, price := range allPrices {
		group, exists := platformMap[price.PlatformName]
		if !exists {
			group = &domain.PriceByPlatform{
				PlatformName:  price.PlatformName,
				CurrentPrice:  price.Price.String(),
				PriceHistory:  []*domain.WishlistItemPrice{},
				IsLowestPrice: false,
			}
			platformMap[price.PlatformName] = group
			platformOrder = append(platformOrder, price.PlatformName)

			// Track lowest current price (first price per platform is the current one)
			if firstPrice || price.Price.LessThan(lowestPrice) {
				lowestPrice = price.Price
				lowestPlatform = price.PlatformName
				firstPrice = false
			}
		}
		group.PriceHistory = append(group.PriceHistory, price)
	}

	// Mark the lowest price platform
	if lowestPlatform != "" {
		platformMap[lowestPlatform].IsLowestPrice = true
	}

	// Convert to slice maintaining order
	result := make([]*domain.PriceByPlatform, len(platformOrder))
	for i, name := range platformOrder {
		result[i] = platformMap[name]
	}

	return result, nil
}

// GetBestPriceForItem retrieves the lowest current price among all platforms for an item
func (s *WishlistPriceService) GetBestPriceForItem(workspaceID int32, itemID int32) (*string, error) {
	return s.priceRepo.GetBestPriceForItem(workspaceID, itemID)
}

// DeletePrice hard-deletes a price entry (for error correction)
func (s *WishlistPriceService) DeletePrice(workspaceID int32, id int32) error {
	// Verify price exists and belongs to workspace
	_, err := s.priceRepo.GetByID(workspaceID, id)
	if err != nil {
		return err
	}

	return s.priceRepo.Delete(workspaceID, id)
}
