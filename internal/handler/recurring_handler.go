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
	"github.com/shopspring/decimal"
)

// RecurringHandler handles recurring transaction HTTP requests
type RecurringHandler struct {
	recurringService *service.RecurringService
}

// NewRecurringHandler creates a new RecurringHandler
func NewRecurringHandler(recurringService *service.RecurringService) *RecurringHandler {
	return &RecurringHandler{
		recurringService: recurringService,
	}
}

// CreateRecurringRequest represents the create recurring transaction request body
type CreateRecurringRequest struct {
	Name       string `json:"name"`
	Amount     string `json:"amount"`
	AccountID  int32  `json:"accountId"`
	Type       string `json:"type"`
	CategoryID *int32 `json:"categoryId,omitempty"`
	Frequency  string `json:"frequency"`
	DueDay     int32  `json:"dueDay"`
}

// UpdateRecurringRequest represents the update recurring transaction request body
type UpdateRecurringRequest struct {
	Name       string `json:"name"`
	Amount     string `json:"amount"`
	AccountID  int32  `json:"accountId"`
	Type       string `json:"type"`
	CategoryID *int32 `json:"categoryId,omitempty"`
	Frequency  string `json:"frequency"`
	DueDay     int32  `json:"dueDay"`
	IsActive   bool   `json:"isActive"`
}

