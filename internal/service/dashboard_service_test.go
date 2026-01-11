package service

import (
	"testing"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/shopspring/decimal"
)

func TestDashboardService_GetSummary(t *testing.T) {
	// Use fixed dates for consistent testing
	testYear := 2025
	testMonth := 1
	startDate := time.Date(testYear, time.Month(testMonth), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, -1)
	txDate := time.Date(testYear, time.Month(testMonth), 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name                 string
		workspaceID          int32
		setupAccounts        func(*testutil.MockAccountRepository)
		setupTransactions    func(*testutil.MockTransactionRepository)
		setupMonth           func(*testutil.MockMonthRepository)
		wantTotalBalance     string
		wantInHandBalance    string
		wantDisposableIncome string
		wantErr              bool
	}{
		{
			name:        "calculates total balance correctly with assets and liabilities",
			workspaceID: 1,
			setupAccounts: func(m *testutil.MockAccountRepository) {
				m.AddAccount(&domain.Account{
					ID:             1,
					WorkspaceID:    1,
					Name:           "Bank",
					AccountType:    domain.AccountTypeAsset,
					Template:       domain.TemplateBank,
					InitialBalance: decimal.NewFromInt(10000),
				})
				m.AddAccount(&domain.Account{
					ID:             2,
					WorkspaceID:    1,
					Name:           "Credit Card",
					AccountType:    domain.AccountTypeLiability,
					Template:       domain.TemplateCreditCard,
					InitialBalance: decimal.Zero,
				})
			},
			setupTransactions: func(m *testutil.MockTransactionRepository) {
				// Bank: Income +5000
				m.AddTransaction(&domain.Transaction{
					ID:              1,
					WorkspaceID:     1,
					AccountID:       1,
					Name:            "Salary",
					Amount:          decimal.NewFromInt(5000),
					Type:            domain.TransactionTypeIncome,
					TransactionDate: txDate,
					IsPaid:          true,
				})
				// Bank: Paid expense -1000
				m.AddTransaction(&domain.Transaction{
					ID:              2,
					WorkspaceID:     1,
					AccountID:       1,
					Name:            "Groceries",
					Amount:          decimal.NewFromInt(1000),
					Type:            domain.TransactionTypeExpense,
					TransactionDate: txDate,
					IsPaid:          true,
				})
				// Credit Card: Expense -500 (unpaid CC debt)
				m.AddTransaction(&domain.Transaction{
					ID:              3,
					WorkspaceID:     1,
					AccountID:       2,
					Name:            "Online Shopping",
					Amount:          decimal.NewFromInt(500),
					Type:            domain.TransactionTypeExpense,
					TransactionDate: txDate,
					IsPaid:          false,
				})
			},
			setupMonth: func(m *testutil.MockMonthRepository) {
				m.AddMonth(&domain.Month{
					ID:              1,
					WorkspaceID:     1,
					Year:            testYear,
					Month:           testMonth,
					StartDate:       startDate,
					EndDate:         endDate,
					StartingBalance: decimal.NewFromInt(10000),
					CreatedAt:       startDate,
					UpdatedAt:       startDate,
				})
			},
			// Bank: 10000 + 5000 - 1000 = 14000
			// CC: 0 - 500 = -500 (debt)
			// Total = 14000 + (-500) = 13500
			wantTotalBalance: "13500.00",
			// In-hand = starting + income - paid expenses = 10000 + 5000 - 1000 = 14000
			wantInHandBalance: "14000.00",
			// Disposable = In-hand - unpaid expenses = 14000 - 500 = 13500
			wantDisposableIncome: "13500.00",
			wantErr:              false,
		},
		{
			name:        "calculates in-hand balance excluding unpaid expenses",
			workspaceID: 1,
			setupAccounts: func(m *testutil.MockAccountRepository) {
				m.AddAccount(&domain.Account{
					ID:             1,
					WorkspaceID:    1,
					Name:           "Bank",
					AccountType:    domain.AccountTypeAsset,
					Template:       domain.TemplateBank,
					InitialBalance: decimal.NewFromInt(5000),
				})
			},
			setupTransactions: func(m *testutil.MockTransactionRepository) {
				// Paid expense
				m.AddTransaction(&domain.Transaction{
					ID:              1,
					WorkspaceID:     1,
					AccountID:       1,
					Name:            "Groceries",
					Amount:          decimal.NewFromInt(500),
					Type:            domain.TransactionTypeExpense,
					TransactionDate: txDate,
					IsPaid:          true,
				})
				// Unpaid expense (should NOT affect in-hand balance)
				m.AddTransaction(&domain.Transaction{
					ID:              2,
					WorkspaceID:     1,
					AccountID:       1,
					Name:            "Utility Bill",
					Amount:          decimal.NewFromInt(200),
					Type:            domain.TransactionTypeExpense,
					TransactionDate: txDate,
					IsPaid:          false,
				})
			},
			setupMonth: func(m *testutil.MockMonthRepository) {
				m.AddMonth(&domain.Month{
					ID:              1,
					WorkspaceID:     1,
					Year:            testYear,
					Month:           testMonth,
					StartDate:       startDate,
					EndDate:         endDate,
					StartingBalance: decimal.NewFromInt(5000),
					CreatedAt:       startDate,
					UpdatedAt:       startDate,
				})
			},
			// Total: 5000 - 500 - 200 = 4300
			wantTotalBalance: "4300.00",
			// In-hand = 5000 + 0 - 500 (only paid) = 4500
			wantInHandBalance: "4500.00",
			// Disposable = In-hand - unpaid = 4500 - 200 = 4300
			wantDisposableIncome: "4300.00",
			wantErr:              false,
		},
		{
			name:        "handles zero accounts",
			workspaceID: 1,
			setupAccounts: func(m *testutil.MockAccountRepository) {
				// No accounts
			},
			setupTransactions: func(m *testutil.MockTransactionRepository) {
				// No transactions
			},
			setupMonth: func(m *testutil.MockMonthRepository) {
				m.AddMonth(&domain.Month{
					ID:              1,
					WorkspaceID:     1,
					Year:            testYear,
					Month:           testMonth,
					StartDate:       startDate,
					EndDate:         endDate,
					StartingBalance: decimal.Zero,
					CreatedAt:       startDate,
					UpdatedAt:       startDate,
				})
			},
			wantTotalBalance:     "0.00",
			wantInHandBalance:    "0.00",
			wantDisposableIncome: "0.00",
			wantErr:              false,
		},
		{
			name:        "handles only assets with income",
			workspaceID: 1,
			setupAccounts: func(m *testutil.MockAccountRepository) {
				m.AddAccount(&domain.Account{
					ID:             1,
					WorkspaceID:    1,
					Name:           "Bank",
					AccountType:    domain.AccountTypeAsset,
					Template:       domain.TemplateBank,
					InitialBalance: decimal.NewFromInt(10000),
				})
			},
			setupTransactions: func(m *testutil.MockTransactionRepository) {
				m.AddTransaction(&domain.Transaction{
					ID:              1,
					WorkspaceID:     1,
					AccountID:       1,
					Name:            "Salary",
					Amount:          decimal.NewFromInt(3000),
					Type:            domain.TransactionTypeIncome,
					TransactionDate: txDate,
					IsPaid:          true,
				})
			},
			setupMonth: func(m *testutil.MockMonthRepository) {
				m.AddMonth(&domain.Month{
					ID:              1,
					WorkspaceID:     1,
					Year:            testYear,
					Month:           testMonth,
					StartDate:       startDate,
					EndDate:         endDate,
					StartingBalance: decimal.NewFromInt(10000),
					CreatedAt:       startDate,
					UpdatedAt:       startDate,
				})
			},
			// Total = 10000 + 3000 = 13000
			wantTotalBalance: "13000.00",
			// In-hand = 10000 + 3000 - 0 = 13000
			wantInHandBalance: "13000.00",
			// Disposable = In-hand - unpaid = 13000 - 0 = 13000
			wantDisposableIncome: "13000.00",
			wantErr:              false,
		},
		{
			name:        "handles only liabilities (negative total)",
			workspaceID: 1,
			setupAccounts: func(m *testutil.MockAccountRepository) {
				m.AddAccount(&domain.Account{
					ID:             1,
					WorkspaceID:    1,
					Name:           "Credit Card",
					AccountType:    domain.AccountTypeLiability,
					Template:       domain.TemplateCreditCard,
					InitialBalance: decimal.Zero,
				})
			},
			setupTransactions: func(m *testutil.MockTransactionRepository) {
				m.AddTransaction(&domain.Transaction{
					ID:              1,
					WorkspaceID:     1,
					AccountID:       1,
					Name:            "Shopping",
					Amount:          decimal.NewFromInt(1000),
					Type:            domain.TransactionTypeExpense,
					TransactionDate: txDate,
					IsPaid:          false,
				})
			},
			setupMonth: func(m *testutil.MockMonthRepository) {
				m.AddMonth(&domain.Month{
					ID:              1,
					WorkspaceID:     1,
					Year:            testYear,
					Month:           testMonth,
					StartDate:       startDate,
					EndDate:         endDate,
					StartingBalance: decimal.Zero,
					CreatedAt:       startDate,
					UpdatedAt:       startDate,
				})
			},
			// CC calculated_balance = 0 - 1000 = -1000 (debt)
			// Total = -1000
			wantTotalBalance: "-1000.00",
			// In-hand = 0 + 0 - 0 (no paid expenses) = 0
			wantInHandBalance: "0.00",
			// Disposable = In-hand - unpaid = 0 - 1000 = -1000
			wantDisposableIncome: "-1000.00",
			wantErr:              false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			accountRepo := testutil.NewMockAccountRepository()
			transactionRepo := testutil.NewMockTransactionRepository()
			monthRepo := testutil.NewMockMonthRepository()
			loanPaymentRepo := testutil.NewMockLoanPaymentRepository()

			if tt.setupAccounts != nil {
				tt.setupAccounts(accountRepo)
			}
			if tt.setupTransactions != nil {
				tt.setupTransactions(transactionRepo)
			}
			if tt.setupMonth != nil {
				tt.setupMonth(monthRepo)
			}

			// Create services
			calcService := NewCalculationService(accountRepo, transactionRepo)
			monthService := NewMonthService(monthRepo, transactionRepo, calcService)
			dashboardService := NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)

			// Execute
			summary, err := dashboardService.GetSummaryForMonth(tt.workspaceID, testYear, testMonth)

			// Assert
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSummary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if summary.TotalBalance.StringFixed(2) != tt.wantTotalBalance {
					t.Errorf("GetSummary() TotalBalance = %v, want %v", summary.TotalBalance.StringFixed(2), tt.wantTotalBalance)
				}
				if summary.InHandBalance.StringFixed(2) != tt.wantInHandBalance {
					t.Errorf("GetSummary() InHandBalance = %v, want %v", summary.InHandBalance.StringFixed(2), tt.wantInHandBalance)
				}
				if summary.DisposableIncome.StringFixed(2) != tt.wantDisposableIncome {
					t.Errorf("GetSummary() DisposableIncome = %v, want %v", summary.DisposableIncome.StringFixed(2), tt.wantDisposableIncome)
				}
				if summary.Month == nil {
					t.Error("GetSummary() Month should not be nil")
				}
			}
		})
	}
}

