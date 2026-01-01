package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/service"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
)

func TestCreateAccount_Success_BankAccount(t *testing.T) {
	e := echo.New()
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountService := service.NewAccountService(accountRepo)
	calculationService := service.NewCalculationService(accountRepo, transactionRepo)
	handler := NewAccountHandler(accountService, calculationService)

	reqBody := `{"name": "My Savings", "template": "bank", "initialBalance": "1000.50"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.CreateAccount(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rec.Code)
	}

	var response AccountResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Name != "My Savings" {
		t.Errorf("Expected name 'My Savings', got %s", response.Name)
	}

	if response.Template != "bank" {
		t.Errorf("Expected template 'bank', got %s", response.Template)
	}

	if response.AccountType != "asset" {
		t.Errorf("Expected account type 'asset', got %s", response.AccountType)
	}

	if response.InitialBalance != "1000.50" {
		t.Errorf("Expected initial balance '1000.50', got %s", response.InitialBalance)
	}
}

func TestCreateAccount_Success_CreditCard(t *testing.T) {
	e := echo.New()
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountService := service.NewAccountService(accountRepo)
	calculationService := service.NewCalculationService(accountRepo, transactionRepo)
	handler := NewAccountHandler(accountService, calculationService)

	reqBody := `{"name": "Visa Card", "template": "credit_card"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.CreateAccount(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rec.Code)
	}

	var response AccountResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.AccountType != "liability" {
		t.Errorf("Expected account type 'liability', got %s", response.AccountType)
	}

	if response.InitialBalance != "0.00" {
		t.Errorf("Expected initial balance '0.00', got %s", response.InitialBalance)
	}
}

