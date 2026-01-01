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

// CCPayableBreakdownResponse is the JSON response for payable breakdown
type CCPayableBreakdownResponse struct {
	ThisMonth      []CCPayableByAccountEntry `json:"thisMonth"`
	NextMonth      []CCPayableByAccountEntry `json:"nextMonth"`
	ThisMonthTotal string                    `json:"thisMonthTotal"`
	NextMonthTotal string                    `json:"nextMonthTotal"`
	GrandTotal     string                    `json:"grandTotal"`
}

// CCPayableByAccountEntry represents a grouped account in the breakdown
type CCPayableByAccountEntry struct {
	AccountID    int32                       `json:"accountId"`
	AccountName  string                      `json:"accountName"`
	Total        string                      `json:"total"`
	Transactions []CCPayableTransactionEntry `json:"transactions"`
}

// CCPayableTransactionEntry represents a single transaction in the breakdown
type CCPayableTransactionEntry struct {
	ID               int32  `json:"id"`
	Name             string `json:"name"`
	Amount           string `json:"amount"`
	TransactionDate  string `json:"transactionDate"`
	SettlementIntent string `json:"settlementIntent"`
	AccountID        int32  `json:"accountId"`
	AccountName      string `json:"accountName"`
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

// GetPayableBreakdown returns CC transactions grouped by settlement intent and account
// @Summary Get CC payable breakdown
// @Description Returns all unpaid CC transactions grouped by this_month/next_month and account
// @Tags cc
// @Accept json
// @Produce json
// @Success 200 {object} CCPayableBreakdownResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /cc/payable/breakdown [get]
func (h *CCHandler) GetPayableBreakdown(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	breakdown, err := h.ccService.GetPayableBreakdown(workspaceID)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get CC payable breakdown")
		return NewInternalError(c, "Failed to get CC payable breakdown")
	}

	// Convert domain types to response types
	response := CCPayableBreakdownResponse{
		ThisMonth:      convertAccountSlice(breakdown.ThisMonth),
		NextMonth:      convertAccountSlice(breakdown.NextMonth),
		ThisMonthTotal: breakdown.ThisMonthTotal.StringFixed(2),
		NextMonthTotal: breakdown.NextMonthTotal.StringFixed(2),
		GrandTotal:     breakdown.GrandTotal.StringFixed(2),
	}

	return c.JSON(http.StatusOK, response)
}

// convertAccountSlice converts domain CCPayableByAccount slice to response entries
func convertAccountSlice(accounts []domain.CCPayableByAccount) []CCPayableByAccountEntry {
	result := make([]CCPayableByAccountEntry, len(accounts))
	for i, acc := range accounts {
		result[i] = CCPayableByAccountEntry{
			AccountID:    acc.AccountID,
			AccountName:  acc.AccountName,
			Total:        acc.Total.StringFixed(2),
			Transactions: convertTransactionSlice(acc.Transactions),
		}
	}
	return result
}

// convertTransactionSlice converts domain CCPayableTransaction slice to response entries
func convertTransactionSlice(transactions []domain.CCPayableTransaction) []CCPayableTransactionEntry {
	result := make([]CCPayableTransactionEntry, len(transactions))
	for i, txn := range transactions {
		result[i] = CCPayableTransactionEntry{
			ID:               txn.ID,
			Name:             txn.Name,
			Amount:           txn.Amount.StringFixed(2),
			TransactionDate:  txn.TransactionDate.Format("2006-01-02"),
			SettlementIntent: string(txn.SettlementIntent),
			AccountID:        txn.AccountID,
			AccountName:      txn.AccountName,
		}
	}
	return result
}

// CreateCCPayment creates a CC payment transaction
// @Summary Create CC payment
// @Description Creates a CC payment (income on CC account, optional expense on source bank account)
// @Tags cc
// @Accept json
// @Produce json
// @Param request body CreateCCPaymentRequest true "CC payment request"
// @Success 201 {object} CCPaymentResponseEntry
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
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
