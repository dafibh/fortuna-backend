package service

import (
	"strings"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
)

// BudgetCategoryService handles budget category business logic
type BudgetCategoryService struct {
	categoryRepo domain.BudgetCategoryRepository
}

// NewBudgetCategoryService creates a new BudgetCategoryService
func NewBudgetCategoryService(categoryRepo domain.BudgetCategoryRepository) *BudgetCategoryService {
	return &BudgetCategoryService{categoryRepo: categoryRepo}
}

// CreateCategory creates a new budget category
func (s *BudgetCategoryService) CreateCategory(workspaceID int32, name string) (*domain.BudgetCategory, error) {
	// Validate name
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrNameRequired
	}
	if len(name) > domain.MaxBudgetCategoryNameLength {
		return nil, domain.ErrNameTooLong
	}

	category := &domain.BudgetCategory{
		WorkspaceID: workspaceID,
		Name:        name,
	}

	return s.categoryRepo.Create(category)
}

// GetCategories retrieves all budget categories for a workspace
func (s *BudgetCategoryService) GetCategories(workspaceID int32) ([]*domain.BudgetCategory, error) {
	return s.categoryRepo.GetAllByWorkspace(workspaceID)
}

// GetCategoryByID retrieves a budget category by ID within a workspace
func (s *BudgetCategoryService) GetCategoryByID(workspaceID int32, id int32) (*domain.BudgetCategory, error) {
	return s.categoryRepo.GetByID(workspaceID, id)
}

// UpdateCategory updates a budget category's name
func (s *BudgetCategoryService) UpdateCategory(workspaceID int32, id int32, name string) (*domain.BudgetCategory, error) {
	// Validate name
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrNameRequired
	}
	if len(name) > domain.MaxBudgetCategoryNameLength {
		return nil, domain.ErrNameTooLong
	}

	return s.categoryRepo.Update(workspaceID, id, name)
}

// DeleteCategory soft-deletes a budget category
func (s *BudgetCategoryService) DeleteCategory(workspaceID int32, id int32) error {
	// Verify category exists before deleting
	_, err := s.categoryRepo.GetByID(workspaceID, id)
	if err != nil {
		return err
	}
	return s.categoryRepo.SoftDelete(workspaceID, id)
}

// CanDeleteResponse contains information about whether a category can be safely deleted
type CanDeleteResponse struct {
	HasTransactions  bool  `json:"hasTransactions"`
	TransactionCount int64 `json:"transactionCount"`
}

// CanDelete checks if a budget category can be safely deleted (no transactions assigned)
func (s *BudgetCategoryService) CanDelete(workspaceID int32, id int32) (*CanDeleteResponse, error) {
	// Verify category exists
	_, err := s.categoryRepo.GetByID(workspaceID, id)
	if err != nil {
		return nil, err
	}

	hasTransactions, err := s.categoryRepo.HasTransactions(workspaceID, id)
	if err != nil {
		return nil, err
	}

	// NOTE: Until Story 4.2 adds category_id to transactions, this will always return false/0
	return &CanDeleteResponse{
		HasTransactions:  hasTransactions,
		TransactionCount: 0, // Will be populated after Story 4.2
	}, nil
}
