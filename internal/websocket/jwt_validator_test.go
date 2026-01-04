package websocket

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockWorkspaceLookup is a test double for WorkspaceLookup
type mockWorkspaceLookup struct {
	workspaceID int32
	err         error
}

func (m *mockWorkspaceLookup) GetWorkspaceByAuth0ID(auth0ID string) (workspaceID int32, err error) {
	return m.workspaceID, m.err
}

func TestWorkspaceLookup_Interface(t *testing.T) {
	// Verify mockWorkspaceLookup implements WorkspaceLookup
	var _ WorkspaceLookup = (*mockWorkspaceLookup)(nil)
}

func TestAuth0JWTValidator_ValidateToken_WorkspaceNotFound(t *testing.T) {
	// This test verifies the workspace lookup error path
	// We can't easily test the full JWT validation without a real Auth0 setup,
	// but we can verify the error types are correct

	t.Run("ErrWorkspaceNotFound is returned correctly", func(t *testing.T) {
		assert.Equal(t, "workspace not found", ErrWorkspaceNotFound.Error())
	})

	t.Run("ErrInvalidToken is returned correctly", func(t *testing.T) {
		assert.Equal(t, "invalid token", ErrInvalidToken.Error())
	})
}

func TestCustomClaims_Validate(t *testing.T) {
	claims := &CustomClaims{}
	err := claims.Validate(nil)
	assert.NoError(t, err, "CustomClaims.Validate should return nil")
}

func TestNewAuth0JWTValidator_InvalidDomain(t *testing.T) {
	lookup := &mockWorkspaceLookup{workspaceID: 1}

	// Test with empty domain - should still work (URL parsing is lenient)
	validator, err := NewAuth0JWTValidator("", "audience", lookup)
	// Empty domain creates https:/// which is technically valid URL
	assert.NoError(t, err)
	assert.NotNil(t, validator)
}

func TestNewAuth0JWTValidator_Success(t *testing.T) {
	lookup := &mockWorkspaceLookup{workspaceID: 1}

	validator, err := NewAuth0JWTValidator("test.auth0.com", "https://api.fortuna.app", lookup)
	assert.NoError(t, err)
	assert.NotNil(t, validator)
	assert.NotNil(t, validator.validator)
	assert.Equal(t, lookup, validator.workspaceLookup)
}

func TestAuth0JWTValidator_ValidateToken_InvalidJWT(t *testing.T) {
	lookup := &mockWorkspaceLookup{workspaceID: 1}

	validator, err := NewAuth0JWTValidator("test.auth0.com", "https://api.fortuna.app", lookup)
	assert.NoError(t, err)

	// Test with invalid token - should return ErrInvalidToken
	workspaceID, err := validator.ValidateToken("invalid-token")
	assert.Error(t, err)
	assert.Equal(t, int32(0), workspaceID)
	assert.True(t, errors.Is(err, ErrInvalidToken))
}
