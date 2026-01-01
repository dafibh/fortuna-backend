package handler

import (
	"net/http"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/middleware"
	"github.com/dafibh/fortuna/fortuna-backend/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
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