func TestDashboardService_WorkspaceIsolation(t *testing.T) {
	// Fixed dates for testing
	testYear := 2025
	testMonth := 1
	startDate := time.Date(testYear, time.Month(testMonth), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, -1)
	txDate := time.Date(testYear, time.Month(testMonth), 15, 0, 0, 0, 0, time.UTC)

	// Setup mocks
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()

	// Workspace 1: Has accounts and transactions
	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    1,
		Name:           "Bank WS1",
		AccountType:    domain.AccountTypeAsset,
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromInt(10000),
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              1,
		WorkspaceID:     1,
		AccountID:       1,
		Name:            "Income WS1",
		Amount:          decimal.NewFromInt(5000),
		Type:            domain.TransactionTypeIncome,
		TransactionDate: txDate,
		IsPaid:          true,
	})
	monthRepo.AddMonth(&domain.Month{
		ID:              1,
		WorkspaceID:     1,
		Year:            testYear,
		Month:           testMonth,
		StartDate:       startDate,
		EndDate:         endDate,
		StartingBalance: decimal.NewFromInt(10000),
		CreatedAt:       startDate,
		UpdatedAt:       startDate,
	})

	// Workspace 2: Different data
	accountRepo.AddAccount(&domain.Account{
		ID:             2,
		WorkspaceID:    2,
		Name:           "Bank WS2",
		AccountType:    domain.AccountTypeAsset,
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromInt(1000),
	})
	monthRepo.AddMonth(&domain.Month{
		ID:              2,
		WorkspaceID:     2,
		Year:            testYear,
		Month:           testMonth,
		StartDate:       startDate,
		EndDate:         endDate,
		StartingBalance: decimal.NewFromInt(1000),
		CreatedAt:       startDate,
		UpdatedAt:       startDate,
	})

	// Create services
	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()
	calcService := NewCalculationService(accountRepo, transactionRepo)
	monthService := NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)

	// Get summaries for both workspaces using fixed dates
	summary1, err := dashboardService.GetSummaryForMonth(1, testYear, testMonth)
	if err != nil {
		t.Fatalf("GetSummaryForMonth(1) error = %v", err)
	}

	summary2, err := dashboardService.GetSummaryForMonth(2, testYear, testMonth)
	if err != nil {
		t.Fatalf("GetSummaryForMonth(2) error = %v", err)
	}

	// Workspace 1 should have 15000 total (10000 + 5000)
	if summary1.TotalBalance.StringFixed(2) != "15000.00" {
		t.Errorf("Workspace 1 TotalBalance = %v, want 15000.00", summary1.TotalBalance.StringFixed(2))
	}

	// Workspace 2 should have 1000 total
	if summary2.TotalBalance.StringFixed(2) != "1000.00" {
		t.Errorf("Workspace 2 TotalBalance = %v, want 1000.00", summary2.TotalBalance.StringFixed(2))
	}
}

