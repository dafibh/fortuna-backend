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
)

// WishlistItemHandler handles wishlist item-related HTTP requests
type WishlistItemHandler struct {
	itemService *service.WishlistItemService
}

// NewWishlistItemHandler creates a new WishlistItemHandler
func NewWishlistItemHandler(itemService *service.WishlistItemService) *WishlistItemHandler {
	return &WishlistItemHandler{itemService: itemService}
}

// CreateWishlistItemRequest represents the create item request body
type CreateWishlistItemRequest struct {
	Title        string  `json:"title"`
	Description  *string `json:"description"`
	ExternalLink *string `json:"externalLink"`
	ImageURL     *string `json:"imageUrl"`
}

// UpdateWishlistItemRequest represents the update item request body
type UpdateWishlistItemRequest struct {
	Title        string  `json:"title"`
	Description  *string `json:"description"`
	ExternalLink *string `json:"externalLink"`
	ImageURL     *string `json:"imageUrl"`
}

// MoveWishlistItemRequest represents the move item request body
type MoveWishlistItemRequest struct {
	TargetWishlistID int32 `json:"targetWishlistId"`
}

// WishlistItemResponse represents a wishlist item in API responses
type WishlistItemResponse struct {
	ID           int32   `json:"id"`
	WishlistID   int32   `json:"wishlistId"`
	Title        string  `json:"title"`
	Description  *string `json:"description,omitempty"`
	ExternalLink *string `json:"externalLink,omitempty"`
	ImageURL     *string `json:"imageUrl,omitempty"`
	BestPrice    *string `json:"bestPrice,omitempty"`
	NoteCount    int     `json:"noteCount"`
	CreatedAt    string  `json:"createdAt"`
	UpdatedAt    string  `json:"updatedAt"`
}

// CreateItem handles POST /api/v1/wishlists/:id/items
func (h *WishlistItemHandler) CreateItem(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	wishlistID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid wishlist ID", nil)
	}

	var req CreateWishlistItemRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	input := service.CreateWishlistItemInput{
		Title:        req.Title,
		Description:  req.Description,
		ExternalLink: req.ExternalLink,
		ImageURL:     req.ImageURL,
	}

	item, err := h.itemService.CreateItem(workspaceID, int32(wishlistID), input)
	if err != nil {
		if errors.Is(err, domain.ErrWishlistNotFound) {
			return NewNotFoundError(c, "Wishlist not found")
		}
		if errors.Is(err, domain.ErrWishlistItemTitleEmpty) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "title", Message: "Title is required"},
			})
		}
		if errors.Is(err, domain.ErrWishlistItemTitleLong) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "title", Message: "Title must be 255 characters or less"},
			})
		}
		if errors.Is(err, domain.ErrWishlistItemInvalidURL) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "externalLink", Message: "Must be a valid URL"},
			})
		}
		if errors.Is(err, domain.ErrWishlistItemInvalidImageURL) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "imageUrl", Message: "Must be a valid URL"},
			})
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("wishlist_id", wishlistID).Msg("Failed to create wishlist item")
		return NewInternalError(c, "Failed to create wishlist item")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("item_id", item.ID).Str("title", item.Title).Msg("Wishlist item created")

	return c.JSON(http.StatusCreated, toWishlistItemResponse(item))
}

