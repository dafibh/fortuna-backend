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

// createTestLoanService creates a LoanService with mock repositories for testing
// Sets up a default bank account with ID 1 in workspaceID 1
func createTestLoanService(loanRepo *testutil.MockLoanRepository, providerRepo *testutil.MockLoanProviderRepository) *service.LoanService {
	svc, _ := createTestLoanServiceWithTransactionRepo(loanRepo, providerRepo)
	return svc
}

func createTestLoanServiceWithTransactionRepo(loanRepo *testutil.MockLoanRepository, providerRepo *testutil.MockLoanProviderRepository) (*service.LoanService, *testutil.MockTransactionRepository) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Add a bank account (ID 1) for testing
	accountRepo.Create(&domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test Bank Account",
		Template:    domain.TemplateBank,
		AccountType: domain.AccountTypeAsset,
	})

	// Add a CC account (ID 2) for testing settlement intent
	accountRepo.Create(&domain.Account{
		ID:          2,
		WorkspaceID: 1,
		Name:        "Test Credit Card",
		Template:    domain.TemplateCreditCard,
		AccountType: domain.AccountTypeLiability,
	})

	return service.NewLoanService(nil, loanRepo, providerRepo, transactionRepo, accountRepo), transactionRepo
}

func TestCreateLoan_Success(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	loanService := createTestLoanService(loanRepo, providerRepo)
	handler := NewLoanHandler(loanService)

	// Add a provider
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         1,
		Name:                "SPayLater",
		CutoffDay:           25,
		DefaultInterestRate: decimal.Zero,
	})

	reqBody := `{
		"providerId": 1,
		"itemName": "iPhone Case",
		"totalAmount": "300.00",
		"numMonths": 3,
		"purchaseDate": "2024-03-20",
		"accountId": 1
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loans", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.CreateLoan(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rec.Code)
	}

	var response LoanResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.ItemName != "iPhone Case" {
		t.Errorf("Expected item name 'iPhone Case', got %s", response.ItemName)
	}

	if response.MonthlyPayment != "100.00" {
		t.Errorf("Expected monthly payment '100.00', got %s", response.MonthlyPayment)
	}

	if response.FirstPaymentYear != 2024 || response.FirstPaymentMonth != 3 {
		t.Errorf("Expected first payment 2024-03, got %d-%d", response.FirstPaymentYear, response.FirstPaymentMonth)
	}
}

func TestCreateLoan_WithInterestRateOverride(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	loanService := createTestLoanService(loanRepo, providerRepo)
	handler := NewLoanHandler(loanService)

	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         1,
		Name:                "Bank",
		CutoffDay:           15,
		DefaultInterestRate: decimal.NewFromInt(5),
	})

	reqBody := `{
		"providerId": 1,
		"itemName": "Laptop",
		"totalAmount": "1000.00",
		"numMonths": 10,
		"purchaseDate": "2024-03-10",
		"interestRate": "10.00",
		"accountId": 1
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loans", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.CreateLoan(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rec.Code)
	}

	var response LoanResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Interest should be 10% (override), monthly = 1000 * 1.10 / 10 = 110
	if response.InterestRate != "10.00" {
		t.Errorf("Expected interest rate '10.00', got %s", response.InterestRate)
	}

	if response.MonthlyPayment != "110.00" {
		t.Errorf("Expected monthly payment '110.00', got %s", response.MonthlyPayment)
	}
}

