package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/middleware"
	"github.com/dafibh/fortuna/fortuna-backend/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// BudgetCategoryHandler handles budget category HTTP requests
type BudgetCategoryHandler struct {
	categoryService *service.BudgetCategoryService
}

// NewBudgetCategoryHandler creates a new BudgetCategoryHandler
func NewBudgetCategoryHandler(categoryService *service.BudgetCategoryService) *BudgetCategoryHandler {
	return &BudgetCategoryHandler{categoryService: categoryService}
}

// CreateBudgetCategoryRequest represents the create category request body
type CreateBudgetCategoryRequest struct {
	Name string `json:"name"`
}

// UpdateBudgetCategoryRequest represents the update category request body
type UpdateBudgetCategoryRequest struct {
	Name string `json:"name"`
}

// BudgetCategoryResponse represents a budget category in API responses
type BudgetCategoryResponse struct {
	ID          int32   `json:"id"`
	WorkspaceID int32   `json:"workspaceId"`
	Name        string  `json:"name"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
	DeletedAt   *string `json:"deletedAt,omitempty"`
}

// CanDeleteResponse represents the can-delete check response
type CanDeleteResponse struct {
	HasTransactions  bool  `json:"hasTransactions"`
	TransactionCount int64 `json:"transactionCount"`
}

// CreateCategory handles POST /api/v1/budget-categories
func (h *BudgetCategoryHandler) CreateCategory(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	var req CreateBudgetCategoryRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	category, err := h.categoryService.CreateCategory(workspaceID, req.Name)
	if err != nil {
		if errors.Is(err, domain.ErrNameRequired) {
			return NewValidationError(c, "Category name is required", []ValidationError{
				{Field: "name", Message: "Name cannot be empty"},
			})
		}
		if errors.Is(err, domain.ErrNameTooLong) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "name", Message: "Name must be 100 characters or less"},
			})
		}
		if errors.Is(err, domain.ErrBudgetCategoryAlreadyExists) {
			return NewConflictError(c, "A category with this name already exists")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to create budget category")
		return NewInternalError(c, "Failed to create category")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("category_id", category.ID).Str("name", category.Name).Msg("Budget category created")

	return c.JSON(http.StatusCreated, toBudgetCategoryResponse(category))
}

// GetCategories handles GET /api/v1/budget-categories
func (h *BudgetCategoryHandler) GetCategories(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	categories, err := h.categoryService.GetCategories(workspaceID)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get budget categories")
		return NewInternalError(c, "Failed to get categories")
	}

	response := make([]BudgetCategoryResponse, len(categories))
	for i, category := range categories {
		response[i] = toBudgetCategoryResponse(category)
	}

	return c.JSON(http.StatusOK, response)
}

// UpdateCategory handles PUT /api/v1/budget-categories/:id
func (h *BudgetCategoryHandler) UpdateCategory(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid category ID", nil)
	}

	var req UpdateBudgetCategoryRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	category, err := h.categoryService.UpdateCategory(workspaceID, int32(id), req.Name)
	if err != nil {
		if errors.Is(err, domain.ErrBudgetCategoryNotFound) {
			return NewNotFoundError(c, "Category not found")
		}
		if errors.Is(err, domain.ErrNameRequired) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "name", Message: "Name is required"},
			})
		}
		if errors.Is(err, domain.ErrNameTooLong) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "name", Message: "Name must be 100 characters or less"},
			})
		}
		if errors.Is(err, domain.ErrBudgetCategoryAlreadyExists) {
			return NewConflictError(c, "A category with this name already exists")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("category_id", id).Msg("Failed to update budget category")
		return NewInternalError(c, "Failed to update category")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("category_id", category.ID).Str("name", category.Name).Msg("Budget category updated")
	return c.JSON(http.StatusOK, toBudgetCategoryResponse(category))
}

// DeleteCategory handles DELETE /api/v1/budget-categories/:id
func (h *BudgetCategoryHandler) DeleteCategory(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid category ID", nil)
	}

	if err := h.categoryService.DeleteCategory(workspaceID, int32(id)); err != nil {
		if errors.Is(err, domain.ErrBudgetCategoryNotFound) {
			return NewNotFoundError(c, "Category not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("category_id", id).Msg("Failed to delete budget category")
		return NewInternalError(c, "Failed to delete category")
	}

	log.Info().Int32("workspace_id", workspaceID).Int("category_id", id).Msg("Budget category deleted (soft)")
	return c.NoContent(http.StatusNoContent)
}

// CanDeleteCategory handles GET /api/v1/budget-categories/:id/can-delete
func (h *BudgetCategoryHandler) CanDeleteCategory(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid category ID", nil)
	}

	result, err := h.categoryService.CanDelete(workspaceID, int32(id))
	if err != nil {
		if errors.Is(err, domain.ErrBudgetCategoryNotFound) {
			return NewNotFoundError(c, "Category not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("category_id", id).Msg("Failed to check category deletion")
		return NewInternalError(c, "Failed to check category")
	}

	return c.JSON(http.StatusOK, CanDeleteResponse{
		HasTransactions:  result.HasTransactions,
		TransactionCount: result.TransactionCount,
	})
}

// Helper function to convert domain.BudgetCategory to BudgetCategoryResponse
func toBudgetCategoryResponse(category *domain.BudgetCategory) BudgetCategoryResponse {
	resp := BudgetCategoryResponse{
		ID:          category.ID,
		WorkspaceID: category.WorkspaceID,
		Name:        category.Name,
		CreatedAt:   category.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   category.UpdatedAt.Format(time.RFC3339),
	}
	if category.DeletedAt != nil {
		deletedAt := category.DeletedAt.Format(time.RFC3339)
		resp.DeletedAt = &deletedAt
	}
	return resp
}
