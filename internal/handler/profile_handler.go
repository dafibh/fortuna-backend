package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/middleware"
	"github.com/dafibh/fortuna/fortuna-backend/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// ProfileHandler handles profile-related HTTP requests
type ProfileHandler struct {
	profileService *service.ProfileService
}

// NewProfileHandler creates a new ProfileHandler
func NewProfileHandler(profileService *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{profileService: profileService}
}

// ProfileResponse represents the profile response
type ProfileResponse struct {
	ID         string  `json:"id"`
	Email      string  `json:"email"`
	Name       *string `json:"name"`
	PictureURL *string `json:"pictureUrl"`
}

// UpdateProfileRequest represents the update profile request
type UpdateProfileRequest struct {
	Name string `json:"name"`
}

// GetProfile handles GET /profile
func (h *ProfileHandler) GetProfile(c echo.Context) error {
	auth0ID := middleware.GetAuth0ID(c)
	if auth0ID == "" {
		return NewUnauthorizedError(c, "Authentication required")
	}

	user, err := h.profileService.GetProfile(auth0ID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return NewNotFoundError(c, "User not found")
		}
		log.Error().Err(err).Str("auth0_id", auth0ID).Msg("Failed to get profile")
		return NewInternalError(c, "Failed to get profile")
	}

	return c.JSON(http.StatusOK, ProfileResponse{
		ID:         user.ID.String(),
		Email:      user.Email,
		Name:       user.Name,
		PictureURL: user.PictureURL,
	})
}

// UpdateProfile handles PUT /profile
func (h *ProfileHandler) UpdateProfile(c echo.Context) error {
	auth0ID := middleware.GetAuth0ID(c)
	if auth0ID == "" {
		return NewUnauthorizedError(c, "Authentication required")
	}

	var req UpdateProfileRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	// Validate name
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "name", Message: "Name is required"},
		})
	}
	if len(name) > 255 {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "name", Message: "Name must be 255 characters or less"},
		})
	}

	user, err := h.profileService.UpdateProfile(auth0ID, name)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return NewNotFoundError(c, "User not found")
		}
		log.Error().Err(err).Str("auth0_id", auth0ID).Msg("Failed to update profile")
		return NewInternalError(c, "Failed to update profile")
	}

	log.Info().Str("auth0_id", auth0ID).Str("name", name).Msg("Profile updated")

	return c.JSON(http.StatusOK, ProfileResponse{
		ID:         user.ID.String(),
		Email:      user.Email,
		Name:       user.Name,
		PictureURL: user.PictureURL,
	})
}
