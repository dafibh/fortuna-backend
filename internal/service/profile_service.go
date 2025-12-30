package service

import (
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
)

// ProfileService handles profile-related business logic
type ProfileService struct {
	userRepo domain.UserRepository
}

// NewProfileService creates a new ProfileService
func NewProfileService(userRepo domain.UserRepository) *ProfileService {
	return &ProfileService{userRepo: userRepo}
}

// GetProfile retrieves a user's profile by Auth0 ID
func (s *ProfileService) GetProfile(auth0ID string) (*domain.User, error) {
	return s.userRepo.GetByAuth0ID(auth0ID)
}

// UpdateProfile updates a user's name by Auth0 ID
func (s *ProfileService) UpdateProfile(auth0ID string, name string) (*domain.User, error) {
	return s.userRepo.UpdateName(auth0ID, name)
}
