package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/middleware"
	"github.com/dafibh/fortuna/fortuna-backend/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// SettlementHandler handles settlement HTTP requests
type SettlementHandler struct {
	settlementService *service.SettlementService
}

// NewSettlementHandler creates a new SettlementHandler
func NewSettlementHandler(settlementService *service.SettlementService) *SettlementHandler {
	return &SettlementHandler{
		settlementService: settlementService,
	}
}

// SettlementRequest represents the JSON request for creating a settlement
type SettlementRequest struct {
	TransactionIDs    []int32 `json:"transactionIds"`
	SourceAccountID   int32   `json:"sourceAccountId"`
	TargetCCAccountID int32   `json:"targetCcAccountId"`
}

// SettlementResponse represents the JSON response for a successful settlement
type SettlementResponse struct {
	TransferID   int32  `json:"transferId"`
	SettledCount int    `json:"settledCount"`
	TotalAmount  string `json:"totalAmount"`
	SettledAt    string `json:"settledAt"`
}

// Create creates a new settlement
// @Summary Create settlement
// @Description Atomically settles CC transactions and creates a transfer transaction from source bank account
// @Tags settlements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body SettlementRequest true "Settlement request"
// @Success 201 {object} SettlementResponse
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Failure 404 {object} ProblemDetails
// @Failure 409 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /settlements [post]
func (h *SettlementHandler) Create(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	var req SettlementRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	// Validate required fields
	if len(req.TransactionIDs) == 0 {
		return NewValidationError(c, "At least one transaction ID is required", nil)
	}
	if req.SourceAccountID == 0 {
		return NewValidationError(c, "Source account ID is required", nil)
	}
	if req.TargetCCAccountID == 0 {
		return NewValidationError(c, "Target CC account ID is required", nil)
	}

	// Build domain input
	input := domain.SettlementInput{
		TransactionIDs:    req.TransactionIDs,
		SourceAccountID:   req.SourceAccountID,
		TargetCCAccountID: req.TargetCCAccountID,
	}

	result, err := h.settlementService.Settle(workspaceID, input)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to create settlement")
		return h.handleServiceError(c, err)
	}

	// TODO: Emit WebSocket event for real-time updates
	// wsHub.Broadcast(domain.Event{
	//     Type: "settlement.created",
	//     Payload: result,
	// })

	return c.JSON(http.StatusCreated, SettlementResponse{
		TransferID:   result.TransferID,
		SettledCount: result.SettledCount,
		TotalAmount:  result.TotalAmount.StringFixed(2),
		SettledAt:    result.SettledAt.Format(time.RFC3339),
	})
}

// handleServiceError maps domain errors to appropriate HTTP responses
func (h *SettlementHandler) handleServiceError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrTransactionsNotFound):
		return NewNotFoundError(c, "One or more transactions not found")
	case errors.Is(err, domain.ErrTransactionNotBilled):
		return NewConflictError(c, "All transactions must be in billed state to settle")
	case errors.Is(err, domain.ErrTransactionNotSettleable):
		return NewConflictError(c, "All transactions must be credit card transactions with settlement intent")
	case errors.Is(err, domain.ErrInvalidSourceAccount):
		return NewValidationError(c, "Source account cannot be a credit card", nil)
	case errors.Is(err, domain.ErrInvalidTargetAccount):
		return NewValidationError(c, "Target account must be a credit card", nil)
	case errors.Is(err, domain.ErrAccountNotFound):
		return NewNotFoundError(c, "Account not found")
	case errors.Is(err, domain.ErrEmptySettlement):
		return NewValidationError(c, "At least one transaction must be selected for settlement", nil)
	default:
		return NewInternalError(c, "Settlement failed")
	}
}