func TestCreateLoan_EmptyItemName(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	loanService := createTestLoanService(loanRepo, providerRepo)
	handler := NewLoanHandler(loanService)

	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Provider",
		CutoffDay:   25,
	})

	reqBody := `{
		"providerId": 1,
		"itemName": "",
		"totalAmount": "100.00",
		"numMonths": 3,
		"purchaseDate": "2024-03-20",
		"accountId": 1
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loans", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.CreateLoan(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestCreateLoan_InvalidProvider(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	loanService := createTestLoanService(loanRepo, providerRepo)
	handler := NewLoanHandler(loanService)

	reqBody := `{
		"providerId": 999,
		"itemName": "Test",
		"totalAmount": "100.00",
		"numMonths": 3,
		"purchaseDate": "2024-03-20",
		"accountId": 1
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loans", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.CreateLoan(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestGetLoans_Success(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	loanService := createTestLoanService(loanRepo, providerRepo)
	handler := NewLoanHandler(loanService)

	// Set up loans with stats
	loanRepo.SetLoansWithStats([]*domain.LoanWithStats{
		{
			Loan: domain.Loan{
				ID:          1,
				WorkspaceID: 1,
				ProviderID:  1,
				ItemName:    "Loan 1",
				TotalAmount: decimal.NewFromInt(100),
			},
			TotalCount:       3,
			PaidCount:        1,
			RemainingBalance: decimal.NewFromInt(66),
			Progress:         33.33,
		},
		{
			Loan: domain.Loan{
				ID:          2,
				WorkspaceID: 1,
				ProviderID:  1,
				ItemName:    "Loan 2",
				TotalAmount: decimal.NewFromInt(200),
			},
			TotalCount:       6,
			PaidCount:        2,
			RemainingBalance: decimal.NewFromInt(133),
			Progress:         33.33,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/loans", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.GetLoans(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response []LoanWithStatsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("Expected 2 loans, got %d", len(response))
	}

	// Check stats fields are present
	if response[0].TotalCount != 3 || response[0].PaidCount != 1 {
		t.Errorf("Expected 3 total, 1 paid, got %d total, %d paid", response[0].TotalCount, response[0].PaidCount)
	}
}

func TestGetLoans_WithStatusFilter(t *testing.T) {
	tests := []struct {
		name           string
		statusParam    string
		expectedStatus int
		setupMock      func(*testutil.MockLoanRepository)
		expectedCount  int
	}{
		{
			name:           "filter active loans",
			statusParam:    "active",
			expectedStatus: http.StatusOK,
			setupMock: func(repo *testutil.MockLoanRepository) {
				repo.SetActiveWithStats([]*domain.LoanWithStats{
					{
						Loan: domain.Loan{
							ID:          1,
							WorkspaceID: 1,
							ItemName:    "Active Loan",
						},
						RemainingBalance: decimal.NewFromInt(100),
					},
				})
			},
			expectedCount: 1,
		},
		{
			name:           "filter completed loans",
			statusParam:    "completed",
			expectedStatus: http.StatusOK,
			setupMock: func(repo *testutil.MockLoanRepository) {
				repo.SetCompletedWithStats([]*domain.LoanWithStats{
					{
						Loan: domain.Loan{
							ID:          2,
							WorkspaceID: 1,
							ItemName:    "Completed Loan",
						},
						RemainingBalance: decimal.Zero,
					},
				})
			},
			expectedCount: 1,
		},
		{
			name:           "filter all loans",
			statusParam:    "all",
			expectedStatus: http.StatusOK,
			setupMock: func(repo *testutil.MockLoanRepository) {
				repo.SetLoansWithStats([]*domain.LoanWithStats{
					{Loan: domain.Loan{ID: 1, WorkspaceID: 1, ItemName: "Loan 1"}},
					{Loan: domain.Loan{ID: 2, WorkspaceID: 1, ItemName: "Loan 2"}},
				})
			},
			expectedCount: 2,
		},
		{
			name:           "invalid status parameter",
			statusParam:    "invalid",
			expectedStatus: http.StatusBadRequest,
			setupMock:      func(repo *testutil.MockLoanRepository) {},
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			loanRepo := testutil.NewMockLoanRepository()
			providerRepo := testutil.NewMockLoanProviderRepository()
			loanService := createTestLoanService(loanRepo, providerRepo)
			handler := NewLoanHandler(loanService)

			tt.setupMock(loanRepo)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/loans?status="+tt.statusParam, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

			err := handler.GetLoans(c)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response []LoanWithStatsResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if len(response) != tt.expectedCount {
					t.Errorf("Expected %d loans, got %d", tt.expectedCount, len(response))
				}
			}
		})
	}
}

