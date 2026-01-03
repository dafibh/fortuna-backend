package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/middleware"
	"github.com/dafibh/fortuna/fortuna-backend/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

// WishlistPriceHandler handles wishlist price-related HTTP requests
type WishlistPriceHandler struct {
	priceService *service.WishlistPriceService
}

// NewWishlistPriceHandler creates a new WishlistPriceHandler
func NewWishlistPriceHandler(priceService *service.WishlistPriceService) *WishlistPriceHandler {
	return &WishlistPriceHandler{priceService: priceService}
}

// CreatePriceRequest represents the create price request body
type CreatePriceRequest struct {
	PlatformName string  `json:"platformName"`
	Price        string  `json:"price"`
	PriceDate    *string `json:"priceDate"` // Optional, defaults to today (YYYY-MM-DD)
	ImageURL     *string `json:"imageUrl"`
}

// PriceEntryResponse represents a price entry in API responses
type PriceEntryResponse struct {
	ID           int32   `json:"id"`
	ItemID       int32   `json:"itemId"`
	PlatformName string  `json:"platformName"`
	Price        string  `json:"price"`
	PriceDate    string  `json:"priceDate"` // YYYY-MM-DD
	ImageURL     *string `json:"imageUrl,omitempty"`
	CreatedAt    string  `json:"createdAt"`
}

// PriceChangeResponse represents price change between current and previous
type PriceChangeResponse struct {
	Amount    string `json:"amount"`
	Percent   string `json:"percent"`
	Direction string `json:"direction"` // "up", "down", or "unchanged"
}

// PlatformPriceGroupResponse represents prices grouped by platform
type PlatformPriceGroupResponse struct {
	PlatformName    string               `json:"platformName"`
	CurrentPrice    string               `json:"currentPrice"`
	PreviousPrice   *string              `json:"previousPrice,omitempty"`
	PriceChange     *PriceChangeResponse `json:"priceChange,omitempty"`
	CurrentImageURL *string              `json:"currentImageUrl,omitempty"`
	PriceHistory    []PriceEntryResponse `json:"priceHistory"`
	IsLowestPrice   bool                 `json:"isLowestPrice"`
}

// CreatePrice handles POST /api/v1/wishlist-items/:id/prices
func (h *WishlistPriceHandler) CreatePrice(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	itemID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid item ID", nil)
	}

	var req CreatePriceRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	// Parse price as decimal
	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "price", Message: "Invalid price format"},
		})
	}

	// Parse price date if provided
	var priceDate *time.Time
	if req.PriceDate != nil && *req.PriceDate != "" {
		parsed, err := time.Parse("2006-01-02", *req.PriceDate)
		if err != nil {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "priceDate", Message: "Invalid date format (expected YYYY-MM-DD)"},
			})
		}
		priceDate = &parsed
	}

	input := service.CreatePriceInput{
		PlatformName: req.PlatformName,
		Price:        price,
		PriceDate:    priceDate,
		ImageURL:     req.ImageURL,
	}

	priceEntry, err := h.priceService.CreatePrice(workspaceID, int32(itemID), input)
	if err != nil {
		if errors.Is(err, domain.ErrWishlistItemNotFound) {
			return NewNotFoundError(c, "Wishlist item not found")
		}
		if errors.Is(err, domain.ErrPricePlatformEmpty) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "platformName", Message: "Platform name is required"},
			})
		}
		if errors.Is(err, domain.ErrPricePlatformLong) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "platformName", Message: "Platform name must be 100 characters or less"},
			})
		}
		if errors.Is(err, domain.ErrPriceNotPositive) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "price", Message: "Price must be greater than zero"},
			})
		}
		if errors.Is(err, domain.ErrPriceDateFuture) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "priceDate", Message: "Price date cannot be in the future"},
			})
		}
		if errors.Is(err, domain.ErrPriceInvalidImageURL) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "imageUrl", Message: "Must be a valid URL"},
			})
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("item_id", itemID).Msg("Failed to create price entry")
		return NewInternalError(c, "Failed to create price entry")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("price_id", priceEntry.ID).Str("platform", priceEntry.PlatformName).Msg("Price entry created")

	return c.JSON(http.StatusCreated, toPriceEntryResponse(priceEntry))
}

