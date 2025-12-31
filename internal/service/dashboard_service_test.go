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
		name              string
		workspaceID       int32
		setupAccounts     func(*testutil.MockAccountRepository)
		setupTransactions func(*testutil.MockTransactionRepository)
		setupMonth        func(*testutil.MockMonthRepository)
		wantTotalBalance  string
		wantInHandBalance string
		wantErr           bool
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
			wantErr:           false,
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
			wantErr:           false,
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
			wantTotalBalance:  "0.00",
			wantInHandBalance: "0.00",
			wantErr:           false,
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
			wantErr:           false,
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
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			accountRepo := testutil.NewMockAccountRepository()
			transactionRepo := testutil.NewMockTransactionRepository()
			monthRepo := testutil.NewMockMonthRepository()

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
			dashboardService := NewDashboardService(accountRepo, transactionRepo, monthService, calcService)

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
	calcService := NewCalculationService(accountRepo, transactionRepo)
	monthService := NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := NewDashboardService(accountRepo, transactionRepo, monthService, calcService)

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
