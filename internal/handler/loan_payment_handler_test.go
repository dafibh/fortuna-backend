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

func TestGetPaymentsByLoanID_Success(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	paymentService := service.NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)
	handler := NewLoanPaymentHandler(paymentService)

	// Add a loan
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: 1,
		ProviderID:  1,
		ItemName:    "Test Loan",
	})

	// Add payments
	paymentRepo.AddPayment(&domain.LoanPayment{
		ID:            1,
		LoanID:        1,
		PaymentNumber: 1,
		Amount:        decimal.NewFromInt(100),
		DueYear:       2024,
		DueMonth:      1,
		Paid:          false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	})
	paymentRepo.AddPayment(&domain.LoanPayment{
		ID:            2,
		LoanID:        1,
		PaymentNumber: 2,
		Amount:        decimal.NewFromInt(100),
		DueYear:       2024,
		DueMonth:      2,
		Paid:          true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/loans/1/payments", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loanId")
	c.SetParamValues("1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.GetPaymentsByLoanID(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response []LoanPaymentResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("Expected 2 payments, got %d", len(response))
	}
}

func TestGetPaymentsByLoanID_LoanNotFound(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	paymentService := service.NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)
	handler := NewLoanPaymentHandler(paymentService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/loans/999/payments", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loanId")
	c.SetParamValues("999")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.GetPaymentsByLoanID(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestGetPaymentsByLoanID_InvalidLoanID(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	paymentService := service.NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)
	handler := NewLoanPaymentHandler(paymentService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/loans/invalid/payments", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loanId")
	c.SetParamValues("invalid")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.GetPaymentsByLoanID(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestGetPaymentsByLoanID_WorkspaceIsolation(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	paymentService := service.NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)
	handler := NewLoanPaymentHandler(paymentService)

	// Add a loan in workspace 2
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: 2,
		ProviderID:  1,
		ItemName:    "Workspace 2 Loan",
	})

	// Try to access from workspace 1
	req := httptest.NewRequest(http.MethodGet, "/api/v1/loans/1/payments", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loanId")
	c.SetParamValues("1")

	setupAuthContextWithWorkspace(c, "auth0|user1", "user1@example.com", "User 1", "", 1)

	err := handler.GetPaymentsByLoanID(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Workspace 1 should not see workspace 2's loan, expected 404 but got %d", rec.Code)
	}
}

func TestUpdatePaymentAmount_Success(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	paymentService := service.NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)
	handler := NewLoanPaymentHandler(paymentService)

	// Add loan and payment
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: 1,
		ProviderID:  1,
		ItemName:    "Test Loan",
	})
	paymentRepo.AddPayment(&domain.LoanPayment{
		ID:            1,
		LoanID:        1,
		PaymentNumber: 1,
		Amount:        decimal.NewFromInt(100),
		DueYear:       2024,
		DueMonth:      1,
		Paid:          false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	})

	reqBody := `{"amount": "150.00"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/loans/1/payments/1", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loanId", "paymentId")
	c.SetParamValues("1", "1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.UpdatePaymentAmount(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response LoanPaymentResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Amount != "150.00" {
		t.Errorf("Expected amount '150.00', got %s", response.Amount)
	}
}

func TestUpdatePaymentAmount_LoanNotFound(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	paymentService := service.NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)
	handler := NewLoanPaymentHandler(paymentService)

	reqBody := `{"amount": "150.00"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/loans/999/payments/1", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loanId", "paymentId")
	c.SetParamValues("999", "1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.UpdatePaymentAmount(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestUpdatePaymentAmount_PaymentNotFound(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	paymentService := service.NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)
	handler := NewLoanPaymentHandler(paymentService)

	// Add loan but no payment
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: 1,
		ProviderID:  1,
		ItemName:    "Test Loan",
	})

	reqBody := `{"amount": "150.00"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/loans/1/payments/999", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loanId", "paymentId")
	c.SetParamValues("1", "999")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.UpdatePaymentAmount(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestUpdatePaymentAmount_InvalidAmount(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	paymentService := service.NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)
	handler := NewLoanPaymentHandler(paymentService)

	// Add loan and payment
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: 1,
		ProviderID:  1,
		ItemName:    "Test Loan",
	})
	paymentRepo.AddPayment(&domain.LoanPayment{
		ID:            1,
		LoanID:        1,
		PaymentNumber: 1,
		Amount:        decimal.NewFromInt(100),
		DueYear:       2024,
		DueMonth:      1,
		Paid:          false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	})

	reqBody := `{"amount": "not-a-number"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/loans/1/payments/1", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loanId", "paymentId")
	c.SetParamValues("1", "1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.UpdatePaymentAmount(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestUpdatePaymentAmount_NegativeAmount(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	paymentService := service.NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)
	handler := NewLoanPaymentHandler(paymentService)

	// Add loan and payment
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: 1,
		ProviderID:  1,
		ItemName:    "Test Loan",
	})
	paymentRepo.AddPayment(&domain.LoanPayment{
		ID:            1,
		LoanID:        1,
		PaymentNumber: 1,
		Amount:        decimal.NewFromInt(100),
		DueYear:       2024,
		DueMonth:      1,
		Paid:          false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	})

	reqBody := `{"amount": "-50.00"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/loans/1/payments/1", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loanId", "paymentId")
	c.SetParamValues("1", "1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.UpdatePaymentAmount(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestTogglePaymentPaid_Success(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	paymentService := service.NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)
	handler := NewLoanPaymentHandler(paymentService)

	// Add loan and payment
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: 1,
		ProviderID:  1,
		ItemName:    "Test Loan",
	})
	paymentRepo.AddPayment(&domain.LoanPayment{
		ID:            1,
		LoanID:        1,
		PaymentNumber: 1,
		Amount:        decimal.NewFromInt(100),
		DueYear:       2024,
		DueMonth:      1,
		Paid:          false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	})

	reqBody := `{"paid": true}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/loans/1/payments/1/toggle-paid", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loanId", "paymentId")
	c.SetParamValues("1", "1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.TogglePaymentPaid(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response LoanPaymentResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Paid {
		t.Error("Expected paid to be true")
	}
}

