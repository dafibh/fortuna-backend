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
	transactionService      *service.TransactionService
	transactionGroupService *service.TransactionGroupService
}

// NewTransactionHandler creates a new TransactionHandler
func NewTransactionHandler(transactionService *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
	}
}

// SetTransactionGroupService sets the group service for auto-detection on GetTransactions
func (h *TransactionHandler) SetTransactionGroupService(groupService *service.TransactionGroupService) {
	h.transactionGroupService = groupService
}

// CreateTransactionRequest represents the create transaction request body
type CreateTransactionRequest struct {
	AccountID        int32   `json:"accountId"`
	Name             string  `json:"name"`
	Amount           string  `json:"amount"`
	Type             string  `json:"type"`
	Date             *string `json:"date,omitempty"`
	IsPaid           *bool   `json:"isPaid,omitempty"`
	Notes            *string `json:"notes,omitempty"`
	CategoryID       *int32  `json:"categoryId,omitempty"`
	SettlementIntent *string `json:"settlementIntent,omitempty"` // v2: "immediate" or "deferred"
}

// TransactionResponse represents a transaction in API responses
type TransactionResponse struct {
	ID              int32   `json:"id"`
	WorkspaceID     int32   `json:"workspaceId"`
	AccountID       int32   `json:"accountId"`
	Name            string  `json:"name"`
	Amount          string  `json:"amount"`
	Type            string  `json:"type"`
	TransactionDate string  `json:"transactionDate"`
	IsPaid          bool    `json:"isPaid"`
	Notes           *string `json:"notes,omitempty"`
	TransferPairID  *string `json:"transferPairId,omitempty"`
	CategoryID      *int32  `json:"categoryId,omitempty"`
	CategoryName    *string `json:"categoryName,omitempty"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`

	// Recurring/Projection fields
	Source      string `json:"source"`               // "manual", "recurring", or "import"
	TemplateID  *int32 `json:"templateId,omitempty"` // ID of recurring template that generated this
	IsProjected bool   `json:"isProjected"`          // true if this is a projected (not yet actual) transaction
	IsModified  bool   `json:"isModified"`           // true if projected instance differs from template

	// CC Lifecycle fields (v2 simplified - ccState computed from isPaid and billedAt)
	CCState          *string `json:"ccState,omitempty"`          // Computed: "pending", "billed", or "settled"
	BilledAt         *string `json:"billedAt,omitempty"`         // Timestamp when marked as billed
	SettlementIntent *string `json:"settlementIntent,omitempty"` // "immediate" or "deferred"

	// Transaction Grouping fields
	GroupID   *int32  `json:"groupId,omitempty"`   // ID of the transaction group
	GroupName *string `json:"groupName,omitempty"` // Name of the transaction group
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

	// Parse settlement intent if provided (v2)
	var settlementIntent *domain.SettlementIntent
	if req.SettlementIntent != nil && *req.SettlementIntent != "" {
		intent := domain.SettlementIntent(*req.SettlementIntent)
		if intent != domain.SettlementIntentImmediate && intent != domain.SettlementIntentDeferred {
			return NewValidationError(c, "Invalid settlementIntent", []ValidationError{
				{Field: "settlementIntent", Message: "Must be one of: immediate, deferred"},
			})
		}
		settlementIntent = &intent
	}

	input := service.CreateTransactionInput{
		AccountID:        req.AccountID,
		Name:             req.Name,
		Amount:           amount,
		Type:             domain.TransactionType(req.Type),
		TransactionDate:  transactionDate,
		IsPaid:           req.IsPaid,
		Notes:            req.Notes,
		CategoryID:       req.CategoryID,
		SettlementIntent: settlementIntent,
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
// @Param month query string false "Filter by month (YYYY-MM format, overrides startDate/endDate)"
// @Param startDate query string false "Start date (YYYY-MM-DD)"
// @Param endDate query string false "End date (YYYY-MM-DD)"
// @Param type query string false "Transaction type (income or expense)"
// @Param ccStatus query string false "Filter by CC status (pending, billed, or settled)"
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
	monthStr := c.QueryParam("month")
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

	// Parse month parameter (YYYY-MM format) - overrides startDate/endDate
	if monthStr != "" {
		parsed, err := time.Parse("2006-01", monthStr)
		if err != nil {
			return NewValidationError(c, "Invalid month format (use YYYY-MM)", nil)
		}
		// Set start and end to cover the entire month
		startOfMonth := time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.UTC)
		endOfMonth := startOfMonth.AddDate(0, 1, -1) // Last day of month
		filters.StartDate = &startOfMonth
		filters.EndDate = &endOfMonth

		// Fire-and-forget: auto-detect BNPL groups for this month
		if h.transactionGroupService != nil {
			_ = h.transactionGroupService.EnsureAutoGroups(workspaceID, monthStr)
		}
	} else {
		// Use startDate/endDate if month not provided
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
	}

	if typeStr != "" {
		transactionType := domain.TransactionType(typeStr)
		if transactionType != domain.TransactionTypeIncome && transactionType != domain.TransactionTypeExpense {
			return NewValidationError(c, "Invalid type (must be 'income' or 'expense')", nil)
		}
		filters.Type = &transactionType
	}

	ccStatusStr := c.QueryParam("ccStatus")
	if ccStatusStr != "" {
		ccStatus := domain.CCState(ccStatusStr)
		if ccStatus != domain.CCStatePending && ccStatus != domain.CCStateBilled && ccStatus != domain.CCStateSettled {
			return NewValidationError(c, "Invalid ccStatus (must be 'pending', 'billed', or 'settled')", nil)
		}
		filters.CCStatus = &ccStatus
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

	// Enrich projected transactions with modification status
	h.transactionService.EnrichWithModificationStatus(workspaceID, result.Data)

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

// ToggleBilled godoc
// @Summary Toggle CC transaction billed status
// @Description Toggle the billed status of a CC transaction (pending <-> billed)
// @Tags transactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Transaction ID"
// @Success 200 {object} TransactionResponse
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Failure 404 {object} ProblemDetails
// @Failure 409 {object} ProblemDetails
// @Router /transactions/{id}/toggle-billed [patch]
func (h *TransactionHandler) ToggleBilled(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid transaction ID", nil)
	}

	transaction, err := h.transactionService.ToggleBilled(workspaceID, int32(id))
	if err != nil {
		if errors.Is(err, domain.ErrTransactionNotFound) {
			return NewNotFoundError(c, "Transaction not found")
		}
		if errors.Is(err, domain.ErrNotCCTransaction) {
			return NewValidationError(c, "Transaction is not a credit card transaction", nil)
		}
		if errors.Is(err, domain.ErrInvalidCCStateTransition) {
			return NewConflictError(c, "Cannot toggle billed status for settled transactions")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("transaction_id", id).Msg("Failed to toggle billed status")
		return NewInternalError(c, "Failed to toggle billed status")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("transaction_id", transaction.ID).Msg("Transaction billed status toggled")
	return c.JSON(http.StatusOK, toTransactionResponse(transaction))
}

// UpdateTransactionRequest represents the request body for updating a transaction
type UpdateTransactionRequest struct {
	AccountID        int32   `json:"accountId"`
	Name             string  `json:"name"`
	Amount           string  `json:"amount"`
	Type             string  `json:"type"`
	Date             string  `json:"date"`
	Notes            *string `json:"notes,omitempty"`
	CategoryID       *int32  `json:"categoryId,omitempty"`
	SettlementIntent *string `json:"settlementIntent,omitempty"` // "immediate" or "deferred"
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

	// Parse settlement intent if provided
	var settlementIntent *domain.SettlementIntent
	if req.SettlementIntent != nil && *req.SettlementIntent != "" {
		intent := domain.SettlementIntent(*req.SettlementIntent)
		if intent != domain.SettlementIntentImmediate && intent != domain.SettlementIntentDeferred {
			return NewValidationError(c, "Invalid settlementIntent", []ValidationError{
				{Field: "settlementIntent", Message: "Must be one of: immediate, deferred"},
			})
		}
		settlementIntent = &intent
	}

	input := service.UpdateTransactionInput{
		AccountID:        req.AccountID,
		Name:             req.Name,
		Amount:           amount,
		Type:             domain.TransactionType(req.Type),
		TransactionDate:  transactionDate,
		Notes:            req.Notes,
		CategoryID:       req.CategoryID,
		SettlementIntent: settlementIntent,
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

// Helper function to convert domain.Transaction to TransactionResponse
func toTransactionResponse(transaction *domain.Transaction) TransactionResponse {
	// Default source to "manual" if not set
	source := transaction.Source
	if source == "" {
		source = "manual"
	}

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

		// Recurring/Projection fields (v2)
		Source:      source,
		TemplateID:  transaction.TemplateID,
		IsProjected: transaction.IsProjected,
		IsModified:  transaction.IsModified,
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
	// CC Lifecycle fields (v2)
	if transaction.CCState != nil {
		ccState := string(*transaction.CCState)
		resp.CCState = &ccState
	}
	if transaction.BilledAt != nil {
		billedAt := transaction.BilledAt.Format(time.RFC3339)
		resp.BilledAt = &billedAt
	}
	// SettledAt removed in v2 - settlement status is determined by isPaid
	if transaction.SettlementIntent != nil {
		settlementIntent := string(*transaction.SettlementIntent)
		resp.SettlementIntent = &settlementIntent
	}
	// Transaction Grouping fields
	if transaction.GroupID != nil {
		resp.GroupID = transaction.GroupID
	}
	if transaction.GroupName != nil {
		resp.GroupName = transaction.GroupName
	}
	return resp
}

// CCMetricsResponse represents CC metrics in API responses
type CCMetricsResponse struct {
	Pending     string `json:"pending"`     // Sum of pending CC transactions
	Outstanding string `json:"outstanding"` // Sum of billed CC transactions with deferred intent (balance to settle)
	Purchases   string `json:"purchases"`   // Sum of all CC transactions (pending + billed + settled)
}

// GetCCMetrics handles GET /api/v1/transactions/cc-metrics
// @Summary Get CC metrics
// @Description Get CC metrics (pending, billed, month total) for the current or specified month
// @Tags transactions
// @Produce json
// @Security BearerAuth
// @Param month query string false "Month in YYYY-MM format (defaults to current month)"
// @Success 200 {object} CCMetricsResponse
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Router /transactions/cc-metrics [get]
func (h *TransactionHandler) GetCCMetrics(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	// Parse month parameter, default to current month
	monthStr := c.QueryParam("month")
	var month time.Time
	if monthStr != "" {
		parsed, err := time.Parse("2006-01", monthStr)
		if err != nil {
			return NewValidationError(c, "Invalid month format. Use YYYY-MM", nil)
		}
		month = parsed
	} else {
		month = time.Now()
	}

	metrics, err := h.transactionService.GetCCMetrics(workspaceID, month)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Str("month", monthStr).Msg("Failed to get CC metrics")
		return NewInternalError(c, "Failed to get CC metrics")
	}

	return c.JSON(http.StatusOK, CCMetricsResponse{
		Pending:     metrics.Pending.StringFixed(2),
		Outstanding: metrics.Outstanding.StringFixed(2),
		Purchases:   metrics.Purchases.StringFixed(2),
	})
}

// BatchToggleBilledRequest represents the request body for batch toggling billed status
type BatchToggleBilledRequest struct {
	IDs []int32 `json:"ids"`
}

// BatchToggleBilledResponse represents the response for batch toggle operation
type BatchToggleBilledResponse struct {
	Updated []TransactionResponse `json:"updated"`
	Count   int                   `json:"count"`
}

// BatchToggleBilled godoc
// @Summary Batch toggle CC transactions to billed status
// @Description Toggle multiple CC transactions from pending to billed in a single request
// @Tags transactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body BatchToggleBilledRequest true "Transaction IDs to toggle"
// @Success 200 {object} BatchToggleBilledResponse
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Router /transactions/batch-toggle-billed [post]
func (h *TransactionHandler) BatchToggleBilled(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	var req BatchToggleBilledRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	if len(req.IDs) == 0 {
		return NewValidationError(c, "At least one transaction ID is required", nil)
	}

	if len(req.IDs) > 100 {
		return NewValidationError(c, "Maximum 100 transactions per batch", nil)
	}

	transactions, err := h.transactionService.BatchToggleToBilled(workspaceID, req.IDs)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("count", len(req.IDs)).Msg("Failed to batch toggle billed status")
		return NewInternalError(c, "Failed to batch toggle billed status")
	}

	response := BatchToggleBilledResponse{
		Updated: make([]TransactionResponse, len(transactions)),
		Count:   len(transactions),
	}
	for i, tx := range transactions {
		response.Updated[i] = toTransactionResponse(tx)
	}

	log.Info().Int32("workspace_id", workspaceID).Int("count", len(transactions)).Msg("Batch toggle billed completed")
	return c.JSON(http.StatusOK, response)
}

// DeferredGroup represents a group of deferred transactions by month
type DeferredGroup struct {
	Month        string                `json:"month"`        // "2026-01"
	MonthLabel   string                `json:"monthLabel"`   // "January"
	TotalAmount  string                `json:"totalAmount"`
	ItemCount    int                   `json:"itemCount"`
	Transactions []TransactionResponse `json:"transactions"`
}

// GetDeferredToSettle returns all billed+deferred transactions grouped by month
// @Summary Get deferred transactions to settle
// @Description Returns all billed, deferred CC transactions that need settlement, grouped by month
// @Tags transactions
// @Produce json
// @Security BearerAuth
// @Success 200 {array} DeferredGroup
// @Failure 401 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /transactions/deferred-to-settle [get]
func (h *TransactionHandler) GetDeferredToSettle(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	transactions, err := h.transactionService.GetDeferredForSettlement(workspaceID)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get deferred transactions")
		return NewInternalError(c, "Failed to get deferred transactions")
	}

	// Group transactions by month
	groups := groupTransactionsByMonth(transactions)

	return c.JSON(http.StatusOK, groups)
}

// ImmediateGroup represents billed transactions with immediate intent for current month
type ImmediateGroup struct {
	Month        string                `json:"month"`        // "2026-01"
	MonthLabel   string                `json:"monthLabel"`   // "January"
	TotalAmount  string                `json:"totalAmount"`
	ItemCount    int                   `json:"itemCount"`
	Transactions []TransactionResponse `json:"transactions"`
}

// GetImmediateToSettle returns billed transactions with immediate intent for the current month
// @Summary Get immediate transactions to settle
// @Description Returns billed CC transactions with immediate intent (Pay This Month) for the current month
// @Tags transactions
// @Produce json
// @Param month query string false "Month in YYYY-MM format (defaults to current month)"
// @Security BearerAuth
// @Success 200 {object} ImmediateGroup
// @Failure 401 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /transactions/immediate-to-settle [get]
func (h *TransactionHandler) GetImmediateToSettle(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	// Parse month parameter (default to current month)
	monthStr := c.QueryParam("month")
	var month time.Time
	if monthStr != "" {
		parsed, err := time.Parse("2006-01", monthStr)
		if err != nil {
			return NewValidationError(c, "Invalid month format. Use YYYY-MM", nil)
		}
		month = parsed
	} else {
		now := time.Now()
		month = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	transactions, err := h.transactionService.GetImmediateForSettlement(workspaceID, month)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get immediate transactions")
		return NewInternalError(c, "Failed to get immediate transactions")
	}

	// Calculate total
	total := decimal.Zero
	txResponses := make([]TransactionResponse, len(transactions))
	for i, tx := range transactions {
		total = total.Add(tx.Amount)
		txResponses[i] = toTransactionResponse(tx)
	}

	group := ImmediateGroup{
		Month:        month.Format("2006-01"),
		MonthLabel:   month.Format("January"),
		TotalAmount:  total.StringFixed(2),
		ItemCount:    len(transactions),
		Transactions: txResponses,
	}

	return c.JSON(http.StatusOK, group)
}

// PendingDeferredGroup represents pending deferred CC transactions for a month
type PendingDeferredGroup struct {
	Month        string                `json:"month"`        // "2026-01"
	MonthLabel   string                `json:"monthLabel"`   // "January"
	TotalAmount  string                `json:"totalAmount"`
	ItemCount    int                   `json:"itemCount"`
	Transactions []TransactionResponse `json:"transactions"`
}

// GetPendingDeferred returns pending (not yet billed) deferred CC transactions
// @Summary Get pending deferred CC transactions
// @Description Returns pending CC transactions with deferred intent (Pay Next Month) for visibility
// @Tags transactions
// @Produce json
// @Param month query string false "Month in YYYY-MM format (defaults to current month)"
// @Security BearerAuth
// @Success 200 {object} PendingDeferredGroup
// @Failure 401 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /transactions/pending-deferred [get]
func (h *TransactionHandler) GetPendingDeferred(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	// Parse month parameter (default to current month)
	monthStr := c.QueryParam("month")
	var month time.Time
	if monthStr != "" {
		parsed, err := time.Parse("2006-01", monthStr)
		if err != nil {
			return NewValidationError(c, "Invalid month format. Use YYYY-MM", nil)
		}
		month = parsed
	} else {
		now := time.Now()
		month = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	transactions, err := h.transactionService.GetPendingDeferredCC(workspaceID, month)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get pending deferred transactions")
		return NewInternalError(c, "Failed to get pending deferred transactions")
	}

	// Calculate total
	total := decimal.Zero
	txResponses := make([]TransactionResponse, len(transactions))
	for i, tx := range transactions {
		total = total.Add(tx.Amount)
		txResponses[i] = toTransactionResponse(tx)
	}

	group := PendingDeferredGroup{
		Month:        month.Format("2006-01"),
		MonthLabel:   month.Format("January"),
		TotalAmount:  total.StringFixed(2),
		ItemCount:    len(transactions),
		Transactions: txResponses,
	}

	return c.JSON(http.StatusOK, group)
}

// OverdueGroupResponse represents an overdue group in API responses
type OverdueGroupResponse struct {
	Month         string                `json:"month"`         // "2025-11"
	MonthLabel    string                `json:"monthLabel"`    // "November 2025"
	MonthsOverdue int                   `json:"monthsOverdue"` // Number of months overdue
	TotalAmount   string                `json:"totalAmount"`
	ItemCount     int                   `json:"itemCount"`
	Transactions  []TransactionResponse `json:"transactions"`
}

// GetOverdue returns overdue CC transactions grouped by month
// @Summary Get overdue CC transactions
// @Description Returns CC transactions that are billed but overdue (2+ months), grouped by month
// @Tags transactions
// @Produce json
// @Security BearerAuth
// @Success 200 {array} OverdueGroupResponse
// @Failure 401 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /transactions/overdue [get]
func (h *TransactionHandler) GetOverdue(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	groups, err := h.transactionService.GetOverdue(workspaceID)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get overdue transactions")
		return NewInternalError(c, "Failed to get overdue transactions")
	}

	// Convert to response format
	response := make([]OverdueGroupResponse, len(groups))
	for i, group := range groups {
		transactions := make([]TransactionResponse, len(group.Transactions))
		for j, tx := range group.Transactions {
			transactions[j] = toTransactionResponse(tx)
		}
		response[i] = OverdueGroupResponse{
			Month:         group.Month,
			MonthLabel:    group.MonthLabel,
			MonthsOverdue: group.MonthsOverdue,
			TotalAmount:   group.TotalAmount.StringFixed(2),
			ItemCount:     group.ItemCount,
			Transactions:  transactions,
		}
	}

	return c.JSON(http.StatusOK, response)
}

// UpdateAmountRequest represents the update amount request body
type UpdateAmountRequest struct {
	Amount string `json:"amount"`
}

// UpdateAmount godoc
// @Summary Update transaction amount
// @Description Update only the amount field of a transaction (for overdue items)
// @Tags transactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Transaction ID"
// @Param request body UpdateAmountRequest true "New amount"
// @Success 200 {object} TransactionResponse
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Failure 404 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /transactions/{id}/amount [patch]
func (h *TransactionHandler) UpdateAmount(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid transaction ID", nil)
	}

	var req UpdateAmountRequest
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

	transaction, err := h.transactionService.UpdateAmount(workspaceID, int32(id), amount)
	if err != nil {
		if errors.Is(err, domain.ErrTransactionNotFound) {
			return NewNotFoundError(c, "Transaction not found")
		}
		if errors.Is(err, domain.ErrInvalidAmount) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "amount", Message: "Amount must be positive"},
			})
		}
		log.Error().Err(err).Int32("id", int32(id)).Msg("Failed to update transaction amount")
		return NewInternalError(c, "Failed to update transaction amount")
	}

	return c.JSON(http.StatusOK, toTransactionResponse(transaction))
}