func TestDashboardService_DaysRemainingAndDailyBudget(t *testing.T) {
	// Test that DaysRemaining and DailyBudget are calculated correctly
	// Note: DaysRemaining depends on current date, so we test relative behavior

	testYear := 2025
	testMonth := 1
	startDate := time.Date(testYear, time.Month(testMonth), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, -1)
	txDate := time.Date(testYear, time.Month(testMonth), 15, 0, 0, 0, 0, time.UTC)

	// Setup mocks
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()

	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    1,
		Name:           "Bank",
		AccountType:    domain.AccountTypeAsset,
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromInt(10000),
	})

	// Add income
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              1,
		WorkspaceID:     1,
		AccountID:       1,
		Name:            "Salary",
		Amount:          decimal.NewFromInt(3100),
		Type:            domain.TransactionTypeIncome,
		TransactionDate: txDate,
		IsPaid:          true,
	})

	// Add unpaid expense
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              2,
		WorkspaceID:     1,
		AccountID:       1,
		Name:            "Utility Bill",
		Amount:          decimal.NewFromInt(100),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: txDate,
		IsPaid:          false,
	})

	monthRepo.AddMonth(&domain.Month{
		ID:              1,
		WorkspaceID:     1,
		Year:            testYear,
		Month:           testMonth,
		StartDate:       startDate,
		EndDate:         endDate,
		StartingBalance: decimal.NewFromInt(10000),
		CreatedAt:       startDate,
		UpdatedAt:       startDate,
	})

	// Create services
	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()
	calcService := NewCalculationService(accountRepo, transactionRepo)
	monthService := NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)

	summary, err := dashboardService.GetSummaryForMonth(1, testYear, testMonth)
	if err != nil {
		t.Fatalf("GetSummaryForMonth() error = %v", err)
	}

	// Verify InHand = Starting + Income - Paid = 10000 + 3100 - 0 = 13100
	if summary.InHandBalance.StringFixed(2) != "13100.00" {
		t.Errorf("InHandBalance = %v, want 13100.00", summary.InHandBalance.StringFixed(2))
	}

	// Verify Disposable = InHand - Unpaid = 13100 - 100 = 13000
	if summary.DisposableIncome.StringFixed(2) != "13000.00" {
		t.Errorf("DisposableIncome = %v, want 13000.00", summary.DisposableIncome.StringFixed(2))
	}

	// DaysRemaining should be >= 0
	if summary.DaysRemaining < 0 {
		t.Errorf("DaysRemaining = %v, should be >= 0", summary.DaysRemaining)
	}

	// If DaysRemaining > 0, DailyBudget should be Disposable / Days
	if summary.DaysRemaining > 0 {
		expectedDailyBudget := summary.DisposableIncome.Div(decimal.NewFromInt(int64(summary.DaysRemaining)))
		if !summary.DailyBudget.Equal(expectedDailyBudget) {
			t.Errorf("DailyBudget = %v, want %v", summary.DailyBudget.StringFixed(2), expectedDailyBudget.StringFixed(2))
		}
	}

	// If viewing past month (DaysRemaining = 0), DailyBudget should be 0
	if summary.DaysRemaining == 0 {
		if !summary.DailyBudget.IsZero() {
			t.Errorf("DailyBudget for past month = %v, want 0", summary.DailyBudget.StringFixed(2))
		}
	}
}

func TestDashboardService_NegativeDisposableIncome(t *testing.T) {
	// Test negative disposable income scenario
	testYear := 2025
	testMonth := 1
	startDate := time.Date(testYear, time.Month(testMonth), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, -1)
	txDate := time.Date(testYear, time.Month(testMonth), 15, 0, 0, 0, 0, time.UTC)

	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()

	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    1,
		Name:           "Bank",
		AccountType:    domain.AccountTypeAsset,
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromInt(1000),
	})

	// Large unpaid expense that exceeds in-hand
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              1,
		WorkspaceID:     1,
		AccountID:       1,
		Name:            "Big Bill",
		Amount:          decimal.NewFromInt(5000),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: txDate,
		IsPaid:          false,
	})

	monthRepo.AddMonth(&domain.Month{
		ID:              1,
		WorkspaceID:     1,
		Year:            testYear,
		Month:           testMonth,
		StartDate:       startDate,
		EndDate:         endDate,
		StartingBalance: decimal.NewFromInt(1000),
	})

	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()
	calcService := NewCalculationService(accountRepo, transactionRepo)
	monthService := NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)

	summary, err := dashboardService.GetSummaryForMonth(1, testYear, testMonth)
	if err != nil {
		t.Fatalf("GetSummaryForMonth() error = %v", err)
	}

	// InHand = 1000 + 0 - 0 = 1000
	if summary.InHandBalance.StringFixed(2) != "1000.00" {
		t.Errorf("InHandBalance = %v, want 1000.00", summary.InHandBalance.StringFixed(2))
	}

	// Disposable = 1000 - 5000 = -4000 (negative!)
	if summary.DisposableIncome.StringFixed(2) != "-4000.00" {
		t.Errorf("DisposableIncome = %v, want -4000.00", summary.DisposableIncome.StringFixed(2))
	}

	// DailyBudget should also be negative if days > 0
	if summary.DaysRemaining > 0 && !summary.DailyBudget.IsNegative() {
		t.Errorf("DailyBudget should be negative when disposable is negative, got %v", summary.DailyBudget.StringFixed(2))
	}
}