func TestTogglePaymentPaid_LoanNotFound(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	paymentService := service.NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)
	handler := NewLoanPaymentHandler(paymentService)

	reqBody := `{"paid": true}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/loans/999/payments/1/toggle-paid", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loanId", "paymentId")
	c.SetParamValues("999", "1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.TogglePaymentPaid(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestTogglePaymentPaid_PaymentNotFound(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	paymentService := service.NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)
	handler := NewLoanPaymentHandler(paymentService)

	// Add loan but no payment
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: 1,
		ProviderID:  1,
		ItemName:    "Test Loan",
	})

	reqBody := `{"paid": true}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/loans/1/payments/999/toggle-paid", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loanId", "paymentId")
	c.SetParamValues("1", "999")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.TogglePaymentPaid(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestTogglePaymentPaid_WithCustomDate(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	paymentService := service.NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)
	handler := NewLoanPaymentHandler(paymentService)

	// Add loan and payment
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: 1,
		ProviderID:  1,
		ItemName:    "Test Loan",
	})
	paymentRepo.AddPayment(&domain.LoanPayment{
		ID:            1,
		LoanID:        1,
		PaymentNumber: 1,
		Amount:        decimal.NewFromInt(100),
		DueYear:       2024,
		DueMonth:      1,
		Paid:          false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	})

	reqBody := `{"paid": true, "paidDate": "2024-06-15"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/loans/1/payments/1/toggle-paid", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loanId", "paymentId")
	c.SetParamValues("1", "1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.TogglePaymentPaid(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response LoanPaymentResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Paid {
		t.Error("Expected paid to be true")
	}

	// Verify the paid date was set to the custom date
	if response.PaidDate == nil {
		t.Error("Expected paidDate to be set")
	} else if !strings.HasPrefix(*response.PaidDate, "2024-06-15") {
		t.Errorf("Expected paidDate to start with '2024-06-15', got %s", *response.PaidDate)
	}
}

func TestTogglePaymentPaid_InvalidDateFormat(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	paymentService := service.NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)
	handler := NewLoanPaymentHandler(paymentService)

	// Add loan and payment
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: 1,
		ProviderID:  1,
		ItemName:    "Test Loan",
	})
	paymentRepo.AddPayment(&domain.LoanPayment{
		ID:            1,
		LoanID:        1,
		PaymentNumber: 1,
		Amount:        decimal.NewFromInt(100),
		DueYear:       2024,
		DueMonth:      1,
		Paid:          false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	})

	reqBody := `{"paid": true, "paidDate": "invalid-date"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/loans/1/payments/1/toggle-paid", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loanId", "paymentId")
	c.SetParamValues("1", "1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.TogglePaymentPaid(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestTogglePaymentPaid_WorkspaceIsolation(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	paymentService := service.NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)
	handler := NewLoanPaymentHandler(paymentService)

	// Add loan in workspace 2
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: 2,
		ProviderID:  1,
		ItemName:    "Workspace 2 Loan",
	})
	paymentRepo.AddPayment(&domain.LoanPayment{
		ID:            1,
		LoanID:        1,
		PaymentNumber: 1,
		Amount:        decimal.NewFromInt(100),
		DueYear:       2024,
		DueMonth:      1,
		Paid:          false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	})

	// Try to toggle from workspace 1
	reqBody := `{"paid": true}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/loans/1/payments/1/toggle-paid", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loanId", "paymentId")
	c.SetParamValues("1", "1")

	setupAuthContextWithWorkspace(c, "auth0|user1", "user1@example.com", "User 1", "", 1)

	err := handler.TogglePaymentPaid(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Workspace 1 should not toggle workspace 2's payment, expected 404 but got %d", rec.Code)
	}
}

func TestUpdatePaymentAmount_WorkspaceIsolation(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	paymentService := service.NewLoanPaymentService(nil, paymentRepo, loanRepo, nil)
	handler := NewLoanPaymentHandler(paymentService)

	// Add loan in workspace 2
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: 2,
		ProviderID:  1,
		ItemName:    "Workspace 2 Loan",
	})
	paymentRepo.AddPayment(&domain.LoanPayment{
		ID:            1,
		LoanID:        1,
		PaymentNumber: 1,
		Amount:        decimal.NewFromInt(100),
		DueYear:       2024,
		DueMonth:      1,
		Paid:          false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	})

	// Try to update from workspace 1
	reqBody := `{"amount": "150.00"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/loans/1/payments/1", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loanId", "paymentId")
	c.SetParamValues("1", "1")

	setupAuthContextWithWorkspace(c, "auth0|user1", "user1@example.com", "User 1", "", 1)

	err := handler.UpdatePaymentAmount(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Workspace 1 should not update workspace 2's payment, expected 404 but got %d", rec.Code)
	}
}