func TestGetLoan_Success(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	loanService := createTestLoanService(loanRepo, providerRepo)
	handler := NewLoanHandler(loanService)

	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: 1,
		ProviderID:  1,
		ItemName:    "Test Loan",
		TotalAmount: decimal.NewFromInt(100),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/loans/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.GetLoan(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response LoanResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.ItemName != "Test Loan" {
		t.Errorf("Expected 'Test Loan', got %s", response.ItemName)
	}
}

func TestGetLoan_NotFound(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	loanService := createTestLoanService(loanRepo, providerRepo)
	handler := NewLoanHandler(loanService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/loans/999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.GetLoan(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestDeleteLoan_Success(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	loanService := createTestLoanService(loanRepo, providerRepo)
	handler := NewLoanHandler(loanService)

	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: 1,
		ItemName:    "Test Loan",
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/loans/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.DeleteLoan(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", rec.Code)
	}
}

func TestDeleteLoan_NotFound(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	loanService := createTestLoanService(loanRepo, providerRepo)
	handler := NewLoanHandler(loanService)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/loans/999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.DeleteLoan(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestPreviewLoan_Success(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	loanService := createTestLoanService(loanRepo, providerRepo)
	handler := NewLoanHandler(loanService)

	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         1,
		Name:                "SPayLater",
		CutoffDay:           25,
		DefaultInterestRate: decimal.Zero,
	})

	reqBody := `{
		"providerId": 1,
		"totalAmount": "300.00",
		"numMonths": 3,
		"purchaseDate": "2024-03-20"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loans/preview", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.PreviewLoan(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response PreviewLoanResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.MonthlyPayment != "100.00" {
		t.Errorf("Expected monthly payment '100.00', got %s", response.MonthlyPayment)
	}

	if response.FirstPaymentYear != 2024 || response.FirstPaymentMonth != 3 {
		t.Errorf("Expected first payment 2024-03, got %d-%d", response.FirstPaymentYear, response.FirstPaymentMonth)
	}
}

func TestPreviewLoan_InvalidProvider(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	loanService := createTestLoanService(loanRepo, providerRepo)
	handler := NewLoanHandler(loanService)

	reqBody := `{
		"providerId": 999,
		"totalAmount": "100.00",
		"numMonths": 3,
		"purchaseDate": "2024-03-20"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loans/preview", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.PreviewLoan(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestGetLoans_WorkspaceIsolation(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	loanService := createTestLoanService(loanRepo, providerRepo)
	handler := NewLoanHandler(loanService)

	// Set up loans with stats for workspace 1
	loanRepo.SetLoansWithStats([]*domain.LoanWithStats{
		{
			Loan: domain.Loan{
				ID:          1,
				WorkspaceID: 1,
				ProviderID:  1,
				ItemName:    "Workspace 1 Loan",
				TotalAmount: decimal.NewFromInt(100),
			},
		},
	})

	// Query as workspace 1 - should only see workspace 1's loan
	req := httptest.NewRequest(http.MethodGet, "/api/v1/loans", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setupAuthContextWithWorkspace(c, "auth0|user1", "user1@example.com", "User 1", "", 1)

	err := handler.GetLoans(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var response1 []LoanWithStatsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response1); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response1) != 1 {
		t.Errorf("Workspace 1 should see 1 loan, got %d", len(response1))
	}
	if len(response1) > 0 && response1[0].ItemName != "Workspace 1 Loan" {
		t.Errorf("Workspace 1 should see 'Workspace 1 Loan', got %s", response1[0].ItemName)
	}
}

// UpdateLoan tests

func TestUpdateLoan_Success(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	loanService := createTestLoanService(loanRepo, providerRepo)
	handler := NewLoanHandler(loanService)

	// Add provider that the loan references
	providerRepo.AddProvider(&domain.LoanProvider{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test Provider",
	})

	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: 1,
		ProviderID:  1,
		ItemName:    "Original Name",
		TotalAmount: decimal.NewFromInt(100),
	})

	reqBody := `{"itemName": "Updated Name", "notes": "New notes"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/loans/1", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.UpdateLoan(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response LoanResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.ItemName != "Updated Name" {
		t.Errorf("Expected 'Updated Name', got %s", response.ItemName)
	}
}

func TestUpdateLoan_EmptyItemName(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	loanService := createTestLoanService(loanRepo, providerRepo)
	handler := NewLoanHandler(loanService)

	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: 1,
		ProviderID:  1,
		ItemName:    "Original Name",
	})

	reqBody := `{"itemName": ""}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/loans/1", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.UpdateLoan(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestUpdateLoan_NotFound(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	loanService := createTestLoanService(loanRepo, providerRepo)
	handler := NewLoanHandler(loanService)

	reqBody := `{"itemName": "New Name"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/loans/999", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.UpdateLoan(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestUpdateLoan_WorkspaceIsolation(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	loanService := createTestLoanService(loanRepo, providerRepo)
	handler := NewLoanHandler(loanService)

	// Create a loan in workspace 2
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: 2,
		ProviderID:  1,
		ItemName:    "Workspace 2 Loan",
	})

	// Try to update from workspace 1
	reqBody := `{"itemName": "Hacked Name"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/loans/1", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	setupAuthContextWithWorkspace(c, "auth0|user1", "user1@example.com", "User 1", "", 1)

	err := handler.UpdateLoan(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Workspace 1 should not update workspace 2's loan, expected 404 but got %d", rec.Code)
	}
}

func TestGetLoan_WorkspaceIsolation(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	loanService := createTestLoanService(loanRepo, providerRepo)
	handler := NewLoanHandler(loanService)

	// Create a loan in workspace 2
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: 2,
		ProviderID:  1,
		ItemName:    "Workspace 2 Loan",
		TotalAmount: decimal.NewFromInt(100),
	})

	// Try to access from workspace 1 - should get not found
	req := httptest.NewRequest(http.MethodGet, "/api/v1/loans/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setupAuthContextWithWorkspace(c, "auth0|user1", "user1@example.com", "User 1", "", 1)

	err := handler.GetLoan(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Workspace 1 should not see workspace 2's loan, expected 404 but got %d", rec.Code)
	}

	// Access from workspace 2 - should succeed
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/loans/1", nil)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	c2.SetParamNames("id")
	c2.SetParamValues("1")
	setupAuthContextWithWorkspace(c2, "auth0|user2", "user2@example.com", "User 2", "", 2)

	err = handler.GetLoan(c2)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec2.Code != http.StatusOK {
		t.Errorf("Workspace 2 should see its own loan, expected 200 but got %d", rec2.Code)
	}
}

// GetDeleteCheck tests

func TestGetDeleteCheck_Success(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	loanService, transactionRepo := createTestLoanServiceWithTransactionRepo(loanRepo, providerRepo)
	handler := NewLoanHandler(loanService)

	loanID := int32(1)
	workspaceID := int32(1)
	loanRepo.AddLoan(&domain.Loan{
		ID:             loanID,
		WorkspaceID:    workspaceID,
		ProviderID:     1,
		ItemName:       "Test Loan",
		TotalAmount:    decimal.NewFromInt(300),
		NumMonths:      3,
		MonthlyPayment: decimal.NewFromInt(100),
	})

	// v2: Add transactions for the loan: 1 paid, 2 unpaid
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          1,
		WorkspaceID: workspaceID,
		LoanID:      &loanID,
		Amount:      decimal.NewFromInt(-100),
		IsPaid:      true,
		Name:        "Test Loan Payment 1",
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          2,
		WorkspaceID: workspaceID,
		LoanID:      &loanID,
		Amount:      decimal.NewFromInt(-100),
		IsPaid:      false,
		Name:        "Test Loan Payment 2",
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          3,
		WorkspaceID: workspaceID,
		LoanID:      &loanID,
		Amount:      decimal.NewFromInt(-100),
		IsPaid:      false,
		Name:        "Test Loan Payment 3",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/loans/1/delete-check", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.GetDeleteCheck(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response DeleteCheckResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.LoanID != 1 {
		t.Errorf("Expected loan ID 1, got %d", response.LoanID)
	}

	if response.ItemName != "Test Loan" {
		t.Errorf("Expected 'Test Loan', got %s", response.ItemName)
	}

	// v2: Stats come from transactions table
	if response.PaidCount != 1 {
		t.Errorf("Expected paid count 1, got %d", response.PaidCount)
	}

	if response.UnpaidCount != 2 {
		t.Errorf("Expected unpaid count 2, got %d", response.UnpaidCount)
	}

	if response.TotalAmount != "300.00" {
		t.Errorf("Expected total amount '300.00', got %s", response.TotalAmount)
	}
}

func TestGetDeleteCheck_NotFound(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	loanService := createTestLoanService(loanRepo, providerRepo)
	handler := NewLoanHandler(loanService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/loans/999/delete-check", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.GetDeleteCheck(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestGetDeleteCheck_WorkspaceIsolation(t *testing.T) {
	e := echo.New()
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	loanService := createTestLoanService(loanRepo, providerRepo)
	handler := NewLoanHandler(loanService)

	// Create a loan in workspace 2
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: 2,
		ProviderID:  1,
		ItemName:    "Workspace 2 Loan",
	})

	// Try to access from workspace 1
	req := httptest.NewRequest(http.MethodGet, "/api/v1/loans/1/delete-check", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	setupAuthContextWithWorkspace(c, "auth0|user1", "user1@example.com", "User 1", "", 1)

	err := handler.GetDeleteCheck(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Workspace 1 should not see workspace 2's loan, expected 404 but got %d", rec.Code)
	}
}
