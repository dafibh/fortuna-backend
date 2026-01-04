package handler

import (
	"errors"
	"net/http"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/middleware"
	"github.com/dafibh/fortuna/fortuna-backend/internal/service"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// APITokenHandler handles API token-related HTTP requests
type APITokenHandler struct {
	apiTokenService *service.APITokenService
	authService     *service.AuthService
}

// NewAPITokenHandler creates a new APITokenHandler
func NewAPITokenHandler(apiTokenService *service.APITokenService, authService *service.AuthService) *APITokenHandler {
	return &APITokenHandler{
		apiTokenService: apiTokenService,
		authService:     authService,
	}
}

// CreateAPITokenRequest represents the create token request body
type CreateAPITokenRequest struct {
	Description string `json:"description"`
}

// CreateAPIToken godoc
// @Summary Create an API token
// @Description Create a new API token for programmatic access (JWT auth only)
// @Tags api-tokens
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateAPITokenRequest true "Token creation request"
// @Success 201 {object} domain.CreateAPITokenResponse
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Router /api-tokens [post]
func (h *APITokenHandler) CreateAPIToken(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	// Get user ID from auth0 ID
	auth0ID := middleware.GetAuth0ID(c)
	if auth0ID == "" {
		return NewUnauthorizedError(c, "Authentication required")
	}

	user, err := h.authService.GetUserByAuth0ID(auth0ID)
	if err != nil {
		log.Error().Err(err).Str("auth0_id", auth0ID).Msg("Failed to get user")
		return NewUnauthorizedError(c, "User not found")
	}

	var req CreateAPITokenRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	// Validate description
	if req.Description == "" {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "description", Message: "Description is required"},
		})
	}
	if len(req.Description) > 255 {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "description", Message: "Description must be 255 characters or less"},
		})
	}

	// Create the token
	result, err := h.apiTokenService.Create(c.Request().Context(), user.ID, workspaceID, req.Description)
	if err != nil {
		if errors.Is(err, domain.ErrTooManyAPITokens) {
			return NewValidationError(c, "Maximum number of API tokens reached (10)", nil)
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to create API token")
		return NewInternalError(c, "Failed to create API token")
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Str("token_id", result.ID.String()).
		Str("description", req.Description).
		Msg("API token created")

	return c.JSON(http.StatusCreated, result)
}

// GetAPITokens godoc
// @Summary List API tokens
// @Description Get all API tokens for the authenticated workspace (JWT auth only)
// @Tags api-tokens
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} domain.APITokenResponse
// @Failure 401 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /api-tokens [get]
func (h *APITokenHandler) GetAPITokens(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	tokens, err := h.apiTokenService.GetByWorkspace(c.Request().Context(), workspaceID)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get API tokens")
		return NewInternalError(c, "Failed to get API tokens")
	}

	return c.JSON(http.StatusOK, tokens)
}

// RevokeAPIToken godoc
// @Summary Revoke an API token
// @Description Revoke/delete an API token (JWT auth only)
// @Tags api-tokens
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Token ID (UUID)"
// @Success 204 "No Content"
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Failure 404 {object} ProblemDetails
// @Router /api-tokens/{id} [delete]
func (h *APITokenHandler) RevokeAPIToken(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	tokenID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid token ID", nil)
	}

	if err := h.apiTokenService.Revoke(c.Request().Context(), workspaceID, tokenID); err != nil {
		if errors.Is(err, domain.ErrAPITokenNotFound) {
			return NewNotFoundError(c, "API token not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Str("token_id", tokenID.String()).Msg("Failed to revoke API token")
		return NewInternalError(c, "Failed to revoke API token")
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Str("token_id", tokenID.String()).
		Msg("API token revoked")

	return c.NoContent(http.StatusNoContent)
}
