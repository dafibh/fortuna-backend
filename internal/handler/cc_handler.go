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
	"github.com/shopspring/decimal"
)

// CCHandler handles credit card related HTTP requests
type CCHandler struct {
	ccService *service.CCService
}

// NewCCHandler creates a new CCHandler
func NewCCHandler(ccService *service.CCService) *CCHandler {
	return &CCHandler{
		ccService: ccService,
	}
}

// CreateCCPaymentRequest is the JSON request for creating a CC payment
type CreateCCPaymentRequest struct {
	CCAccountID     int32   `json:"ccAccountId"`
	Amount          string  `json:"amount"`
	TransactionDate string  `json:"transactionDate"`
	SourceAccountID *int32  `json:"sourceAccountId,omitempty"`
	Notes           string  `json:"notes,omitempty"`
}

// CCPaymentResponseEntry is the JSON response for a created CC payment
type CCPaymentResponseEntry struct {
	CCTransaction     *TransactionResponseEntry `json:"ccTransaction"`
	SourceTransaction *TransactionResponseEntry `json:"sourceTransaction,omitempty"`
}

// TransactionResponseEntry is a simplified transaction response
type TransactionResponseEntry struct {
	ID              int32   `json:"id"`
	AccountID       int32   `json:"accountId"`
	Name            string  `json:"name"`
	Amount          string  `json:"amount"`
	Type            string  `json:"type"`
	TransactionDate string  `json:"transactionDate"`
	IsPaid          bool    `json:"isPaid"`
	IsCCPayment     bool    `json:"isCcPayment"`
	Notes           *string `json:"notes,omitempty"`
}

// CreateCCPayment creates a CC payment transaction
// @Summary Create CC payment
// @Description Creates a CC payment (income on CC account, optional expense on source bank account)
// @Tags cc
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateCCPaymentRequest true "CC payment request"
// @Success 201 {object} CCPaymentResponseEntry
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Failure 404 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /cc/payments [post]
func (h *CCHandler) CreateCCPayment(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	var req CreateCCPaymentRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	// Validate required fields
	if req.CCAccountID == 0 {
		return NewValidationError(c, "CC account ID is required", nil)
	}

	// Parse amount
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return NewValidationError(c, "Invalid amount format", nil)
	}

	// Parse transaction date
	transactionDate, err := time.Parse("2006-01-02", req.TransactionDate)
	if err != nil {
		return NewValidationError(c, "Invalid transaction date format (expected YYYY-MM-DD)", nil)
	}

	// Build domain request
	domainReq := &domain.CreateCCPaymentRequest{
		CCAccountID:     req.CCAccountID,
		Amount:          amount,
		TransactionDate: transactionDate,
		SourceAccountID: req.SourceAccountID,
		Notes:           req.Notes,
	}

	result, err := h.ccService.CreateCCPayment(workspaceID, domainReq)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to create CC payment")

		if errors.Is(err, domain.ErrAccountNotFound) {
			return NewNotFoundError(c, "Account not found")
		}
		if errors.Is(err, domain.ErrInvalidAccountType) {
			return NewValidationError(c, "Target account must be a credit card", nil)
		}
		if errors.Is(err, domain.ErrInvalidSourceAccount) {
			return NewValidationError(c, "Source account cannot be a credit card", nil)
		}
		if errors.Is(err, domain.ErrInvalidAmount) {
			return NewValidationError(c, "Amount must be positive", nil)
		}

		return NewInternalError(c, "Failed to create CC payment")
	}

	// Convert to response
	response := CCPaymentResponseEntry{
		CCTransaction: convertDomainTransaction(result.CCTransaction),
	}
	if result.SourceTransaction != nil {
		response.SourceTransaction = convertDomainTransaction(result.SourceTransaction)
	}

	return c.JSON(http.StatusCreated, response)
}

// convertDomainTransaction converts a domain Transaction to a response entry
func convertDomainTransaction(t *domain.Transaction) *TransactionResponseEntry {
	if t == nil {
		return nil
	}
	return &TransactionResponseEntry{
		ID:              t.ID,
		AccountID:       t.AccountID,
		Name:            t.Name,
		Amount:          t.Amount.StringFixed(2),
		Type:            string(t.Type),
		TransactionDate: t.TransactionDate.Format("2006-01-02"),
		IsPaid:          t.IsPaid,
		IsCCPayment:     t.IsCCPayment,
		Notes:           t.Notes,
	}
}
