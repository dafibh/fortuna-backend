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
// @Security BearerAuth
// @Success 200 {object} CCPayableBreakdownResponse
// @Failure 401 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
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

// =====================================================
// V2 SETTLEMENT HANDLERS
// =====================================================

// SettlementRequest is the JSON request for settling CC transactions
type SettlementRequest struct {
	TransactionIDs    []int32 `json:"transactionIds"`
	SourceAccountID   int32   `json:"sourceAccountId"`
	TargetCCAccountID int32   `json:"targetCcAccountId"`
}

// SettlementResponse is the JSON response for a settlement operation
type SettlementResponse struct {
	TransferID   int32  `json:"transferId"`
	SettledCount int    `json:"settledCount"`
	TotalAmount  string `json:"totalAmount"`
	SettledAt    string `json:"settledAt"`
}

// DeferredGroupResponse represents a group of deferred CC transactions for a month
type DeferredGroupResponse struct {
	Year         int                         `json:"year"`
	Month        int                         `json:"month"`
	MonthLabel   string                      `json:"monthLabel"`
	Total        string                      `json:"total"`
	ItemCount    int                         `json:"itemCount"`
	IsOverdue    bool                        `json:"isOverdue"`
	Transactions []DeferredTransactionEntry  `json:"transactions"`
}

// DeferredTransactionEntry represents a deferred CC transaction
type DeferredTransactionEntry struct {
	ID              int32   `json:"id"`
	AccountID       int32   `json:"accountId"`
	AccountName     string  `json:"accountName"`
	Name            string  `json:"name"`
	Amount          string  `json:"amount"`
	TransactionDate string  `json:"transactionDate"`
	OriginYear      int     `json:"originYear"`
	OriginMonth     int     `json:"originMonth"`
}

// SettleCCTransactions settles multiple CC transactions atomically
// @Summary Settle CC transactions
// @Description Atomically settles selected CC transactions and creates a bank transfer
// @Tags cc
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body SettlementRequest true "Settlement request"
// @Success 200 {object} SettlementResponse
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Failure 404 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /cc/settlements [post]
func (h *CCHandler) SettleCCTransactions(c echo.Context) error {
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

	domainReq := &domain.SettlementRequest{
		TransactionIDs:    req.TransactionIDs,
		SourceAccountID:   req.SourceAccountID,
		TargetCCAccountID: req.TargetCCAccountID,
	}

	result, err := h.ccService.SettleCCTransactions(workspaceID, domainReq)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to settle CC transactions")

		if errors.Is(err, domain.ErrAccountNotFound) {
			return NewNotFoundError(c, "Account not found")
		}
		if errors.Is(err, domain.ErrTransactionNotFound) {
			return NewNotFoundError(c, "One or more transactions not found")
		}
		if errors.Is(err, domain.ErrInvalidAccountType) {
			return NewValidationError(c, "Invalid account type", nil)
		}
		if errors.Is(err, domain.ErrInvalidSourceAccount) {
			return NewValidationError(c, "Source account cannot be a credit card", nil)
		}
		if errors.Is(err, domain.ErrInvalidCCState) {
			return NewValidationError(c, "All transactions must be in billed state", nil)
		}
		if errors.Is(err, domain.ErrInvalidSettlementIntent) {
			return NewValidationError(c, "All transactions must have deferred settlement intent", nil)
		}

		return NewInternalError(c, "Failed to settle CC transactions")
	}

	response := SettlementResponse{
		TransferID:   result.TransferID,
		SettledCount: result.SettledCount,
		TotalAmount:  result.TotalAmount.StringFixed(2),
		SettledAt:    result.SettledAt.Format(time.RFC3339),
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Int("settled_count", result.SettledCount).
		Str("total_amount", result.TotalAmount.StringFixed(2)).
		Msg("CC transactions settled")

	return c.JSON(http.StatusOK, response)
}

// OverdueSummaryResponse is the JSON response for overdue CC summary
type OverdueSummaryResponse struct {
	HasOverdue  bool                    `json:"hasOverdue"`
	TotalAmount string                  `json:"totalAmount"`
	ItemCount   int                     `json:"itemCount"`
	Groups      []DeferredGroupResponse `json:"groups"`
}

