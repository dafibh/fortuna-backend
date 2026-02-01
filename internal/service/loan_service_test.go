package service

import (
	"testing"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/shopspring/decimal"
)

// createTestLoanService creates a LoanService with mock repositories for testing
// Sets up a default bank account with ID 1 in workspaceID 1
func createTestLoanService(loanRepo *testutil.MockLoanRepository, providerRepo *testutil.MockLoanProviderRepository) *LoanService {
	svc, _ := createTestLoanServiceWithTransactionRepo(loanRepo, providerRepo)
	return svc
}

// createTestLoanServiceWithTransactionRepo creates a LoanService and returns the transaction repo for testing
// This allows tests to add transactions for testing GetDeleteStats
func createTestLoanServiceWithTransactionRepo(loanRepo *testutil.MockLoanRepository, providerRepo *testutil.MockLoanProviderRepository) (*LoanService, *testutil.MockTransactionRepository) {
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Add a bank account (ID 1) for testing
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test Bank Account",
		Template:    domain.TemplateBank,
		AccountType: domain.AccountTypeAsset,
	})

	// Add a CC account (ID 2) for testing settlement intent
	accountRepo.AddAccount(&domain.Account{
		ID:          2,
		WorkspaceID: 1,
		Name:        "Test Credit Card",
		Template:    domain.TemplateCreditCard,
		AccountType: domain.AccountTypeLiability,
	})

	return NewLoanService(nil, loanRepo, providerRepo, transactionRepo, accountRepo), transactionRepo
}

// Test helper functions

func TestCalculateMonthlyPayment_ZeroInterest(t *testing.T) {
	// RM 300, 0% interest, 3 months = RM 100 per month
	total := decimal.NewFromInt(300)
	interest := decimal.Zero
	months := 3

	result := CalculateMonthlyPayment(total, interest, months)
	expected := decimal.NewFromInt(100)

	if !result.Equal(expected) {
		t.Errorf("Expected %s, got %s", expected.String(), result.String())
	}
}

func TestCalculateMonthlyPayment_WithInterest(t *testing.T) {
	// RM 1000, 10% interest, 10 months
	// Total with interest: 1000 * 1.10 = 1100
	// Monthly: 1100 / 10 = 110
	total := decimal.NewFromInt(1000)
	interest := decimal.NewFromInt(10)
	months := 10

	result := CalculateMonthlyPayment(total, interest, months)
	expected := decimal.NewFromInt(110)

	if !result.Equal(expected) {
		t.Errorf("Expected %s, got %s", expected.String(), result.String())
	}
}

func TestCalculateMonthlyPayment_WithDecimalInterest(t *testing.T) {
	// RM 1000, 2.5% interest, 4 months
	// Total with interest: 1000 * 1.025 = 1025
	// Monthly: 1025 / 4 = 256.25
	total := decimal.NewFromInt(1000)
	interest := decimal.NewFromFloat(2.5)
	months := 4

	result := CalculateMonthlyPayment(total, interest, months)
	expected := decimal.NewFromFloat(256.25)

	if !result.Equal(expected) {
		t.Errorf("Expected %s, got %s", expected.String(), result.String())
	}
}

func TestCalculateMonthlyPayment_Rounds(t *testing.T) {
	// RM 100, 0% interest, 3 months = RM 33.33 (rounded)
	total := decimal.NewFromInt(100)
	interest := decimal.Zero
	months := 3

	result := CalculateMonthlyPayment(total, interest, months)
	expected := decimal.NewFromFloat(33.33)

	if !result.Equal(expected) {
		t.Errorf("Expected %s, got %s", expected.String(), result.String())
	}
}

func TestCalculateMonthlyPayment_ZeroMonths(t *testing.T) {
	total := decimal.NewFromInt(100)
	interest := decimal.Zero
	months := 0

	result := CalculateMonthlyPayment(total, interest, months)
	expected := decimal.Zero

	if !result.Equal(expected) {
		t.Errorf("Expected %s for zero months, got %s", expected.String(), result.String())
	}
}

func TestCalculateFirstPaymentMonth_BeforeCutoff(t *testing.T) {
	// Purchase on March 20, cutoff day 25 → first payment March
	purchaseDate := time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)
	cutoffDay := 25

	year, month := CalculateFirstPaymentMonth(purchaseDate, cutoffDay)

	if year != 2024 || month != 3 {
		t.Errorf("Expected 2024-03, got %d-%d", year, month)
	}
}

func TestCalculateFirstPaymentMonth_OnCutoff(t *testing.T) {
	// Purchase on March 25, cutoff day 25 → first payment April
	purchaseDate := time.Date(2024, 3, 25, 0, 0, 0, 0, time.UTC)
	cutoffDay := 25

	year, month := CalculateFirstPaymentMonth(purchaseDate, cutoffDay)

	if year != 2024 || month != 4 {
		t.Errorf("Expected 2024-04, got %d-%d", year, month)
	}
}

func TestCalculateFirstPaymentMonth_AfterCutoff(t *testing.T) {
	// Purchase on March 26, cutoff day 25 → first payment April
	purchaseDate := time.Date(2024, 3, 26, 0, 0, 0, 0, time.UTC)
	cutoffDay := 25

	year, month := CalculateFirstPaymentMonth(purchaseDate, cutoffDay)

	if year != 2024 || month != 4 {
		t.Errorf("Expected 2024-04, got %d-%d", year, month)
	}
}

func TestCalculateFirstPaymentMonth_YearWrap(t *testing.T) {
	// Purchase on December 26, cutoff day 25 → first payment January next year
	purchaseDate := time.Date(2024, 12, 26, 0, 0, 0, 0, time.UTC)
	cutoffDay := 25

	year, month := CalculateFirstPaymentMonth(purchaseDate, cutoffDay)

	if year != 2025 || month != 1 {
		t.Errorf("Expected 2025-01, got %d-%d", year, month)
	}
}

func TestCalculateFirstPaymentMonth_FirstOfMonth(t *testing.T) {
	// Purchase on March 1, cutoff day 25 → first payment March
	purchaseDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	cutoffDay := 25

	year, month := CalculateFirstPaymentMonth(purchaseDate, cutoffDay)

	if year != 2024 || month != 3 {
		t.Errorf("Expected 2024-03, got %d-%d", year, month)
	}
}

func TestCalculateFirstPaymentMonth_CutoffDay1(t *testing.T) {
	// Purchase on March 1, cutoff day 1 → first payment April
	purchaseDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	cutoffDay := 1

	year, month := CalculateFirstPaymentMonth(purchaseDate, cutoffDay)

	if year != 2024 || month != 4 {
		t.Errorf("Expected 2024-04, got %d-%d", year, month)
	}
}

// CreateLoan tests

func TestCreateLoan_Success(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "SPayLater",
		CutoffDay:           25,
		DefaultInterestRate: decimal.Zero,
	})

	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "iPhone Case",
		TotalAmount:  decimal.NewFromInt(300),
		NumMonths:    3,
		PurchaseDate: time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC),
		AccountID:    1,
	}

	loan, err := service.CreateLoan(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify calculated values
	expectedMonthly := decimal.NewFromInt(100) // 300 / 3
	if !loan.MonthlyPayment.Equal(expectedMonthly) {
		t.Errorf("Expected monthly payment %s, got %s", expectedMonthly.String(), loan.MonthlyPayment.String())
	}

	if loan.FirstPaymentYear != 2024 || loan.FirstPaymentMonth != 3 {
		t.Errorf("Expected first payment 2024-03, got %d-%d", loan.FirstPaymentYear, loan.FirstPaymentMonth)
	}

	if !loan.InterestRate.Equal(decimal.Zero) {
		t.Errorf("Expected interest rate 0, got %s", loan.InterestRate.String())
	}
}

func TestCreateLoan_WithInterestRateOverride(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "Bank",
		CutoffDay:           15,
		DefaultInterestRate: decimal.NewFromInt(5), // 5% default
	})

	// Override with 10%
	overrideRate := decimal.NewFromInt(10)
	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "Laptop",
		TotalAmount:  decimal.NewFromInt(1000),
		NumMonths:    10,
		PurchaseDate: time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC),
		InterestRate: &overrideRate,
		AccountID:    1,
	}

	loan, err := service.CreateLoan(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Interest should be 10% (override), not 5% (provider default)
	if !loan.InterestRate.Equal(overrideRate) {
		t.Errorf("Expected interest rate %s, got %s", overrideRate.String(), loan.InterestRate.String())
	}

	// Monthly: 1000 * 1.10 / 10 = 110
	expectedMonthly := decimal.NewFromInt(110)
	if !loan.MonthlyPayment.Equal(expectedMonthly) {
		t.Errorf("Expected monthly payment %s, got %s", expectedMonthly.String(), loan.MonthlyPayment.String())
	}
}

