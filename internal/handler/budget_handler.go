package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/middleware"
	"github.com/dafibh/fortuna/fortuna-backend/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

// BudgetHandler handles budget allocation HTTP requests
type BudgetHandler struct {
	allocationService *service.BudgetAllocationService
}

// NewBudgetHandler creates a new BudgetHandler
func NewBudgetHandler(allocationService *service.BudgetAllocationService) *BudgetHandler {
	return &BudgetHandler{allocationService: allocationService}
}

// AllocationInput represents a single allocation in batch requests
type AllocationInput struct {
	CategoryID int32  `json:"categoryId"`
	Amount     string `json:"amount"`
}

// SetAllocationsRequest represents the batch update request body
type SetAllocationsRequest struct {
	Allocations []AllocationInput `json:"allocations"`
}

// SetAllocationRequest represents the single update request body
type SetAllocationRequest struct {
	Amount string `json:"amount"`
}

// BudgetCategoryWithAllocationResponse represents a category with its allocation
type BudgetCategoryWithAllocationResponse struct {
	CategoryID   int32  `json:"categoryId"`
	CategoryName string `json:"categoryName"`
	Allocated    string `json:"allocated"`
}

// BudgetProgressResponse represents a category with budget progress
type BudgetProgressResponse struct {
	CategoryID   int32  `json:"categoryId"`
	CategoryName string `json:"categoryName"`
	Allocated    string `json:"allocated"`
	Spent        string `json:"spent"`
	Remaining    string `json:"remaining"`
	Percentage   string `json:"percentage"`
	Status       string `json:"status"`
}

// BudgetMonthResponse represents the budget data for a month (allocation only)
type BudgetMonthResponse struct {
	Year           int                                    `json:"year"`
	Month          int                                    `json:"month"`
	TotalAllocated string                                 `json:"totalAllocated"`
	Categories     []BudgetCategoryWithAllocationResponse `json:"categories"`
}

// MonthlyBudgetSummaryResponse represents the budget progress data for a month
type MonthlyBudgetSummaryResponse struct {
	Year           int                      `json:"year"`
	Month          int                      `json:"month"`
	TotalAllocated string                   `json:"totalAllocated"`
	TotalSpent     string                   `json:"totalSpent"`
	TotalRemaining string                   `json:"totalRemaining"`
	Categories     []BudgetProgressResponse `json:"categories"`
}

// GetAllocations handles GET /api/v1/budgets/:year/:month
func (h *BudgetHandler) GetAllocations(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	year, err := strconv.Atoi(c.Param("year"))
	if err != nil || year < 1900 || year > 2100 {
		return NewValidationError(c, "Invalid year", nil)
	}

	month, err := strconv.Atoi(c.Param("month"))
	if err != nil || month < 1 || month > 12 {
		return NewValidationError(c, "Invalid month", nil)
	}

	result, err := h.allocationService.GetMonthlyProgress(workspaceID, year, month)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("year", year).Int("month", month).Msg("Failed to get budget progress")
		return NewInternalError(c, "Failed to get budget progress")
	}

	return c.JSON(http.StatusOK, toMonthlyBudgetSummaryResponse(result))
}