func TestDashboardService_GetCCPayable(t *testing.T) {
	workspaceID := int32(1)

	tests := []struct {
		name              string
		setupTransactions func(*testutil.MockTransactionRepository)
		wantThisMonth     string
		wantNextMonth     string
		wantTotal         string
		wantErr           bool
	}{
		{
			name: "returns correct totals for mixed settlement intents",
			setupTransactions: func(m *testutil.MockTransactionRepository) {
				thisMonth := domain.CCSettlementThisMonth
				nextMonth := domain.CCSettlementNextMonth

				// This month: 1500
				m.AddTransaction(&domain.Transaction{
					ID:                 1,
					WorkspaceID:        workspaceID,
					AccountID:          1,
					Name:               "Groceries",
					Amount:             decimal.NewFromInt(1000),
					Type:               domain.TransactionTypeExpense,
					TransactionDate:    time.Now(),
					IsPaid:             false,
					CCSettlementIntent: &thisMonth,
				})
				m.AddTransaction(&domain.Transaction{
					ID:                 2,
					WorkspaceID:        workspaceID,
					AccountID:          1,
					Name:               "Shopping",
					Amount:             decimal.NewFromInt(500),
					Type:               domain.TransactionTypeExpense,
					TransactionDate:    time.Now(),
					IsPaid:             false,
					CCSettlementIntent: &thisMonth,
				})
				// Next month: 750
				m.AddTransaction(&domain.Transaction{
					ID:                 3,
					WorkspaceID:        workspaceID,
					AccountID:          1,
					Name:               "Electronics",
					Amount:             decimal.NewFromInt(750),
					Type:               domain.TransactionTypeExpense,
					TransactionDate:    time.Now(),
					IsPaid:             false,
					CCSettlementIntent: &nextMonth,
				})
			},
			wantThisMonth: "1500.00",
			wantNextMonth: "750.00",
			wantTotal:     "2250.00",
			wantErr:       false,
		},
		{
			name: "returns zeros when no CC transactions",
			setupTransactions: func(m *testutil.MockTransactionRepository) {
				// No CC transactions
			},
			wantThisMonth: "0.00",
			wantNextMonth: "0.00",
			wantTotal:     "0.00",
			wantErr:       false,
		},
		{
			name: "excludes paid transactions",
			setupTransactions: func(m *testutil.MockTransactionRepository) {
				thisMonth := domain.CCSettlementThisMonth

				// Unpaid - should be counted
				m.AddTransaction(&domain.Transaction{
					ID:                 1,
					WorkspaceID:        workspaceID,
					AccountID:          1,
					Name:               "Unpaid",
					Amount:             decimal.NewFromInt(100),
					Type:               domain.TransactionTypeExpense,
					TransactionDate:    time.Now(),
					IsPaid:             false,
					CCSettlementIntent: &thisMonth,
				})
				// Paid - should be excluded
				m.AddTransaction(&domain.Transaction{
					ID:                 2,
					WorkspaceID:        workspaceID,
					AccountID:          1,
					Name:               "Paid",
					Amount:             decimal.NewFromInt(200),
					Type:               domain.TransactionTypeExpense,
					TransactionDate:    time.Now(),
					IsPaid:             true,
					CCSettlementIntent: &thisMonth,
				})
			},
			wantThisMonth: "100.00",
			wantNextMonth: "0.00",
			wantTotal:     "100.00",
			wantErr:       false,
		},
		{
			name: "only this_month transactions",
			setupTransactions: func(m *testutil.MockTransactionRepository) {
				thisMonth := domain.CCSettlementThisMonth
				m.AddTransaction(&domain.Transaction{
					ID:                 1,
					WorkspaceID:        workspaceID,
					AccountID:          1,
					Name:               "Only This Month",
					Amount:             decimal.NewFromInt(300),
					Type:               domain.TransactionTypeExpense,
					TransactionDate:    time.Now(),
					IsPaid:             false,
					CCSettlementIntent: &thisMonth,
				})
			},
			wantThisMonth: "300.00",
			wantNextMonth: "0.00",
			wantTotal:     "300.00",
			wantErr:       false,
		},
		{
			name: "only next_month transactions",
			setupTransactions: func(m *testutil.MockTransactionRepository) {
				nextMonth := domain.CCSettlementNextMonth
				m.AddTransaction(&domain.Transaction{
					ID:                 1,
					WorkspaceID:        workspaceID,
					AccountID:          1,
					Name:               "Only Next Month",
					Amount:             decimal.NewFromInt(400),
					Type:               domain.TransactionTypeExpense,
					TransactionDate:    time.Now(),
					IsPaid:             false,
					CCSettlementIntent: &nextMonth,
				})
			},
			wantThisMonth: "0.00",
			wantNextMonth: "400.00",
			wantTotal:     "400.00",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accountRepo := testutil.NewMockAccountRepository()
			transactionRepo := testutil.NewMockTransactionRepository()
			monthRepo := testutil.NewMockMonthRepository()

			if tt.setupTransactions != nil {
				tt.setupTransactions(transactionRepo)
			}

			loanPaymentRepo := testutil.NewMockLoanPaymentRepository()
			calcService := NewCalculationService(accountRepo, transactionRepo)
			monthService := NewMonthService(monthRepo, transactionRepo, calcService)
			dashboardService := NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)

			summary, err := dashboardService.GetCCPayable(workspaceID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetCCPayable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if summary.ThisMonth.StringFixed(2) != tt.wantThisMonth {
					t.Errorf("GetCCPayable() ThisMonth = %v, want %v", summary.ThisMonth.StringFixed(2), tt.wantThisMonth)
				}
				if summary.NextMonth.StringFixed(2) != tt.wantNextMonth {
					t.Errorf("GetCCPayable() NextMonth = %v, want %v", summary.NextMonth.StringFixed(2), tt.wantNextMonth)
				}
				if summary.Total.StringFixed(2) != tt.wantTotal {
					t.Errorf("GetCCPayable() Total = %v, want %v", summary.Total.StringFixed(2), tt.wantTotal)
				}
			}
		})
	}
}