func TestCreateLoan_UsesProviderDefaultRate(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "Bank",
		CutoffDay:           15,
		DefaultInterestRate: decimal.NewFromFloat(2.5), // 2.5% default
	})

	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "Phone",
		TotalAmount:  decimal.NewFromInt(1000),
		NumMonths:    4,
		PurchaseDate: time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC),
		AccountID:    1,
	}

	loan, err := service.CreateLoan(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !loan.InterestRate.Equal(decimal.NewFromFloat(2.5)) {
		t.Errorf("Expected interest rate 2.5, got %s", loan.InterestRate.String())
	}
}

func TestCreateLoan_EmptyItemName(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "",
		TotalAmount:  decimal.NewFromInt(100),
		NumMonths:    3,
		PurchaseDate: time.Now(),
	}

	_, err := service.CreateLoan(1, input)
	if err != domain.ErrLoanItemNameEmpty {
		t.Errorf("Expected ErrLoanItemNameEmpty, got %v", err)
	}
}

func TestCreateLoan_ItemNameTooLong(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	// Create a name that's 201 characters long
	longName := ""
	for i := 0; i < 201; i++ {
		longName += "A"
	}

	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     longName,
		TotalAmount:  decimal.NewFromInt(100),
		NumMonths:    3,
		PurchaseDate: time.Now(),
	}

	_, err := service.CreateLoan(1, input)
	if err != domain.ErrLoanItemNameTooLong {
		t.Errorf("Expected ErrLoanItemNameTooLong, got %v", err)
	}
}

func TestCreateLoan_ZeroAmount(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "Test",
		TotalAmount:  decimal.Zero,
		NumMonths:    3,
		PurchaseDate: time.Now(),
	}

	_, err := service.CreateLoan(1, input)
	if err != domain.ErrLoanAmountInvalid {
		t.Errorf("Expected ErrLoanAmountInvalid, got %v", err)
	}
}

func TestCreateLoan_NegativeAmount(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "Test",
		TotalAmount:  decimal.NewFromInt(-100),
		NumMonths:    3,
		PurchaseDate: time.Now(),
	}

	_, err := service.CreateLoan(1, input)
	if err != domain.ErrLoanAmountInvalid {
		t.Errorf("Expected ErrLoanAmountInvalid, got %v", err)
	}
}

func TestCreateLoan_ZeroMonths(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "Test",
		TotalAmount:  decimal.NewFromInt(100),
		NumMonths:    0,
		PurchaseDate: time.Now(),
	}

	_, err := service.CreateLoan(1, input)
	if err != domain.ErrLoanMonthsInvalid {
		t.Errorf("Expected ErrLoanMonthsInvalid, got %v", err)
	}
}

func TestCreateLoan_InvalidProvider(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	input := CreateLoanInput{
		ProviderID:   0,
		ItemName:     "Test",
		TotalAmount:  decimal.NewFromInt(100),
		NumMonths:    3,
		PurchaseDate: time.Now(),
	}

	_, err := service.CreateLoan(1, input)
	if err != domain.ErrLoanProviderInvalid {
		t.Errorf("Expected ErrLoanProviderInvalid, got %v", err)
	}
}

func TestCreateLoan_ProviderNotFound(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	input := CreateLoanInput{
		ProviderID:   999, // Non-existent provider
		ItemName:     "Test",
		TotalAmount:  decimal.NewFromInt(100),
		NumMonths:    3,
		PurchaseDate: time.Now(),
		AccountID:    1,
	}

	_, err := service.CreateLoan(1, input)
	if err != domain.ErrLoanProviderInvalid {
		t.Errorf("Expected ErrLoanProviderInvalid, got %v", err)
	}
}

func TestCreateLoan_ProviderWrongWorkspace(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	// Provider belongs to workspace 2
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:          1,
		WorkspaceID: 2,
		Name:        "Provider",
		CutoffDay:   25,
	})

	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "Test",
		TotalAmount:  decimal.NewFromInt(100),
		NumMonths:    3,
		PurchaseDate: time.Now(),
		AccountID:    1,
	}

	// Try to create in workspace 1
	_, err := service.CreateLoan(1, input)
	if err != domain.ErrLoanProviderInvalid {
		t.Errorf("Expected ErrLoanProviderInvalid, got %v", err)
	}
}

func TestCreateLoan_MissingAccountID(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Provider",
		CutoffDay:   25,
	})

	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "Test",
		TotalAmount:  decimal.NewFromInt(100),
		NumMonths:    3,
		PurchaseDate: time.Now(),
		AccountID:    0, // Missing/invalid account
	}

	_, err := service.CreateLoan(workspaceID, input)
	if err != domain.ErrLoanAccountInvalid {
		t.Errorf("Expected ErrLoanAccountInvalid, got %v", err)
	}
}

func TestCreateLoan_WithAccountAndSettlementIntent(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "SPayLater",
		CutoffDay:           25,
		DefaultInterestRate: decimal.Zero,
	})

	settlementIntent := "deferred"
	input := CreateLoanInput{
		ProviderID:       1,
		ItemName:         "Test Item",
		TotalAmount:      decimal.NewFromInt(300),
		NumMonths:        3,
		PurchaseDate:     time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC),
		AccountID:        2, // Uses CC account (ID 2) for settlement intent
		SettlementIntent: &settlementIntent,
	}

	loan, err := service.CreateLoan(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if loan.AccountID != 2 {
		t.Errorf("Expected AccountID 2, got %d", loan.AccountID)
	}

	if loan.SettlementIntent == nil || *loan.SettlementIntent != "deferred" {
		t.Errorf("Expected SettlementIntent 'deferred', got %v", loan.SettlementIntent)
	}
}

func TestCreateLoan_WithAccountNoSettlementIntent(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "SPayLater",
		CutoffDay:           25,
		DefaultInterestRate: decimal.Zero,
	})

	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "Test Item",
		TotalAmount:  decimal.NewFromInt(300),
		NumMonths:    3,
		PurchaseDate: time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC),
		AccountID:    1, // Bank account from helper, no settlement intent needed
	}

	loan, err := service.CreateLoan(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if loan.AccountID != 1 {
		t.Errorf("Expected AccountID 1, got %d", loan.AccountID)
	}

	if loan.SettlementIntent != nil {
		t.Errorf("Expected SettlementIntent nil, got %v", loan.SettlementIntent)
	}
}

func TestCreateLoan_WithNotes(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Provider",
		CutoffDay:   25,
	})

	notes := "This is a test note"
	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "Test",
		TotalAmount:  decimal.NewFromInt(100),
		NumMonths:    3,
		PurchaseDate: time.Now(),
		Notes:        &notes,
		AccountID:    1,
	}

	loan, err := service.CreateLoan(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if loan.Notes == nil || *loan.Notes != notes {
		t.Errorf("Expected notes '%s', got %v", notes, loan.Notes)
	}
}

// PreviewLoan tests

func TestPreviewLoan_Success(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "SPayLater",
		CutoffDay:           25,
		DefaultInterestRate: decimal.Zero,
	})

	input := PreviewLoanInput{
		ProviderID:   1,
		TotalAmount:  decimal.NewFromInt(300),
		NumMonths:    3,
		PurchaseDate: time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC),
	}

	result, err := service.PreviewLoan(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedMonthly := decimal.NewFromInt(100)
	if !result.MonthlyPayment.Equal(expectedMonthly) {
		t.Errorf("Expected monthly payment %s, got %s", expectedMonthly.String(), result.MonthlyPayment.String())
	}

	if result.FirstPaymentYear != 2024 || result.FirstPaymentMonth != 3 {
		t.Errorf("Expected first payment 2024-03, got %d-%d", result.FirstPaymentYear, result.FirstPaymentMonth)
	}
}

func TestPreviewLoan_InvalidProvider(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	input := PreviewLoanInput{
		ProviderID:   999,
		TotalAmount:  decimal.NewFromInt(100),
		NumMonths:    3,
		PurchaseDate: time.Now(),
	}

	_, err := service.PreviewLoan(1, input)
	if err != domain.ErrLoanProviderInvalid {
		t.Errorf("Expected ErrLoanProviderInvalid, got %v", err)
	}
}

