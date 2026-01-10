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
	recurringService   *service.RecurringService
	projectionService  *service.ProjectionService
}

// NewTransactionHandler creates a new TransactionHandler
func NewTransactionHandler(transactionService *service.TransactionService, recurringService *service.RecurringService, projectionService *service.ProjectionService) *TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
		recurringService:   recurringService,
		projectionService:  projectionService,
	}
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
	CategoryID         *int32  `json:"categoryId,omitempty"`
}

// TransactionResponse represents a transaction in API responses
type TransactionResponse struct {
	ID                     int32   `json:"id"`
	WorkspaceID            int32   `json:"workspaceId"`
	AccountID              int32   `json:"accountId"`
	Name                   string  `json:"name"`
	Amount             string  `json:"amount"`
	Type               string  `json:"type"`
	TransactionDate    string  `json:"transactionDate"`
	IsPaid             bool    `json:"isPaid"`
	CCSettlementIntent *string `json:"ccSettlementIntent,omitempty"`
	Notes              *string `json:"notes,omitempty"`
	TransferPairID     *string `json:"transferPairId,omitempty"`
	CategoryID         *int32  `json:"categoryId,omitempty"`
	CategoryName       *string `json:"categoryName,omitempty"`

	// V2 Fields
	TemplateID  *int32  `json:"templateId,omitempty"`
	Source      string  `json:"source,omitempty"`
	IsProjected bool    `json:"isProjected"`
	CCState     *string `json:"ccState,omitempty"`
	BilledAt    *string `json:"billedAt,omitempty"`
	SettledAt   *string `json:"settledAt,omitempty"`
	AccountName *string `json:"accountName,omitempty"`

	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// CreateTransferRequest represents the create transfer request body
type CreateTransferRequest struct {
	FromAccountID int32   `json:"fromAccountId"`
	ToAccountID   int32   `json:"toAccountId"`
	Amount        string  `json:"amount"`
	Date          *string `json:"date,omitempty"`
	Notes         *string `json:"notes,omitempty"`
}

// TransferResponse represents a transfer in API responses
type TransferResponse struct {
	FromTransaction TransactionResponse `json:"fromTransaction"`
	ToTransaction   TransactionResponse `json:"toTransaction"`
}

// CreateTransaction godoc
// @Summary Create a transaction
// @Description Create a new income or expense transaction
// @Tags transactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateTransactionRequest true "Transaction creation request"
// @Success 201 {object} TransactionResponse
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Router /transactions [post]
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
		CategoryID:         req.CategoryID,
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
		if errors.Is(err, domain.ErrBudgetCategoryNotFound) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "categoryId", Message: "Category not found"},
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

