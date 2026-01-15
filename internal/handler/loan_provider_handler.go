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

// LoanProviderHandler handles loan provider-related HTTP requests
type LoanProviderHandler struct {
	providerService *service.LoanProviderService
}

// NewLoanProviderHandler creates a new LoanProviderHandler
func NewLoanProviderHandler(providerService *service.LoanProviderService) *LoanProviderHandler {
	return &LoanProviderHandler{providerService: providerService}
}

// CreateLoanProviderRequest represents the create loan provider request body
type CreateLoanProviderRequest struct {
	Name                string `json:"name"`
	CutoffDay           int32  `json:"cutoffDay"`
	DefaultInterestRate string `json:"defaultInterestRate"`
}

// UpdateLoanProviderRequest represents the update loan provider request body
type UpdateLoanProviderRequest struct {
	Name                string  `json:"name"`
	CutoffDay           int32   `json:"cutoffDay"`
	DefaultInterestRate string  `json:"defaultInterestRate"`
	PaymentMode         *string `json:"paymentMode,omitempty"`
}

// LoanProviderResponse represents a loan provider in API responses
type LoanProviderResponse struct {
	ID                  int32   `json:"id"`
	WorkspaceID         int32   `json:"workspaceId"`
	Name                string  `json:"name"`
	CutoffDay           int32   `json:"cutoffDay"`
	DefaultInterestRate string  `json:"defaultInterestRate"`
	PaymentMode         string  `json:"paymentMode"`
	CreatedAt           string  `json:"createdAt"`
	UpdatedAt           string  `json:"updatedAt"`
	DeletedAt           *string `json:"deletedAt,omitempty"`
}

// CreateLoanProvider handles POST /api/v1/loan-providers
func (h *LoanProviderHandler) CreateLoanProvider(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	var req CreateLoanProviderRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	// Parse interest rate (default to 0)
	interestRate := decimal.Zero
	if req.DefaultInterestRate != "" {
		var err error
		interestRate, err = decimal.NewFromString(req.DefaultInterestRate)
		if err != nil {
			return NewValidationError(c, "Invalid interest rate", []ValidationError{
				{Field: "defaultInterestRate", Message: "Must be a valid decimal number"},
			})
		}
	}

	input := service.CreateProviderInput{
		Name:                req.Name,
		CutoffDay:           req.CutoffDay,
		DefaultInterestRate: interestRate,
	}

	provider, err := h.providerService.CreateProvider(workspaceID, input)
	if err != nil {
		if errors.Is(err, domain.ErrLoanProviderNameEmpty) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "name", Message: "Name is required"},
			})
		}
		if errors.Is(err, domain.ErrLoanProviderNameTooLong) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "name", Message: "Name must be 100 characters or less"},
			})
		}
		if errors.Is(err, domain.ErrInvalidCutoffDay) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "cutoffDay", Message: "Cutoff day must be between 1 and 31"},
			})
		}
		if errors.Is(err, domain.ErrInvalidInterestRate) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "defaultInterestRate", Message: "Interest rate must be non-negative"},
			})
		}
		if errors.Is(err, domain.ErrInterestRateTooHigh) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "defaultInterestRate", Message: "Interest rate must be 100% or less"},
			})
		}
		if errors.Is(err, domain.ErrLoanProviderNameExists) {
			return NewConflictError(c, "A loan provider with this name already exists")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to create loan provider")
		return NewInternalError(c, "Failed to create loan provider")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("provider_id", provider.ID).Str("name", provider.Name).Msg("Loan provider created")

	return c.JSON(http.StatusCreated, toLoanProviderResponse(provider))
}

// GetLoanProviders handles GET /api/v1/loan-providers
func (h *LoanProviderHandler) GetLoanProviders(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	providers, err := h.providerService.GetProviders(workspaceID)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get loan providers")
		return NewInternalError(c, "Failed to get loan providers")
	}

	response := make([]LoanProviderResponse, len(providers))
	for i, provider := range providers {
		response[i] = toLoanProviderResponse(provider)
	}

	return c.JSON(http.StatusOK, response)
}

