package middleware

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// CustomClaims contains the custom claims from Auth0 JWT
type CustomClaims struct {
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// Validate implements validator.CustomClaims
func (c CustomClaims) Validate(ctx context.Context) error {
	return nil
}

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// ClaimsKey is the context key for JWT claims
	ClaimsKey contextKey = "claims"
	// Auth0IDKey is the context key for the Auth0 user ID (subject)
	Auth0IDKey contextKey = "auth0_id"
	// WorkspaceIDKey is the context key for the user's workspace ID
	WorkspaceIDKey contextKey = "workspace_id"
)

// WorkspaceProvider provides workspace lookup by Auth0 ID
type WorkspaceProvider interface {
	GetWorkspaceByAuth0ID(auth0ID string) (workspaceID int32, err error)
}

// AuthMiddleware provides JWT validation middleware
type AuthMiddleware struct {
	validator         *validator.Validator
	workspaceProvider WorkspaceProvider
}

// NewAuthMiddleware creates a new AuthMiddleware with Auth0 configuration
func NewAuthMiddleware(domain, audience string, workspaceProvider WorkspaceProvider) (*AuthMiddleware, error) {
	issuerURL, err := url.Parse("https://" + domain + "/")
	if err != nil {
		return nil, err
	}

	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

	jwtValidator, err := validator.New(
		provider.KeyFunc,
		validator.RS256,
		issuerURL.String(),
		[]string{audience},
		validator.WithCustomClaims(func() validator.CustomClaims {
			return &CustomClaims{}
		}),
		validator.WithAllowedClockSkew(time.Minute),
	)
	if err != nil {
		return nil, err
	}

	return &AuthMiddleware{
		validator:         jwtValidator,
		workspaceProvider: workspaceProvider,
	}, nil
}

// Authenticate returns an Echo middleware that validates JWT tokens
func (m *AuthMiddleware) Authenticate() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}

			// Check Bearer prefix
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header format")
			}

			token := parts[1]

			// Validate the token
			claims, err := m.validator.ValidateToken(c.Request().Context(), token)
			if err != nil {
				log.Debug().Err(err).Msg("Token validation failed")
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}

			validatedClaims, ok := claims.(*validator.ValidatedClaims)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid claims")
			}

			auth0ID := validatedClaims.RegisteredClaims.Subject

			// Store claims in context
			ctx := context.WithValue(c.Request().Context(), ClaimsKey, validatedClaims)
			ctx = context.WithValue(ctx, Auth0IDKey, auth0ID)

			// Fetch workspace by auth0_id and inject into context
			if m.workspaceProvider != nil {
				workspaceID, err := m.workspaceProvider.GetWorkspaceByAuth0ID(auth0ID)
				if err != nil {
					log.Debug().Err(err).Str("auth0_id", auth0ID).Msg("Workspace lookup failed")
					return echo.NewHTTPError(http.StatusUnauthorized, "workspace not found")
				}
				ctx = context.WithValue(ctx, WorkspaceIDKey, workspaceID)
			}

			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

// GetAuth0ID extracts the Auth0 user ID from the context
func GetAuth0ID(c echo.Context) string {
	if id, ok := c.Request().Context().Value(Auth0IDKey).(string); ok {
		return id
	}
	return ""
}

// GetClaims extracts the validated claims from the context
func GetClaims(c echo.Context) *validator.ValidatedClaims {
	if claims, ok := c.Request().Context().Value(ClaimsKey).(*validator.ValidatedClaims); ok {
		return claims
	}
	return nil
}

// GetCustomClaims extracts the custom claims from the context
func GetCustomClaims(c echo.Context) *CustomClaims {
	claims := GetClaims(c)
	if claims == nil {
		return nil
	}
	if custom, ok := claims.CustomClaims.(*CustomClaims); ok {
		return custom
	}
	return nil
}

// GetWorkspaceID extracts the workspace ID from the context
func GetWorkspaceID(c echo.Context) int32 {
	if id, ok := c.Request().Context().Value(WorkspaceIDKey).(int32); ok {
		return id
	}
	return 0
}
