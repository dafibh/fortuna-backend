package service

import (
	"testing"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/google/uuid"
)

func TestAuthenticateUser_NewUser(t *testing.T) {
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	service := NewAuthService(userRepo, workspaceRepo)

	auth0ID := "auth0|12345"
	email := "test@example.com"
	name := "Test User"

	result, err := service.AuthenticateUser(auth0ID, email, &name, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if !result.IsNewUser {
		t.Error("Expected IsNewUser to be true for new user")
	}

	if result.User == nil {
		t.Fatal("Expected user, got nil")
	}

	if result.User.Auth0ID != auth0ID {
		t.Errorf("Expected auth0ID %s, got %s", auth0ID, result.User.Auth0ID)
	}

	if result.User.Email != email {
		t.Errorf("Expected email %s, got %s", email, result.User.Email)
	}

	if result.Workspace == nil {
		t.Fatal("Expected workspace, got nil")
	}

	if result.Workspace.Name != "Personal" {
		t.Errorf("Expected workspace name 'Personal', got %s", result.Workspace.Name)
	}
}

func TestAuthenticateUser_ExistingUser(t *testing.T) {
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	service := NewAuthService(userRepo, workspaceRepo)

	// Create existing user and workspace
	auth0ID := "auth0|existing"
	email := "existing@example.com"
	name := "Existing User"

	existingUser := &domain.User{
		ID:      uuid.New(),
		Auth0ID: auth0ID,
		Email:   email,
		Name:    &name,
	}
	userRepo.AddUser(existingUser)

	existingWorkspace := &domain.Workspace{
		ID:     1,
		UserID: existingUser.ID,
		Name:   "My Workspace",
	}
	workspaceRepo.AddWorkspace(existingWorkspace, "")

	result, err := service.AuthenticateUser(auth0ID, email, &name, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.IsNewUser {
		t.Error("Expected IsNewUser to be false for existing user")
	}

	if result.Workspace.Name != "My Workspace" {
		t.Errorf("Expected existing workspace name 'My Workspace', got %s", result.Workspace.Name)
	}
}

func TestGetUserByID(t *testing.T) {
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	service := NewAuthService(userRepo, workspaceRepo)

	// Create a user
	userID := uuid.New()
	user := &domain.User{
		ID:      userID,
		Auth0ID: "auth0|test",
		Email:   "test@example.com",
	}
	userRepo.AddUser(user)

	// Test getting user
	found, err := service.GetUserByID(userID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if found.ID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, found.ID)
	}

	// Test user not found
	_, err = service.GetUserByID(uuid.New())
	if err != domain.ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestGetUserByAuth0ID(t *testing.T) {
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	service := NewAuthService(userRepo, workspaceRepo)

	auth0ID := "auth0|findme"
	user := &domain.User{
		ID:      uuid.New(),
		Auth0ID: auth0ID,
		Email:   "findme@example.com",
	}
	userRepo.AddUser(user)

	found, err := service.GetUserByAuth0ID(auth0ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if found.Auth0ID != auth0ID {
		t.Errorf("Expected auth0ID %s, got %s", auth0ID, found.Auth0ID)
	}

	// Test not found
	_, err = service.GetUserByAuth0ID("auth0|notexist")
	if err != domain.ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestGetWorkspaceByAuth0ID(t *testing.T) {
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	service := NewAuthService(userRepo, workspaceRepo)

	// Setup: Create a user and workspace
	auth0ID := "auth0|workspace-test"
	user := &domain.User{
		ID:      uuid.New(),
		Auth0ID: auth0ID,
		Email:   "workspace@example.com",
	}
	userRepo.AddUser(user)

	workspace := &domain.Workspace{
		ID:     1,
		UserID: user.ID,
		Name:   "Test Workspace",
	}
	workspaceRepo.AddWorkspace(workspace, auth0ID)

	t.Run("returns workspace for valid auth0_id", func(t *testing.T) {
		found, err := service.GetWorkspaceByAuth0ID(auth0ID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if found.ID != workspace.ID {
			t.Errorf("Expected workspace ID %d, got %d", workspace.ID, found.ID)
		}
		if found.Name != "Test Workspace" {
			t.Errorf("Expected workspace name 'Test Workspace', got %s", found.Name)
		}
	})

	t.Run("returns error for unknown auth0_id", func(t *testing.T) {
		_, err := service.GetWorkspaceByAuth0ID("auth0|unknown")
		if err != domain.ErrWorkspaceNotFound {
			t.Errorf("Expected ErrWorkspaceNotFound, got %v", err)
		}
	})
}
