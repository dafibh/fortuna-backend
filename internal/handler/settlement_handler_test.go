package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/middleware"
	"github.com/dafibh/fortuna/fortuna-backend/internal/service"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
)

func TestSettlementHandler_Create_Success(t *testing.T) {
	// Setup
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Create accounts
	bankAccount := &domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Bank",
		Template:    domain.TemplateBank,
	}
	accountRepo.AddAccount(bankAccount)

	ccAccount := &domain.Account{
		ID:          2,
		WorkspaceID: 1,
		Name:        "CC",
		Template:    domain.TemplateCreditCard,
	}
	accountRepo.AddAccount(ccAccount)

	// Create billed transactions
	billedState := domain.CCStateBilled
	deferredIntent := domain.SettlementIntentDeferred
	tx := &domain.Transaction{
		ID:               1,
		WorkspaceID:      1,
		AccountID:        2,
		Amount:           decimal.NewFromFloat(100.00),
		CCState:          &billedState,
		SettlementIntent: &deferredIntent,
	}
	transactionRepo.AddTransaction(tx)

	// Track created transfer
	transactionRepo.CreateFn = func(tx *domain.Transaction) (*domain.Transaction, error) {
		tx.ID = 99
		tx.CreatedAt = time.Now()
		tx.UpdatedAt = time.Now()
		return tx, nil
	}

	settlementService := service.NewSettlementService(transactionRepo, accountRepo)
	handler := NewSettlementHandler(settlementService)

	// Create request
	reqBody := SettlementRequest{
		TransactionIDs:    []int32{1},
		SourceAccountID:   1,
		TargetCCAccountID: 2,
	}
	body, _ := json.Marshal(reqBody)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settlements", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	ctx := context.WithValue(c.Request().Context(), middleware.WorkspaceIDKey, int32(1))
	c.SetRequest(c.Request().WithContext(ctx))

	// Execute
	err := handler.Create(c)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var response SettlementResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if response.TransferID != 99 {
		t.Errorf("expected transfer ID 99, got %d", response.TransferID)
	}
	if response.SettledCount != 1 {
		t.Errorf("expected settled count 1, got %d", response.SettledCount)
	}
	if response.TotalAmount != "100.00" {
		t.Errorf("expected total amount 100.00, got %s", response.TotalAmount)
	}
}

