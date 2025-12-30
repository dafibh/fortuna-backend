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

// TransactionHandler handles transaction-related HTTP requests
type TransactionHandler struct {
	transactionService *service.TransactionService
}

// NewTransactionHandler creates a new TransactionHandler
func NewTransactionHandler(transactionService *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{transactionService: transactionService}
}

// CreateTransactionRequest represents the create transaction request body
type CreateTransactionRequest struct {
	AccountID          int32   `json:"accountId"`
	Name               string  `json:"name"`
	Amount             string  `json:"amount"`
	Type               string  `json:"type"`
	Date               *string `json:"date,omitempty"`
	IsPaid             *bool   `json:"isPaid,omitempty"`
	CCSettlementIntent *string `json:"ccSettlementIntent,omitempty"`
	Notes              *string `json:"notes,omitempty"`
}

// TransactionResponse represents a transaction in API responses
type TransactionResponse struct {
	ID                 int32   `json:"id"`
	WorkspaceID        int32   `json:"workspaceId"`
	AccountID          int32   `json:"accountId"`
	Name               string  `json:"name"`
	Amount             string  `json:"amount"`
	Type               string  `json:"type"`
	TransactionDate    string  `json:"transactionDate"`
	IsPaid             bool    `json:"isPaid"`
	CCSettlementIntent *string `json:"ccSettlementIntent,omitempty"`
	Notes              *string `json:"notes,omitempty"`
	CreatedAt          string  `json:"createdAt"`
	UpdatedAt          string  `json:"updatedAt"`
}

// CreateTransaction handles POST /api/v1/transactions
func (h *TransactionHandler) CreateTransaction(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	var req CreateTransactionRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	// Validate accountId early to avoid unnecessary database lookup
	if req.AccountID <= 0 {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "accountId", Message: "Account ID is required"},
		})
	}

	// Parse amount
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return NewValidationError(c, "Invalid amount", []ValidationError{
			{Field: "amount", Message: "Must be a valid decimal number"},
		})
	}

	// Parse transaction date if provided
	var transactionDate *time.Time
	if req.Date != nil && *req.Date != "" {
		parsed, err := time.Parse("2006-01-02", *req.Date)
		if err != nil {
			return NewValidationError(c, "Invalid date", []ValidationError{
				{Field: "date", Message: "Must be in YYYY-MM-DD format"},
			})
		}
		transactionDate = &parsed
	}

	// Parse CC settlement intent if provided
	var ccSettlementIntent *domain.CCSettlementIntent
	if req.CCSettlementIntent != nil && *req.CCSettlementIntent != "" {
		intent := domain.CCSettlementIntent(*req.CCSettlementIntent)
		if intent != domain.CCSettlementThisMonth && intent != domain.CCSettlementNextMonth {
			return NewValidationError(c, "Invalid ccSettlementIntent", []ValidationError{
				{Field: "ccSettlementIntent", Message: "Must be one of: this_month, next_month"},
			})
		}
		ccSettlementIntent = &intent
	}

	input := service.CreateTransactionInput{
		AccountID:          req.AccountID,
		Name:               req.Name,
		Amount:             amount,
		Type:               domain.TransactionType(req.Type),
		TransactionDate:    transactionDate,
		IsPaid:             req.IsPaid,
		CCSettlementIntent: ccSettlementIntent,
		Notes:              req.Notes,
	}

	transaction, err := h.transactionService.CreateTransaction(workspaceID, input)
	if err != nil {
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
				{Field: "type", Message: "Type must be one of: income, expense"},
			})
		}
		if errors.Is(err, domain.ErrAccountNotFound) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "accountId", Message: "Account not found"},
			})
		}
		if errors.Is(err, domain.ErrNotesTooLong) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "notes", Message: "Notes must be 1000 characters or less"},
			})
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to create transaction")
		return NewInternalError(c, "Failed to create transaction")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("transaction_id", transaction.ID).Str("name", transaction.Name).Msg("Transaction created")

	return c.JSON(http.StatusCreated, toTransactionResponse(transaction))
}