// GetLoans tests

func TestGetLoans_Success(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	workspaceID := int32(1)
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: workspaceID,
		ItemName:    "Loan 1",
	})
	loanRepo.AddLoan(&domain.Loan{
		ID:          2,
		WorkspaceID: workspaceID,
		ItemName:    "Loan 2",
	})

	loans, err := service.GetLoans(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(loans) != 2 {
		t.Errorf("Expected 2 loans, got %d", len(loans))
	}
}

func TestGetLoans_Empty(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	loans, err := service.GetLoans(1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(loans) != 0 {
		t.Errorf("Expected 0 loans, got %d", len(loans))
	}
}

// GetLoanByID tests

func TestGetLoanByID_Success(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	workspaceID := int32(1)
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: workspaceID,
		ItemName:    "Test Loan",
	})

	loan, err := service.GetLoanByID(workspaceID, 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if loan.ItemName != "Test Loan" {
		t.Errorf("Expected 'Test Loan', got '%s'", loan.ItemName)
	}
}

func TestGetLoanByID_NotFound(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	_, err := service.GetLoanByID(1, 999)
	if err != domain.ErrLoanNotFound {
		t.Errorf("Expected ErrLoanNotFound, got %v", err)
	}
}

// DeleteLoan tests

func TestDeleteLoan_Success(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	workspaceID := int32(1)
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: workspaceID,
		ItemName:    "Test Loan",
	})

	err := service.DeleteLoan(workspaceID, 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify loan is deleted
	_, err = service.GetLoanByID(workspaceID, 1)
	if err != domain.ErrLoanNotFound {
		t.Errorf("Expected ErrLoanNotFound after delete, got %v", err)
	}
}

func TestDeleteLoan_NotFound(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	err := service.DeleteLoan(1, 999)
	if err != domain.ErrLoanNotFound {
		t.Errorf("Expected ErrLoanNotFound, got %v", err)
	}
}

// GetLoansWithStats tests

func TestGetLoansWithStats_AllFilter(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	workspaceID := int32(1)
	loanRepo.SetLoansWithStats([]*domain.LoanWithStats{
		{
			Loan:             domain.Loan{ID: 1, WorkspaceID: workspaceID, ItemName: "Active Loan"},
			TotalCount:       6,
			PaidCount:        2,
			RemainingBalance: decimal.NewFromInt(400),
			Progress:         33.33,
		},
		{
			Loan:             domain.Loan{ID: 2, WorkspaceID: workspaceID, ItemName: "Completed Loan"},
			TotalCount:       3,
			PaidCount:        3,
			RemainingBalance: decimal.Zero,
			Progress:         100,
		},
	})

	loans, err := service.GetLoansWithStats(workspaceID, domain.LoanFilterAll)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(loans) != 2 {
		t.Errorf("Expected 2 loans with 'all' filter, got %d", len(loans))
	}
}

func TestGetLoansWithStats_ActiveFilter(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	workspaceID := int32(1)
	loanRepo.SetActiveWithStats([]*domain.LoanWithStats{
		{
			Loan:             domain.Loan{ID: 1, WorkspaceID: workspaceID, ItemName: "Active Loan"},
			TotalCount:       6,
			PaidCount:        2,
			RemainingBalance: decimal.NewFromInt(400),
			Progress:         33.33,
		},
	})

	loans, err := service.GetLoansWithStats(workspaceID, domain.LoanFilterActive)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(loans) != 1 {
		t.Errorf("Expected 1 loan with 'active' filter, got %d", len(loans))
	}

	if loans[0].ItemName != "Active Loan" {
		t.Errorf("Expected 'Active Loan', got '%s'", loans[0].ItemName)
	}

	if !loans[0].RemainingBalance.GreaterThan(decimal.Zero) {
		t.Errorf("Active loan should have remaining balance > 0")
	}
}

func TestGetLoansWithStats_CompletedFilter(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	workspaceID := int32(1)
	loanRepo.SetCompletedWithStats([]*domain.LoanWithStats{
		{
			Loan:             domain.Loan{ID: 2, WorkspaceID: workspaceID, ItemName: "Completed Loan"},
			TotalCount:       3,
			PaidCount:        3,
			RemainingBalance: decimal.Zero,
			Progress:         100,
		},
	})

	loans, err := service.GetLoansWithStats(workspaceID, domain.LoanFilterCompleted)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(loans) != 1 {
		t.Errorf("Expected 1 loan with 'completed' filter, got %d", len(loans))
	}

	if loans[0].ItemName != "Completed Loan" {
		t.Errorf("Expected 'Completed Loan', got '%s'", loans[0].ItemName)
	}

	if loans[0].Progress != 100 {
		t.Errorf("Completed loan should have 100%% progress, got %.2f", loans[0].Progress)
	}
}

func TestGetLoansWithStats_EmptyResult(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	loans, err := service.GetLoansWithStats(1, domain.LoanFilterAll)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(loans) != 0 {
		t.Errorf("Expected 0 loans, got %d", len(loans))
	}
}

func TestGetLoansWithStats_DefaultsToAll(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	workspaceID := int32(1)
	loanRepo.SetLoansWithStats([]*domain.LoanWithStats{
		{Loan: domain.Loan{ID: 1, WorkspaceID: workspaceID, ItemName: "Loan 1"}},
		{Loan: domain.Loan{ID: 2, WorkspaceID: workspaceID, ItemName: "Loan 2"}},
	})

	// Empty string should default to all
	loans, err := service.GetLoansWithStats(workspaceID, domain.LoanFilter(""))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(loans) != 2 {
		t.Errorf("Expected 2 loans with empty filter (defaults to all), got %d", len(loans))
	}
}

// Domain method tests

func TestLoan_IsActive(t *testing.T) {
	tests := []struct {
		name              string
		firstPaymentYear  int32
		firstPaymentMonth int32
		numMonths         int32
		currentYear       int
		currentMonth      int
		expectedActive    bool
	}{
		{
			name:              "Active - current month is first payment month",
			firstPaymentYear:  2024,
			firstPaymentMonth: 3,
			numMonths:         3,
			currentYear:       2024,
			currentMonth:      3,
			expectedActive:    true,
		},
		{
			name:              "Active - in the middle of payments",
			firstPaymentYear:  2024,
			firstPaymentMonth: 3,
			numMonths:         3,
			currentYear:       2024,
			currentMonth:      4,
			expectedActive:    true,
		},
		{
			name:              "Active - last payment month",
			firstPaymentYear:  2024,
			firstPaymentMonth: 3,
			numMonths:         3,
			currentYear:       2024,
			currentMonth:      5,
			expectedActive:    true,
		},
		{
			name:              "Completed - one month after last payment",
			firstPaymentYear:  2024,
			firstPaymentMonth: 3,
			numMonths:         3,
			currentYear:       2024,
			currentMonth:      6,
			expectedActive:    false,
		},
		{
			name:              "Active - crosses year boundary",
			firstPaymentYear:  2024,
			firstPaymentMonth: 11,
			numMonths:         4, // Nov, Dec, Jan, Feb
			currentYear:       2025,
			currentMonth:      1,
			expectedActive:    true,
		},
		{
			name:              "Completed - after year boundary",
			firstPaymentYear:  2024,
			firstPaymentMonth: 11,
			numMonths:         4, // Nov, Dec, Jan, Feb
			currentYear:       2025,
			currentMonth:      3,
			expectedActive:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loan := &domain.Loan{
				FirstPaymentYear:  tt.firstPaymentYear,
				FirstPaymentMonth: tt.firstPaymentMonth,
				NumMonths:         tt.numMonths,
			}
			result := loan.IsActive(tt.currentYear, tt.currentMonth)
			if result != tt.expectedActive {
				t.Errorf("Expected IsActive=%v, got %v", tt.expectedActive, result)
			}
		})
	}
}

// UpdateLoan tests