func TestSettlementHandler_Create_MissingWorkspace(t *testing.T) {
	handler := NewSettlementHandler(nil)

	reqBody := SettlementRequest{
		TransactionIDs:    []int32{1},
		SourceAccountID:   1,
		TargetCCAccountID: 2,
	}
	body, _ := json.Marshal(reqBody)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settlements", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Not setting workspace ID

	err := handler.Create(c)

	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestSettlementHandler_Create_InvalidJSON(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	settlementService := service.NewSettlementService(transactionRepo, accountRepo)
	handler := NewSettlementHandler(settlementService)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settlements", bytes.NewReader([]byte("invalid json")))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	ctx := context.WithValue(c.Request().Context(), middleware.WorkspaceIDKey, int32(1))
	c.SetRequest(c.Request().WithContext(ctx))

	err := handler.Create(c)

	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestSettlementHandler_Create_EmptyTransactionIDs(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	settlementService := service.NewSettlementService(transactionRepo, accountRepo)
	handler := NewSettlementHandler(settlementService)

	reqBody := SettlementRequest{
		TransactionIDs:    []int32{},
		SourceAccountID:   1,
		TargetCCAccountID: 2,
	}
	body, _ := json.Marshal(reqBody)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settlements", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	ctx := context.WithValue(c.Request().Context(), middleware.WorkspaceIDKey, int32(1))
	c.SetRequest(c.Request().WithContext(ctx))

	err := handler.Create(c)

	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestSettlementHandler_Create_TransactionsNotFound(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Create accounts but no transactions
	bankAccount := &domain.Account{ID: 1, WorkspaceID: 1, Template: domain.TemplateBank}
	accountRepo.AddAccount(bankAccount)
	ccAccount := &domain.Account{ID: 2, WorkspaceID: 1, Template: domain.TemplateCreditCard}
	accountRepo.AddAccount(ccAccount)

	settlementService := service.NewSettlementService(transactionRepo, accountRepo)
	handler := NewSettlementHandler(settlementService)

	reqBody := SettlementRequest{
		TransactionIDs:    []int32{999},
		SourceAccountID:   1,
		TargetCCAccountID: 2,
	}
	body, _ := json.Marshal(reqBody)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settlements", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	ctx := context.WithValue(c.Request().Context(), middleware.WorkspaceIDKey, int32(1))
	c.SetRequest(c.Request().WithContext(ctx))

	err := handler.Create(c)

	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestSettlementHandler_Create_TransactionNotBilled(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Create accounts
	bankAccount := &domain.Account{ID: 1, WorkspaceID: 1, Template: domain.TemplateBank}
	accountRepo.AddAccount(bankAccount)
	ccAccount := &domain.Account{ID: 2, WorkspaceID: 1, Template: domain.TemplateCreditCard}
	accountRepo.AddAccount(ccAccount)

	// Create pending transaction (not billed)
	pendingState := domain.CCStatePending
	tx := &domain.Transaction{
		ID:          1,
		WorkspaceID: 1,
		AccountID:   2,
		Amount:      decimal.NewFromFloat(50.00),
		CCState:     &pendingState,
	}
	transactionRepo.AddTransaction(tx)

	settlementService := service.NewSettlementService(transactionRepo, accountRepo)
	handler := NewSettlementHandler(settlementService)

	reqBody := SettlementRequest{
		TransactionIDs:    []int32{1},
		SourceAccountID:   1,
		TargetCCAccountID: 2,
	}
	body, _ := json.Marshal(reqBody)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settlements", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	ctx := context.WithValue(c.Request().Context(), middleware.WorkspaceIDKey, int32(1))
	c.SetRequest(c.Request().WithContext(ctx))

	err := handler.Create(c)

	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}
}

func TestSettlementHandler_Create_InvalidSourceAccount(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Create source as CC (invalid)
	sourceCC := &domain.Account{ID: 1, WorkspaceID: 1, Template: domain.TemplateCreditCard}
	accountRepo.AddAccount(sourceCC)
	targetCC := &domain.Account{ID: 2, WorkspaceID: 1, Template: domain.TemplateCreditCard}
	accountRepo.AddAccount(targetCC)

	// Create valid billed transaction
	billedState := domain.CCStateBilled
	deferredIntent := domain.SettlementIntentDeferred
	tx := &domain.Transaction{
		ID:               1,
		WorkspaceID:      1,
		AccountID:        2,
		Amount:           decimal.NewFromFloat(50.00),
		CCState:          &billedState,
		SettlementIntent: &deferredIntent,
	}
	transactionRepo.AddTransaction(tx)

	settlementService := service.NewSettlementService(transactionRepo, accountRepo)
	handler := NewSettlementHandler(settlementService)

	reqBody := SettlementRequest{
		TransactionIDs:    []int32{1},
		SourceAccountID:   1,
		TargetCCAccountID: 2,
	}
	body, _ := json.Marshal(reqBody)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settlements", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	ctx := context.WithValue(c.Request().Context(), middleware.WorkspaceIDKey, int32(1))
	c.SetRequest(c.Request().WithContext(ctx))

	err := handler.Create(c)

	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestSettlementHandler_Create_InvalidTargetAccount(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Create valid source bank
	bankAccount := &domain.Account{ID: 1, WorkspaceID: 1, Template: domain.TemplateBank}
	accountRepo.AddAccount(bankAccount)
	// Create invalid target (not CC)
	anotherBank := &domain.Account{ID: 2, WorkspaceID: 1, Template: domain.TemplateBank}
	accountRepo.AddAccount(anotherBank)

	// Create billed transaction
	billedState := domain.CCStateBilled
	deferredIntent := domain.SettlementIntentDeferred
	tx := &domain.Transaction{
		ID:               1,
		WorkspaceID:      1,
		AccountID:        2,
		Amount:           decimal.NewFromFloat(50.00),
		CCState:          &billedState,
		SettlementIntent: &deferredIntent,
	}
	transactionRepo.AddTransaction(tx)

	settlementService := service.NewSettlementService(transactionRepo, accountRepo)
	handler := NewSettlementHandler(settlementService)

	reqBody := SettlementRequest{
		TransactionIDs:    []int32{1},
		SourceAccountID:   1,
		TargetCCAccountID: 2,
	}
	body, _ := json.Marshal(reqBody)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settlements", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	ctx := context.WithValue(c.Request().Context(), middleware.WorkspaceIDKey, int32(1))
	c.SetRequest(c.Request().WithContext(ctx))

	err := handler.Create(c)

	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