// GetTransactions godoc
// @Summary List transactions
// @Description Get paginated transactions with optional filters
// @Tags transactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param accountId query int false "Filter by account ID"
// @Param startDate query string false "Start date (YYYY-MM-DD)"
// @Param endDate query string false "End date (YYYY-MM-DD)"
// @Param type query string false "Transaction type (income or expense)"
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Items per page" default(20)
// @Success 200 {object} PaginatedTransactionsResponse
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Router /transactions [get]
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

	// On-access projection generation:
	// When user navigates to a specific month, generate projections if needed.
	// This ensures projections exist before fetching transactions.
	now := time.Now()
	currentYear, currentMonth, _ := now.Date()

	// Determine which months are being requested
	if filters.StartDate != nil && filters.EndDate != nil {
		// Generate projections for each month in the date range (if in future)
		if h.projectionService != nil {
			startY, startM, _ := filters.StartDate.Date()
			endY, endM, _ := filters.EndDate.Date()

			// Iterate through each month in the range
			for y, m := startY, startM; y < endY || (y == endY && m <= endM); {
				// Only generate for future months (projections)
				isFuture := y > currentYear || (y == currentYear && m > currentMonth)
				if isFuture {
					if _, err := h.projectionService.GenerateProjectionsForMonth(workspaceID, y, m); err != nil {
						log.Warn().Err(err).
							Int32("workspace_id", workspaceID).
							Int("year", y).
							Int("month", int(m)).
							Msg("Failed to generate projections for month")
					}
				}

				// Move to next month
				m++
				if m > 12 {
					m = 1
					y++
				}
			}
		}

		// For current month, use the legacy recurring generation (creates actual transactions)
		if h.recurringService != nil {
			includesCurrentMonth := true
			endOfCurrentMonth := time.Date(currentYear, currentMonth+1, 0, 23, 59, 59, 0, time.UTC)
			startOfCurrentMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.UTC)

			if filters.StartDate.After(endOfCurrentMonth) || filters.EndDate.Before(startOfCurrentMonth) {
				includesCurrentMonth = false
			}

			if includesCurrentMonth {
				if _, err := h.recurringService.GenerateRecurringTransactions(workspaceID, currentYear, currentMonth); err != nil {
					log.Warn().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to generate recurring transactions")
				}
			}
		}
	} else if h.recurringService != nil {
		// No date filters - generate for current month only (legacy behavior)
		if _, err := h.recurringService.GenerateRecurringTransactions(workspaceID, currentYear, currentMonth); err != nil {
			log.Warn().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to generate recurring transactions")
		}
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

// TogglePaidStatus godoc
// @Summary Toggle transaction paid status
// @Description Toggle the paid/unpaid status of a transaction
// @Tags transactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Transaction ID"
// @Success 200 {object} TransactionResponse
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Failure 404 {object} ProblemDetails
// @Router /transactions/{id}/toggle-paid [patch]
func (h *TransactionHandler) TogglePaidStatus(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid transaction ID", nil)
	}

	transaction, err := h.transactionService.TogglePaidStatus(workspaceID, int32(id))
	if err != nil {
		if errors.Is(err, domain.ErrTransactionNotFound) {
			return NewNotFoundError(c, "Transaction not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("transaction_id", id).Msg("Failed to toggle paid status")
		return NewInternalError(c, "Failed to toggle paid status")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("transaction_id", transaction.ID).Bool("is_paid", transaction.IsPaid).Msg("Transaction paid status toggled")
	return c.JSON(http.StatusOK, toTransactionResponse(transaction))
}

// UpdateSettlementIntentRequest represents the request body for updating settlement intent
type UpdateSettlementIntentRequest struct {
	Intent string `json:"intent"`
}

// UpdateSettlementIntent handles PATCH /api/v1/transactions/:id/settlement-intent
func (h *TransactionHandler) UpdateSettlementIntent(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid transaction ID", nil)
	}

	var req UpdateSettlementIntentRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	if req.Intent == "" {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "intent", Message: "Intent is required"},
		})
	}

	intent := domain.CCSettlementIntent(req.Intent)
	transaction, err := h.transactionService.UpdateSettlementIntent(workspaceID, int32(id), intent)
	if err != nil {
		if errors.Is(err, domain.ErrTransactionNotFound) {
			return NewNotFoundError(c, "Transaction not found")
		}
		if errors.Is(err, domain.ErrInvalidSettlementIntent) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "intent", Message: "Must be one of: this_month, next_month"},
			})
		}
		if errors.Is(err, domain.ErrTransactionAlreadyPaid) {
			return NewValidationError(c, "Cannot change settlement intent for paid transactions", nil)
		}
		if errors.Is(err, domain.ErrSettlementIntentNotApplicable) {
			return NewValidationError(c, "Settlement intent only applies to credit card transactions", nil)
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("transaction_id", id).Msg("Failed to update settlement intent")
		return NewInternalError(c, "Failed to update settlement intent")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("transaction_id", transaction.ID).Str("intent", string(intent)).Msg("Transaction settlement intent updated")
	return c.JSON(http.StatusOK, toTransactionResponse(transaction))
}

// UpdateTransactionRequest represents the request body for updating a transaction
type UpdateTransactionRequest struct {
	AccountID          int32   `json:"accountId"`
	Name               string  `json:"name"`
	Amount             string  `json:"amount"`
	Type               string  `json:"type"`
	Date               string  `json:"date"`
	CCSettlementIntent *string `json:"ccSettlementIntent,omitempty"`
	Notes              *string `json:"notes,omitempty"`
	CategoryID         *int32  `json:"categoryId,omitempty"`
}