func TestDashboardService_Projection_FutureMonth(t *testing.T) {
	// Test that future months return projection data
	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	// Calculate future month (2 months ahead)
	futureYear := currentYear
	futureMonth := currentMonth + 2
	if futureMonth > 12 {
		futureMonth -= 12
		futureYear++
	}

	// Setup for current month (use Local to match service behavior)
	currentStartDate := time.Date(currentYear, time.Month(currentMonth), 1, 0, 0, 0, 0, time.Local)
	currentEndDate := currentStartDate.AddDate(0, 1, -1)

	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()

	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    1,
		Name:           "Bank",
		AccountType:    domain.AccountTypeAsset,
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromInt(10000),
	})

	// Current month with closing balance of 15000
	monthRepo.AddMonth(&domain.Month{
		ID:              1,
		WorkspaceID:     1,
		Year:            currentYear,
		Month:           currentMonth,
		StartDate:       currentStartDate,
		EndDate:         currentEndDate,
		StartingBalance: decimal.NewFromInt(10000),
	})

	// Add income transaction to increase closing balance
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              1,
		WorkspaceID:     1,
		AccountID:       1,
		Name:            "Salary",
		Amount:          decimal.NewFromInt(5000),
		Type:            domain.TransactionTypeIncome,
		TransactionDate: currentStartDate.AddDate(0, 0, 15),
		IsPaid:          true,
	})

	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()
	calcService := NewCalculationService(accountRepo, transactionRepo)
	monthService := NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)

	// Get projection for future month
	summary, err := dashboardService.GetSummaryForMonth(1, futureYear, futureMonth)
	if err != nil {
		t.Fatalf("GetSummaryForMonth() error = %v", err)
	}

	// Verify it's marked as projection
	if !summary.IsProjection {
		t.Error("Future month should have IsProjection = true")
	}

	// Verify projection limit is set
	if summary.ProjectionLimitMonths != domain.MaxProjectionMonths {
		t.Errorf("ProjectionLimitMonths = %v, want %v", summary.ProjectionLimitMonths, domain.MaxProjectionMonths)
	}

	// Verify projection details are populated
	if summary.Projection == nil {
		t.Fatal("Projection should not be nil for future month")
	}

	// MVP: All projection values should be zero
	if !summary.Projection.RecurringIncome.IsZero() {
		t.Errorf("RecurringIncome = %v, want 0", summary.Projection.RecurringIncome)
	}
	if !summary.Projection.RecurringExpenses.IsZero() {
		t.Errorf("RecurringExpenses = %v, want 0", summary.Projection.RecurringExpenses)
	}
	if !summary.Projection.LoanPayments.IsZero() {
		t.Errorf("LoanPayments = %v, want 0", summary.Projection.LoanPayments)
	}

	// Verify CC payable is zero for projected months
	if summary.CCPayable == nil {
		t.Fatal("CCPayable should not be nil for projection")
	}
	if !summary.CCPayable.ThisMonth.IsZero() {
		t.Errorf("CCPayable.ThisMonth = %v, want 0", summary.CCPayable.ThisMonth)
	}
	if !summary.CCPayable.NextMonth.IsZero() {
		t.Errorf("CCPayable.NextMonth = %v, want 0", summary.CCPayable.NextMonth)
	}
	if !summary.CCPayable.Total.IsZero() {
		t.Errorf("CCPayable.Total = %v, want 0", summary.CCPayable.Total)
	}

	// Verify month data is set correctly
	if summary.Month == nil {
		t.Fatal("Month should not be nil")
	}
	// Note: CalculatedMonth embeds Month, so Month.Month accesses embedded struct's Month field
	if summary.Month.Year != futureYear || summary.Month.Month.Month != futureMonth {
		t.Errorf("Month = %d/%d, want %d/%d", summary.Month.Year, summary.Month.Month.Month, futureYear, futureMonth)
	}
}

func TestDashboardService_Projection_CurrentMonthNotProjection(t *testing.T) {
	// Test that current month returns isProjection: false
	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	// Use Local to match service behavior
	startDate := time.Date(currentYear, time.Month(currentMonth), 1, 0, 0, 0, 0, time.Local)
	endDate := startDate.AddDate(0, 1, -1)

	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()

	monthRepo.AddMonth(&domain.Month{
		ID:              1,
		WorkspaceID:     1,
		Year:            currentYear,
		Month:           currentMonth,
		StartDate:       startDate,
		EndDate:         endDate,
		StartingBalance: decimal.NewFromInt(5000),
	})

	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()
	calcService := NewCalculationService(accountRepo, transactionRepo)
	monthService := NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)

	summary, err := dashboardService.GetSummaryForMonth(1, currentYear, currentMonth)
	if err != nil {
		t.Fatalf("GetSummaryForMonth() error = %v", err)
	}

	// Current month should NOT be a projection
	if summary.IsProjection {
		t.Error("Current month should have IsProjection = false")
	}

	// Projection details should be nil for current month
	if summary.Projection != nil {
		t.Error("Projection should be nil for current month")
	}
}

func TestDashboardService_Projection_PastMonthNotProjection(t *testing.T) {
	// Test that past month returns isProjection: false
	pastYear := 2024
	pastMonth := 6

	// Use Local to match service behavior
	startDate := time.Date(pastYear, time.Month(pastMonth), 1, 0, 0, 0, 0, time.Local)
	endDate := startDate.AddDate(0, 1, -1)

	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()

	monthRepo.AddMonth(&domain.Month{
		ID:              1,
		WorkspaceID:     1,
		Year:            pastYear,
		Month:           pastMonth,
		StartDate:       startDate,
		EndDate:         endDate,
		StartingBalance: decimal.NewFromInt(5000),
	})

	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()
	calcService := NewCalculationService(accountRepo, transactionRepo)
	monthService := NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)

	summary, err := dashboardService.GetSummaryForMonth(1, pastYear, pastMonth)
	if err != nil {
		t.Fatalf("GetSummaryForMonth() error = %v", err)
	}

	// Past month should NOT be a projection
	if summary.IsProjection {
		t.Error("Past month should have IsProjection = false")
	}
}

