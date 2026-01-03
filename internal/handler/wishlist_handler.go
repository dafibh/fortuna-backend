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

// WishlistHandler handles wishlist-related HTTP requests
type WishlistHandler struct {
	wishlistService *service.WishlistService
}

// NewWishlistHandler creates a new WishlistHandler
func NewWishlistHandler(wishlistService *service.WishlistService) *WishlistHandler {
	return &WishlistHandler{wishlistService: wishlistService}
}

// CreateWishlistRequest represents the create wishlist request body
type CreateWishlistRequest struct {
	Name string `json:"name"`
}

// UpdateWishlistRequest represents the update wishlist request body
type UpdateWishlistRequest struct {
	Name string `json:"name"`
}

// WishlistResponse represents a wishlist in API responses
type WishlistResponse struct {
	ID          int32   `json:"id"`
	WorkspaceID int32   `json:"workspaceId"`
	Name        string  `json:"name"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
	DeletedAt   *string `json:"deletedAt,omitempty"`
}

// CreateWishlist handles POST /api/v1/wishlists
func (h *WishlistHandler) CreateWishlist(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	var req CreateWishlistRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	input := service.CreateWishlistInput{
		Name: req.Name,
	}

	wishlist, err := h.wishlistService.CreateWishlist(workspaceID, input)
	if err != nil {
		if errors.Is(err, domain.ErrWishlistNameEmpty) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "name", Message: "Name is required"},
			})
		}
		if errors.Is(err, domain.ErrWishlistNameTooLong) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "name", Message: "Name must be 255 characters or less"},
			})
		}
		if errors.Is(err, domain.ErrWishlistNameExists) {
			return NewConflictError(c, "A wishlist with this name already exists")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to create wishlist")
		return NewInternalError(c, "Failed to create wishlist")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("wishlist_id", wishlist.ID).Str("name", wishlist.Name).Msg("Wishlist created")

	return c.JSON(http.StatusCreated, toWishlistResponse(wishlist))
}

// GetWishlists handles GET /api/v1/wishlists
func (h *WishlistHandler) GetWishlists(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	wishlists, err := h.wishlistService.GetWishlists(workspaceID)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get wishlists")
		return NewInternalError(c, "Failed to get wishlists")
	}

	response := make([]WishlistResponse, len(wishlists))
	for i, wishlist := range wishlists {
		response[i] = toWishlistResponse(wishlist)
	}

	return c.JSON(http.StatusOK, response)
}

// GetWishlist handles GET /api/v1/wishlists/:id
func (h *WishlistHandler) GetWishlist(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid wishlist ID", nil)
	}

	wishlist, err := h.wishlistService.GetWishlistByID(workspaceID, int32(id))
	if err != nil {
		if errors.Is(err, domain.ErrWishlistNotFound) {
			return NewNotFoundError(c, "Wishlist not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("wishlist_id", id).Msg("Failed to get wishlist")
		return NewInternalError(c, "Failed to get wishlist")
	}

	return c.JSON(http.StatusOK, toWishlistResponse(wishlist))
}

// UpdateWishlist handles PUT /api/v1/wishlists/:id
func (h *WishlistHandler) UpdateWishlist(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid wishlist ID", nil)
	}

	var req UpdateWishlistRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	input := service.UpdateWishlistInput{
		Name: req.Name,
	}

	wishlist, err := h.wishlistService.UpdateWishlist(workspaceID, int32(id), input)
	if err != nil {
		if errors.Is(err, domain.ErrWishlistNotFound) {
			return NewNotFoundError(c, "Wishlist not found")
		}
		if errors.Is(err, domain.ErrWishlistNameEmpty) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "name", Message: "Name is required"},
			})
		}
		if errors.Is(err, domain.ErrWishlistNameTooLong) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "name", Message: "Name must be 255 characters or less"},
			})
		}
		if errors.Is(err, domain.ErrWishlistNameExists) {
			return NewConflictError(c, "A wishlist with this name already exists")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("wishlist_id", id).Msg("Failed to update wishlist")
		return NewInternalError(c, "Failed to update wishlist")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("wishlist_id", wishlist.ID).Str("name", wishlist.Name).Msg("Wishlist updated")
	return c.JSON(http.StatusOK, toWishlistResponse(wishlist))
}

// DeleteWishlist handles DELETE /api/v1/wishlists/:id
func (h *WishlistHandler) DeleteWishlist(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid wishlist ID", nil)
	}

	if err := h.wishlistService.DeleteWishlist(workspaceID, int32(id)); err != nil {
		if errors.Is(err, domain.ErrWishlistNotFound) {
			return NewNotFoundError(c, "Wishlist not found")
		}
		if errors.Is(err, domain.ErrWishlistHasItems) {
			return NewConflictError(c, "Cannot delete wishlist with items")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("wishlist_id", id).Msg("Failed to delete wishlist")
		return NewInternalError(c, "Failed to delete wishlist")
	}

	log.Info().Int32("workspace_id", workspaceID).Int("wishlist_id", id).Msg("Wishlist deleted (soft)")
	return c.NoContent(http.StatusNoContent)
}

// Helper function to convert domain.Wishlist to WishlistResponse
func toWishlistResponse(wishlist *domain.Wishlist) WishlistResponse {
	resp := WishlistResponse{
		ID:          wishlist.ID,
		WorkspaceID: wishlist.WorkspaceID,
		Name:        wishlist.Name,
		CreatedAt:   wishlist.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   wishlist.UpdatedAt.Format(time.RFC3339),
	}
	if wishlist.DeletedAt != nil {
		deletedAt := wishlist.DeletedAt.Format(time.RFC3339)
		resp.DeletedAt = &deletedAt
	}
	return resp
}