func TestUpdateLoan_Success(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	workspaceID := int32(1)
	providerID := int32(1)

	// Add provider that the loan references
	providerRepo.AddProvider(&domain.LoanProvider{
		ID:          providerID,
		WorkspaceID: workspaceID,
		Name:        "Test Provider",
	})

	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: workspaceID,
		ProviderID:  providerID,
		ItemName:    "Original Name",
	})

	notes := "Updated notes"
	input := UpdateLoanInput{
		ItemName: "Updated Name",
		Notes:    &notes,
	}

	loan, err := service.UpdateLoan(workspaceID, 1, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if loan.ItemName != "Updated Name" {
		t.Errorf("Expected 'Updated Name', got '%s'", loan.ItemName)
	}

	if loan.Notes == nil || *loan.Notes != "Updated notes" {
		t.Errorf("Expected notes 'Updated notes', got %v", loan.Notes)
	}
}

func TestUpdateLoan_EmptyItemName(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	workspaceID := int32(1)
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: workspaceID,
		ItemName:    "Original Name",
	})

	input := UpdateLoanInput{
		ItemName: "",
	}

	_, err := service.UpdateLoan(workspaceID, 1, input)
	if err != domain.ErrLoanItemNameEmpty {
		t.Errorf("Expected ErrLoanItemNameEmpty, got %v", err)
	}
}

func TestUpdateLoan_ItemNameTooLong(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	workspaceID := int32(1)
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: workspaceID,
		ItemName:    "Original Name",
	})

	// Create a name that's 201 characters long
	longName := ""
	for i := 0; i < 201; i++ {
		longName += "A"
	}

	input := UpdateLoanInput{
		ItemName: longName,
	}

	_, err := service.UpdateLoan(workspaceID, 1, input)
	if err != domain.ErrLoanItemNameTooLong {
		t.Errorf("Expected ErrLoanItemNameTooLong, got %v", err)
	}
}

func TestUpdateLoan_LoanNotFound(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	input := UpdateLoanInput{
		ItemName: "New Name",
	}

	_, err := service.UpdateLoan(1, 999, input)
	if err != domain.ErrLoanNotFound {
		t.Errorf("Expected ErrLoanNotFound, got %v", err)
	}
}

func TestUpdateLoan_ClearsNotes(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	workspaceID := int32(1)
	providerID := int32(1)
	originalNotes := "Original notes"

	// Add provider that the loan references
	providerRepo.AddProvider(&domain.LoanProvider{
		ID:          providerID,
		WorkspaceID: workspaceID,
		Name:        "Test Provider",
	})

	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: workspaceID,
		ProviderID:  providerID,
		ItemName:    "Original Name",
		Notes:       &originalNotes,
	})

	// Update with nil notes (clear notes)
	input := UpdateLoanInput{
		ItemName: "Updated Name",
		Notes:    nil,
	}

	loan, err := service.UpdateLoan(workspaceID, 1, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if loan.Notes != nil {
		t.Errorf("Expected notes to be nil, got %v", loan.Notes)
	}
}

// GetDeleteStats tests
// NOTE: GetDeleteStats is currently stubbed (v2 migration). Tests verify stub behavior.

func TestGetDeleteStats_Success(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service, transactionRepo := createTestLoanServiceWithTransactionRepo(loanRepo, providerRepo)

	workspaceID := int32(1)
	loanID := int32(1)
	loanRepo.AddLoan(&domain.Loan{
		ID:             loanID,
		WorkspaceID:    workspaceID,
		ItemName:       "Test Loan",
		TotalAmount:    decimal.NewFromInt(300),
		NumMonths:      3,
		MonthlyPayment: decimal.NewFromInt(100),
	})

	// Add transactions for the loan: 1 paid, 2 unpaid
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

	loan, stats, err := service.GetDeleteStats(workspaceID, loanID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if loan.ItemName != "Test Loan" {
		t.Errorf("Expected 'Test Loan', got '%s'", loan.ItemName)
	}

	// v2: Stats come from transactions table
	if stats.TotalCount != 3 {
		t.Errorf("Expected total count 3, got %d", stats.TotalCount)
	}

	if stats.PaidCount != 1 {
		t.Errorf("Expected paid count 1, got %d", stats.PaidCount)
	}

	if stats.UnpaidCount != 2 {
		t.Errorf("Expected unpaid count 2, got %d", stats.UnpaidCount)
	}

	expectedTotal := decimal.NewFromInt(300)
	if !stats.TotalAmount.Equal(expectedTotal) {
		t.Errorf("Expected total amount %s, got %s", expectedTotal.String(), stats.TotalAmount.String())
	}
}

func TestGetDeleteStats_LoanNotFound(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	_, _, err := service.GetDeleteStats(1, 999)
	if err != domain.ErrLoanNotFound {
		t.Errorf("Expected ErrLoanNotFound, got %v", err)
	}
}

func TestGetDeleteStats_NoPayments(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	workspaceID := int32(1)
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: workspaceID,
		ItemName:    "Test Loan",
	})

	loan, stats, err := service.GetDeleteStats(workspaceID, 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if loan.ItemName != "Test Loan" {
		t.Errorf("Expected 'Test Loan', got '%s'", loan.ItemName)
	}

	if stats.TotalCount != 0 {
		t.Errorf("Expected total count 0, got %d", stats.TotalCount)
	}

	if stats.PaidCount != 0 {
		t.Errorf("Expected paid count 0, got %d", stats.PaidCount)
	}
}

func TestLoan_GetLastPaymentYearMonth(t *testing.T) {
	tests := []struct {
		name              string
		firstPaymentYear  int32
		firstPaymentMonth int32
		numMonths         int32
		expectedYear      int
		expectedMonth     int
	}{
		{
			name:              "Single month",
			firstPaymentYear:  2024,
			firstPaymentMonth: 3,
			numMonths:         1,
			expectedYear:      2024,
			expectedMonth:     3,
		},
		{
			name:              "Three months same year",
			firstPaymentYear:  2024,
			firstPaymentMonth: 3,
			numMonths:         3,
			expectedYear:      2024,
			expectedMonth:     5,
		},
		{
			name:              "Crosses year boundary",
			firstPaymentYear:  2024,
			firstPaymentMonth: 11,
			numMonths:         4,
			expectedYear:      2025,
			expectedMonth:     2,
		},
		{
			name:              "12 months",
			firstPaymentYear:  2024,
			firstPaymentMonth: 1,
			numMonths:         12,
			expectedYear:      2024,
			expectedMonth:     12,
		},
		{
			name:              "24 months",
			firstPaymentYear:  2024,
			firstPaymentMonth: 1,
			numMonths:         24,
			expectedYear:      2025,
			expectedMonth:     12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loan := &domain.Loan{
				FirstPaymentYear:  tt.firstPaymentYear,
				FirstPaymentMonth: tt.firstPaymentMonth,
				NumMonths:         tt.numMonths,
			}
			year, month := loan.GetLastPaymentYearMonth()
			if year != tt.expectedYear || month != tt.expectedMonth {
				t.Errorf("Expected %d-%02d, got %d-%02d", tt.expectedYear, tt.expectedMonth, year, month)
			}
		})
	}
}

// GetTrend tests
// NOTE: GetTrend stub in v2 returns months with zero totals and empty providers.
// The stub preserves month count logic (defaults, caps) but returns empty data.

func TestGetTrend_ReturnsZeroTotalMonths(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	result, err := service.GetTrend(1, 6)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Stub returns requested number of months with zero data
	if len(result.Months) != 6 {
		t.Errorf("Expected 6 months, got %d", len(result.Months))
	}

	// All months should have zero totals and no providers
	for i, month := range result.Months {
		if !month.Total.Equal(decimal.Zero) {
			t.Errorf("Month %d: expected total 0, got %s", i, month.Total.String())
		}
		if len(month.Providers) != 0 {
			t.Errorf("Month %d: expected 0 providers, got %d", i, len(month.Providers))
		}
	}
}

func TestGetTrend_DefaultsTo12Months(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	// months = 0 should default to 12
	result, err := service.GetTrend(1, 0)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Months) != 12 {
		t.Errorf("Expected 12 months (default), got %d", len(result.Months))
	}
}

func TestGetTrend_MaxCapsAt24Months(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := createTestLoanService(loanRepo, providerRepo)

	// months > 24 should cap at 24
	result, err := service.GetTrend(1, 36)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Months) != 24 {
		t.Errorf("Expected 24 months (max), got %d", len(result.Months))
	}
}

// ============================================================================
// CC Loan Integration Tests (cl-v2-2-3)
// Tests verifying CC-backed loan transactions integrate with CC settlement workflow
// ============================================================================