func TestDashboardService_Projection_LimitExceeded(t *testing.T) {
	// Test that requesting >12 months ahead returns error
	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	// Calculate 13 months ahead
	futureYear := currentYear + 1
	futureMonth := currentMonth + 1
	if futureMonth > 12 {
		futureMonth -= 12
		futureYear++
	}

	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()
	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()

	calcService := NewCalculationService(accountRepo, transactionRepo)
	monthService := NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)

	_, err := dashboardService.GetSummaryForMonth(1, futureYear, futureMonth)
	if err == nil {
		t.Error("Expected error for projection beyond 12 months")
	}

	if err != ErrProjectionLimitExceeded {
		t.Errorf("Expected ErrProjectionLimitExceeded, got %v", err)
	}
}

func TestDashboardService_Projection_ExactlyAtLimit(t *testing.T) {
	// Test that requesting exactly 12 months ahead succeeds (boundary test)
	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	// Calculate exactly 12 months ahead
	futureYear := currentYear + 1
	futureMonth := currentMonth
	// No adjustment needed - 12 months ahead is same month next year

	// Setup current month for balance chaining
	currentStartDate := time.Date(currentYear, time.Month(currentMonth), 1, 0, 0, 0, 0, time.Local)
	currentEndDate := currentStartDate.AddDate(0, 1, -1)

	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()

	monthRepo.AddMonth(&domain.Month{
		ID:              1,
		WorkspaceID:     1,
		Year:            currentYear,
		Month:           currentMonth,
		StartDate:       currentStartDate,
		EndDate:         currentEndDate,
		StartingBalance: decimal.NewFromInt(5000),
	})

	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()
	calcService := NewCalculationService(accountRepo, transactionRepo)
	monthService := NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)

	// Should succeed - exactly at limit
	summary, err := dashboardService.GetSummaryForMonth(1, futureYear, futureMonth)
	if err != nil {
		t.Fatalf("Expected success for exactly 12 months ahead, got error: %v", err)
	}

	if !summary.IsProjection {
		t.Error("Should be marked as projection")
	}
}

func TestDashboardService_Projection_BalanceChaining(t *testing.T) {
	// Test that future month uses current month's closing balance
	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	// Next month
	futureYear := currentYear
	futureMonth := currentMonth + 1
	if futureMonth > 12 {
		futureMonth = 1
		futureYear++
	}

	// Use Local to match service behavior
	currentStartDate := time.Date(currentYear, time.Month(currentMonth), 1, 0, 0, 0, 0, time.Local)
	currentEndDate := currentStartDate.AddDate(0, 1, -1)

	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()

	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    1,
		Name:           "Bank",
		AccountType:    domain.AccountTypeAsset,
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromInt(10000),
	})

	// Current month with specific closing balance
	monthRepo.AddMonth(&domain.Month{
		ID:              1,
		WorkspaceID:     1,
		Year:            currentYear,
		Month:           currentMonth,
		StartDate:       currentStartDate,
		EndDate:         currentEndDate,
		StartingBalance: decimal.NewFromInt(10000),
	})

	// Add transactions that affect closing balance
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              1,
		WorkspaceID:     1,
		AccountID:       1,
		Name:            "Income",
		Amount:          decimal.NewFromInt(5000),
		Type:            domain.TransactionTypeIncome,
		TransactionDate: currentStartDate.AddDate(0, 0, 5),
		IsPaid:          true,
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              2,
		WorkspaceID:     1,
		AccountID:       1,
		Name:            "Expense",
		Amount:          decimal.NewFromInt(2000),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: currentStartDate.AddDate(0, 0, 10),
		IsPaid:          true,
	})

	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()
	calcService := NewCalculationService(accountRepo, transactionRepo)
	monthService := NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)

	// Get projection for next month
	summary, err := dashboardService.GetSummaryForMonth(1, futureYear, futureMonth)
	if err != nil {
		t.Fatalf("GetSummaryForMonth() error = %v", err)
	}

	// Expected closing balance: 10000 + 5000 - 2000 = 13000
	// Future month should use this as starting balance
	expectedBalance := decimal.NewFromInt(13000)
	if !summary.Month.StartingBalance.Equal(expectedBalance) {
		t.Errorf("Projected StartingBalance = %v, want %v", summary.Month.StartingBalance, expectedBalance)
	}

	// For MVP, closing = starting (no changes)
	if !summary.Month.ClosingBalance.Equal(expectedBalance) {
		t.Errorf("Projected ClosingBalance = %v, want %v (same as starting for MVP)", summary.Month.ClosingBalance, expectedBalance)
	}
}

func TestDashboardService_GetSummary_IncludesCCPayable(t *testing.T) {
	testYear := 2025
	testMonth := 1
	startDate := time.Date(testYear, time.Month(testMonth), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, -1)
	txDate := time.Date(testYear, time.Month(testMonth), 15, 0, 0, 0, 0, time.UTC)

	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()

	// Setup accounts
	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    1,
		Name:           "Credit Card",
		AccountType:    domain.AccountTypeLiability,
		Template:       domain.TemplateCreditCard,
		InitialBalance: decimal.Zero,
	})

	// Setup CC transactions with settlement intent
	thisMonth := domain.CCSettlementThisMonth
	nextMonth := domain.CCSettlementNextMonth

	transactionRepo.AddTransaction(&domain.Transaction{
		ID:                 1,
		WorkspaceID:        1,
		AccountID:          1,
		Name:               "Groceries",
		Amount:             decimal.NewFromInt(500),
		Type:               domain.TransactionTypeExpense,
		TransactionDate:    txDate,
		IsPaid:             false,
		CCSettlementIntent: &thisMonth,
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:                 2,
		WorkspaceID:        1,
		AccountID:          1,
		Name:               "Shopping",
		Amount:             decimal.NewFromInt(300),
		Type:               domain.TransactionTypeExpense,
		TransactionDate:    txDate,
		IsPaid:             false,
		CCSettlementIntent: &nextMonth,
	})

	monthRepo.AddMonth(&domain.Month{
		ID:              1,
		WorkspaceID:     1,
		Year:            testYear,
		Month:           testMonth,
		StartDate:       startDate,
		EndDate:         endDate,
		StartingBalance: decimal.Zero,
	})

	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()
	calcService := NewCalculationService(accountRepo, transactionRepo)
	monthService := NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)

	summary, err := dashboardService.GetSummaryForMonth(1, testYear, testMonth)
	if err != nil {
		t.Fatalf("GetSummaryForMonth() error = %v", err)
	}

	// Verify CCPayable is included in the summary
	if summary.CCPayable == nil {
		t.Fatal("GetSummaryForMonth() CCPayable should not be nil")
	}

	if summary.CCPayable.ThisMonth.StringFixed(2) != "500.00" {
		t.Errorf("CCPayable.ThisMonth = %v, want 500.00", summary.CCPayable.ThisMonth.StringFixed(2))
	}
	if summary.CCPayable.NextMonth.StringFixed(2) != "300.00" {
		t.Errorf("CCPayable.NextMonth = %v, want 300.00", summary.CCPayable.NextMonth.StringFixed(2))
	}
	if summary.CCPayable.Total.StringFixed(2) != "800.00" {
		t.Errorf("CCPayable.Total = %v, want 800.00", summary.CCPayable.Total.StringFixed(2))
	}
}

