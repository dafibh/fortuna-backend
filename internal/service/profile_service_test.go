package service

import (
	"testing"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/google/uuid"
)

func TestGetProfile_Success(t *testing.T) {
	userRepo := testutil.NewMockUserRepository()
	profileService := NewProfileService(userRepo)

	// Pre-create user
	auth0ID := "auth0|profile123"
	name := "Test User"
	existingUser := &domain.User{
		ID:      uuid.New(),
		Auth0ID: auth0ID,
		Email:   "test@example.com",
		Name:    &name,
	}
	userRepo.AddUser(existingUser)

	user, err := profileService.GetProfile(auth0ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %s", user.Email)
	}

	if *user.Name != name {
		t.Errorf("Expected name '%s', got '%s'", name, *user.Name)
	}
}

func TestGetProfile_UserNotFound(t *testing.T) {
	userRepo := testutil.NewMockUserRepository()
	profileService := NewProfileService(userRepo)

	_, err := profileService.GetProfile("auth0|nonexistent")
	if err != domain.ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestUpdateProfile_Success(t *testing.T) {
	userRepo := testutil.NewMockUserRepository()
	profileService := NewProfileService(userRepo)

	// Pre-create user
	auth0ID := "auth0|update123"
	oldName := "Old Name"
	existingUser := &domain.User{
		ID:      uuid.New(),
		Auth0ID: auth0ID,
		Email:   "update@example.com",
		Name:    &oldName,
	}
	userRepo.AddUser(existingUser)

	newName := "New Name"
	user, err := profileService.UpdateProfile(auth0ID, newName)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if *user.Name != newName {
		t.Errorf("Expected name '%s', got '%s'", newName, *user.Name)
	}

	// Email should remain unchanged
	if user.Email != "update@example.com" {
		t.Errorf("Expected email to remain 'update@example.com', got %s", user.Email)
	}
}

func TestUpdateProfile_UserNotFound(t *testing.T) {
	userRepo := testutil.NewMockUserRepository()
	profileService := NewProfileService(userRepo)

	_, err := profileService.UpdateProfile("auth0|nonexistent", "New Name")
	if err != domain.ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}