// RecurringResponse represents a recurring transaction in API responses
type RecurringResponse struct {
	ID          int32   `json:"id"`
	WorkspaceID int32   `json:"workspaceId"`
	Name        string  `json:"name"`
	Amount      string  `json:"amount"`
	AccountID   int32   `json:"accountId"`
	Type        string  `json:"type"`
	CategoryID  *int32  `json:"categoryId,omitempty"`
	Frequency   string  `json:"frequency"`
	DueDay      int32   `json:"dueDay"`
	IsActive    bool    `json:"isActive"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
	DeletedAt   *string `json:"deletedAt,omitempty"`
}

// RecurringListResponse represents the list response
type RecurringListResponse struct {
	Data []RecurringResponse `json:"data"`
}

// CreateRecurring handles POST /api/v1/recurring-transactions
func (h *RecurringHandler) CreateRecurring(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	var req CreateRecurringRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	// Parse amount
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return NewValidationError(c, "Invalid amount", []ValidationError{
			{Field: "amount", Message: "Must be a valid decimal number"},
		})
	}

	input := service.CreateRecurringInput{
		Name:       req.Name,
		Amount:     amount,
		AccountID:  req.AccountID,
		Type:       domain.TransactionType(req.Type),
		CategoryID: req.CategoryID,
		Frequency:  domain.Frequency(req.Frequency),
		DueDay:     req.DueDay,
	}

	rt, err := h.recurringService.CreateRecurring(workspaceID, input)
	if err != nil {
		return h.handleServiceError(c, err, workspaceID, "create recurring transaction")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("recurring_id", rt.ID).Str("name", rt.Name).Msg("Recurring transaction created")

	return c.JSON(http.StatusCreated, toRecurringResponse(rt))
}

// GetRecurringTransactions handles GET /api/v1/recurring-transactions
func (h *RecurringHandler) GetRecurringTransactions(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	// Check for active query param
	var activeOnly *bool
	if activeParam := c.QueryParam("active"); activeParam != "" {
		active := activeParam == "true"
		activeOnly = &active
	}

	rts, err := h.recurringService.ListRecurring(workspaceID, activeOnly)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get recurring transactions")
		return NewInternalError(c, "Failed to get recurring transactions")
	}

	response := make([]RecurringResponse, len(rts))
	for i, rt := range rts {
		response[i] = toRecurringResponse(rt)
	}

	return c.JSON(http.StatusOK, RecurringListResponse{Data: response})
}

// GetRecurringTransaction handles GET /api/v1/recurring-transactions/:id
func (h *RecurringHandler) GetRecurringTransaction(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid recurring transaction ID", nil)
	}

	rt, err := h.recurringService.GetRecurringByID(workspaceID, int32(id))
	if err != nil {
		if errors.Is(err, domain.ErrRecurringNotFound) {
			return NewNotFoundError(c, "Recurring transaction not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("recurring_id", id).Msg("Failed to get recurring transaction")
		return NewInternalError(c, "Failed to get recurring transaction")
	}

	return c.JSON(http.StatusOK, toRecurringResponse(rt))
}

// UpdateRecurring handles PUT /api/v1/recurring-transactions/:id
func (h *RecurringHandler) UpdateRecurring(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid recurring transaction ID", nil)
	}

	var req UpdateRecurringRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	// Parse amount
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return NewValidationError(c, "Invalid amount", []ValidationError{
			{Field: "amount", Message: "Must be a valid decimal number"},
		})
	}

	input := service.UpdateRecurringInput{
		Name:       req.Name,
		Amount:     amount,
		AccountID:  req.AccountID,
		Type:       domain.TransactionType(req.Type),
		CategoryID: req.CategoryID,
		Frequency:  domain.Frequency(req.Frequency),
		DueDay:     req.DueDay,
		IsActive:   req.IsActive,
	}

	rt, err := h.recurringService.UpdateRecurring(workspaceID, int32(id), input)
	if err != nil {
		return h.handleServiceError(c, err, workspaceID, "update recurring transaction")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("recurring_id", rt.ID).Str("name", rt.Name).Msg("Recurring transaction updated")

	return c.JSON(http.StatusOK, toRecurringResponse(rt))
}

// DeleteRecurring handles DELETE /api/v1/recurring-transactions/:id
func (h *RecurringHandler) DeleteRecurring(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid recurring transaction ID", nil)
	}

	if err := h.recurringService.DeleteRecurring(workspaceID, int32(id)); err != nil {
		if errors.Is(err, domain.ErrRecurringNotFound) {
			return NewNotFoundError(c, "Recurring transaction not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("recurring_id", id).Msg("Failed to delete recurring transaction")
		return NewInternalError(c, "Failed to delete recurring transaction")
	}

	log.Info().Int32("workspace_id", workspaceID).Int("recurring_id", id).Msg("Recurring transaction deleted (soft)")
	return c.NoContent(http.StatusNoContent)
}

// handleServiceError handles common service errors
func (h *RecurringHandler) handleServiceError(c echo.Context, err error, workspaceID int32, operation string) error {
	if errors.Is(err, domain.ErrRecurringNotFound) {
		return NewNotFoundError(c, "Recurring transaction not found")
	}
	if errors.Is(err, domain.ErrNameRequired) {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "name", Message: "Name is required"},
		})
	}
	if errors.Is(err, domain.ErrNameTooLong) {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "name", Message: "Name must be 255 characters or less"},
		})
	}
	if errors.Is(err, domain.ErrInvalidAmount) {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "amount", Message: "Amount must be positive"},
		})
	}
	if errors.Is(err, domain.ErrInvalidTransactionType) {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "type", Message: "Type must be 'income' or 'expense'"},
		})
	}
	if errors.Is(err, domain.ErrInvalidFrequency) {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "frequency", Message: "Frequency must be 'monthly'"},
		})
	}
	if errors.Is(err, domain.ErrInvalidDueDay) {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "dueDay", Message: "Due day must be between 1 and 31"},
		})
	}
	if errors.Is(err, domain.ErrAccountNotFound) {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "accountId", Message: "Account not found"},
		})
	}
	if errors.Is(err, domain.ErrBudgetCategoryNotFound) {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "categoryId", Message: "Category not found"},
		})
	}
	log.Error().Err(err).Int32("workspace_id", workspaceID).Str("operation", operation).Msg("Failed to " + operation)
	return NewInternalError(c, "Failed to "+operation)
}

// Helper function to convert domain.RecurringTransaction to RecurringResponse
func toRecurringResponse(rt *domain.RecurringTransaction) RecurringResponse {
	resp := RecurringResponse{
		ID:          rt.ID,
		WorkspaceID: rt.WorkspaceID,
		Name:        rt.Name,
		Amount:      rt.Amount.StringFixed(2),
		AccountID:   rt.AccountID,
		Type:        string(rt.Type),
		CategoryID:  rt.CategoryID,
		Frequency:   string(rt.Frequency),
		DueDay:      rt.DueDay,
		IsActive:    rt.IsActive,
		CreatedAt:   rt.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   rt.UpdatedAt.Format(time.RFC3339),
	}
	if rt.DeletedAt != nil {
		deletedAt := rt.DeletedAt.Format(time.RFC3339)
		resp.DeletedAt = &deletedAt
	}
	return resp
}
