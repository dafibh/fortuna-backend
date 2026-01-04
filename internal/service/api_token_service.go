package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const (
	// tokenPrefix is the prefix for all API tokens
	tokenPrefix = "fort_"
	// tokenRandomBytes is the number of random bytes for the token (32 bytes = 256 bits)
	tokenRandomBytes = 32
	// tokenPrefixLength is the length of the displayable prefix (e.g., "fort_abc...xyz")
	tokenPrefixLength = 8
	// maxTokensPerWorkspace is the maximum number of active tokens per workspace
	maxTokensPerWorkspace = 10
)

// APITokenService handles API token business logic
type APITokenService struct {
	repo domain.APITokenRepository
}

// NewAPITokenService creates a new APITokenService
func NewAPITokenService(repo domain.APITokenRepository) *APITokenService {
	return &APITokenService{repo: repo}
}

// Create creates a new API token and returns the full token (shown only once)
func (s *APITokenService) Create(ctx context.Context, userID uuid.UUID, workspaceID int32, description string) (*domain.CreateAPITokenResponse, error) {
	// Check token limit per workspace
	existingTokens, err := s.repo.GetByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if len(existingTokens) >= maxTokensPerWorkspace {
		return nil, domain.ErrTooManyAPITokens
	}

	// Generate secure random token
	rawToken, err := generateSecureToken()
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate secure token")
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Create full token with prefix
	fullToken := tokenPrefix + rawToken

	// Hash the token for storage
	hash := hashToken(fullToken)

	// Extract displayable prefix (first 8 chars after fort_)
	displayPrefix := tokenPrefix + rawToken[:tokenPrefixLength] + "..."

	// Create domain token
	token := &domain.APIToken{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Description: description,
		TokenHash:   hash,
		TokenPrefix: displayPrefix,
	}

	if err := s.repo.Create(ctx, token); err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to create API token")
		return nil, err
	}

	log.Info().
		Str("token_id", token.ID.String()).
		Int32("workspace_id", workspaceID).
		Str("description", description).
		Msg("API token created")

	return &domain.CreateAPITokenResponse{
		ID:          token.ID,
		Description: description,
		TokenPrefix: displayPrefix,
		Token:       fullToken,
		CreatedAt:   token.CreatedAt,
		Warning:     "Make sure to copy your API token now. You won't be able to see it again!",
	}, nil
}

// GetByWorkspace retrieves all active API tokens for a workspace
func (s *APITokenService) GetByWorkspace(ctx context.Context, workspaceID int32) ([]*domain.APITokenResponse, error) {
	tokens, err := s.repo.GetByWorkspace(ctx, workspaceID)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get API tokens")
		return nil, err
	}

	// Convert to response DTOs (without sensitive data)
	result := make([]*domain.APITokenResponse, len(tokens))
	for i, t := range tokens {
		result[i] = &domain.APITokenResponse{
			ID:          t.ID,
			Description: t.Description,
			TokenPrefix: t.TokenPrefix,
			CreatedAt:   t.CreatedAt,
			LastUsedAt:  t.LastUsedAt,
		}
	}
	return result, nil
}

// Revoke revokes an API token
func (s *APITokenService) Revoke(ctx context.Context, workspaceID int32, tokenID uuid.UUID) error {
	if err := s.repo.Revoke(ctx, workspaceID, tokenID); err != nil {
		log.Error().Err(err).
			Int32("workspace_id", workspaceID).
			Str("token_id", tokenID.String()).
			Msg("Failed to revoke API token")
		return err
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Str("token_id", tokenID.String()).
		Msg("API token revoked")

	return nil
}

// ValidateToken validates an API token and returns the associated token data
// Note: This method is implemented for Story 10.3 (API Authentication)
func (s *APITokenService) ValidateToken(ctx context.Context, token string) (*domain.APIToken, error) {
	// Validate token format - must start with fort_ prefix
	if len(token) < len(tokenPrefix) || token[:len(tokenPrefix)] != tokenPrefix {
		return nil, domain.ErrAPITokenNotFound
	}

	// Hash the provided token
	hash := hashToken(token)

	// Look up by hash
	apiToken, err := s.repo.GetByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	// Update last used timestamp asynchronously
	go func() {
		if updateErr := s.repo.UpdateLastUsed(context.Background(), apiToken.ID); updateErr != nil {
			log.Error().Err(updateErr).Str("token_id", apiToken.ID.String()).Msg("Failed to update last_used_at")
		}
	}()

	return apiToken, nil
}

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken() (string, error) {
	bytes := make([]byte, tokenRandomBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Use URL-safe base64 encoding without padding
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// hashToken creates a SHA-256 hash of the token
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", hash)
}