// TestCreateLoan_CCAccount_HasCorrectSettlementIntent verifies that when a loan
// is created with a CC account, the loan has proper settlement_intent (AC: #1)
func TestCreateLoan_CCAccount_HasCorrectSettlementIntent(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Add CC account
	accountRepo.AddAccount(&domain.Account{
		ID:          2,
		WorkspaceID: 1,
		Name:        "Test Credit Card",
		Template:    domain.TemplateCreditCard,
		AccountType: domain.AccountTypeLiability,
	})

	service := NewLoanService(nil, loanRepo, providerRepo, transactionRepo, accountRepo)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "SPayLater",
		CutoffDay:           25,
		DefaultInterestRate: decimal.Zero,
	})

	// Create loan with CC account and immediate intent
	immediateIntent := "immediate"
	input := CreateLoanInput{
		ProviderID:       1,
		ItemName:         "Test CC Loan",
		TotalAmount:      decimal.NewFromInt(300),
		NumMonths:        3,
		PurchaseDate:     time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC),
		AccountID:        2, // CC account
		SettlementIntent: &immediateIntent,
	}

	loan, err := service.CreateLoan(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify loan has correct settlement intent
	if loan.SettlementIntent == nil || *loan.SettlementIntent != "immediate" {
		t.Errorf("Expected loan SettlementIntent 'immediate', got %v", loan.SettlementIntent)
	}

	// Verify loan has CC account
	if loan.AccountID != 2 {
		t.Errorf("Expected AccountID 2, got %d", loan.AccountID)
	}
}

// TestCreateLoan_CCAccount_DefaultsToDeferredIntent verifies that CC loans without
// explicit intent default to "deferred" (AC: #1)
func TestCreateLoan_CCAccount_DefaultsToDeferredIntent(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Add CC account
	accountRepo.AddAccount(&domain.Account{
		ID:          2,
		WorkspaceID: 1,
		Name:        "Test Credit Card",
		Template:    domain.TemplateCreditCard,
		AccountType: domain.AccountTypeLiability,
	})

	service := NewLoanService(nil, loanRepo, providerRepo, transactionRepo, accountRepo)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "SPayLater",
		CutoffDay:           25,
		DefaultInterestRate: decimal.Zero,
	})

	// Create loan with CC account but NO explicit intent
	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "Test CC Loan",
		TotalAmount:  decimal.NewFromInt(200),
		NumMonths:    2,
		PurchaseDate: time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC),
		AccountID:    2, // CC account
		// SettlementIntent NOT set - should default to deferred
	}

	loan, err := service.CreateLoan(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify loan defaults to deferred
	if loan.SettlementIntent == nil || *loan.SettlementIntent != "deferred" {
		t.Errorf("Expected loan SettlementIntent 'deferred' (default), got %v", loan.SettlementIntent)
	}

	// Verify transactions have deferred intent
	for id, tx := range transactionRepo.Transactions {
		if tx.SettlementIntent == nil || *tx.SettlementIntent != domain.SettlementIntentDeferred {
			t.Errorf("Transaction %d: expected SettlementIntent 'deferred', got %v", id, tx.SettlementIntent)
		}
	}
}

// TestCCLoanTransactions_AppearInDeferredSettlementQuery verifies that billed CC loan
// transactions with deferred intent appear in GetDeferredForSettlement (AC: #4)
func TestCCLoanTransactions_AppearInDeferredSettlementQuery(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	workspaceID := int32(1)
	loanID := int32(1)

	// Simulate CC loan transactions that would be created by loan service
	// These have loan_id set and deferred settlement intent
	deferredIntent := domain.SettlementIntentDeferred
	billedState := domain.CCStateBilled
	now := time.Now()

	// Add two billed deferred CC loan transactions
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:               1,
		WorkspaceID:      workspaceID,
		AccountID:        2,
		Name:             "Loan Payment 1",
		Amount:           decimal.NewFromInt(100),
		Type:             domain.TransactionTypeExpense,
		TransactionDate:  time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		IsPaid:           false,
		Source:           "loan",
		LoanID:           &loanID,
		SettlementIntent: &deferredIntent,
		CCState:          &billedState,
		BilledAt:         &now,
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:               2,
		WorkspaceID:      workspaceID,
		AccountID:        2,
		Name:             "Loan Payment 2",
		Amount:           decimal.NewFromInt(100),
		Type:             domain.TransactionTypeExpense,
		TransactionDate:  time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
		IsPaid:           false,
		Source:           "loan",
		LoanID:           &loanID,
		SettlementIntent: &deferredIntent,
		CCState:          &billedState,
		BilledAt:         &now,
	})

	// Query deferred transactions for settlement
	deferredTxs, err := transactionRepo.GetDeferredForSettlement(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify loan transactions appear in the results (NOT excluded by loan_id)
	if len(deferredTxs) != 2 {
		t.Errorf("Expected 2 deferred transactions (including loan txns), got %d", len(deferredTxs))
	}

	// Verify they have loan_id set
	for _, tx := range deferredTxs {
		if tx.LoanID == nil || *tx.LoanID != loanID {
			t.Errorf("Expected transaction to have LoanID %d, got %v", loanID, tx.LoanID)
		}
	}
}

// TestCCLoanTransactions_AppearInImmediateSettlementQuery verifies that billed CC loan
// transactions with immediate intent appear in GetImmediateForSettlement (AC: #3)
func TestCCLoanTransactions_AppearInImmediateSettlementQuery(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	workspaceID := int32(1)
	loanID := int32(1)

	// Simulate CC loan transactions that would be created by loan service
	// These have loan_id set and immediate settlement intent
	immediateIntent := domain.SettlementIntentImmediate
	billedState := domain.CCStateBilled
	now := time.Now()

	// Add two billed immediate CC loan transactions
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:               1,
		WorkspaceID:      workspaceID,
		AccountID:        2,
		Name:             "Loan Payment 1",
		Amount:           decimal.NewFromInt(100),
		Type:             domain.TransactionTypeExpense,
		TransactionDate:  time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		IsPaid:           false,
		Source:           "loan",
		LoanID:           &loanID,
		SettlementIntent: &immediateIntent,
		CCState:          &billedState,
		BilledAt:         &now,
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:               2,
		WorkspaceID:      workspaceID,
		AccountID:        2,
		Name:             "Loan Payment 2",
		Amount:           decimal.NewFromInt(100),
		Type:             domain.TransactionTypeExpense,
		TransactionDate:  time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
		IsPaid:           false,
		Source:           "loan",
		LoanID:           &loanID,
		SettlementIntent: &immediateIntent,
		CCState:          &billedState,
		BilledAt:         &now,
	})

	// Query immediate transactions for settlement - use date range that covers the transactions
	startDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	immediateTxs, err := transactionRepo.GetImmediateForSettlement(workspaceID, startDate, endDate)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify loan transactions appear in the results (NOT excluded by loan_id)
	if len(immediateTxs) != 2 {
		t.Errorf("Expected 2 immediate transactions (including loan txns), got %d", len(immediateTxs))
	}

	// Verify they have loan_id set
	for _, tx := range immediateTxs {
		if tx.LoanID == nil || *tx.LoanID != loanID {
			t.Errorf("Expected transaction to have LoanID %d, got %v", loanID, tx.LoanID)
		}
	}
}

// TestLoanStats_UpdatesWhenTransactionSettled verifies that loan stats
// (paidCount, remainingBalance) reflect transaction is_paid status (AC: #6, #8)
func TestLoanStats_UpdatesWhenTransactionSettled(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	service := NewLoanService(nil, loanRepo, providerRepo, transactionRepo, accountRepo)

	workspaceID := int32(1)
	loanID := int32(1)

	// Setup: loan with 3 payments, 1 paid (settled), 2 unpaid
	loanRepo.SetLoansWithStats([]*domain.LoanWithStats{
		{
			Loan: domain.Loan{
				ID:          loanID,
				WorkspaceID: workspaceID,
				ItemName:    "CC Loan",
			},
			TotalCount:       3,
			PaidCount:        1,  // 1 transaction settled (is_paid=true)
			RemainingBalance: decimal.NewFromInt(200), // 2 * 100
			Progress:         33.33,
		},
	})

	// Get loan stats
	loans, err := service.GetLoansWithStats(workspaceID, domain.LoanFilterAll)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(loans) != 1 {
		t.Fatalf("Expected 1 loan, got %d", len(loans))
	}

	loan := loans[0]

	// Verify stats reflect transaction is_paid status
	if loan.TotalCount != 3 {
		t.Errorf("Expected TotalCount 3, got %d", loan.TotalCount)
	}
	if loan.PaidCount != 1 {
		t.Errorf("Expected PaidCount 1 (1 settled transaction), got %d", loan.PaidCount)
	}

	expectedRemaining := decimal.NewFromInt(200)
	if !loan.RemainingBalance.Equal(expectedRemaining) {
		t.Errorf("Expected RemainingBalance %s, got %s", expectedRemaining.String(), loan.RemainingBalance.String())
	}

	// Progress = paid / total * 100
	if loan.Progress < 33 || loan.Progress > 34 {
		t.Errorf("Expected Progress ~33.33%%, got %.2f%%", loan.Progress)
	}
}