// ListItems handles GET /api/v1/wishlists/:id/items
func (h *WishlistItemHandler) ListItems(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	wishlistID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid wishlist ID", nil)
	}

	items, err := h.itemService.GetItemsByWishlistWithStats(workspaceID, int32(wishlistID))
	if err != nil {
		if errors.Is(err, domain.ErrWishlistNotFound) {
			return NewNotFoundError(c, "Wishlist not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("wishlist_id", wishlistID).Msg("Failed to list wishlist items")
		return NewInternalError(c, "Failed to list wishlist items")
	}

	response := make([]WishlistItemResponse, len(items))
	for i, item := range items {
		response[i] = toWishlistItemWithStatsResponse(item)
	}

	return c.JSON(http.StatusOK, response)
}

// GetItem handles GET /api/v1/wishlist-items/:id
func (h *WishlistItemHandler) GetItem(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid item ID", nil)
	}

	item, err := h.itemService.GetItemByID(workspaceID, int32(id))
	if err != nil {
		if errors.Is(err, domain.ErrWishlistItemNotFound) {
			return NewNotFoundError(c, "Wishlist item not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("item_id", id).Msg("Failed to get wishlist item")
		return NewInternalError(c, "Failed to get wishlist item")
	}

	return c.JSON(http.StatusOK, toWishlistItemResponse(item))
}

// UpdateItem handles PUT /api/v1/wishlist-items/:id
func (h *WishlistItemHandler) UpdateItem(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid item ID", nil)
	}

	var req UpdateWishlistItemRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	input := service.UpdateWishlistItemInput{
		Title:        req.Title,
		Description:  req.Description,
		ExternalLink: req.ExternalLink,
		ImageURL:     req.ImageURL,
	}

	item, err := h.itemService.UpdateItem(workspaceID, int32(id), input)
	if err != nil {
		if errors.Is(err, domain.ErrWishlistItemNotFound) {
			return NewNotFoundError(c, "Wishlist item not found")
		}
		if errors.Is(err, domain.ErrWishlistItemTitleEmpty) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "title", Message: "Title is required"},
			})
		}
		if errors.Is(err, domain.ErrWishlistItemTitleLong) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "title", Message: "Title must be 255 characters or less"},
			})
		}
		if errors.Is(err, domain.ErrWishlistItemInvalidURL) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "externalLink", Message: "Must be a valid URL"},
			})
		}
		if errors.Is(err, domain.ErrWishlistItemInvalidImageURL) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "imageUrl", Message: "Must be a valid URL"},
			})
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("item_id", id).Msg("Failed to update wishlist item")
		return NewInternalError(c, "Failed to update wishlist item")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("item_id", item.ID).Msg("Wishlist item updated")

	return c.JSON(http.StatusOK, toWishlistItemResponse(item))
}

// MoveItem handles PATCH /api/v1/wishlist-items/:id/move
func (h *WishlistItemHandler) MoveItem(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid item ID", nil)
	}

	var req MoveWishlistItemRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	if req.TargetWishlistID == 0 {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "targetWishlistId", Message: "Target wishlist ID is required"},
		})
	}

	item, err := h.itemService.MoveItem(workspaceID, int32(id), req.TargetWishlistID)
	if err != nil {
		if errors.Is(err, domain.ErrWishlistItemNotFound) {
			return NewNotFoundError(c, "Wishlist item not found")
		}
		if errors.Is(err, domain.ErrWishlistNotFound) {
			return NewNotFoundError(c, "Target wishlist not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("item_id", id).Msg("Failed to move wishlist item")
		return NewInternalError(c, "Failed to move wishlist item")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("item_id", item.ID).Int32("target_wishlist", req.TargetWishlistID).Msg("Wishlist item moved")

	return c.JSON(http.StatusOK, toWishlistItemResponse(item))
}

// DeleteItem handles DELETE /api/v1/wishlist-items/:id
func (h *WishlistItemHandler) DeleteItem(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid item ID", nil)
	}

	if err := h.itemService.DeleteItem(workspaceID, int32(id)); err != nil {
		if errors.Is(err, domain.ErrWishlistItemNotFound) {
			return NewNotFoundError(c, "Wishlist item not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("item_id", id).Msg("Failed to delete wishlist item")
		return NewInternalError(c, "Failed to delete wishlist item")
	}

	log.Info().Int32("workspace_id", workspaceID).Int("item_id", id).Msg("Wishlist item deleted (soft)")

	return c.NoContent(http.StatusNoContent)
}

// Helper function to convert domain.WishlistItem to WishlistItemResponse
func toWishlistItemResponse(item *domain.WishlistItem) WishlistItemResponse {
	return WishlistItemResponse{
		ID:           item.ID,
		WishlistID:   item.WishlistID,
		Title:        item.Title,
		Description:  item.Description,
		ExternalLink: item.ExternalLink,
		ImageURL:     item.ImageURL,
		BestPrice:    nil,
		NoteCount:    0,
		CreatedAt:    item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    item.UpdatedAt.Format(time.RFC3339),
	}
}

// Helper function to convert domain.WishlistItemWithStats to WishlistItemResponse
func toWishlistItemWithStatsResponse(item *domain.WishlistItemWithStats) WishlistItemResponse {
	return WishlistItemResponse{
		ID:           item.ID,
		WishlistID:   item.WishlistID,
		Title:        item.Title,
		Description:  item.Description,
		ExternalLink: item.ExternalLink,
		ImageURL:     item.ImageURL,
		BestPrice:    item.BestPrice,
		NoteCount:    item.NoteCount,
		CreatedAt:    item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    item.UpdatedAt.Format(time.RFC3339),
	}
}
