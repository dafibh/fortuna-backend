package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/middleware"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

// RecurringTemplateHandler handles recurring template HTTP requests (v2)
type RecurringTemplateHandler struct {
	service domain.RecurringTemplateService
}

// NewRecurringTemplateHandler creates a new RecurringTemplateHandler
func NewRecurringTemplateHandler(service domain.RecurringTemplateService) *RecurringTemplateHandler {
	return &RecurringTemplateHandler{
		service: service,
	}
}

// CreateTemplateRequest represents the create recurring template request body
type CreateTemplateRequest struct {
	Description       string  `json:"description"`
	Amount            string  `json:"amount"`
	CategoryID        int32   `json:"categoryId"`
	AccountID         int32   `json:"accountId"`
	Frequency         string  `json:"frequency"`
	StartDate         string  `json:"startDate"`
	EndDate           *string `json:"endDate,omitempty"`
	LinkTransactionID *int32  `json:"linkTransactionId,omitempty"`
}

// UpdateTemplateRequest represents the update recurring template request body
type UpdateTemplateRequest struct {
	Description string  `json:"description"`
	Amount      string  `json:"amount"`
	CategoryID  int32   `json:"categoryId"`
	AccountID   int32   `json:"accountId"`
	Frequency   string  `json:"frequency"`
	StartDate   string  `json:"startDate"`
	EndDate     *string `json:"endDate,omitempty"`
}