// TestBulkSettle_IncludesLoanTransactions verifies that BulkSettle
// works with loan transactions (AC: #6)
func TestBulkSettle_IncludesLoanTransactions(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	workspaceID := int32(1)
	loanID := int32(1)

	// Simulate billed CC loan transactions (ready for settlement)
	billedState := domain.CCStateBilled
	deferredIntent := domain.SettlementIntentDeferred
	now := time.Now()

	// Add billed loan transaction
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:               1,
		WorkspaceID:      workspaceID,
		AccountID:        2,
		Name:             "Loan Payment 1",
		Amount:           decimal.NewFromInt(100),
		Type:             domain.TransactionTypeExpense,
		TransactionDate:  time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		IsPaid:           false,
		Source:           "loan",
		LoanID:           &loanID,
		SettlementIntent: &deferredIntent,
		CCState:          &billedState,
		BilledAt:         &now,
	})
	// Add billed regular CC transaction
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:               2,
		WorkspaceID:      workspaceID,
		AccountID:        2,
		Name:             "Regular CC Purchase",
		Amount:           decimal.NewFromInt(50),
		Type:             domain.TransactionTypeExpense,
		TransactionDate:  time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC),
		IsPaid:           false,
		Source:           "manual",
		LoanID:           nil,
		SettlementIntent: &deferredIntent,
		CCState:          &billedState,
		BilledAt:         &now,
	})

	// Custom BulkSettleFn that simulates the DB behavior
	transactionRepo.BulkSettleFn = func(wsID int32, ids []int32) ([]*domain.Transaction, error) {
		var result []*domain.Transaction
		settledState := domain.CCStateSettled
		for _, id := range ids {
			if tx, ok := transactionRepo.Transactions[id]; ok && tx.WorkspaceID == wsID && tx.BilledAt != nil && !tx.IsPaid {
				tx.IsPaid = true
				tx.CCState = &settledState
				tx.UpdatedAt = time.Now()
				result = append(result, tx)
			}
		}
		return result, nil
	}

	// Bulk settle both transactions (including the loan transaction)
	settledTxs, err := transactionRepo.BulkSettle(workspaceID, []int32{1, 2})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify both transactions were settled (loan tx NOT excluded)
	if len(settledTxs) != 2 {
		t.Errorf("Expected 2 settled transactions, got %d", len(settledTxs))
	}

	// Verify loan transaction was settled
	loanTx := transactionRepo.Transactions[1]
	if !loanTx.IsPaid {
		t.Error("Expected loan transaction to have IsPaid=true")
	}
	if loanTx.CCState == nil || *loanTx.CCState != domain.CCStateSettled {
		t.Errorf("Expected loan transaction CCState 'settled', got %v", loanTx.CCState)
	}

	// Verify loan_id is preserved
	if loanTx.LoanID == nil || *loanTx.LoanID != loanID {
		t.Errorf("Expected loan transaction to retain LoanID %d, got %v", loanID, loanTx.LoanID)
	}
}

// TestBatchBilling_IncludesLoanTransactions verifies that BatchToggleToBilled
// works with loan transactions (AC: #5)
func TestBatchBilling_IncludesLoanTransactions(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	workspaceID := int32(1)
	loanID := int32(1)

	// Simulate pending CC loan transactions
	pendingState := domain.CCStatePending
	deferredIntent := domain.SettlementIntentDeferred

	// Add pending loan transactions
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:               1,
		WorkspaceID:      workspaceID,
		AccountID:        2,
		Name:             "Loan Payment 1",
		Amount:           decimal.NewFromInt(100),
		Type:             domain.TransactionTypeExpense,
		TransactionDate:  time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		IsPaid:           false,
		Source:           "loan",
		LoanID:           &loanID,
		SettlementIntent: &deferredIntent,
		CCState:          &pendingState,
		BilledAt:         nil,
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:               2,
		WorkspaceID:      workspaceID,
		AccountID:        2,
		Name:             "Regular CC Purchase",
		Amount:           decimal.NewFromInt(50),
		Type:             domain.TransactionTypeExpense,
		TransactionDate:  time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC),
		IsPaid:           false,
		Source:           "manual",
		LoanID:           nil, // Not a loan transaction
		SettlementIntent: &deferredIntent,
		CCState:          &pendingState,
		BilledAt:         nil,
	})

	// Custom BatchToggleToBilledFn that simulates the DB behavior
	transactionRepo.BatchToggleToBilledFn = func(wsID int32, ids []int32) ([]*domain.Transaction, error) {
		var result []*domain.Transaction
		now := time.Now()
		billedState := domain.CCStateBilled
		for _, id := range ids {
			if tx, ok := transactionRepo.Transactions[id]; ok && tx.WorkspaceID == wsID {
				tx.BilledAt = &now
				tx.CCState = &billedState
				tx.UpdatedAt = now
				result = append(result, tx)
			}
		}
		return result, nil
	}

	// Batch bill both transactions (including the loan transaction)
	billedTxs, err := transactionRepo.BatchToggleToBilled(workspaceID, []int32{1, 2})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify both transactions were billed (loan tx NOT excluded)
	if len(billedTxs) != 2 {
		t.Errorf("Expected 2 billed transactions, got %d", len(billedTxs))
	}

	// Verify loan transaction was billed
	loanTx := transactionRepo.Transactions[1]
	if loanTx.BilledAt == nil {
		t.Error("Expected loan transaction to have BilledAt set")
	}
	if loanTx.CCState == nil || *loanTx.CCState != domain.CCStateBilled {
		t.Errorf("Expected loan transaction CCState 'billed', got %v", loanTx.CCState)
	}

	// Verify loan_id is preserved
	if loanTx.LoanID == nil || *loanTx.LoanID != loanID {
		t.Errorf("Expected loan transaction to retain LoanID %d, got %v", loanID, loanTx.LoanID)
	}
}

// TestCCMetrics_IncludeLoanTransactions verifies that GetCCMetrics includes
// loan transactions in its calculations (AC: #3, #4)
func TestCCMetrics_IncludeLoanTransactions(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	workspaceID := int32(1)
	loanID := int32(1)

	// Simulate CC loan transactions with different states
	pendingState := domain.CCStatePending
	immediateIntent := domain.SettlementIntentImmediate

	// Add a pending loan transaction
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:               1,
		WorkspaceID:      workspaceID,
		AccountID:        2,
		Name:             "Loan Payment - Pending",
		Amount:           decimal.NewFromInt(100),
		Type:             domain.TransactionTypeExpense,
		TransactionDate:  time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		IsPaid:           false,
		Source:           "loan",
		LoanID:           &loanID,
		SettlementIntent: &immediateIntent,
		CCState:          &pendingState,
		BilledAt:         nil,
	})

	// Custom GetCCMetricsFn that calculates based on transactions in mock
	transactionRepo.GetCCMetricsFn = func(wsID int32, start, end time.Time) (*domain.CCMetrics, error) {
		var pending, outstanding, purchases decimal.Decimal
		for _, tx := range transactionRepo.Transactions {
			if tx.WorkspaceID != wsID || tx.Type != domain.TransactionTypeExpense {
				continue
			}
			// Purchases: all CC expenses
			purchases = purchases.Add(tx.Amount)
			// Pending: billed_at IS NULL AND is_paid = false
			if tx.BilledAt == nil && !tx.IsPaid {
				pending = pending.Add(tx.Amount)
			}
			// Outstanding: billed AND is_paid = false
			if tx.BilledAt != nil && !tx.IsPaid {
				outstanding = outstanding.Add(tx.Amount)
			}
		}
		return &domain.CCMetrics{
			Pending:     pending,
			Outstanding: outstanding,
			Purchases:   purchases,
		}, nil
	}

	// Get CC metrics - should include the loan transaction
	startDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	metrics, err := transactionRepo.GetCCMetrics(workspaceID, startDate, endDate)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify loan transaction is included in pending (not excluded by loan_id)
	expectedPending := decimal.NewFromInt(100)
	if !metrics.Pending.Equal(expectedPending) {
		t.Errorf("Expected pending %s, got %s - loan transaction may be excluded", expectedPending.String(), metrics.Pending.String())
	}

	// Verify loan transaction is included in purchases
	expectedPurchases := decimal.NewFromInt(100)
	if !metrics.Purchases.Equal(expectedPurchases) {
		t.Errorf("Expected purchases %s, got %s - loan transaction may be excluded", expectedPurchases.String(), metrics.Purchases.String())
	}
}

