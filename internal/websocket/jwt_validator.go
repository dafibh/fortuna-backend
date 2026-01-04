package websocket

import (
	"context"
	"errors"
	"net/url"
	"time"

	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
)

// ErrInvalidToken is returned when JWT validation fails
var ErrInvalidToken = errors.New("invalid token")

// ErrWorkspaceNotFound is returned when workspace lookup fails
var ErrWorkspaceNotFound = errors.New("workspace not found")

// WorkspaceLookup provides workspace lookup by Auth0 ID
type WorkspaceLookup interface {
	GetWorkspaceByAuth0ID(auth0ID string) (workspaceID int32, err error)
}

// CustomClaims contains the custom claims from Auth0 JWT
type CustomClaims struct{}

// Validate implements validator.CustomClaims
func (c CustomClaims) Validate(ctx context.Context) error {
	return nil
}

// Auth0JWTValidator validates Auth0 JWT tokens for WebSocket connections
type Auth0JWTValidator struct {
	validator       *validator.Validator
	workspaceLookup WorkspaceLookup
}

// NewAuth0JWTValidator creates a new Auth0JWTValidator
func NewAuth0JWTValidator(domain, audience string, workspaceLookup WorkspaceLookup) (*Auth0JWTValidator, error) {
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

	return &Auth0JWTValidator{
		validator:       jwtValidator,
		workspaceLookup: workspaceLookup,
	}, nil
}

// ValidateToken validates a JWT token and returns the associated workspace ID
func (v *Auth0JWTValidator) ValidateToken(token string) (workspaceID int32, err error) {
	ctx := context.Background()

	claims, err := v.validator.ValidateToken(ctx, token)
	if err != nil {
		return 0, ErrInvalidToken
	}

	validatedClaims, ok := claims.(*validator.ValidatedClaims)
	if !ok {
		return 0, ErrInvalidToken
	}

	auth0ID := validatedClaims.RegisteredClaims.Subject

	// Lookup workspace by Auth0 ID
	wsID, err := v.workspaceLookup.GetWorkspaceByAuth0ID(auth0ID)
	if err != nil {
		return 0, ErrWorkspaceNotFound
	}

	return wsID, nil
}
