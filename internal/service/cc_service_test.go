package service

import (
	"testing"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/shopspring/decimal"
)

func TestGetPayableBreakdown_Success_MixedIntents(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	ccService := NewCCService(transactionRepo)

	workspaceID := int32(1)

	// Configure mock with mixed settlement intents
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
				SettlementIntent: domain.CCSettlementThisMonth,
				AccountID:        1,
				AccountName:      "Maybank CC",
			},
			{
				ID:               3,
				Name:             "Dining",
				Amount:           decimal.NewFromFloat(50.00),
				TransactionDate:  time.Now(),
				SettlementIntent: domain.CCSettlementNextMonth,
				AccountID:        2,
				AccountName:      "CIMB CC",
			},
		}, nil
	}

	result, err := ccService.GetPayableBreakdown(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify this month totals
	if !result.ThisMonthTotal.Equal(decimal.NewFromFloat(300.00)) {
		t.Errorf("Expected this month total 300.00, got %s", result.ThisMonthTotal.String())
	}

	// Verify next month totals
	if !result.NextMonthTotal.Equal(decimal.NewFromFloat(50.00)) {
		t.Errorf("Expected next month total 50.00, got %s", result.NextMonthTotal.String())
	}

	// Verify grand total
	if !result.GrandTotal.Equal(decimal.NewFromFloat(350.00)) {
		t.Errorf("Expected grand total 350.00, got %s", result.GrandTotal.String())
	}

	// Verify account grouping for this month
	if len(result.ThisMonth) != 1 {
		t.Fatalf("Expected 1 account in this month, got %d", len(result.ThisMonth))
	}
	if result.ThisMonth[0].AccountName != "Maybank CC" {
		t.Errorf("Expected Maybank CC, got %s", result.ThisMonth[0].AccountName)
	}
	if len(result.ThisMonth[0].Transactions) != 2 {
		t.Errorf("Expected 2 transactions in Maybank CC, got %d", len(result.ThisMonth[0].Transactions))
	}

	// Verify account grouping for next month
	if len(result.NextMonth) != 1 {
		t.Fatalf("Expected 1 account in next month, got %d", len(result.NextMonth))
	}
	if result.NextMonth[0].AccountName != "CIMB CC" {
		t.Errorf("Expected CIMB CC, got %s", result.NextMonth[0].AccountName)
	}
}