// PaginatedTransactionsResponse represents paginated transactions in API responses
type PaginatedTransactionsResponse struct {
	Data       []TransactionResponse `json:"data"`
	Page       int32                 `json:"page"`
	PageSize   int32                 `json:"pageSize"`
	TotalItems int64                 `json:"totalItems"`
	TotalPages int32                 `json:"totalPages"`
}

// GetTransactions handles GET /api/v1/transactions
func (h *TransactionHandler) GetTransactions(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	// Parse filters and pagination
	filters := &domain.TransactionFilters{
		Page:     1,
		PageSize: domain.DefaultPageSize,
	}

	accountIDStr := c.QueryParam("accountId")
	startDateStr := c.QueryParam("startDate")
	endDateStr := c.QueryParam("endDate")
	typeStr := c.QueryParam("type")
	pageStr := c.QueryParam("page")
	pageSizeStr := c.QueryParam("pageSize")

	if accountIDStr != "" {
		var accountID int32
		if _, err := parseIntParam(accountIDStr, &accountID); err != nil {
			return NewValidationError(c, "Invalid accountId", nil)
		}
		filters.AccountID = &accountID
	}

	if startDateStr != "" {
		parsed, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			return NewValidationError(c, "Invalid startDate format (use YYYY-MM-DD)", nil)
		}
		filters.StartDate = &parsed
	}

	if endDateStr != "" {
		parsed, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			return NewValidationError(c, "Invalid endDate format (use YYYY-MM-DD)", nil)
		}
		filters.EndDate = &parsed
	}

	if typeStr != "" {
		transactionType := domain.TransactionType(typeStr)
		if transactionType != domain.TransactionTypeIncome && transactionType != domain.TransactionTypeExpense {
			return NewValidationError(c, "Invalid type (must be 'income' or 'expense')", nil)
		}
		filters.Type = &transactionType
	}

	if pageStr != "" {
		var page int32
		if _, err := parseIntParam(pageStr, &page); err != nil || page < 1 {
			return NewValidationError(c, "Invalid page (must be positive integer)", nil)
		}
		filters.Page = page
	}

	if pageSizeStr != "" {
		var pageSize int32
		if _, err := parseIntParam(pageSizeStr, &pageSize); err != nil || pageSize < 1 {
			return NewValidationError(c, "Invalid pageSize (must be positive integer)", nil)
		}
		if pageSize > domain.MaxPageSize {
			pageSize = domain.MaxPageSize
		}
		filters.PageSize = pageSize
	}

	result, err := h.transactionService.GetTransactions(workspaceID, filters)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get transactions")
		return NewInternalError(c, "Failed to get transactions")
	}

	response := PaginatedTransactionsResponse{
		Data:       make([]TransactionResponse, len(result.Data)),
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalItems: result.TotalItems,
		TotalPages: result.TotalPages,
	}
	for i, transaction := range result.Data {
		response.Data[i] = toTransactionResponse(transaction)
	}

	return c.JSON(http.StatusOK, response)
}

// Helper function to parse int query params with overflow protection
func parseIntParam(s string, out *int32) (bool, error) {
	if s == "" {
		return false, nil
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return false, errors.New("invalid integer")
	}
	*out = int32(v)
	return true, nil
}

// Helper function to convert domain.Transaction to TransactionResponse
func toTransactionResponse(transaction *domain.Transaction) TransactionResponse {
	resp := TransactionResponse{
		ID:              transaction.ID,
		WorkspaceID:     transaction.WorkspaceID,
		AccountID:       transaction.AccountID,
		Name:            transaction.Name,
		Amount:          transaction.Amount.StringFixed(2),
		Type:            string(transaction.Type),
		TransactionDate: transaction.TransactionDate.Format("2006-01-02"),
		IsPaid:          transaction.IsPaid,
		CreatedAt:       transaction.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       transaction.UpdatedAt.Format(time.RFC3339),
	}
	if transaction.CCSettlementIntent != nil {
		intent := string(*transaction.CCSettlementIntent)
		resp.CCSettlementIntent = &intent
	}
	if transaction.Notes != nil {
		resp.Notes = transaction.Notes
	}
	return resp
}
