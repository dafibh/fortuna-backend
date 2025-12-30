package service

import (
	"errors"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// AuthService handles authentication-related business logic
type AuthService struct {
	userRepo      domain.UserRepository
	workspaceRepo domain.WorkspaceRepository
}

// NewAuthService creates a new AuthService
func NewAuthService(userRepo domain.UserRepository, workspaceRepo domain.WorkspaceRepository) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		workspaceRepo: workspaceRepo,
	}
}

// AuthResult represents the result of an authentication operation
type AuthResult struct {
	User        *domain.User
	Workspace   *domain.Workspace
	IsNewUser   bool
}

// AuthenticateUser handles the authentication flow after Auth0 callback
// Creates user and workspace if they don't exist
func (s *AuthService) AuthenticateUser(auth0ID, email string, name, pictureURL *string) (*AuthResult, error) {
	// Try to get or create user
	user, err := s.userRepo.CreateOrGetByAuth0ID(auth0ID, email, name, pictureURL)
	if err != nil {
		log.Error().Err(err).Str("auth0_id", auth0ID).Msg("Failed to create or get user")
		return nil, err
	}

	// Check if this is a new user by trying to get their workspace
	workspace, err := s.workspaceRepo.GetByUserID(user.ID)
	if err != nil {
		if errors.Is(err, domain.ErrWorkspaceNotFound) {
			// New user - create default workspace
			workspace, err = s.createDefaultWorkspace(user.ID)
			if err != nil {
				log.Error().Err(err).Str("user_id", user.ID.String()).Msg("Failed to create default workspace")
				return nil, err
			}
			log.Info().Str("user_id", user.ID.String()).Msg("Created new user with default workspace")
			return &AuthResult{
				User:      user,
				Workspace: workspace,
				IsNewUser: true,
			}, nil
		}
		log.Error().Err(err).Str("user_id", user.ID.String()).Msg("Failed to get workspace")
		return nil, err
	}

	log.Info().Str("user_id", user.ID.String()).Msg("Existing user authenticated")
	return &AuthResult{
		User:      user,
		Workspace: workspace,
		IsNewUser: false,
	}, nil
}

// GetUserByID retrieves a user by their ID
func (s *AuthService) GetUserByID(id uuid.UUID) (*domain.User, error) {
	return s.userRepo.GetByID(id)
}

// GetUserByAuth0ID retrieves a user by their Auth0 ID
func (s *AuthService) GetUserByAuth0ID(auth0ID string) (*domain.User, error) {
	return s.userRepo.GetByAuth0ID(auth0ID)
}

// GetWorkspaceByUserID retrieves a user's workspace
func (s *AuthService) GetWorkspaceByUserID(userID uuid.UUID) (*domain.Workspace, error) {
	return s.workspaceRepo.GetByUserID(userID)
}

// GetWorkspaceByAuth0ID retrieves a user's workspace by their Auth0 ID
func (s *AuthService) GetWorkspaceByAuth0ID(auth0ID string) (*domain.Workspace, error) {
	return s.workspaceRepo.GetByUserAuth0ID(auth0ID)
}

// GetWorkspaceByID retrieves a workspace by its ID
func (s *AuthService) GetWorkspaceByID(id int32) (*domain.Workspace, error) {
	return s.workspaceRepo.GetByID(id)
}

func (s *AuthService) createDefaultWorkspace(userID uuid.UUID) (*domain.Workspace, error) {
	workspace := &domain.Workspace{
		UserID: userID,
		Name:   "Personal",
	}
	return s.workspaceRepo.Create(workspace)
}
