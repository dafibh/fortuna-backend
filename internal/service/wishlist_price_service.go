package service

import (
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
	ImagePath    *string
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

	price := &domain.WishlistItemPrice{
		ItemID:       itemID,
		PlatformName: platformName,
		Price:        input.Price,
		PriceDate:    priceDate,
		ImagePath:    input.ImagePath,
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

// GetPricesGroupedByPlatform retrieves prices grouped by platform with current price, history, and price change
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
				PlatformName:     price.PlatformName,
				CurrentPrice:     price.Price.String(),
				CurrentImagePath: price.ImagePath, // First (current) entry's image
				PriceHistory:     []*domain.WishlistItemPrice{},
				IsLowestPrice:    false,
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

	// Calculate price change for each platform (current vs previous)
	for _, group := range platformMap {
		if len(group.PriceHistory) >= 2 {
			currentPrice := group.PriceHistory[0].Price
			previousPrice := group.PriceHistory[1].Price
			prevPriceStr := previousPrice.String()
			group.PreviousPrice = &prevPriceStr
			group.PriceChange = CalculatePriceChange(currentPrice, previousPrice)
		}
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

// GetPlatformHistory retrieves all price history for a specific platform on an item
func (s *WishlistPriceService) GetPlatformHistory(workspaceID int32, itemID int32, platformName string) ([]*domain.WishlistItemPrice, error) {
	// Verify item exists and belongs to workspace
	_, err := s.itemRepo.GetByID(workspaceID, itemID)
	if err != nil {
		return nil, err
	}

	return s.priceRepo.GetPriceHistoryByPlatform(workspaceID, itemID, platformName)
}

// CalculatePriceChange calculates the change between current and previous price
func CalculatePriceChange(current, previous decimal.Decimal) *domain.PriceChange {
	if previous.IsZero() {
		return nil // No previous price to compare
	}

	diff := current.Sub(previous)
	percent := diff.Div(previous).Mul(decimal.NewFromInt(100))

	direction := "unchanged"
	if diff.IsPositive() {
		direction = "up"
	} else if diff.IsNegative() {
		direction = "down"
	}

	return &domain.PriceChange{
		Amount:    diff.StringFixed(2),
		Percent:   percent.StringFixed(1),
		Direction: direction,
	}
}
