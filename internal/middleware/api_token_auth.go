package middleware

import (
	"context"
	"strings"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

const (
	// APITokenIDKey is the context key for the API token ID
	APITokenIDKey contextKey = "api_token_id"
	// UserIDKey is the context key for the user ID (from API token)
	UserIDKey contextKey = "user_id"
	// IsAPITokenAuthKey is the context key indicating API token authentication
	IsAPITokenAuthKey contextKey = "is_api_token_auth"
)

// APITokenValidator provides API token validation
type APITokenValidator interface {
	ValidateToken(ctx context.Context, token string) (*domain.APIToken, error)
}

// APITokenAuthMiddleware provides API token authentication middleware
type APITokenAuthMiddleware struct {
	validator APITokenValidator
}

// NewAPITokenAuthMiddleware creates a new APITokenAuthMiddleware
func NewAPITokenAuthMiddleware(validator APITokenValidator) *APITokenAuthMiddleware {
	return &APITokenAuthMiddleware{validator: validator}
}

// Authenticate returns an Echo middleware that validates API tokens
func (m *APITokenAuthMiddleware) Authenticate() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return unauthorizedError(c, "Missing authorization header")
			}

			// Check Bearer prefix
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return unauthorizedError(c, "Invalid authorization header format")
			}

			token := parts[1]

			// Validate token format - must start with "fort_"
			if !strings.HasPrefix(token, "fort_") {
				return unauthorizedError(c, "Invalid token format")
			}

			// Validate the token
			apiToken, err := m.validator.ValidateToken(c.Request().Context(), token)
			if err != nil {
				if err == domain.ErrAPITokenNotFound {
					log.Debug().Msg("API token not found or revoked")
					return unauthorizedError(c, "Invalid or expired API token")
				}
				log.Error().Err(err).Msg("Token validation failed")
				return unauthorizedError(c, "Token validation failed")
			}

			// Set context values
			ctx := c.Request().Context()
			ctx = context.WithValue(ctx, WorkspaceIDKey, apiToken.WorkspaceID)
			ctx = context.WithValue(ctx, UserIDKey, apiToken.UserID)
			ctx = context.WithValue(ctx, APITokenIDKey, apiToken.ID)
			ctx = context.WithValue(ctx, IsAPITokenAuthKey, true)

			c.SetRequest(c.Request().WithContext(ctx))

			log.Debug().
				Int32("workspace_id", apiToken.WorkspaceID).
				Str("token_id", apiToken.ID.String()).
				Msg("API token authentication successful")

			return next(c)
		}
	}
}

// GetUserID extracts the user ID from the context (set by API token auth)
func GetUserID(c echo.Context) uuid.UUID {
	if id, ok := c.Request().Context().Value(UserIDKey).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

// GetAPITokenID extracts the API token ID from the context
func GetAPITokenID(c echo.Context) uuid.UUID {
	if id, ok := c.Request().Context().Value(APITokenIDKey).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

// IsAPITokenAuth checks if the request was authenticated via API token
func IsAPITokenAuth(c echo.Context) bool {
	if isAPIToken, ok := c.Request().Context().Value(IsAPITokenAuthKey).(bool); ok {
		return isAPIToken
	}
	return false
}