func TestCreateAccount_MissingWorkspaceID(t *testing.T) {
	e := echo.New()
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountService := service.NewAccountService(accountRepo)
	calculationService := service.NewCalculationService(accountRepo, transactionRepo)
	handler := NewAccountHandler(accountService, calculationService)

	reqBody := `{"name": "My Account", "template": "bank"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// No workspace ID set
	setupAuthContext(c, "auth0|test", "test@example.com", "Test User", "")

	err := handler.CreateAccount(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestCreateAccount_MissingName(t *testing.T) {
	e := echo.New()
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountService := service.NewAccountService(accountRepo)
	calculationService := service.NewCalculationService(accountRepo, transactionRepo)
	handler := NewAccountHandler(accountService, calculationService)

	reqBody := `{"name": "", "template": "bank"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.CreateAccount(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var problemDetails ProblemDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &problemDetails); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if problemDetails.Type != ErrorTypeValidation {
		t.Errorf("Expected error type %s, got %s", ErrorTypeValidation, problemDetails.Type)
	}

	if len(problemDetails.Errors) != 1 || problemDetails.Errors[0].Field != "name" {
		t.Error("Expected validation error for 'name' field")
	}
}

func TestCreateAccount_InvalidTemplate(t *testing.T) {
	e := echo.New()
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountService := service.NewAccountService(accountRepo)
	calculationService := service.NewCalculationService(accountRepo, transactionRepo)
	handler := NewAccountHandler(accountService, calculationService)

	reqBody := `{"name": "Invalid", "template": "invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.CreateAccount(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var problemDetails ProblemDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &problemDetails); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(problemDetails.Errors) != 1 || problemDetails.Errors[0].Field != "template" {
		t.Error("Expected validation error for 'template' field")
	}
}

func TestCreateAccount_InvalidInitialBalance(t *testing.T) {
	e := echo.New()
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountService := service.NewAccountService(accountRepo)
	calculationService := service.NewCalculationService(accountRepo, transactionRepo)
	handler := NewAccountHandler(accountService, calculationService)

	reqBody := `{"name": "My Account", "template": "bank", "initialBalance": "not-a-number"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.CreateAccount(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var problemDetails ProblemDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &problemDetails); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(problemDetails.Errors) != 1 || problemDetails.Errors[0].Field != "initialBalance" {
		t.Error("Expected validation error for 'initialBalance' field")
	}
}

func TestGetAccounts_Success(t *testing.T) {
	e := echo.New()
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountService := service.NewAccountService(accountRepo)
	calculationService := service.NewCalculationService(accountRepo, transactionRepo)
	handler := NewAccountHandler(accountService, calculationService)

	workspaceID := int32(1)

	// Add test accounts
	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    workspaceID,
		Name:           "Bank Account",
		AccountType:    domain.AccountTypeAsset,
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromFloat(1000.00),
	})
	accountRepo.AddAccount(&domain.Account{
		ID:             2,
		WorkspaceID:    workspaceID,
		Name:           "Credit Card",
		AccountType:    domain.AccountTypeLiability,
		Template:       domain.TemplateCreditCard,
		InitialBalance: decimal.Zero,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.GetAccounts(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response []AccountResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(response))
	}
}

func TestGetAccounts_EmptyList(t *testing.T) {
	e := echo.New()
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountService := service.NewAccountService(accountRepo)
	calculationService := service.NewCalculationService(accountRepo, transactionRepo)
	handler := NewAccountHandler(accountService, calculationService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.GetAccounts(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response []AccountResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response) != 0 {
		t.Errorf("Expected 0 accounts, got %d", len(response))
	}
}

func TestGetAccounts_MissingWorkspaceID(t *testing.T) {
	e := echo.New()
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountService := service.NewAccountService(accountRepo)
	calculationService := service.NewCalculationService(accountRepo, transactionRepo)
	handler := NewAccountHandler(accountService, calculationService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// No workspace ID set
	setupAuthContext(c, "auth0|test", "test@example.com", "Test User", "")

	err := handler.GetAccounts(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestGetAccounts_WorkspaceIsolation(t *testing.T) {
	e := echo.New()
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountService := service.NewAccountService(accountRepo)
	calculationService := service.NewCalculationService(accountRepo, transactionRepo)
	handler := NewAccountHandler(accountService, calculationService)

	// Add account to workspace 1
	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    1,
		Name:           "Workspace 1 Account",
		AccountType:    domain.AccountTypeAsset,
		Template:       domain.TemplateBank,
		InitialBalance: decimal.Zero,
	})

	// Add account to workspace 2
	accountRepo.AddAccount(&domain.Account{
		ID:             2,
		WorkspaceID:    2,
		Name:           "Workspace 2 Account",
		AccountType:    domain.AccountTypeAsset,
		Template:       domain.TemplateBank,
		InitialBalance: decimal.Zero,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Request from workspace 1 - should only see workspace 1's account
	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.GetAccounts(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var response []AccountResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response) != 1 {
		t.Errorf("Expected 1 account, got %d", len(response))
	}

	if response[0].Name != "Workspace 1 Account" {
		t.Errorf("Expected 'Workspace 1 Account', got %s", response[0].Name)
	}
}

// GetCCSummary tests

func TestGetCCSummary_Success(t *testing.T) {
	e := echo.New()
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountService := service.NewAccountService(accountRepo)
	calculationService := service.NewCalculationService(accountRepo, transactionRepo)
	handler := NewAccountHandler(accountService, calculationService)

	workspaceID := int32(1)

	// Configure mock to return CC outstanding data
	accountRepo.GetCCOutstandingSummaryFn = func(wsID int32) (*domain.CCOutstandingSummary, error) {
		return &domain.CCOutstandingSummary{
			TotalOutstanding: decimal.NewFromFloat(5250.00),
			CCAccountCount:   2,
		}, nil
	}

	accountRepo.GetPerAccountOutstandingFn = func(wsID int32) ([]*domain.PerAccountOutstanding, error) {
		return []*domain.PerAccountOutstanding{
			{AccountID: 1, AccountName: "Maybank CC", OutstandingBalance: decimal.NewFromFloat(2500.00)},
			{AccountID: 2, AccountName: "CIMB CC", OutstandingBalance: decimal.NewFromFloat(2750.00)},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/cc-summary", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.GetCCSummary(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response CCOutstandingResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.TotalOutstanding != "5250.00" {
		t.Errorf("Expected total outstanding '5250.00', got %s", response.TotalOutstanding)
	}

	if response.CCAccountCount != 2 {
		t.Errorf("Expected 2 CC accounts, got %d", response.CCAccountCount)
	}

	if len(response.PerAccount) != 2 {
		t.Errorf("Expected 2 per-account entries, got %d", len(response.PerAccount))
	}

	if response.PerAccount[0].AccountName != "Maybank CC" {
		t.Errorf("Expected first account 'Maybank CC', got %s", response.PerAccount[0].AccountName)
	}
}

func TestGetCCSummary_MissingWorkspaceID(t *testing.T) {
	e := echo.New()
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountService := service.NewAccountService(accountRepo)
	calculationService := service.NewCalculationService(accountRepo, transactionRepo)
	handler := NewAccountHandler(accountService, calculationService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/cc-summary", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// No workspace ID set
	setupAuthContext(c, "auth0|test", "test@example.com", "Test User", "")

	err := handler.GetCCSummary(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestGetCCSummary_NoAccounts(t *testing.T) {
	e := echo.New()
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountService := service.NewAccountService(accountRepo)
	calculationService := service.NewCalculationService(accountRepo, transactionRepo)
	handler := NewAccountHandler(accountService, calculationService)

	workspaceID := int32(1)

	// Default mock returns zeros (no CC accounts)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/cc-summary", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.GetCCSummary(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response CCOutstandingResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.TotalOutstanding != "0.00" {
		t.Errorf("Expected total outstanding '0.00', got %s", response.TotalOutstanding)
	}

	if response.CCAccountCount != 0 {
		t.Errorf("Expected 0 CC accounts, got %d", response.CCAccountCount)
	}
}
