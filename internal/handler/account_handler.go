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

// AccountHandler handles account-related HTTP requests
type AccountHandler struct {
	accountService     *service.AccountService
	calculationService *service.CalculationService
}

// NewAccountHandler creates a new AccountHandler
func NewAccountHandler(accountService *service.AccountService, calculationService *service.CalculationService) *AccountHandler {
	return &AccountHandler{
		accountService:     accountService,
		calculationService: calculationService,
	}
}

// CreateAccountRequest represents the create account request body
type CreateAccountRequest struct {
	Name           string `json:"name"`
	Template       string `json:"template"`
	InitialBalance string `json:"initialBalance,omitempty"`
}

// UpdateAccountRequest represents the update account request body
type UpdateAccountRequest struct {
	Name string `json:"name"`
}

// AccountResponse represents an account in API responses
type AccountResponse struct {
	ID                int32   `json:"id"`
	WorkspaceID       int32   `json:"workspaceId"`
	Name              string  `json:"name"`
	AccountType       string  `json:"accountType"`
	Template          string  `json:"template"`
	InitialBalance    string  `json:"initialBalance"`
	CalculatedBalance string  `json:"calculatedBalance"`
	CCOutstanding     *string `json:"ccOutstanding,omitempty"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
	DeletedAt         *string `json:"deletedAt,omitempty"`
}

// CCOutstandingResponse represents the CC summary API response
type CCOutstandingResponse struct {
	TotalOutstanding string                       `json:"totalOutstanding"`
	CCAccountCount   int32                        `json:"ccAccountCount"`
	PerAccount       []PerAccountOutstandingEntry `json:"perAccount"`
}

// PerAccountOutstandingEntry represents a single account's outstanding balance
type PerAccountOutstandingEntry struct {
	AccountID          int32  `json:"accountId"`
	AccountName        string `json:"accountName"`
	OutstandingBalance string `json:"outstandingBalance"`
}

// CreateAccount handles POST /api/v1/accounts
func (h *AccountHandler) CreateAccount(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	var req CreateAccountRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	// Parse initial balance (default to 0)
	initialBalance := decimal.Zero
	if req.InitialBalance != "" {
		var err error
		initialBalance, err = decimal.NewFromString(req.InitialBalance)
		if err != nil {
			return NewValidationError(c, "Invalid initial balance", []ValidationError{
				{Field: "initialBalance", Message: "Must be a valid decimal number"},
			})
		}
	}

	input := service.CreateAccountInput{
		Name:           req.Name,
		Template:       domain.AccountTemplate(req.Template),
		InitialBalance: initialBalance,
	}

	account, err := h.accountService.CreateAccount(workspaceID, input)
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
		if errors.Is(err, domain.ErrInvalidTemplate) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "template", Message: "Template must be one of: bank, cash, ewallet, credit_card"},
			})
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to create account")
		return NewInternalError(c, "Failed to create account")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("account_id", account.ID).Str("name", account.Name).Msg("Account created")

	return c.JSON(http.StatusCreated, toAccountResponse(account))
}

// GetAccounts handles GET /api/v1/accounts
func (h *AccountHandler) GetAccounts(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	// Check for includeArchived query param
	includeArchived := c.QueryParam("includeArchived") == "true"

	accounts, err := h.accountService.GetAccounts(workspaceID, includeArchived)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get accounts")
		return NewInternalError(c, "Failed to get accounts")
	}

	// Calculate balances for all accounts
	balances, err := h.calculationService.CalculateAccountBalances(workspaceID)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to calculate balances")
		return NewInternalError(c, "Failed to calculate balances")
	}

	response := make([]AccountResponse, len(accounts))
	for i, account := range accounts {
		balance := balances[account.ID]
		if balance == nil {
			balance = &service.AccountBalanceResult{
				AccountID:         account.ID,
				InitialBalance:    account.InitialBalance,
				CalculatedBalance: account.InitialBalance,
			}
		}
		response[i] = toAccountResponseWithBalance(account, balance)
	}

	return c.JSON(http.StatusOK, response)
}

// UpdateAccount handles PUT /api/v1/accounts/:id
func (h *AccountHandler) UpdateAccount(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid account ID", nil)
	}

	var req UpdateAccountRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	account, err := h.accountService.UpdateAccount(workspaceID, int32(id), req.Name)
	if err != nil {
		if errors.Is(err, domain.ErrAccountNotFound) {
			return NewNotFoundError(c, "Account not found")
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
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("account_id", id).Msg("Failed to update account")
		return NewInternalError(c, "Failed to update account")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("account_id", account.ID).Str("name", account.Name).Msg("Account updated")
	return c.JSON(http.StatusOK, toAccountResponse(account))
}

// DeleteAccount handles DELETE /api/v1/accounts/:id
func (h *AccountHandler) DeleteAccount(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid account ID", nil)
	}

	if err := h.accountService.DeleteAccount(workspaceID, int32(id)); err != nil {
		if errors.Is(err, domain.ErrAccountNotFound) {
			return NewNotFoundError(c, "Account not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("account_id", id).Msg("Failed to delete account")
		return NewInternalError(c, "Failed to delete account")
	}

	log.Info().Int32("workspace_id", workspaceID).Int("account_id", id).Msg("Account deleted (soft)")
	return c.NoContent(http.StatusNoContent)
}

// GetCCSummary handles GET /api/v1/accounts/cc-summary
func (h *AccountHandler) GetCCSummary(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	result, err := h.accountService.GetCCOutstanding(workspaceID)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get CC outstanding summary")
		return NewInternalError(c, "Failed to get CC outstanding summary")
	}

	// Convert to response format
	perAccount := make([]PerAccountOutstandingEntry, len(result.PerAccount))
	for i, acc := range result.PerAccount {
		perAccount[i] = PerAccountOutstandingEntry{
			AccountID:          acc.AccountID,
			AccountName:        acc.AccountName,
			OutstandingBalance: acc.OutstandingBalance.StringFixed(2),
		}
	}

	response := CCOutstandingResponse{
		TotalOutstanding: result.TotalOutstanding.StringFixed(2),
		CCAccountCount:   result.CCAccountCount,
		PerAccount:       perAccount,
	}

	return c.JSON(http.StatusOK, response)
}

// Helper function to convert domain.Account to AccountResponse (without balance calculation)
func toAccountResponse(account *domain.Account) AccountResponse {
	resp := AccountResponse{
		ID:                account.ID,
		WorkspaceID:       account.WorkspaceID,
		Name:              account.Name,
		AccountType:       string(account.AccountType),
		Template:          string(account.Template),
		InitialBalance:    account.InitialBalance.StringFixed(2),
		CalculatedBalance: account.InitialBalance.StringFixed(2), // Default to initial if no calculation
		CreatedAt:         account.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         account.UpdatedAt.Format(time.RFC3339),
	}
	if account.DeletedAt != nil {
		deletedAt := account.DeletedAt.Format(time.RFC3339)
		resp.DeletedAt = &deletedAt
	}
	return resp
}

// Helper function to convert domain.Account to AccountResponse with calculated balance
func toAccountResponseWithBalance(account *domain.Account, balance *service.AccountBalanceResult) AccountResponse {
	resp := AccountResponse{
		ID:                account.ID,
		WorkspaceID:       account.WorkspaceID,
		Name:              account.Name,
		AccountType:       string(account.AccountType),
		Template:          string(account.Template),
		InitialBalance:    account.InitialBalance.StringFixed(2),
		CalculatedBalance: balance.CalculatedBalance.StringFixed(2),
		CreatedAt:         account.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         account.UpdatedAt.Format(time.RFC3339),
	}

	// Include CC outstanding for credit card accounts
	if account.Template == domain.TemplateCreditCard && !balance.CCOutstanding.IsZero() {
		outstanding := balance.CCOutstanding.StringFixed(2)
		resp.CCOutstanding = &outstanding
	}

	if account.DeletedAt != nil {
		deletedAt := account.DeletedAt.Format(time.RFC3339)
		resp.DeletedAt = &deletedAt
	}
	return resp
}