// groupTransactionsByMonth groups transactions by their transaction month
func groupTransactionsByMonth(transactions []*domain.Transaction) []DeferredGroup {
	// Map to group transactions by year-month
	monthGroups := make(map[string]*DeferredGroup)
	monthOrder := []string{} // Track order of months

	for _, tx := range transactions {
		yearMonth := tx.TransactionDate.Format("2006-01")
		monthLabel := tx.TransactionDate.Format("January")

		if _, exists := monthGroups[yearMonth]; !exists {
			monthGroups[yearMonth] = &DeferredGroup{
				Month:        yearMonth,
				MonthLabel:   monthLabel,
				TotalAmount:  "0.00",
				ItemCount:    0,
				Transactions: []TransactionResponse{},
			}
			monthOrder = append(monthOrder, yearMonth)
		}

		group := monthGroups[yearMonth]
		group.Transactions = append(group.Transactions, toTransactionResponse(tx))
		group.ItemCount++

		// Update total
		currentTotal, _ := decimal.NewFromString(group.TotalAmount)
		group.TotalAmount = currentTotal.Add(tx.Amount).StringFixed(2)
	}

	// Convert map to slice in order
	result := make([]DeferredGroup, 0, len(monthOrder))
	for _, ym := range monthOrder {
		result = append(result, *monthGroups[ym])
	}

	return result
}
