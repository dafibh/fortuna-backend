package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// APIToken represents an API access token for programmatic access
type APIToken struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"userId"`
	WorkspaceID int32      `json:"workspaceId"`
	Description string     `json:"description"`
	TokenHash   string     `json:"-"` // Never expose hash
	TokenPrefix string     `json:"tokenPrefix"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	RevokedAt   *time.Time `json:"revokedAt,omitempty"`
}

// CreateAPITokenRequest represents the request to create a new API token
type CreateAPITokenRequest struct {
	Description string `json:"description" validate:"required,max=255"`
}

// APITokenResponse represents a token in list responses (excludes sensitive data)
type APITokenResponse struct {
	ID          uuid.UUID  `json:"id"`
	Description string     `json:"description"`
	TokenPrefix string     `json:"tokenPrefix"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
}

// CreateAPITokenResponse includes the full token for one-time display
type CreateAPITokenResponse struct {
	ID          uuid.UUID `json:"id"`
	Description string    `json:"description"`
	TokenPrefix string    `json:"tokenPrefix"`
	Token       string    `json:"token"` // Full token - shown only once!
	CreatedAt   time.Time `json:"createdAt"`
	Warning     string    `json:"warning"`
}

// APITokenRepository defines the interface for API token persistence
type APITokenRepository interface {
	Create(ctx context.Context, token *APIToken) error
	GetByWorkspace(ctx context.Context, workspaceID int32) ([]*APIToken, error)
	GetByID(ctx context.Context, workspaceID int32, id uuid.UUID) (*APIToken, error)
	GetByHash(ctx context.Context, hash string) (*APIToken, error)
	Revoke(ctx context.Context, workspaceID int32, id uuid.UUID) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
}
