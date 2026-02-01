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
)

// TransactionGroupHandler handles transaction group HTTP requests
type TransactionGroupHandler struct {
	groupService *service.TransactionGroupService
}

// NewTransactionGroupHandler creates a new TransactionGroupHandler
func NewTransactionGroupHandler(groupService *service.TransactionGroupService) *TransactionGroupHandler {
	return &TransactionGroupHandler{
		groupService: groupService,
	}
}

// CreateGroupRequest represents the create group request body
type CreateGroupRequest struct {
	Name           string  `json:"name"`
	TransactionIDs []int32 `json:"transactionIds"`
}

// RenameGroupRequest represents the rename group request body
type RenameGroupRequest struct {
	Name string `json:"name"`
}

// MembershipRequest represents the add/remove transactions request body
type MembershipRequest struct {
	TransactionIDs []int32 `json:"transactionIds"`
}

// GroupDeletedResponse represents the response when a group is auto-deleted
type GroupDeletedResponse struct {
	Deleted bool  `json:"deleted"`
	GroupID int32 `json:"groupId"`
}

// GroupResponse represents a transaction group in API responses
type GroupResponse struct {
	ID             int32  `json:"id"`
	Name           string `json:"name"`
	Month          string `json:"month"`
	AutoDetected   bool   `json:"autoDetected"`
	LoanProviderID *int32 `json:"loanProviderId,omitempty"`
	TotalAmount    string `json:"totalAmount"`
	ChildCount     int32  `json:"childCount"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

// CreateGroup handles POST /api/v1/transaction-groups
func (h *TransactionGroupHandler) CreateGroup(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	var req CreateGroupRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	if len(req.TransactionIDs) == 0 {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "transactionIds", Message: "At least one transaction ID is required"},
		})
	}

	group, err := h.groupService.CreateGroup(workspaceID, req.Name, req.TransactionIDs)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Int32("group_id", group.ID).
		Str("action", "create_group").
		Msg("Transaction group created")

	return c.JSON(http.StatusCreated, toGroupResponse(group))
}

// RenameGroup handles PUT /api/v1/transaction-groups/:id
func (h *TransactionGroupHandler) RenameGroup(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid group ID", nil)
	}

	var req RenameGroupRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	group, err := h.groupService.RenameGroup(workspaceID, int32(id), req.Name)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Int32("group_id", int32(id)).
		Str("action", "rename_group").
		Msg("Transaction group renamed")

	return c.JSON(http.StatusOK, toGroupResponse(group))
}

// AddTransactions handles POST /api/v1/transaction-groups/:id/transactions
func (h *TransactionGroupHandler) AddTransactions(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid group ID", nil)
	}

	var req MembershipRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	if len(req.TransactionIDs) == 0 {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "transactionIds", Message: "At least one transaction ID is required"},
		})
	}

	group, err := h.groupService.AddTransactionsToGroup(workspaceID, int32(id), req.TransactionIDs)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Int32("group_id", int32(id)).
		Str("action", "add_transactions_to_group").
		Msg("Transactions added to group")

	return c.JSON(http.StatusOK, toGroupResponse(group))
}

// RemoveTransactions handles DELETE /api/v1/transaction-groups/:id/transactions
func (h *TransactionGroupHandler) RemoveTransactions(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid group ID", nil)
	}

	var req MembershipRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	if len(req.TransactionIDs) == 0 {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "transactionIds", Message: "At least one transaction ID is required"},
		})
	}

	group, wasDeleted, err := h.groupService.RemoveTransactionsFromGroup(workspaceID, int32(id), req.TransactionIDs)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	if wasDeleted {
		log.Info().
			Int32("workspace_id", workspaceID).
			Int32("group_id", int32(id)).
			Str("action", "auto_delete_empty_group").
			Msg("Group auto-deleted after removing last children")

		return c.JSON(http.StatusOK, GroupDeletedResponse{
			Deleted: true,
			GroupID: int32(id),
		})
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Int32("group_id", int32(id)).
		Str("action", "remove_transactions_from_group").
		Msg("Transactions removed from group")

	return c.JSON(http.StatusOK, toGroupResponse(group))
}

// DeleteGroup handles DELETE /api/v1/transaction-groups/:id?mode=ungroup|delete_all
func (h *TransactionGroupHandler) DeleteGroup(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid group ID", nil)
	}

	mode := c.QueryParam("mode")
	if mode != "ungroup" && mode != "delete_all" {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "mode", Message: "Required: ungroup or delete_all"},
		})
	}

	switch mode {
	case "ungroup":
		result, err := h.groupService.UngroupGroup(workspaceID, int32(id))
		if err != nil {
			return h.handleServiceError(c, err)
		}
		log.Info().
			Int32("workspace_id", workspaceID).
			Int32("group_id", int32(id)).
			Str("action", "ungroup").
			Msg("Transaction group ungrouped")
		return c.JSON(http.StatusOK, result)
	case "delete_all":
		result, err := h.groupService.DeleteGroupWithChildren(workspaceID, int32(id))
		if err != nil {
			return h.handleServiceError(c, err)
		}
		log.Info().
			Int32("workspace_id", workspaceID).
			Int32("group_id", int32(id)).
			Str("action", "delete_all").
			Msg("Transaction group and children deleted")
		return c.JSON(http.StatusOK, result)
	}
	return nil
}

// GetGroupsByMonth handles GET /api/v1/transaction-groups?month=YYYY-MM
func (h *TransactionGroupHandler) GetGroupsByMonth(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	month := c.QueryParam("month")
	if month == "" {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "month", Message: "Month parameter is required (YYYY-MM)"},
		})
	}

	groups, err := h.groupService.GetGroupsByMonth(workspaceID, month)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	responses := make([]GroupResponse, len(groups))
	for i, g := range groups {
		responses[i] = toGroupResponse(g)
	}

	return c.JSON(http.StatusOK, responses)
}

// handleServiceError maps domain errors to RFC 7807 responses
func (h *TransactionGroupHandler) handleServiceError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrMonthBoundaryViolation):
		return c.JSON(http.StatusUnprocessableEntity, ProblemDetails{
			Type:     ErrorTypeMonthBoundary,
			Title:    "Month Boundary Violation",
			Status:   http.StatusUnprocessableEntity,
			Detail:   err.Error(),
			Instance: c.Request().URL.Path,
		})
	case errors.Is(err, domain.ErrAlreadyGrouped):
		return c.JSON(http.StatusConflict, ProblemDetails{
			Type:     ErrorTypeAlreadyGrouped,
			Title:    "Already Grouped",
			Status:   http.StatusConflict,
			Detail:   err.Error(),
			Instance: c.Request().URL.Path,
		})
	case errors.Is(err, domain.ErrTransactionNotInGroup):
		return NewValidationError(c, err.Error(), nil)
	case errors.Is(err, domain.ErrGroupNotFound):
		return NewNotFoundError(c, "Transaction group not found")
	case errors.Is(err, domain.ErrGroupNameEmpty):
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "name", Message: "Group name is required"},
		})
	case errors.Is(err, domain.ErrTransactionNotFound):
		return NewNotFoundError(c, "One or more transactions not found")
	default:
		log.Error().Err(err).Msg("Transaction group operation failed")
		return NewInternalError(c, "Operation failed")
	}
}

// toGroupResponse converts a domain TransactionGroup to a GroupResponse
func toGroupResponse(group *domain.TransactionGroup) GroupResponse {
	resp := GroupResponse{
		ID:           group.ID,
		Name:         group.Name,
		Month:        group.Month,
		AutoDetected: group.AutoDetected,
		TotalAmount:  group.TotalAmount.StringFixed(2),
		ChildCount:   group.ChildCount,
		CreatedAt:    group.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    group.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if group.LoanProviderID != nil {
		resp.LoanProviderID = group.LoanProviderID
	}
	return resp
}
