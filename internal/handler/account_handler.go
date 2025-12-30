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

// AccountHandler handles account-related HTTP requests
type AccountHandler struct {
	accountService *service.AccountService
}

// NewAccountHandler creates a new AccountHandler
func NewAccountHandler(accountService *service.AccountService) *AccountHandler {
	return &AccountHandler{accountService: accountService}
}

// CreateAccountRequest represents the create account request body
type CreateAccountRequest struct {
	Name           string `json:"name"`
	Template       string `json:"template"`
	InitialBalance string `json:"initialBalance,omitempty"`
}

// AccountResponse represents an account in API responses
type AccountResponse struct {
	ID             int32  `json:"id"`
	WorkspaceID    int32  `json:"workspaceId"`
	Name           string `json:"name"`
	AccountType    string `json:"accountType"`
	Template       string `json:"template"`
	InitialBalance string `json:"initialBalance"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
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

	accounts, err := h.accountService.GetAccounts(workspaceID)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get accounts")
		return NewInternalError(c, "Failed to get accounts")
	}

	response := make([]AccountResponse, len(accounts))
	for i, account := range accounts {
		response[i] = toAccountResponse(account)
	}

	return c.JSON(http.StatusOK, response)
}

// Helper function to convert domain.Account to AccountResponse
func toAccountResponse(account *domain.Account) AccountResponse {
	return AccountResponse{
		ID:             account.ID,
		WorkspaceID:    account.WorkspaceID,
		Name:           account.Name,
		AccountType:    string(account.AccountType),
		Template:       string(account.Template),
		InitialBalance: account.InitialBalance.StringFixed(2),
		CreatedAt:      account.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      account.UpdatedAt.Format(time.RFC3339),
	}
}
