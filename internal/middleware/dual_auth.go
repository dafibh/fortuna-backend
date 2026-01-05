package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// DualAuthMiddleware provides middleware that accepts both JWT and API token authentication
type DualAuthMiddleware struct {
	jwtAuth      *AuthMiddleware
	apiTokenAuth *APITokenAuthMiddleware
}

// NewDualAuthMiddleware creates a new DualAuthMiddleware
func NewDualAuthMiddleware(jwtAuth *AuthMiddleware, apiTokenAuth *APITokenAuthMiddleware) *DualAuthMiddleware {
	return &DualAuthMiddleware{
		jwtAuth:      jwtAuth,
		apiTokenAuth: apiTokenAuth,
	}
}

// Authenticate returns an Echo middleware that tries JWT first, then API token
func (m *DualAuthMiddleware) Authenticate() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return unauthorizedError(c, "Missing authorization header")
			}

			var token string

			// Check if header starts with "Bearer " prefix
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				token = parts[1]
			} else if strings.HasPrefix(authHeader, "fort_") {
				// Accept API tokens without Bearer prefix (for Swagger/simple clients)
				token = authHeader
			} else {
				return unauthorizedError(c, "Invalid authorization header format")
			}

			// Determine auth type based on token format
			if strings.HasPrefix(token, "fort_") {
				// This is an API token
				log.Debug().Msg("Attempting API token authentication")
				return m.apiTokenAuth.authenticateWithToken(token)(next)(c)
			}

			// Try JWT authentication
			log.Debug().Msg("Attempting JWT authentication")
			return m.jwtAuth.Authenticate()(next)(c)
		}
	}
}

// JWTOnly returns a middleware that only accepts JWT authentication
// Use this for routes that should not allow API token access
func (m *DualAuthMiddleware) JWTOnly() echo.MiddlewareFunc {
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

			// Reject API tokens on JWT-only routes
			if strings.HasPrefix(token, "fort_") {
				log.Debug().Msg("API token rejected on JWT-only route")
				return unauthorizedError(c, "This endpoint requires session authentication")
			}

			// Use JWT authentication
			return m.jwtAuth.Authenticate()(next)(c)
		}
	}
}

// APITokenOnly returns a middleware that only accepts API token authentication
// Use this for routes that should only allow API token access
func (m *DualAuthMiddleware) APITokenOnly() echo.MiddlewareFunc {
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

			// Require API token
			if !strings.HasPrefix(token, "fort_") {
				log.Debug().Msg("Non-API token rejected on API-token-only route")
				return unauthorizedError(c, "This endpoint requires API token authentication")
			}

			// Use API token authentication
			return m.apiTokenAuth.Authenticate()(next)(c)
		}
	}
}