// TemplateResponse represents a recurring template in API responses
type TemplateResponse struct {
	ID          int32   `json:"id"`
	WorkspaceID int32   `json:"workspaceId"`
	Description string  `json:"description"`
	Amount      string  `json:"amount"`
	CategoryID  int32   `json:"categoryId"`
	AccountID   int32   `json:"accountId"`
	Frequency   string  `json:"frequency"`
	StartDate   string  `json:"startDate"`
	EndDate     *string `json:"endDate,omitempty"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

// TemplateListResponse represents the list response
type TemplateListResponse struct {
	Data []TemplateResponse `json:"data"`
}

// CreateTemplate handles POST /api/v1/recurring-templates
// @Summary Create a recurring template
// @Description Creates a new recurring template with projection generation
// @Tags Recurring Templates
// @Accept json
// @Produce json
// @Param template body CreateTemplateRequest true "Template data"
// @Success 201 {object} TemplateResponse
// @Failure 400 {object} ProblemDetail
// @Failure 401 {object} ProblemDetail
// @Security BearerAuth
// @Router /recurring-templates [post]
func (h *RecurringTemplateHandler) CreateTemplate(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	var req CreateTemplateRequest
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

	// Parse start date
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return NewValidationError(c, "Invalid start date", []ValidationError{
			{Field: "startDate", Message: "Must be in YYYY-MM-DD format"},
		})
	}

	input := domain.CreateRecurringTemplateInput{
		WorkspaceID:       workspaceID,
		Description:       req.Description,
		Amount:            amount,
		CategoryID:        req.CategoryID,
		AccountID:         req.AccountID,
		Frequency:         req.Frequency,
		StartDate:         startDate,
		LinkTransactionID: req.LinkTransactionID,
	}

	// Parse optional end date
	if req.EndDate != nil && *req.EndDate != "" {
		endDate, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			return NewValidationError(c, "Invalid end date", []ValidationError{
				{Field: "endDate", Message: "Must be in YYYY-MM-DD format"},
			})
		}
		input.EndDate = &endDate
	}

	template, err := h.service.CreateTemplate(workspaceID, input)
	if err != nil {
		return h.handleServiceError(c, err, workspaceID, "create recurring template")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("template_id", template.ID).Str("description", template.Description).Msg("Recurring template created")

	return c.JSON(http.StatusCreated, toTemplateResponse(template))
}

// ListTemplates handles GET /api/v1/recurring-templates
// @Summary List all recurring templates
// @Description Retrieves all recurring templates for the workspace
// @Tags Recurring Templates
// @Produce json
// @Success 200 {object} TemplateListResponse
// @Failure 401 {object} ProblemDetail
// @Security BearerAuth
// @Router /recurring-templates [get]
func (h *RecurringTemplateHandler) ListTemplates(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	templates, err := h.service.ListTemplates(workspaceID)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to list recurring templates")
		return NewInternalError(c, "Failed to list recurring templates")
	}

	response := make([]TemplateResponse, len(templates))
	for i, t := range templates {
		response[i] = toTemplateResponse(t)
	}

	return c.JSON(http.StatusOK, TemplateListResponse{Data: response})
}

// GetTemplate handles GET /api/v1/recurring-templates/:id
// @Summary Get a recurring template
// @Description Retrieves a single recurring template by ID
// @Tags Recurring Templates
// @Produce json
// @Param id path int true "Template ID"
// @Success 200 {object} TemplateResponse
// @Failure 401 {object} ProblemDetail
// @Failure 404 {object} ProblemDetail
// @Security BearerAuth
// @Router /recurring-templates/{id} [get]
func (h *RecurringTemplateHandler) GetTemplate(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid template ID", nil)
	}

	template, err := h.service.GetTemplate(workspaceID, int32(id))
	if err != nil {
		if errors.Is(err, domain.ErrRecurringTemplateNotFound) {
			return NewNotFoundError(c, "Recurring template not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("template_id", id).Msg("Failed to get recurring template")
		return NewInternalError(c, "Failed to get recurring template")
	}

	return c.JSON(http.StatusOK, toTemplateResponse(template))
}

// UpdateTemplate handles PUT /api/v1/recurring-templates/:id
// @Summary Update a recurring template
// @Description Updates a recurring template and recalculates projections
// @Tags Recurring Templates
// @Accept json
// @Produce json
// @Param id path int true "Template ID"
// @Param template body UpdateTemplateRequest true "Updated template data"
// @Success 200 {object} TemplateResponse
// @Failure 400 {object} ProblemDetail
// @Failure 401 {object} ProblemDetail
// @Failure 404 {object} ProblemDetail
// @Security BearerAuth
// @Router /recurring-templates/{id} [put]
func (h *RecurringTemplateHandler) UpdateTemplate(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid template ID", nil)
	}

	var req UpdateTemplateRequest
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

	// Parse start date
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return NewValidationError(c, "Invalid start date", []ValidationError{
			{Field: "startDate", Message: "Must be in YYYY-MM-DD format"},
		})
	}

	input := domain.UpdateRecurringTemplateInput{
		Description: req.Description,
		Amount:      amount,
		CategoryID:  req.CategoryID,
		AccountID:   req.AccountID,
		Frequency:   req.Frequency,
		StartDate:   startDate,
	}

	// Parse optional end date
	if req.EndDate != nil && *req.EndDate != "" {
		endDate, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			return NewValidationError(c, "Invalid end date", []ValidationError{
				{Field: "endDate", Message: "Must be in YYYY-MM-DD format"},
			})
		}
		input.EndDate = &endDate
	}

	template, err := h.service.UpdateTemplate(workspaceID, int32(id), input)
	if err != nil {
		return h.handleServiceError(c, err, workspaceID, "update recurring template")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("template_id", template.ID).Str("description", template.Description).Msg("Recurring template updated")

	return c.JSON(http.StatusOK, toTemplateResponse(template))
}

// DeleteTemplate handles DELETE /api/v1/recurring-templates/:id
// @Summary Delete a recurring template
// @Description Deletes a recurring template and all its projections
// @Tags Recurring Templates
// @Param id path int true "Template ID"
// @Success 204 "No Content"
// @Failure 401 {object} ProblemDetail
// @Failure 404 {object} ProblemDetail
// @Security BearerAuth
// @Router /recurring-templates/{id} [delete]
func (h *RecurringTemplateHandler) DeleteTemplate(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid template ID", nil)
	}

	if err := h.service.DeleteTemplate(workspaceID, int32(id)); err != nil {
		if errors.Is(err, domain.ErrRecurringTemplateNotFound) {
			return NewNotFoundError(c, "Recurring template not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("template_id", id).Msg("Failed to delete recurring template")
		return NewInternalError(c, "Failed to delete recurring template")
	}

	log.Info().Int32("workspace_id", workspaceID).Int("template_id", id).Msg("Recurring template deleted")
	return c.NoContent(http.StatusNoContent)
}

// handleServiceError handles common service errors
func (h *RecurringTemplateHandler) handleServiceError(c echo.Context, err error, workspaceID int32, operation string) error {
	if errors.Is(err, domain.ErrRecurringTemplateNotFound) {
		return NewNotFoundError(c, "Recurring template not found")
	}
	if errors.Is(err, domain.ErrNameRequired) {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "description", Message: "Description is required"},
		})
	}
	if errors.Is(err, domain.ErrNameTooLong) {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "description", Message: "Description must be 255 characters or less"},
		})
	}
	if errors.Is(err, domain.ErrInvalidAmount) {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "amount", Message: "Amount must be positive"},
		})
	}
	if errors.Is(err, domain.ErrInvalidFrequency) {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "frequency", Message: "Frequency must be 'monthly'"},
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

// toTemplateResponse converts domain.RecurringTemplate to TemplateResponse
func toTemplateResponse(t *domain.RecurringTemplate) TemplateResponse {
	resp := TemplateResponse{
		ID:          t.ID,
		WorkspaceID: t.WorkspaceID,
		Description: t.Description,
		Amount:      t.Amount.StringFixed(2),
		CategoryID:  t.CategoryID,
		AccountID:   t.AccountID,
		Frequency:   t.Frequency,
		StartDate:   t.StartDate.Format("2006-01-02"),
		CreatedAt:   t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   t.UpdatedAt.Format(time.RFC3339),
	}
	if t.EndDate != nil {
		endDate := t.EndDate.Format("2006-01-02")
		resp.EndDate = &endDate
	}
	return resp
}