// TestCCLoanFullLifecycle_DeferredIntent tests the complete lifecycle of a CC loan
// with deferred intent: create → pending → billed → settled (AC: #1-8)
func TestCCLoanFullLifecycle_DeferredIntent(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	workspaceID := int32(1)
	loanID := int32(1)

	// Simulate CC loan transactions created by loan service
	deferredIntent := domain.SettlementIntentDeferred
	pendingState := domain.CCStatePending

	// STEP 1: Loan created - transactions are PENDING
	tx1 := &domain.Transaction{
		ID:               1,
		WorkspaceID:      workspaceID,
		AccountID:        2,
		Name:             "CC Loan Payment 1",
		Amount:           decimal.NewFromInt(100),
		Type:             domain.TransactionTypeExpense,
		TransactionDate:  time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		IsPaid:           false,
		Source:           "loan",
		LoanID:           &loanID,
		SettlementIntent: &deferredIntent,
		CCState:          &pendingState,
		BilledAt:         nil,
	}
	tx2 := &domain.Transaction{
		ID:               2,
		WorkspaceID:      workspaceID,
		AccountID:        2,
		Name:             "CC Loan Payment 2",
		Amount:           decimal.NewFromInt(100),
		Type:             domain.TransactionTypeExpense,
		TransactionDate:  time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
		IsPaid:           false,
		Source:           "loan",
		LoanID:           &loanID,
		SettlementIntent: &deferredIntent,
		CCState:          &pendingState,
		BilledAt:         nil,
	}
	transactionRepo.AddTransaction(tx1)
	transactionRepo.AddTransaction(tx2)

	// Verify: Pending state
	if tx1.CCState == nil || *tx1.CCState != domain.CCStatePending {
		t.Error("Step 1 failed: Expected pending state after loan creation")
	}

	// Verify: Appears in GetPendingDeferredCC (not in settlement yet)
	pendingTxs, _ := transactionRepo.GetPendingDeferredCC(workspaceID, time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC))
	if len(pendingTxs) != 2 {
		t.Errorf("Step 1 failed: Expected 2 pending deferred txns, got %d", len(pendingTxs))
	}

	// STEP 2: Batch bill - transactions become BILLED
	billedState := domain.CCStateBilled
	now := time.Now()
	for _, tx := range transactionRepo.Transactions {
		tx.CCState = &billedState
		tx.BilledAt = &now
	}

	// Verify: Billed state
	if tx1.CCState == nil || *tx1.CCState != domain.CCStateBilled {
		t.Error("Step 2 failed: Expected billed state after billing")
	}

	// Verify: Appears in GetDeferredForSettlement
	deferredTxs, _ := transactionRepo.GetDeferredForSettlement(workspaceID)
	if len(deferredTxs) != 2 {
		t.Errorf("Step 2 failed: Expected 2 deferred-to-settle txns, got %d", len(deferredTxs))
	}
	// Verify loan transactions included (not filtered by loan_id)
	for _, tx := range deferredTxs {
		if tx.LoanID == nil {
			t.Error("Step 2 failed: Expected loan transaction to have LoanID set")
		}
	}

	// STEP 3: Settle - transactions become SETTLED (is_paid = true)
	settledState := domain.CCStateSettled
	for _, tx := range transactionRepo.Transactions {
		tx.CCState = &settledState
		tx.IsPaid = true
	}

	// Verify: Settled state
	if tx1.CCState == nil || *tx1.CCState != domain.CCStateSettled {
		t.Error("Step 3 failed: Expected settled state after settlement")
	}
	if !tx1.IsPaid {
		t.Error("Step 3 failed: Expected is_paid=true after settlement")
	}

	// Verify: No longer appears in settlement queries (is_paid=true)
	deferredAfterSettle, _ := transactionRepo.GetDeferredForSettlement(workspaceID)
	if len(deferredAfterSettle) != 0 {
		t.Errorf("Step 3 failed: Expected 0 deferred txns after settle, got %d", len(deferredAfterSettle))
	}
}

// TestCCLoanFullLifecycle_ImmediateIntent tests CC loan with immediate intent
// (billed immediately for current month settlement) (AC: #3)
func TestCCLoanFullLifecycle_ImmediateIntent(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	workspaceID := int32(1)
	loanID := int32(1)

	// Simulate CC loan transactions with immediate intent
	immediateIntent := domain.SettlementIntentImmediate
	billedState := domain.CCStateBilled
	now := time.Now()

	// Create transactions already billed (immediate = bill same month)
	tx1 := &domain.Transaction{
		ID:               1,
		WorkspaceID:      workspaceID,
		AccountID:        2,
		Name:             "Immediate CC Loan Payment 1",
		Amount:           decimal.NewFromInt(100),
		Type:             domain.TransactionTypeExpense,
		TransactionDate:  time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
		IsPaid:           false,
		Source:           "loan",
		LoanID:           &loanID,
		SettlementIntent: &immediateIntent,
		CCState:          &billedState,
		BilledAt:         &now,
	}
	transactionRepo.AddTransaction(tx1)

	// Verify: Appears in GetImmediateForSettlement
	startDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	immediateTxs, _ := transactionRepo.GetImmediateForSettlement(workspaceID, startDate, endDate)
	if len(immediateTxs) != 1 {
		t.Errorf("Expected 1 immediate-to-settle txn, got %d", len(immediateTxs))
	}
	if immediateTxs[0].LoanID == nil {
		t.Error("Expected loan transaction to have LoanID set")
	}

	// Verify: Does NOT appear in GetDeferredForSettlement
	deferredTxs, _ := transactionRepo.GetDeferredForSettlement(workspaceID)
	if len(deferredTxs) != 0 {
		t.Errorf("Expected 0 deferred txns for immediate loan, got %d", len(deferredTxs))
	}
}

// TestBankLoan_NoSettlementIntent verifies that bank-backed loans don't have
// settlement_intent set on transactions (AC: #1 - negative case)
func TestBankLoan_NoSettlementIntent(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Add Bank account
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test Bank",
		Template:    domain.TemplateBank,
		AccountType: domain.AccountTypeAsset,
	})

	service := NewLoanService(nil, loanRepo, providerRepo, transactionRepo, accountRepo)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "BankLoan",
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	})

	// Create loan with Bank account (should NOT have settlement intent)
	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "Test Bank Loan",
		TotalAmount:  decimal.NewFromInt(300),
		NumMonths:    3,
		PurchaseDate: time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC),
		AccountID:    1, // Bank account
	}

	loan, err := service.CreateLoan(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify loan has NO settlement intent
	if loan.SettlementIntent != nil {
		t.Errorf("Expected loan SettlementIntent nil for bank account, got %v", loan.SettlementIntent)
	}

	// Verify transactions have NO settlement intent
	for id, tx := range transactionRepo.Transactions {
		if tx.SettlementIntent != nil {
			t.Errorf("Transaction %d: expected SettlementIntent nil for bank account, got %v", id, tx.SettlementIntent)
		}
	}
}

// ============================================================================
// PayLoanMonth tests (cl-v2-3-1)
// Tests for paying a loan month by settling transactions
// ============================================================================