// UpdateTransaction godoc
// @Summary Update a transaction
// @Description Update an existing transaction
// @Tags transactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Transaction ID"
// @Param request body CreateTransactionRequest true "Transaction update request"
// @Success 200 {object} TransactionResponse
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Failure 404 {object} ProblemDetails
// @Router /transactions/{id} [put]
func (h *TransactionHandler) UpdateTransaction(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid transaction ID", nil)
	}

	var req UpdateTransactionRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	// Validate accountId early
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

	// Parse date
	transactionDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return NewValidationError(c, "Invalid date", []ValidationError{
			{Field: "date", Message: "Must be in YYYY-MM-DD format"},
		})
	}

	// Parse CC settlement intent if provided
	var ccSettlementIntent *domain.CCSettlementIntent
	if req.CCSettlementIntent != nil && *req.CCSettlementIntent != "" {
		intent := domain.CCSettlementIntent(*req.CCSettlementIntent)
		ccSettlementIntent = &intent
	}

	input := service.UpdateTransactionInput{
		AccountID:          req.AccountID,
		Name:               req.Name,
		Amount:             amount,
		Type:               domain.TransactionType(req.Type),
		TransactionDate:    transactionDate,
		CCSettlementIntent: ccSettlementIntent,
		Notes:              req.Notes,
		CategoryID:         req.CategoryID,
	}

	transaction, err := h.transactionService.UpdateTransaction(workspaceID, int32(id), input)
	if err != nil {
		if errors.Is(err, domain.ErrTransactionNotFound) {
			return NewNotFoundError(c, "Transaction not found")
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
				{Field: "type", Message: "Type must be one of: income, expense"},
			})
		}
		if errors.Is(err, domain.ErrAccountNotFound) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "accountId", Message: "Account not found"},
			})
		}
		if errors.Is(err, domain.ErrInvalidSettlementIntent) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "ccSettlementIntent", Message: "Must be one of: this_month, next_month"},
			})
		}
		if errors.Is(err, domain.ErrNotesTooLong) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "notes", Message: "Notes must be 1000 characters or less"},
			})
		}
		if errors.Is(err, domain.ErrBudgetCategoryNotFound) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "categoryId", Message: "Category not found"},
			})
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("transaction_id", id).Msg("Failed to update transaction")
		return NewInternalError(c, "Failed to update transaction")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("transaction_id", transaction.ID).Msg("Transaction updated")
	return c.JSON(http.StatusOK, toTransactionResponse(transaction))
}

// DeleteTransaction godoc
// @Summary Delete a transaction
// @Description Soft delete a transaction
// @Tags transactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Transaction ID"
// @Success 204 "No Content"
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Failure 404 {object} ProblemDetails
// @Router /transactions/{id} [delete]
func (h *TransactionHandler) DeleteTransaction(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid transaction ID", nil)
	}

	if err := h.transactionService.DeleteTransaction(workspaceID, int32(id)); err != nil {
		if errors.Is(err, domain.ErrTransactionNotFound) {
			return NewNotFoundError(c, "Transaction not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("transaction_id", id).Msg("Failed to delete transaction")
		return NewInternalError(c, "Failed to delete transaction")
	}

	log.Info().Int32("workspace_id", workspaceID).Int("transaction_id", id).Msg("Transaction deleted")
	return c.NoContent(http.StatusNoContent)
}

// CreateTransfer handles POST /api/v1/transfers
func (h *TransactionHandler) CreateTransfer(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	var req CreateTransferRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	// Validate fromAccountId
	if req.FromAccountID <= 0 {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "fromAccountId", Message: "Source account is required"},
		})
	}

	// Validate toAccountId
	if req.ToAccountID <= 0 {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "toAccountId", Message: "Destination account is required"},
		})
	}

	// Parse amount
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return NewValidationError(c, "Invalid amount", []ValidationError{
			{Field: "amount", Message: "Must be a valid decimal number"},
		})
	}

	// Parse date if provided, default to today
	transferDate := time.Now()
	if req.Date != nil && *req.Date != "" {
		parsed, err := time.Parse("2006-01-02", *req.Date)
		if err != nil {
			return NewValidationError(c, "Invalid date", []ValidationError{
				{Field: "date", Message: "Must be in YYYY-MM-DD format"},
			})
		}
		transferDate = parsed
	}

	input := service.CreateTransferInput{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        amount,
		Date:          transferDate,
		Notes:         req.Notes,
	}

	result, err := h.transactionService.CreateTransfer(workspaceID, input)
	if err != nil {
		if errors.Is(err, domain.ErrSameAccountTransfer) {
			return NewValidationError(c, "Cannot transfer to the same account", []ValidationError{
				{Field: "toAccountId", Message: "Must be different from source account"},
			})
		}
		if errors.Is(err, domain.ErrInvalidAmount) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "amount", Message: "Amount must be positive"},
			})
		}
		if errors.Is(err, domain.ErrAccountNotFound) {
			return NewValidationError(c, "Invalid account", nil)
		}
		if errors.Is(err, domain.ErrNotesTooLong) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "notes", Message: "Notes must be 1000 characters or less"},
			})
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to create transfer")
		return NewInternalError(c, "Failed to create transfer")
	}

	log.Info().Int32("workspace_id", workspaceID).Str("pair_id", result.FromTransaction.TransferPairID.String()).Msg("Transfer created")
	return c.JSON(http.StatusCreated, TransferResponse{
		FromTransaction: toTransactionResponse(result.FromTransaction),
		ToTransaction:   toTransactionResponse(result.ToTransaction),
	})
}