// GetOverdueSummary returns a summary of overdue CC transactions for the warning banner
// @Summary Get overdue CC summary
// @Description Returns a summary of CC transactions that are overdue (billed 2+ months ago)
// @Tags cc
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} OverdueSummaryResponse
// @Failure 401 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /cc/overdue [get]
func (h *CCHandler) GetOverdueSummary(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	summary, err := h.ccService.GetOverdueCCSummary(workspaceID)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get overdue CC summary")
		return NewInternalError(c, "Failed to get overdue CC summary")
	}

	// Convert groups to response format
	groups := make([]DeferredGroupResponse, len(summary.Groups))
	for i, group := range summary.Groups {
		transactions := make([]DeferredTransactionEntry, len(group.Transactions))
		for j, tx := range group.Transactions {
			accountName := ""
			if tx.AccountName != nil {
				accountName = *tx.AccountName
			}
			transactions[j] = DeferredTransactionEntry{
				ID:              tx.ID,
				AccountID:       tx.AccountID,
				AccountName:     accountName,
				Name:            tx.Name,
				Amount:          tx.Amount.StringFixed(2),
				TransactionDate: tx.TransactionDate.Format("2006-01-02"),
				OriginYear:      tx.OriginYear,
				OriginMonth:     tx.OriginMonth,
			}
		}
		groups[i] = DeferredGroupResponse{
			Year:         group.Year,
			Month:        group.Month,
			MonthLabel:   group.MonthLabel,
			Total:        group.Total.StringFixed(2),
			ItemCount:    group.ItemCount,
			IsOverdue:    group.IsOverdue,
			Transactions: transactions,
		}
	}

	response := OverdueSummaryResponse{
		HasOverdue:  summary.HasOverdue,
		TotalAmount: summary.TotalAmount.StringFixed(2),
		ItemCount:   summary.ItemCount,
		Groups:      groups,
	}

	return c.JSON(http.StatusOK, response)
}

// UpdateOverdueAmountRequest is the JSON request for updating overdue CC amount
type UpdateOverdueAmountRequest struct {
	Amount string `json:"amount"`
}

// UpdateOverdueAmountResponse is the JSON response for updating overdue CC amount
type UpdateOverdueAmountResponse struct {
	ID     int32  `json:"id"`
	Amount string `json:"amount"`
}

// UpdateOverdueAmount updates the amount of an overdue CC transaction
// @Summary Update overdue CC transaction amount
// @Description Updates the amount of an overdue CC transaction (for interest/late fees)
// @Tags cc
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Transaction ID"
// @Param request body UpdateOverdueAmountRequest true "Amount update request"
// @Success 200 {object} UpdateOverdueAmountResponse
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Failure 404 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /cc/overdue/{id}/amount [patch]
func (h *CCHandler) UpdateOverdueAmount(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	// Parse transaction ID from URL
	transactionID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid transaction ID", nil)
	}

	var req UpdateOverdueAmountRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	// Parse and validate amount
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return NewValidationError(c, "Invalid amount format", nil)
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return NewValidationError(c, "Amount must be positive", nil)
	}

	// Update the amount via service
	updatedAmount, err := h.ccService.UpdateOverdueAmount(workspaceID, int32(transactionID), amount)
	if err != nil {
		log.Error().Err(err).
			Int32("workspace_id", workspaceID).
			Int("transaction_id", transactionID).
			Msg("Failed to update overdue CC amount")

		if errors.Is(err, domain.ErrTransactionNotFound) {
			return NewNotFoundError(c, "Transaction not found")
		}
		if errors.Is(err, domain.ErrInvalidCCState) {
			return NewValidationError(c, "Transaction must be overdue to update amount", nil)
		}

		return NewInternalError(c, "Failed to update overdue CC amount")
	}

	response := UpdateOverdueAmountResponse{
		ID:     int32(transactionID),
		Amount: updatedAmount.StringFixed(2),
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Int("transaction_id", transactionID).
		Str("new_amount", updatedAmount.StringFixed(2)).
		Msg("Overdue CC amount updated")

	return c.JSON(http.StatusOK, response)
}

// GetDeferredGroups returns deferred CC transactions grouped by origin month
// @Summary Get deferred CC groups
// @Description Returns all billed-but-unsettled CC transactions grouped by origin month
// @Tags cc
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} DeferredGroupResponse
// @Failure 401 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /cc/deferred [get]
func (h *CCHandler) GetDeferredGroups(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	groups, err := h.ccService.GetDeferredCCGroups(workspaceID)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get deferred CC groups")
		return NewInternalError(c, "Failed to get deferred CC groups")
	}

	// Convert to response format
	response := make([]DeferredGroupResponse, len(groups))
	for i, group := range groups {
		transactions := make([]DeferredTransactionEntry, len(group.Transactions))
		for j, tx := range group.Transactions {
			accountName := ""
			if tx.AccountName != nil {
				accountName = *tx.AccountName
			}
			transactions[j] = DeferredTransactionEntry{
				ID:              tx.ID,
				AccountID:       tx.AccountID,
				AccountName:     accountName,
				Name:            tx.Name,
				Amount:          tx.Amount.StringFixed(2),
				TransactionDate: tx.TransactionDate.Format("2006-01-02"),
				OriginYear:      tx.OriginYear,
				OriginMonth:     tx.OriginMonth,
			}
		}
		response[i] = DeferredGroupResponse{
			Year:         group.Year,
			Month:        group.Month,
			MonthLabel:   group.MonthLabel,
			Total:        group.Total.StringFixed(2),
			ItemCount:    group.ItemCount,
			IsOverdue:    group.IsOverdue,
			Transactions: transactions,
		}
	}

	return c.JSON(http.StatusOK, response)
}