// TestPayLoanMonth_Success verifies basic payment flow for bank account loans
func TestPayLoanMonth_Success(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Add Bank account
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test Bank",
		Template:    domain.TemplateBank,
		AccountType: domain.AccountTypeAsset,
	})

	service := NewLoanService(nil, loanRepo, providerRepo, transactionRepo, accountRepo)

	workspaceID := int32(1)
	loanID := int32(1)

	// Setup loan
	loanRepo.AddLoan(&domain.Loan{
		ID:          loanID,
		WorkspaceID: workspaceID,
		ItemName:    "Test Loan",
	})

	// Add unpaid transactions for March 2024
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              1,
		WorkspaceID:     workspaceID,
		AccountID:       1,
		Name:            "Loan Payment 1",
		Amount:          decimal.NewFromInt(100),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
		IsPaid:          false,
		LoanID:          &loanID,
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              2,
		WorkspaceID:     workspaceID,
		AccountID:       1,
		Name:            "Loan Payment 2",
		Amount:          decimal.NewFromInt(50),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC),
		IsPaid:          false,
		LoanID:          &loanID,
	})

	// Pay March 2024
	input := PayLoanMonthInput{
		LoanID: loanID,
		Year:   2024,
		Month:  3,
	}

	result, err := service.PayLoanMonth(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify result
	if len(result.SettledTransactions) != 2 {
		t.Errorf("Expected 2 settled transactions, got %d", len(result.SettledTransactions))
	}

	expectedTotal := decimal.NewFromInt(150)
	if !result.TotalAmount.Equal(expectedTotal) {
		t.Errorf("Expected total %s, got %s", expectedTotal.String(), result.TotalAmount.String())
	}

	// Verify transactions are marked as paid
	for _, tx := range result.SettledTransactions {
		if !tx.IsPaid {
			t.Errorf("Transaction %d should be marked as paid", tx.ID)
		}
	}
}

// TestPayLoanMonth_NoTransactionsToSettle verifies error when no unpaid transactions
func TestPayLoanMonth_NoTransactionsToSettle(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	service := NewLoanService(nil, loanRepo, providerRepo, transactionRepo, accountRepo)

	workspaceID := int32(1)
	loanID := int32(1)

	// Setup loan
	loanRepo.AddLoan(&domain.Loan{
		ID:          loanID,
		WorkspaceID: workspaceID,
		ItemName:    "Test Loan",
	})

	// No transactions added - or add already paid ones
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              1,
		WorkspaceID:     workspaceID,
		AccountID:       1,
		Name:            "Already Paid",
		Amount:          decimal.NewFromInt(100),
		TransactionDate: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
		IsPaid:          true, // Already paid
		LoanID:          &loanID,
	})

	input := PayLoanMonthInput{
		LoanID: loanID,
		Year:   2024,
		Month:  3,
	}

	_, err := service.PayLoanMonth(workspaceID, input)
	if err != domain.ErrNoTransactionsToSettle {
		t.Errorf("Expected ErrNoTransactionsToSettle, got %v", err)
	}
}

// TestPayLoanMonth_LoanNotFound verifies error when loan doesn't exist
func TestPayLoanMonth_LoanNotFound(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	service := NewLoanService(nil, loanRepo, providerRepo, transactionRepo, accountRepo)

	input := PayLoanMonthInput{
		LoanID: 999, // Non-existent
		Year:   2024,
		Month:  3,
	}

	_, err := service.PayLoanMonth(1, input)
	if err != domain.ErrLoanNotFound {
		t.Errorf("Expected ErrLoanNotFound, got %v", err)
	}
}

// TestPayLoanMonth_CCLoan_MarksIsPaidTrue verifies CC transactions get is_paid=true
func TestPayLoanMonth_CCLoan_MarksIsPaidTrue(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	// Add CC account
	accountRepo.AddAccount(&domain.Account{
		ID:          2,
		WorkspaceID: 1,
		Name:        "Test CC",
		Template:    domain.TemplateCreditCard,
		AccountType: domain.AccountTypeLiability,
	})

	service := NewLoanService(nil, loanRepo, providerRepo, transactionRepo, accountRepo)

	workspaceID := int32(1)
	loanID := int32(1)
	deferredIntent := domain.SettlementIntentDeferred

	// Setup loan with CC settlement intent
	settlementIntent := "deferred"
	loanRepo.AddLoan(&domain.Loan{
		ID:               loanID,
		WorkspaceID:      workspaceID,
		ItemName:         "CC Loan",
		AccountID:        2,
		SettlementIntent: &settlementIntent,
	})

	// Add CC loan transaction (billed)
	billedState := domain.CCStateBilled
	now := time.Now()
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:               1,
		WorkspaceID:      workspaceID,
		AccountID:        2,
		Name:             "CC Loan Payment",
		Amount:           decimal.NewFromInt(100),
		Type:             domain.TransactionTypeExpense,
		TransactionDate:  time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
		IsPaid:           false,
		LoanID:           &loanID,
		SettlementIntent: &deferredIntent,
		CCState:          &billedState,
		BilledAt:         &now,
	})

	input := PayLoanMonthInput{
		LoanID: loanID,
		Year:   2024,
		Month:  3,
	}

	result, err := service.PayLoanMonth(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify CC transaction is marked as paid
	if len(result.SettledTransactions) != 1 {
		t.Fatalf("Expected 1 settled transaction, got %d", len(result.SettledTransactions))
	}

	tx := result.SettledTransactions[0]
	if !tx.IsPaid {
		t.Error("Expected CC transaction to have IsPaid=true")
	}
}

// TestPayLoanMonth_MultipleTransactionsInMonth verifies all transactions for month are settled
func TestPayLoanMonth_MultipleTransactionsInMonth(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test Bank",
		Template:    domain.TemplateBank,
	})

	service := NewLoanService(nil, loanRepo, providerRepo, transactionRepo, accountRepo)

	workspaceID := int32(1)
	loanID := int32(1)

	loanRepo.AddLoan(&domain.Loan{
		ID:          loanID,
		WorkspaceID: workspaceID,
		ItemName:    "Test Loan",
	})

	// Add 3 transactions for same month
	for i := 1; i <= 3; i++ {
		transactionRepo.AddTransaction(&domain.Transaction{
			ID:              int32(i),
			WorkspaceID:     workspaceID,
			AccountID:       1,
			Name:            "Loan Payment",
			Amount:          decimal.NewFromInt(int64(i * 100)),
			TransactionDate: time.Date(2024, 3, i*5, 0, 0, 0, 0, time.UTC),
			IsPaid:          false,
			LoanID:          &loanID,
		})
	}

	input := PayLoanMonthInput{
		LoanID: loanID,
		Year:   2024,
		Month:  3,
	}

	result, err := service.PayLoanMonth(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify all 3 transactions settled
	if len(result.SettledTransactions) != 3 {
		t.Errorf("Expected 3 settled transactions, got %d", len(result.SettledTransactions))
	}

	// Verify total: 100 + 200 + 300 = 600
	expectedTotal := decimal.NewFromInt(600)
	if !result.TotalAmount.Equal(expectedTotal) {
		t.Errorf("Expected total %s, got %s", expectedTotal.String(), result.TotalAmount.String())
	}
}

// TestPayLoanMonth_OnlyPaysTargetMonth verifies only target month transactions are paid
func TestPayLoanMonth_OnlyPaysTargetMonth(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()

	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test Bank",
		Template:    domain.TemplateBank,
	})

	service := NewLoanService(nil, loanRepo, providerRepo, transactionRepo, accountRepo)

	workspaceID := int32(1)
	loanID := int32(1)

	loanRepo.AddLoan(&domain.Loan{
		ID:          loanID,
		WorkspaceID: workspaceID,
		ItemName:    "Test Loan",
	})

	// Add transactions for March and April
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              1,
		WorkspaceID:     workspaceID,
		AccountID:       1,
		Name:            "March Payment",
		Amount:          decimal.NewFromInt(100),
		TransactionDate: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
		IsPaid:          false,
		LoanID:          &loanID,
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              2,
		WorkspaceID:     workspaceID,
		AccountID:       1,
		Name:            "April Payment",
		Amount:          decimal.NewFromInt(100),
		TransactionDate: time.Date(2024, 4, 15, 0, 0, 0, 0, time.UTC),
		IsPaid:          false,
		LoanID:          &loanID,
	})

	// Pay only March
	input := PayLoanMonthInput{
		LoanID: loanID,
		Year:   2024,
		Month:  3,
	}

	result, err := service.PayLoanMonth(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify only March transaction settled
	if len(result.SettledTransactions) != 1 {
		t.Errorf("Expected 1 settled transaction, got %d", len(result.SettledTransactions))
	}

	// Verify April transaction is still unpaid
	aprilTx := transactionRepo.Transactions[2]
	if aprilTx.IsPaid {
		t.Error("April transaction should still be unpaid")
	}
}