// RecentCategoryResponse represents a recently used category in API responses
type RecentCategoryResponse struct {
	ID       int32  `json:"id"`
	Name     string `json:"name"`
	LastUsed string `json:"lastUsed"`
}

// GetRecentlyUsedCategories handles GET /api/v1/transactions/categories/recent
func (h *TransactionHandler) GetRecentlyUsedCategories(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	categories, err := h.transactionService.GetRecentlyUsedCategories(workspaceID)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get recently used categories")
		return NewInternalError(c, "Failed to get recently used categories")
	}

	response := make([]RecentCategoryResponse, len(categories))
	for i, cat := range categories {
		response[i] = RecentCategoryResponse{
			ID:       cat.ID,
			Name:     cat.Name,
			LastUsed: cat.LastUsed.Format(time.RFC3339),
		}
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

// =====================================================
// V2 CC LIFECYCLE HANDLERS
// =====================================================

// CCTransactionResponse represents a CC transaction with account info in API responses
type CCTransactionResponse struct {
	ID              int32   `json:"id"`
	WorkspaceID     int32   `json:"workspaceId"`
	AccountID       int32   `json:"accountId"`
	AccountName     string  `json:"accountName"`
	Name            string  `json:"name"`
	Amount          string  `json:"amount"`
	TransactionDate string  `json:"transactionDate"`
	CCState         string  `json:"ccState"`
	BilledAt        *string `json:"billedAt,omitempty"`
	SettledAt       *string `json:"settledAt,omitempty"`
	Source          string  `json:"source"`
	IsProjected     bool    `json:"isProjected"`
}

// CCMetricsResponse represents CC metrics in API responses
type CCMetricsResponse struct {
	TotalPurchases string `json:"totalPurchases"` // All CC transactions
	Outstanding    string `json:"outstanding"`    // Billed + deferred, not yet settled
	Pending        string `json:"pending"`        // Not yet billed
}

// ToggleCCBilled handles PATCH /api/v1/transactions/:id/toggle-billed
func (h *TransactionHandler) ToggleCCBilled(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid transaction ID", nil)
	}

	transaction, err := h.transactionService.ToggleCCBilled(workspaceID, int32(id))
	if err != nil {
		if errors.Is(err, domain.ErrTransactionNotFound) {
			return NewNotFoundError(c, "Transaction not found")
		}
		if errors.Is(err, domain.ErrNotCCTransaction) {
			return NewValidationError(c, "This operation only applies to credit card transactions", nil)
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("transaction_id", id).Msg("Failed to toggle CC billed status")
		return NewInternalError(c, "Failed to toggle billed status")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("transaction_id", transaction.ID).Str("cc_state", string(*transaction.CCState)).Msg("CC transaction billed status toggled")
	return c.JSON(http.StatusOK, toTransactionResponse(transaction))
}

// GetPendingCCTransactions handles GET /api/v1/cc/pending
func (h *TransactionHandler) GetPendingCCTransactions(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	yearStr := c.QueryParam("year")
	monthStr := c.QueryParam("month")

	if yearStr == "" || monthStr == "" {
		return NewValidationError(c, "year and month query parameters are required", nil)
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return NewValidationError(c, "Invalid year", nil)
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		return NewValidationError(c, "Invalid month (must be 1-12)", nil)
	}

	transactions, err := h.transactionService.GetPendingCCByMonth(workspaceID, year, month)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("year", year).Int("month", month).Msg("Failed to get pending CC transactions")
		return NewInternalError(c, "Failed to get pending CC transactions")
	}

	response := make([]CCTransactionResponse, len(transactions))
	for i, tx := range transactions {
		response[i] = toCCTransactionResponse(tx)
	}

	return c.JSON(http.StatusOK, response)
}

// GetBilledCCTransactions handles GET /api/v1/cc/billed
func (h *TransactionHandler) GetBilledCCTransactions(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	yearStr := c.QueryParam("year")
	monthStr := c.QueryParam("month")

	if yearStr == "" || monthStr == "" {
		return NewValidationError(c, "year and month query parameters are required", nil)
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return NewValidationError(c, "Invalid year", nil)
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		return NewValidationError(c, "Invalid month (must be 1-12)", nil)
	}

	transactions, err := h.transactionService.GetBilledCCByMonth(workspaceID, year, month)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("year", year).Int("month", month).Msg("Failed to get billed CC transactions")
		return NewInternalError(c, "Failed to get billed CC transactions")
	}

	response := make([]CCTransactionResponse, len(transactions))
	for i, tx := range transactions {
		response[i] = toCCTransactionResponse(tx)
	}

	return c.JSON(http.StatusOK, response)
}

