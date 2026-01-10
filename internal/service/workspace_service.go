package service

import (
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
)

// WorkspaceService handles workspace-related business logic
type WorkspaceService struct {
	workspaceRepo domain.WorkspaceRepository
}

// NewWorkspaceService creates a new WorkspaceService
func NewWorkspaceService(workspaceRepo domain.WorkspaceRepository) *WorkspaceService {
	return &WorkspaceService{workspaceRepo: workspaceRepo}
}

// ClearAllData deletes all data for a workspace (but keeps the workspace itself)
// This is a destructive operation that removes all accounts, transactions, budgets, loans, wishlists, etc.
func (s *WorkspaceService) ClearAllData(workspaceID int32) error {
	// Verify workspace exists
	_, err := s.workspaceRepo.GetByID(workspaceID)
	if err != nil {
		return err
	}

	// Clear all data
	return s.workspaceRepo.ClearAllData(workspaceID)
}

