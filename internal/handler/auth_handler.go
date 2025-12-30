package handler

import (
	"net/http"

	"github.com/dafibh/fortuna/fortuna-backend/internal/middleware"
	"github.com/dafibh/fortuna/fortuna-backend/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// AuthCallbackResponse represents the response from the auth callback
type AuthCallbackResponse struct {
	User      UserResponse      `json:"user"`
	Workspace WorkspaceResponse `json:"workspace"`
	IsNewUser bool              `json:"isNewUser"`
}

// UserResponse represents a user in API responses
type UserResponse struct {
	ID         string  `json:"id"`
	Email      string  `json:"email"`
	Name       *string `json:"name"`
	PictureURL *string `json:"pictureUrl"`
}

// WorkspaceResponse represents a workspace in API responses
type WorkspaceResponse struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

// Callback handles the Auth0 callback after successful authentication
// This endpoint is called by the frontend after receiving the Auth0 token
// POST /auth/callback
func (h *AuthHandler) Callback(c echo.Context) error {
	// Get Auth0 ID from the validated JWT (set by auth middleware)
	auth0ID := middleware.GetAuth0ID(c)
	if auth0ID == "" {
		log.Error().Msg("No Auth0 ID in context - middleware may not be configured")
		return NewUnauthorizedError(c, "Authentication required")
	}

	// Get custom claims for additional user info
	customClaims := middleware.GetCustomClaims(c)
	var email, name, picture string
	if customClaims != nil {
		email = customClaims.Email
		name = customClaims.Name
		picture = customClaims.Picture
	}

	// Email is required for user creation
	if email == "" {
		log.Error().Str("auth0_id", auth0ID).Msg("No email in JWT claims")
		return NewValidationError(c, "Email is required for authentication", []ValidationError{
			{Field: "email", Message: "Email claim is missing from token"},
		})
	}

	// Authenticate/register the user
	var namePtr, picturePtr *string
	if name != "" {
		namePtr = &name
	}
	if picture != "" {
		picturePtr = &picture
	}

	result, err := h.authService.AuthenticateUser(auth0ID, email, namePtr, picturePtr)
	if err != nil {
		log.Error().Err(err).Str("auth0_id", auth0ID).Msg("Failed to authenticate user")
		return NewInternalError(c, "Failed to authenticate user")
	}

	response := AuthCallbackResponse{
		User: UserResponse{
			ID:         result.User.ID.String(),
			Email:      result.User.Email,
			Name:       result.User.Name,
			PictureURL: result.User.PictureURL,
		},
		Workspace: WorkspaceResponse{
			ID:   result.Workspace.ID,
			Name: result.Workspace.Name,
		},
		IsNewUser: result.IsNewUser,
	}

	return c.JSON(http.StatusOK, response)
}

// Me returns the current authenticated user's information
// GET /auth/me
func (h *AuthHandler) Me(c echo.Context) error {
	auth0ID := middleware.GetAuth0ID(c)
	if auth0ID == "" {
		return NewUnauthorizedError(c, "Authentication required")
	}

	// Use workspace ID from context (already fetched by middleware)
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		log.Error().Str("auth0_id", auth0ID).Msg("No workspace ID in context")
		return NewInternalError(c, "Workspace not available")
	}

	user, err := h.authService.GetUserByAuth0ID(auth0ID)
	if err != nil {
		log.Error().Err(err).Str("auth0_id", auth0ID).Msg("Failed to get user")
		return NewNotFoundError(c, "User not found")
	}

	workspace, err := h.authService.GetWorkspaceByID(workspaceID)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get workspace")
		return NewInternalError(c, "Failed to get workspace")
	}

	response := AuthCallbackResponse{
		User: UserResponse{
			ID:         user.ID.String(),
			Email:      user.Email,
			Name:       user.Name,
			PictureURL: user.PictureURL,
		},
		Workspace: WorkspaceResponse{
			ID:   workspace.ID,
			Name: workspace.Name,
		},
		IsNewUser: false,
	}

	return c.JSON(http.StatusOK, response)
}

// LogoutResponse represents the response from logout
type LogoutResponse struct {
	Message string `json:"message"`
}

// Logout handles user logout
// POST /auth/logout
func (h *AuthHandler) Logout(c echo.Context) error {
	auth0ID := middleware.GetAuth0ID(c)
	if auth0ID == "" {
		return NewUnauthorizedError(c, "Authentication required")
	}

	// Log the logout event (useful for audit)
	log.Info().Str("auth0_id", auth0ID).Msg("User logged out")

	// Return success - Auth0 handles actual session termination
	return c.JSON(http.StatusOK, LogoutResponse{
		Message: "Logged out successfully",
	})
}