// GetCCMetrics handles GET /api/v1/cc/metrics
func (h *TransactionHandler) GetCCMetrics(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	yearStr := c.QueryParam("year")
	monthStr := c.QueryParam("month")

	if yearStr == "" || monthStr == "" {
		return NewValidationError(c, "year and month query parameters are required", nil)
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return NewValidationError(c, "Invalid year", nil)
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		return NewValidationError(c, "Invalid month (must be 1-12)", nil)
	}

	metrics, err := h.transactionService.GetCCMetricsByMonth(workspaceID, year, month)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("year", year).Int("month", month).Msg("Failed to get CC metrics")
		return NewInternalError(c, "Failed to get CC metrics")
	}

	response := CCMetricsResponse{
		TotalPurchases: metrics.TotalPurchases.StringFixed(2),
		Outstanding:    metrics.Outstanding.StringFixed(2),
		Pending:        metrics.Pending.StringFixed(2),
	}

	return c.JSON(http.StatusOK, response)
}

// Helper function to convert domain.CCTransactionWithAccount to CCTransactionResponse
func toCCTransactionResponse(tx *domain.CCTransactionWithAccount) CCTransactionResponse {
	resp := CCTransactionResponse{
		ID:              tx.ID,
		WorkspaceID:     tx.WorkspaceID,
		AccountID:       tx.AccountID,
		Name:            tx.Name,
		Amount:          tx.Amount.StringFixed(2),
		TransactionDate: tx.TransactionDate.Format("2006-01-02"),
		Source:          string(tx.Source),
		IsProjected:     tx.IsProjected,
	}
	if tx.AccountName != nil {
		resp.AccountName = *tx.AccountName
	}
	if tx.CCState != nil {
		resp.CCState = string(*tx.CCState)
	}
	if tx.BilledAt != nil {
		billedAt := tx.BilledAt.Format(time.RFC3339)
		resp.BilledAt = &billedAt
	}
	if tx.SettledAt != nil {
		settledAt := tx.SettledAt.Format(time.RFC3339)
		resp.SettledAt = &settledAt
	}
	return resp
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
	if transaction.TransferPairID != nil {
		pairID := transaction.TransferPairID.String()
		resp.TransferPairID = &pairID
	}
	if transaction.CategoryID != nil {
		resp.CategoryID = transaction.CategoryID
	}
	if transaction.CategoryName != nil {
		resp.CategoryName = transaction.CategoryName
	}
	// V2 fields
	if transaction.TemplateID != nil {
		resp.TemplateID = transaction.TemplateID
	}
	resp.Source = string(transaction.Source)
	resp.IsProjected = transaction.IsProjected
	if transaction.CCState != nil {
		state := string(*transaction.CCState)
		resp.CCState = &state
	}
	if transaction.BilledAt != nil {
		billedAt := transaction.BilledAt.Format(time.RFC3339)
		resp.BilledAt = &billedAt
	}
	if transaction.SettledAt != nil {
		settledAt := transaction.SettledAt.Format(time.RFC3339)
		resp.SettledAt = &settledAt
	}
	if transaction.AccountName != nil {
		resp.AccountName = transaction.AccountName
	}
	return resp
}
