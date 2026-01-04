package service

import (
	"context"
	"strings"
	"testing"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/google/uuid"
)

// MockAPITokenRepository is a mock implementation for testing
type MockAPITokenRepository struct {
	tokens    map[string]*domain.APIToken
	createErr error
}

func NewMockAPITokenRepository() *MockAPITokenRepository {
	return &MockAPITokenRepository{
		tokens: make(map[string]*domain.APIToken),
	}
}

func (m *MockAPITokenRepository) Create(ctx context.Context, token *domain.APIToken) error {
	if m.createErr != nil {
		return m.createErr
	}
	token.ID = uuid.New()
	m.tokens[token.TokenHash] = token
	return nil
}

func (m *MockAPITokenRepository) GetByWorkspace(ctx context.Context, workspaceID int32) ([]*domain.APIToken, error) {
	var result []*domain.APIToken
	for _, t := range m.tokens {
		if t.WorkspaceID == workspaceID && t.RevokedAt == nil {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *MockAPITokenRepository) GetByID(ctx context.Context, workspaceID int32, id uuid.UUID) (*domain.APIToken, error) {
	for _, t := range m.tokens {
		if t.ID == id && t.WorkspaceID == workspaceID {
			return t, nil
		}
	}
	return nil, domain.ErrAPITokenNotFound
}

func (m *MockAPITokenRepository) GetByHash(ctx context.Context, hash string) (*domain.APIToken, error) {
	if t, ok := m.tokens[hash]; ok && t.RevokedAt == nil {
		return t, nil
	}
	return nil, domain.ErrAPITokenNotFound
}

func (m *MockAPITokenRepository) Revoke(ctx context.Context, workspaceID int32, id uuid.UUID) error {
	for _, t := range m.tokens {
		if t.ID == id && t.WorkspaceID == workspaceID {
			return nil
		}
	}
	return domain.ErrAPITokenNotFound
}

func (m *MockAPITokenRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	return nil
}

func TestGenerateSecureToken(t *testing.T) {
	token1, err := generateSecureToken()
	if err != nil {
		t.Fatalf("generateSecureToken() error = %v", err)
	}

	// Token should be base64url encoded 32 bytes = 43 characters
	if len(token1) != 43 {
		t.Errorf("Expected token length 43, got %d", len(token1))
	}

	// Generate another token - should be different
	token2, err := generateSecureToken()
	if err != nil {
		t.Fatalf("generateSecureToken() error = %v", err)
	}

	if token1 == token2 {
		t.Error("Two generated tokens should not be equal")
	}
}

func TestHashToken(t *testing.T) {
	token := "fort_testtoken123"
	hash := hashToken(token)

	// SHA-256 produces 64 hex characters
	if len(hash) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash))
	}

	// Same input should produce same hash
	hash2 := hashToken(token)
	if hash != hash2 {
		t.Error("Same token should produce same hash")
	}

	// Different input should produce different hash
	hash3 := hashToken("fort_differenttoken")
	if hash == hash3 {
		t.Error("Different tokens should produce different hashes")
	}
}

func TestAPITokenService_Create(t *testing.T) {
	repo := NewMockAPITokenRepository()
	service := NewAPITokenService(repo)

	userID := uuid.New()
	workspaceID := int32(1)
	description := "Test token"

	result, err := service.Create(context.Background(), userID, workspaceID, description)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify token format
	if !strings.HasPrefix(result.Token, "fort_") {
		t.Errorf("Token should start with 'fort_', got %s", result.Token[:10])
	}

	// Verify token prefix format
	if !strings.HasPrefix(result.TokenPrefix, "fort_") {
		t.Errorf("TokenPrefix should start with 'fort_', got %s", result.TokenPrefix)
	}
	if !strings.HasSuffix(result.TokenPrefix, "...") {
		t.Errorf("TokenPrefix should end with '...', got %s", result.TokenPrefix)
	}

	// Verify description
	if result.Description != description {
		t.Errorf("Expected description %s, got %s", description, result.Description)
	}

	// Verify warning message
	if result.Warning == "" {
		t.Error("Warning message should not be empty")
	}
}

func TestAPITokenService_ValidateToken_InvalidFormat(t *testing.T) {
	repo := NewMockAPITokenRepository()
	service := NewAPITokenService(repo)

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"no prefix", "abc123"},
		{"wrong prefix", "wrong_abc123"},
		{"partial prefix", "for_abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ValidateToken(context.Background(), tt.token)
			if err != domain.ErrAPITokenNotFound {
				t.Errorf("ValidateToken(%s) expected ErrAPITokenNotFound, got %v", tt.token, err)
			}
		})
	}
}

func TestAPITokenService_ValidateToken_ValidFormat(t *testing.T) {
	repo := NewMockAPITokenRepository()
	service := NewAPITokenService(repo)

	// Create a token first
	userID := uuid.New()
	workspaceID := int32(1)
	result, err := service.Create(context.Background(), userID, workspaceID, "Test")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Validate the created token
	token, err := service.ValidateToken(context.Background(), result.Token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if token.WorkspaceID != workspaceID {
		t.Errorf("Expected workspaceID %d, got %d", workspaceID, token.WorkspaceID)
	}
}

func TestAPITokenService_GetByWorkspace(t *testing.T) {
	repo := NewMockAPITokenRepository()
	service := NewAPITokenService(repo)

	userID := uuid.New()
	workspaceID := int32(1)

	// Create two tokens
	_, err := service.Create(context.Background(), userID, workspaceID, "Token 1")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	_, err = service.Create(context.Background(), userID, workspaceID, "Token 2")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get tokens
	tokens, err := service.GetByWorkspace(context.Background(), workspaceID)
	if err != nil {
		t.Fatalf("GetByWorkspace() error = %v", err)
	}

	if len(tokens) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(tokens))
	}
}

func TestAPITokenService_Revoke(t *testing.T) {
	repo := NewMockAPITokenRepository()
	service := NewAPITokenService(repo)

	userID := uuid.New()
	workspaceID := int32(1)

	// Create a token
	result, err := service.Create(context.Background(), userID, workspaceID, "Test")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Revoke it
	err = service.Revoke(context.Background(), workspaceID, result.ID)
	if err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
}

func TestAPITokenService_Revoke_NotFound(t *testing.T) {
	repo := NewMockAPITokenRepository()
	service := NewAPITokenService(repo)

	// Try to revoke non-existent token
	err := service.Revoke(context.Background(), 1, uuid.New())
	if err != domain.ErrAPITokenNotFound {
		t.Errorf("Expected ErrAPITokenNotFound, got %v", err)
	}
}