// Tests for GetFutureSpending service method

func TestDashboardService_GetFutureSpending_BasicAggregation(t *testing.T) {
	now := time.Now()
	workspaceID := int32(1)

	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()
	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()

	// Add account
	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    workspaceID,
		Name:           "Bank Account",
		AccountType:    domain.AccountTypeAsset,
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromInt(10000),
		CreatedAt:      now,
		UpdatedAt:      now,
	})

	// Add expense transactions in current month
	categoryID := int32(1)
	categoryName := "Food"
	txDate := time.Date(now.Year(), now.Month(), 15, 12, 0, 0, 0, time.UTC)

	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              1,
		WorkspaceID:     workspaceID,
		AccountID:       1,
		Name:            "Groceries",
		Amount:          decimal.NewFromInt(500),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: txDate,
		IsPaid:          true,
		CategoryID:      &categoryID,
		CategoryName:    &categoryName,
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              2,
		WorkspaceID:     workspaceID,
		AccountID:       1,
		Name:            "Restaurant",
		Amount:          decimal.NewFromInt(200),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: txDate,
		IsPaid:          true,
		CategoryID:      &categoryID,
		CategoryName:    &categoryName,
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	calcService := NewCalculationService(accountRepo, transactionRepo)
	monthService := NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)

	result, err := dashboardService.GetFutureSpending(workspaceID, 3)
	if err != nil {
		t.Fatalf("GetFutureSpending() error = %v", err)
	}

	// Should have 3 months
	if len(result.Months) != 3 {
		t.Errorf("Expected 3 months, got %d", len(result.Months))
	}

	// First month should have total = 700 (500 + 200)
	if result.Months[0].Total != "700.00" {
		t.Errorf("First month total = %s, want 700.00", result.Months[0].Total)
	}

	// Should have category breakdown
	if len(result.Months[0].ByCategory) != 1 {
		t.Errorf("Expected 1 category, got %d", len(result.Months[0].ByCategory))
	}

	// Category amount should be 700
	if len(result.Months[0].ByCategory) > 0 && result.Months[0].ByCategory[0].Amount != "700.00" {
		t.Errorf("Category amount = %s, want 700.00", result.Months[0].ByCategory[0].Amount)
	}
}

func TestDashboardService_GetFutureSpending_IncludesDeferredCC(t *testing.T) {
	now := time.Now()
	workspaceID := int32(1)

	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()
	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()

	// Add CC account
	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    workspaceID,
		Name:           "Credit Card",
		AccountType:    domain.AccountTypeLiability,
		Template:       domain.TemplateCreditCard,
		InitialBalance: decimal.Zero,
		CreatedAt:      now,
		UpdatedAt:      now,
	})

	// Add deferred CC transaction from last month (billed + deferred)
	ccState := domain.CCStateBilled
	settlementIntent := domain.SettlementIntentDeferred
	lastMonth := now.AddDate(0, -1, 0)

	transactionRepo.AddTransaction(&domain.Transaction{
		ID:               1,
		WorkspaceID:      workspaceID,
		AccountID:        1,
		Name:             "Deferred Purchase",
		Amount:           decimal.NewFromInt(300),
		Type:             domain.TransactionTypeExpense,
		TransactionDate:  lastMonth,
		IsPaid:           false,
		CCState:          &ccState,
		SettlementIntent: &settlementIntent,
		CreatedAt:        now,
		UpdatedAt:        now,
	})

	calcService := NewCalculationService(accountRepo, transactionRepo)
	monthService := NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)

	result, err := dashboardService.GetFutureSpending(workspaceID, 1)
	if err != nil {
		t.Fatalf("GetFutureSpending() error = %v", err)
	}

	// Current month should include deferred CC carried forward
	if len(result.Months) != 1 {
		t.Fatalf("Expected 1 month, got %d", len(result.Months))
	}

	// Total should include the deferred amount
	if result.Months[0].Total != "300.00" {
		t.Errorf("Current month total = %s, want 300.00 (deferred CC)", result.Months[0].Total)
	}
}

func TestDashboardService_GetFutureSpending_ExcludesIncome(t *testing.T) {
	now := time.Now()
	workspaceID := int32(1)

	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()
	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()

	// Add account
	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    workspaceID,
		Name:           "Bank Account",
		AccountType:    domain.AccountTypeAsset,
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromInt(10000),
		CreatedAt:      now,
		UpdatedAt:      now,
	})

	txDate := time.Date(now.Year(), now.Month(), 15, 12, 0, 0, 0, time.UTC)

	// Add expense
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              1,
		WorkspaceID:     workspaceID,
		AccountID:       1,
		Name:            "Expense",
		Amount:          decimal.NewFromInt(100),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: txDate,
		IsPaid:          true,
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	// Add income (should be excluded from spending)
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              2,
		WorkspaceID:     workspaceID,
		AccountID:       1,
		Name:            "Salary",
		Amount:          decimal.NewFromInt(5000),
		Type:            domain.TransactionTypeIncome,
		TransactionDate: txDate,
		IsPaid:          true,
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	calcService := NewCalculationService(accountRepo, transactionRepo)
	monthService := NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)

	result, err := dashboardService.GetFutureSpending(workspaceID, 1)
	if err != nil {
		t.Fatalf("GetFutureSpending() error = %v", err)
	}

	// Total should only include expense, not income
	if result.Months[0].Total != "100.00" {
		t.Errorf("Total = %s, want 100.00 (income should be excluded)", result.Months[0].Total)
	}
}

