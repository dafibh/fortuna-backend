package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	ccService := service.NewCCService(transactionRepo)
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
	ccService := service.NewCCService(transactionRepo)
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
	ccService := service.NewCCService(transactionRepo)
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