// GetLoanProvider handles GET /api/v1/loan-providers/:id
func (h *LoanProviderHandler) GetLoanProvider(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid loan provider ID", nil)
	}

	provider, err := h.providerService.GetProviderByID(workspaceID, int32(id))
	if err != nil {
		if errors.Is(err, domain.ErrLoanProviderNotFound) {
			return NewNotFoundError(c, "Loan provider not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("provider_id", id).Msg("Failed to get loan provider")
		return NewInternalError(c, "Failed to get loan provider")
	}

	return c.JSON(http.StatusOK, toLoanProviderResponse(provider))
}

// UpdateLoanProvider handles PUT /api/v1/loan-providers/:id
func (h *LoanProviderHandler) UpdateLoanProvider(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid loan provider ID", nil)
	}

	var req UpdateLoanProviderRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	// Parse interest rate (default to 0)
	interestRate := decimal.Zero
	if req.DefaultInterestRate != "" {
		interestRate, err = decimal.NewFromString(req.DefaultInterestRate)
		if err != nil {
			return NewValidationError(c, "Invalid interest rate", []ValidationError{
				{Field: "defaultInterestRate", Message: "Must be a valid decimal number"},
			})
		}
	}

	input := service.UpdateProviderInput{
		Name:                req.Name,
		CutoffDay:           req.CutoffDay,
		DefaultInterestRate: interestRate,
		PaymentMode:         req.PaymentMode,
	}

	provider, err := h.providerService.UpdateProvider(workspaceID, int32(id), input)
	if err != nil {
		if errors.Is(err, domain.ErrLoanProviderNotFound) {
			return NewNotFoundError(c, "Loan provider not found")
		}
		if errors.Is(err, domain.ErrLoanProviderNameEmpty) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "name", Message: "Name is required"},
			})
		}
		if errors.Is(err, domain.ErrLoanProviderNameTooLong) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "name", Message: "Name must be 100 characters or less"},
			})
		}
		if errors.Is(err, domain.ErrInvalidCutoffDay) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "cutoffDay", Message: "Cutoff day must be between 1 and 31"},
			})
		}
		if errors.Is(err, domain.ErrInvalidInterestRate) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "defaultInterestRate", Message: "Interest rate must be non-negative"},
			})
		}
		if errors.Is(err, domain.ErrInterestRateTooHigh) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "defaultInterestRate", Message: "Interest rate must be 100% or less"},
			})
		}
		if errors.Is(err, domain.ErrInvalidPaymentMode) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "paymentMode", Message: "Payment mode must be 'per_item' or 'consolidated_monthly'"},
			})
		}
		if errors.Is(err, domain.ErrLoanProviderNameExists) {
			return NewConflictError(c, "A loan provider with this name already exists")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("provider_id", id).Msg("Failed to update loan provider")
		return NewInternalError(c, "Failed to update loan provider")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("provider_id", provider.ID).Str("name", provider.Name).Msg("Loan provider updated")
	return c.JSON(http.StatusOK, toLoanProviderResponse(provider))
}

// DeleteLoanProvider handles DELETE /api/v1/loan-providers/:id
func (h *LoanProviderHandler) DeleteLoanProvider(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid loan provider ID", nil)
	}

	if err := h.providerService.DeleteProvider(workspaceID, int32(id)); err != nil {
		if errors.Is(err, domain.ErrLoanProviderNotFound) {
			return NewNotFoundError(c, "Loan provider not found")
		}
		if errors.Is(err, domain.ErrLoanProviderHasLoans) {
			return NewConflictError(c, "Cannot delete loan provider with active loans")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("provider_id", id).Msg("Failed to delete loan provider")
		return NewInternalError(c, "Failed to delete loan provider")
	}

	log.Info().Int32("workspace_id", workspaceID).Int("provider_id", id).Msg("Loan provider deleted (soft)")
	return c.NoContent(http.StatusNoContent)
}

// Helper function to convert domain.LoanProvider to LoanProviderResponse
func toLoanProviderResponse(provider *domain.LoanProvider) LoanProviderResponse {
	resp := LoanProviderResponse{
		ID:                  provider.ID,
		WorkspaceID:         provider.WorkspaceID,
		Name:                provider.Name,
		CutoffDay:           provider.CutoffDay,
		DefaultInterestRate: provider.DefaultInterestRate.StringFixed(2),
		PaymentMode:         provider.PaymentMode,
		CreatedAt:           provider.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           provider.UpdatedAt.Format(time.RFC3339),
	}
	if provider.DeletedAt != nil {
		deletedAt := provider.DeletedAt.Format(time.RFC3339)
		resp.DeletedAt = &deletedAt
	}
	return resp
}