// SetAllocations handles PUT /api/v1/budgets/:year/:month (batch update)
func (h *BudgetHandler) SetAllocations(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	year, err := strconv.Atoi(c.Param("year"))
	if err != nil || year < 1900 || year > 2100 {
		return NewValidationError(c, "Invalid year", nil)
	}

	month, err := strconv.Atoi(c.Param("month"))
	if err != nil || month < 1 || month > 12 {
		return NewValidationError(c, "Invalid month", nil)
	}

	var req SetAllocationsRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	// Convert request to service input
	allocations := make([]service.AllocationInput, len(req.Allocations))
	for i, a := range req.Allocations {
		amount, err := decimal.NewFromString(a.Amount)
		if err != nil {
			return NewValidationError(c, "Invalid amount format", []ValidationError{
				{Field: "amount", Message: "Must be a valid decimal number"},
			})
		}
		allocations[i] = service.AllocationInput{
			CategoryID: a.CategoryID,
			Amount:     amount,
		}
	}

	result, err := h.allocationService.SetAllocations(workspaceID, year, month, allocations)
	if err != nil {
		if errors.Is(err, domain.ErrBudgetCategoryNotFound) {
			return NewNotFoundError(c, "One or more categories not found")
		}
		if errors.Is(err, domain.ErrInvalidAmount) {
			return NewValidationError(c, "Invalid amount", []ValidationError{
				{Field: "amount", Message: "Amount must be zero or positive"},
			})
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("year", year).Int("month", month).Msg("Failed to set budget allocations")
		return NewInternalError(c, "Failed to set budget allocations")
	}

	log.Info().Int32("workspace_id", workspaceID).Int("year", year).Int("month", month).Int("count", len(allocations)).Msg("Budget allocations updated (batch)")

	return c.JSON(http.StatusOK, toBudgetMonthResponse(result))
}

// GetCategoryTransactions handles GET /api/v1/budgets/:year/:month/:categoryId/transactions
func (h *BudgetHandler) GetCategoryTransactions(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	year, err := strconv.Atoi(c.Param("year"))
	if err != nil || year < 1900 || year > 2100 {
		return NewValidationError(c, "Invalid year", nil)
	}

	month, err := strconv.Atoi(c.Param("month"))
	if err != nil || month < 1 || month > 12 {
		return NewValidationError(c, "Invalid month", nil)
	}

	categoryID, err := strconv.Atoi(c.Param("categoryId"))
	if err != nil {
		return NewValidationError(c, "Invalid category ID", nil)
	}

	result, err := h.allocationService.GetCategoryTransactions(workspaceID, int32(categoryID), year, month)
	if err != nil {
		if errors.Is(err, domain.ErrBudgetCategoryNotFound) {
			return NewNotFoundError(c, "Category not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("year", year).Int("month", month).Int("category_id", categoryID).Msg("Failed to get category transactions")
		return NewInternalError(c, "Failed to get category transactions")
	}

	return c.JSON(http.StatusOK, result)
}

// SetAllocation handles PUT /api/v1/budgets/:year/:month/:categoryId (single update)
func (h *BudgetHandler) SetAllocation(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	year, err := strconv.Atoi(c.Param("year"))
	if err != nil || year < 1900 || year > 2100 {
		return NewValidationError(c, "Invalid year", nil)
	}

	month, err := strconv.Atoi(c.Param("month"))
	if err != nil || month < 1 || month > 12 {
		return NewValidationError(c, "Invalid month", nil)
	}

	categoryID, err := strconv.Atoi(c.Param("categoryId"))
	if err != nil {
		return NewValidationError(c, "Invalid category ID", nil)
	}

	var req SetAllocationRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return NewValidationError(c, "Invalid amount format", []ValidationError{
			{Field: "amount", Message: "Must be a valid decimal number"},
		})
	}

	result, err := h.allocationService.SetAllocation(workspaceID, int32(categoryID), year, month, amount)
	if err != nil {
		if errors.Is(err, domain.ErrBudgetCategoryNotFound) {
			return NewNotFoundError(c, "Category not found")
		}
		if errors.Is(err, domain.ErrInvalidAmount) {
			return NewValidationError(c, "Invalid amount", []ValidationError{
				{Field: "amount", Message: "Amount must be zero or positive"},
			})
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("year", year).Int("month", month).Int("category_id", categoryID).Msg("Failed to set budget allocation")
		return NewInternalError(c, "Failed to set budget allocation")
	}

	log.Info().Int32("workspace_id", workspaceID).Int("year", year).Int("month", month).Int("category_id", categoryID).Str("amount", amount.String()).Msg("Budget allocation updated")

	return c.JSON(http.StatusOK, BudgetCategoryWithAllocationResponse{
		CategoryID:   result.CategoryID,
		CategoryName: result.CategoryName,
		Allocated:    result.Allocated.StringFixed(2),
	})
}

// Helper function to convert service response to API response
func toBudgetMonthResponse(result *service.BudgetMonthResponse) BudgetMonthResponse {
	categories := make([]BudgetCategoryWithAllocationResponse, len(result.Categories))
	for i, cat := range result.Categories {
		categories[i] = BudgetCategoryWithAllocationResponse{
			CategoryID:   cat.CategoryID,
			CategoryName: cat.CategoryName,
			Allocated:    cat.Allocated.StringFixed(2),
		}
	}
	return BudgetMonthResponse{
		Year:           result.Year,
		Month:          result.Month,
		TotalAllocated: result.TotalAllocated.StringFixed(2),
		Categories:     categories,
	}
}

// toMonthlyBudgetSummaryResponse converts domain MonthlyBudgetSummary to API response
func toMonthlyBudgetSummaryResponse(result *domain.MonthlyBudgetSummary) MonthlyBudgetSummaryResponse {
	categories := make([]BudgetProgressResponse, len(result.Categories))
	for i, cat := range result.Categories {
		categories[i] = BudgetProgressResponse{
			CategoryID:   cat.CategoryID,
			CategoryName: cat.CategoryName,
			Allocated:    cat.Allocated.StringFixed(2),
			Spent:        cat.Spent.StringFixed(2),
			Remaining:    cat.Remaining.StringFixed(2),
			Percentage:   cat.Percentage.StringFixed(2),
			Status:       string(cat.Status),
		}
	}
	return MonthlyBudgetSummaryResponse{
		Year:           result.Year,
		Month:          result.Month,
		TotalAllocated: result.TotalAllocated.StringFixed(2),
		TotalSpent:     result.TotalSpent.StringFixed(2),
		TotalRemaining: result.TotalRemaining.StringFixed(2),
		Categories:     categories,
	}
}