// ListPrices handles GET /api/v1/wishlist-items/:id/prices
func (h *WishlistPriceHandler) ListPrices(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	itemID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid item ID", nil)
	}

	groups, err := h.priceService.GetPricesGroupedByPlatform(workspaceID, int32(itemID))
	if err != nil {
		if errors.Is(err, domain.ErrWishlistItemNotFound) {
			return NewNotFoundError(c, "Wishlist item not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("item_id", itemID).Msg("Failed to list prices")
		return NewInternalError(c, "Failed to list prices")
	}

	response := make([]PlatformPriceGroupResponse, len(groups))
	for i, group := range groups {
		response[i] = toPlatformPriceGroupResponse(group)
	}

	return c.JSON(http.StatusOK, response)
}

// GetPlatformHistory handles GET /api/v1/wishlist-items/:id/prices/:platform
func (h *WishlistPriceHandler) GetPlatformHistory(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	itemID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid item ID", nil)
	}

	platformName := c.Param("platform")
	if platformName == "" {
		return NewValidationError(c, "Platform name is required", nil)
	}

	prices, err := h.priceService.GetPlatformHistory(workspaceID, int32(itemID), platformName)
	if err != nil {
		if errors.Is(err, domain.ErrWishlistItemNotFound) {
			return NewNotFoundError(c, "Wishlist item not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("item_id", itemID).Str("platform", platformName).Msg("Failed to get platform history")
		return NewInternalError(c, "Failed to get platform history")
	}

	response := make([]PriceEntryResponse, len(prices))
	for i, price := range prices {
		response[i] = toPriceEntryResponse(price)
	}

	return c.JSON(http.StatusOK, response)
}

// DeletePrice handles DELETE /api/v1/wishlist-item-prices/:id
func (h *WishlistPriceHandler) DeletePrice(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid price ID", nil)
	}

	if err := h.priceService.DeletePrice(workspaceID, int32(id)); err != nil {
		if errors.Is(err, domain.ErrPriceEntryNotFound) {
			return NewNotFoundError(c, "Price entry not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("price_id", id).Msg("Failed to delete price entry")
		return NewInternalError(c, "Failed to delete price entry")
	}

	log.Info().Int32("workspace_id", workspaceID).Int("price_id", id).Msg("Price entry deleted")

	return c.NoContent(http.StatusNoContent)
}

// Helper function to convert domain.WishlistItemPrice to PriceEntryResponse
func toPriceEntryResponse(price *domain.WishlistItemPrice) PriceEntryResponse {
	return PriceEntryResponse{
		ID:           price.ID,
		ItemID:       price.ItemID,
		PlatformName: price.PlatformName,
		Price:        price.Price.String(),
		PriceDate:    price.PriceDate.Format("2006-01-02"),
		ImageURL:     price.ImageURL,
		CreatedAt:    price.CreatedAt.Format(time.RFC3339),
	}
}

// Helper function to convert domain.PriceByPlatform to PlatformPriceGroupResponse
func toPlatformPriceGroupResponse(group *domain.PriceByPlatform) PlatformPriceGroupResponse {
	history := make([]PriceEntryResponse, len(group.PriceHistory))
	for i, price := range group.PriceHistory {
		history[i] = toPriceEntryResponse(price)
	}

	var priceChange *PriceChangeResponse
	if group.PriceChange != nil {
		priceChange = &PriceChangeResponse{
			Amount:    group.PriceChange.Amount,
			Percent:   group.PriceChange.Percent,
			Direction: group.PriceChange.Direction,
		}
	}

	return PlatformPriceGroupResponse{
		PlatformName:    group.PlatformName,
		CurrentPrice:    group.CurrentPrice,
		PreviousPrice:   group.PreviousPrice,
		PriceChange:     priceChange,
		CurrentImageURL: group.CurrentImageURL,
		PriceHistory:    history,
		IsLowestPrice:   group.IsLowestPrice,
	}
}
