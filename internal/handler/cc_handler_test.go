package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/service"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
)

func TestGetPayableBreakdown_Success(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	ccService := service.NewCCService(transactionRepo, accountRepo)
	handler := NewCCHandler(ccService)

	// Configure mock with test data
	transactionRepo.GetCCPayableBreakdownFn = func(wsID int32) ([]*domain.CCPayableTransaction, error) {
		return []*domain.CCPayableTransaction{
			{
				ID:               1,
				Name:             "Groceries",
				Amount:           decimal.NewFromFloat(100.00),
				TransactionDate:  time.Now(),
				SettlementIntent: domain.CCSettlementThisMonth,
				AccountID:        1,
				AccountName:      "Maybank CC",
			},
			{
				ID:               2,
				Name:             "Shopping",
				Amount:           decimal.NewFromFloat(200.00),
				TransactionDate:  time.Now(),
				SettlementIntent: domain.CCSettlementNextMonth,
				AccountID:        2,
				AccountName:      "CIMB CC",
			},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cc/payable/breakdown", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.GetPayableBreakdown(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response CCPayableBreakdownResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.ThisMonthTotal != "100.00" {
		t.Errorf("Expected this month total '100.00', got %s", response.ThisMonthTotal)
	}

	if response.NextMonthTotal != "200.00" {
		t.Errorf("Expected next month total '200.00', got %s", response.NextMonthTotal)
	}

	if response.GrandTotal != "300.00" {
		t.Errorf("Expected grand total '300.00', got %s", response.GrandTotal)
	}

	if len(response.ThisMonth) != 1 {
		t.Errorf("Expected 1 account in this month, got %d", len(response.ThisMonth))
	}

	if len(response.NextMonth) != 1 {
		t.Errorf("Expected 1 account in next month, got %d", len(response.NextMonth))
	}
}

func TestGetPayableBreakdown_NoTransactions(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	ccService := service.NewCCService(transactionRepo, accountRepo)
	handler := NewCCHandler(ccService)

	// Configure mock with empty data
	transactionRepo.GetCCPayableBreakdownFn = func(wsID int32) ([]*domain.CCPayableTransaction, error) {
		return []*domain.CCPayableTransaction{}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cc/payable/breakdown", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.GetPayableBreakdown(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response CCPayableBreakdownResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.ThisMonthTotal != "0.00" {
		t.Errorf("Expected this month total '0.00', got %s", response.ThisMonthTotal)
	}

	if response.NextMonthTotal != "0.00" {
		t.Errorf("Expected next month total '0.00', got %s", response.NextMonthTotal)
	}

	if response.GrandTotal != "0.00" {
		t.Errorf("Expected grand total '0.00', got %s", response.GrandTotal)
	}
}

func TestGetPayableBreakdown_Unauthorized(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	ccService := service.NewCCService(transactionRepo, accountRepo)
	handler := NewCCHandler(ccService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cc/payable/breakdown", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Don't set workspace - should return unauthorized
	err := handler.GetPayableBreakdown(c)
	if err != nil {
		t.Fatalf("Expected no error from handler, got %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestCreateCCPayment_Success(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	ccService := service.NewCCService(transactionRepo, accountRepo)
	handler := NewCCHandler(ccService)

	// Add a CC account
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test CC",
		Template:    domain.TemplateCreditCard,
		AccountType: domain.AccountTypeLiability,
	})

	body := `{"ccAccountId":1,"amount":"100.00","transactionDate":"2025-01-15"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cc/payments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.CreateCCPayment(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rec.Code)
	}

	var response CCPaymentResponseEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.CCTransaction == nil {
		t.Error("Expected CC transaction in response")
	}
	if response.CCTransaction.Amount != "100.00" {
		t.Errorf("Expected amount '100.00', got %s", response.CCTransaction.Amount)
	}
	if response.CCTransaction.Type != "income" {
		t.Errorf("Expected type 'income', got %s", response.CCTransaction.Type)
	}
	if !response.CCTransaction.IsCCPayment {
		t.Error("Expected IsCCPayment to be true")
	}
	if response.SourceTransaction != nil {
		t.Error("Expected no source transaction when sourceAccountId is not provided")
	}
}

func TestCreateCCPayment_WithSourceAccount(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	ccService := service.NewCCService(transactionRepo, accountRepo)
	handler := NewCCHandler(ccService)

	// Add accounts
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test CC",
		Template:    domain.TemplateCreditCard,
		AccountType: domain.AccountTypeLiability,
	})
	accountRepo.AddAccount(&domain.Account{
		ID:          2,
		WorkspaceID: 1,
		Name:        "Test Bank",
		Template:    domain.TemplateBank,
		AccountType: domain.AccountTypeAsset,
	})

	body := `{"ccAccountId":1,"amount":"500.00","transactionDate":"2025-01-15","sourceAccountId":2,"notes":"Monthly payment"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cc/payments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.CreateCCPayment(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rec.Code)
	}

	var response CCPaymentResponseEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.CCTransaction == nil {
		t.Error("Expected CC transaction in response")
	}
	if response.SourceTransaction == nil {
		t.Error("Expected source transaction in response")
	}
	if response.CCTransaction.Type != "income" {
		t.Errorf("Expected CC transaction type 'income', got %s", response.CCTransaction.Type)
	}
	if response.SourceTransaction.Type != "expense" {
		t.Errorf("Expected source transaction type 'expense', got %s", response.SourceTransaction.Type)
	}
}

func TestCreateCCPayment_InvalidAccountType(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	ccService := service.NewCCService(transactionRepo, accountRepo)
	handler := NewCCHandler(ccService)

	// Add a bank account (not a CC)
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test Bank",
		Template:    domain.TemplateBank,
		AccountType: domain.AccountTypeAsset,
	})

	body := `{"ccAccountId":1,"amount":"100.00","transactionDate":"2025-01-15"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cc/payments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.CreateCCPayment(c)
	if err != nil {
		t.Fatalf("Expected no error from handler, got %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestCreateCCPayment_SourceCannotBeCC(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	ccService := service.NewCCService(transactionRepo, accountRepo)
	handler := NewCCHandler(ccService)

	// Add two CC accounts
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Name:        "CC 1",
		Template:    domain.TemplateCreditCard,
		AccountType: domain.AccountTypeLiability,
	})
	accountRepo.AddAccount(&domain.Account{
		ID:          2,
		WorkspaceID: 1,
		Name:        "CC 2",
		Template:    domain.TemplateCreditCard,
		AccountType: domain.AccountTypeLiability,
	})

	// Try to use CC as source account
	body := `{"ccAccountId":1,"amount":"100.00","transactionDate":"2025-01-15","sourceAccountId":2}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cc/payments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.CreateCCPayment(c)
	if err != nil {
		t.Fatalf("Expected no error from handler, got %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}