func TestDashboardService_GetFutureSpending_EmptyMonths(t *testing.T) {
	workspaceID := int32(1)

	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()
	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()

	calcService := NewCalculationService(accountRepo, transactionRepo)
	monthService := NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)

	result, err := dashboardService.GetFutureSpending(workspaceID, 6)
	if err != nil {
		t.Fatalf("GetFutureSpending() error = %v", err)
	}

	// Should have 6 months even with no data
	if len(result.Months) != 6 {
		t.Errorf("Expected 6 months, got %d", len(result.Months))
	}

	// All months should have zero totals
	for i, month := range result.Months {
		if month.Total != "0.00" {
			t.Errorf("Month %d total = %s, want 0.00", i, month.Total)
		}
	}
}

func TestDashboardService_GetFutureSpending_MultipleAccounts(t *testing.T) {
	now := time.Now()
	workspaceID := int32(1)

	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()
	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()

	// Add two accounts
	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    workspaceID,
		Name:           "Bank Account",
		AccountType:    domain.AccountTypeAsset,
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromInt(10000),
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	accountRepo.AddAccount(&domain.Account{
		ID:             2,
		WorkspaceID:    workspaceID,
		Name:           "Cash",
		AccountType:    domain.AccountTypeAsset,
		Template:       domain.TemplateCash,
		InitialBalance: decimal.NewFromInt(500),
		CreatedAt:      now,
		UpdatedAt:      now,
	})

	txDate := time.Date(now.Year(), now.Month(), 15, 12, 0, 0, 0, time.UTC)

	// Add expense from account 1
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              1,
		WorkspaceID:     workspaceID,
		AccountID:       1,
		Name:            "Card Payment",
		Amount:          decimal.NewFromInt(300),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: txDate,
		IsPaid:          true,
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	// Add expense from account 2
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              2,
		WorkspaceID:     workspaceID,
		AccountID:       2,
		Name:            "Cash Payment",
		Amount:          decimal.NewFromInt(100),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: txDate,
		IsPaid:          true,
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	calcService := NewCalculationService(accountRepo, transactionRepo)
	monthService := NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)

	result, err := dashboardService.GetFutureSpending(workspaceID, 1)
	if err != nil {
		t.Fatalf("GetFutureSpending() error = %v", err)
	}

	// Total should be 400 (300 + 100)
	if result.Months[0].Total != "400.00" {
		t.Errorf("Total = %s, want 400.00", result.Months[0].Total)
	}

	// Should have 2 accounts in breakdown
	if len(result.Months[0].ByAccount) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(result.Months[0].ByAccount))
	}
}

// BenchmarkDashboardService_GetFutureSpending benchmarks the GetFutureSpending method
// to verify NFR-P6 compliance (<500ms response time)
func BenchmarkDashboardService_GetFutureSpending(b *testing.B) {
	now := time.Now()
	workspaceID := int32(1)

	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()
	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()

	// Add realistic number of accounts (5 accounts)
	for i := int32(1); i <= 5; i++ {
		accountRepo.AddAccount(&domain.Account{
			ID:             i,
			WorkspaceID:    workspaceID,
			Name:           "Account " + string(rune('A'+i-1)),
			AccountType:    domain.AccountTypeAsset,
			Template:       domain.TemplateBank,
			InitialBalance: decimal.NewFromInt(10000),
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}

	// Add realistic volume of transactions (500 transactions over 12 months)
	categoryID := int32(1)
	categoryName := "Expenses"
	for i := 1; i <= 500; i++ {
		monthOffset := (i % 12)
		txDate := now.AddDate(0, monthOffset, -(i % 28))
		accountID := int32((i % 5) + 1)

		transactionRepo.AddTransaction(&domain.Transaction{
			ID:              int32(i),
			WorkspaceID:     workspaceID,
			AccountID:       accountID,
			Name:            "Transaction " + string(rune(i)),
			Amount:          decimal.NewFromInt(int64(50 + (i % 500))),
			Type:            domain.TransactionTypeExpense,
			TransactionDate: txDate,
			IsPaid:          true,
			CategoryID:      &categoryID,
			CategoryName:    &categoryName,
			CreatedAt:       now,
			UpdatedAt:       now,
		})
	}

	calcService := NewCalculationService(accountRepo, transactionRepo)
	monthService := NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)

	// Reset timer to exclude setup time
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := dashboardService.GetFutureSpending(workspaceID, 12)
		if err != nil {
			b.Fatalf("GetFutureSpending() error = %v", err)
		}
	}
}

// TestDashboardService_GetFutureSpending_Performance verifies NFR-P6 compliance
// Requirement: <500ms response time for future spending endpoint
func TestDashboardService_GetFutureSpending_Performance(t *testing.T) {
	now := time.Now()
	workspaceID := int32(1)

	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()
	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()

	// Add realistic number of accounts
	for i := int32(1); i <= 5; i++ {
		accountRepo.AddAccount(&domain.Account{
			ID:             i,
			WorkspaceID:    workspaceID,
			Name:           "Account " + string(rune('A'+i-1)),
			AccountType:    domain.AccountTypeAsset,
			Template:       domain.TemplateBank,
			InitialBalance: decimal.NewFromInt(10000),
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}

	// Add realistic volume of transactions (500 transactions over 12 months)
	categoryID := int32(1)
	categoryName := "Expenses"
	for i := 1; i <= 500; i++ {
		monthOffset := (i % 12)
		txDate := now.AddDate(0, monthOffset, -(i % 28))
		accountID := int32((i % 5) + 1)

		transactionRepo.AddTransaction(&domain.Transaction{
			ID:              int32(i),
			WorkspaceID:     workspaceID,
			AccountID:       accountID,
			Name:            "Transaction " + string(rune(i)),
			Amount:          decimal.NewFromInt(int64(50 + (i % 500))),
			Type:            domain.TransactionTypeExpense,
			TransactionDate: txDate,
			IsPaid:          true,
			CategoryID:      &categoryID,
			CategoryName:    &categoryName,
			CreatedAt:       now,
			UpdatedAt:       now,
		})
	}

	calcService := NewCalculationService(accountRepo, transactionRepo)
	monthService := NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)

	// Measure execution time
	start := time.Now()
	_, err := dashboardService.GetFutureSpending(workspaceID, 12)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("GetFutureSpending() error = %v", err)
	}

	// NFR-P6: Response time must be <500ms
	maxDuration := 500 * time.Millisecond
	if elapsed > maxDuration {
		t.Errorf("GetFutureSpending() took %v, expected <%v (NFR-P6 violation)", elapsed, maxDuration)
	}

	t.Logf("GetFutureSpending() completed in %v (limit: %v)", elapsed, maxDuration)
}