func TestGetPayableBreakdown_NoTransactions(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	ccService := NewCCService(transactionRepo)

	workspaceID := int32(1)

	// Configure mock with no transactions
	transactionRepo.GetCCPayableBreakdownFn = func(wsID int32) ([]*domain.CCPayableTransaction, error) {
		return []*domain.CCPayableTransaction{}, nil
	}

	result, err := ccService.GetPayableBreakdown(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !result.ThisMonthTotal.IsZero() {
		t.Errorf("Expected zero this month total, got %s", result.ThisMonthTotal.String())
	}
	if !result.NextMonthTotal.IsZero() {
		t.Errorf("Expected zero next month total, got %s", result.NextMonthTotal.String())
	}
	if !result.GrandTotal.IsZero() {
		t.Errorf("Expected zero grand total, got %s", result.GrandTotal.String())
	}
	if len(result.ThisMonth) != 0 {
		t.Errorf("Expected empty this month, got %d accounts", len(result.ThisMonth))
	}
	if len(result.NextMonth) != 0 {
		t.Errorf("Expected empty next month, got %d accounts", len(result.NextMonth))
	}
}

func TestGetPayableBreakdown_OnlyThisMonth(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	ccService := NewCCService(transactionRepo)

	workspaceID := int32(1)

	transactionRepo.GetCCPayableBreakdownFn = func(wsID int32) ([]*domain.CCPayableTransaction, error) {
		return []*domain.CCPayableTransaction{
			{
				ID:               1,
				Name:             "Transaction 1",
				Amount:           decimal.NewFromFloat(500.00),
				TransactionDate:  time.Now(),
				SettlementIntent: domain.CCSettlementThisMonth,
				AccountID:        1,
				AccountName:      "Card A",
			},
		}, nil
	}

	result, err := ccService.GetPayableBreakdown(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !result.ThisMonthTotal.Equal(decimal.NewFromFloat(500.00)) {
		t.Errorf("Expected this month total 500.00, got %s", result.ThisMonthTotal.String())
	}
	if !result.NextMonthTotal.IsZero() {
		t.Errorf("Expected zero next month total, got %s", result.NextMonthTotal.String())
	}
	if len(result.ThisMonth) != 1 {
		t.Errorf("Expected 1 account in this month, got %d", len(result.ThisMonth))
	}
	if len(result.NextMonth) != 0 {
		t.Errorf("Expected 0 accounts in next month, got %d", len(result.NextMonth))
	}
}

func TestGetPayableBreakdown_OnlyNextMonth(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	ccService := NewCCService(transactionRepo)

	workspaceID := int32(1)

	transactionRepo.GetCCPayableBreakdownFn = func(wsID int32) ([]*domain.CCPayableTransaction, error) {
		return []*domain.CCPayableTransaction{
			{
				ID:               1,
				Name:             "Transaction 1",
				Amount:           decimal.NewFromFloat(250.00),
				TransactionDate:  time.Now(),
				SettlementIntent: domain.CCSettlementNextMonth,
				AccountID:        1,
				AccountName:      "Card B",
			},
		}, nil
	}

	result, err := ccService.GetPayableBreakdown(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !result.ThisMonthTotal.IsZero() {
		t.Errorf("Expected zero this month total, got %s", result.ThisMonthTotal.String())
	}
	if !result.NextMonthTotal.Equal(decimal.NewFromFloat(250.00)) {
		t.Errorf("Expected next month total 250.00, got %s", result.NextMonthTotal.String())
	}
	if len(result.ThisMonth) != 0 {
		t.Errorf("Expected 0 accounts in this month, got %d", len(result.ThisMonth))
	}
	if len(result.NextMonth) != 1 {
		t.Errorf("Expected 1 account in next month, got %d", len(result.NextMonth))
	}
}

func TestGetPayableBreakdown_MultipleAccounts(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	ccService := NewCCService(transactionRepo)

	workspaceID := int32(1)

	transactionRepo.GetCCPayableBreakdownFn = func(wsID int32) ([]*domain.CCPayableTransaction, error) {
		return []*domain.CCPayableTransaction{
			{
				ID:               1,
				Name:             "T1",
				Amount:           decimal.NewFromFloat(100.00),
				TransactionDate:  time.Now(),
				SettlementIntent: domain.CCSettlementThisMonth,
				AccountID:        1,
				AccountName:      "Card A",
			},
			{
				ID:               2,
				Name:             "T2",
				Amount:           decimal.NewFromFloat(200.00),
				TransactionDate:  time.Now(),
				SettlementIntent: domain.CCSettlementThisMonth,
				AccountID:        2,
				AccountName:      "Card B",
			},
			{
				ID:               3,
				Name:             "T3",
				Amount:           decimal.NewFromFloat(150.00),
				TransactionDate:  time.Now(),
				SettlementIntent: domain.CCSettlementThisMonth,
				AccountID:        3,
				AccountName:      "Card C",
			},
		}, nil
	}

	result, err := ccService.GetPayableBreakdown(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.ThisMonth) != 3 {
		t.Errorf("Expected 3 accounts in this month, got %d", len(result.ThisMonth))
	}

	// Verify sorted by account name
	if result.ThisMonth[0].AccountName != "Card A" {
		t.Errorf("Expected first account to be Card A, got %s", result.ThisMonth[0].AccountName)
	}
	if result.ThisMonth[1].AccountName != "Card B" {
		t.Errorf("Expected second account to be Card B, got %s", result.ThisMonth[1].AccountName)
	}
	if result.ThisMonth[2].AccountName != "Card C" {
		t.Errorf("Expected third account to be Card C, got %s", result.ThisMonth[2].AccountName)
	}
}

func TestGetPayableBreakdown_RepoError(t *testing.T) {
	transactionRepo := testutil.NewMockTransactionRepository()
	ccService := NewCCService(transactionRepo)

	workspaceID := int32(1)

	transactionRepo.GetCCPayableBreakdownFn = func(wsID int32) ([]*domain.CCPayableTransaction, error) {
		return nil, domain.ErrTransactionNotFound
	}

	_, err := ccService.GetPayableBreakdown(workspaceID)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if err != domain.ErrTransactionNotFound {
		t.Errorf("Expected ErrTransactionNotFound, got %v", err)
	}
}
