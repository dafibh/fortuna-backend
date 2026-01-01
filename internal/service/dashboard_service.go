package service

import (
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/shopspring/decimal"
)

// DashboardService handles dashboard-related business logic
type DashboardService struct {
	accountRepo     domain.AccountRepository
	transactionRepo domain.TransactionRepository
	monthService    *MonthService
	calcService     *CalculationService
}

// NewDashboardService creates a new DashboardService
func NewDashboardService(
	accountRepo domain.AccountRepository,
	transactionRepo domain.TransactionRepository,
	monthService *MonthService,
	calcService *CalculationService,
) *DashboardService {
	return &DashboardService{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
		monthService:    monthService,
		calcService:     calcService,
	}
}

// GetSummary returns the dashboard summary for a workspace for the current month
func (s *DashboardService) GetSummary(workspaceID int32) (*domain.DashboardSummary, error) {
	now := time.Now()
	return s.GetSummaryForMonth(workspaceID, now.Year(), int(now.Month()))
}

// GetSummaryForMonth returns the dashboard summary for a workspace for a specific month
func (s *DashboardService) GetSummaryForMonth(workspaceID int32, year, month int) (*domain.DashboardSummary, error) {
	// 1. Get month data
	monthData, err := s.monthService.GetOrCreateMonth(workspaceID, year, month)
	if err != nil {
		return nil, err
	}

	// 2. Calculate total balance from all accounts (assets - liabilities)
	totalBalance, err := s.calculateTotalBalance(workspaceID)
	if err != nil {
		return nil, err
	}

	// 3. Calculate in-hand balance (starting + income - paid expenses only)
	paidExpenses, err := s.transactionRepo.SumPaidExpensesByDateRange(
		workspaceID, monthData.StartDate, monthData.EndDate)
	if err != nil {
		return nil, err
	}

	inHandBalance := monthData.StartingBalance.Add(monthData.TotalIncome).Sub(paidExpenses)

	// 4. Calculate unpaid expenses for disposable income
	unpaidExpenses, err := s.transactionRepo.SumUnpaidExpensesByDateRange(
		workspaceID, monthData.StartDate, monthData.EndDate)
	if err != nil {
		return nil, err
	}

	// Disposable = In-Hand - Unpaid Expenses
	disposableIncome := inHandBalance.Sub(unpaidExpenses)

	// 5. Calculate days remaining in the month
	daysRemaining := s.calculateDaysRemaining(year, month)

	// 6. Calculate daily budget
	dailyBudget := decimal.Zero
	if daysRemaining > 0 {
		dailyBudget = disposableIncome.Div(decimal.NewFromInt(int64(daysRemaining)))
	}

	// 7. Get CC payable summary
	ccPayable, err := s.GetCCPayable(workspaceID)
	if err != nil {
		return nil, err
	}

	return &domain.DashboardSummary{
		TotalBalance:     totalBalance,
		InHandBalance:    inHandBalance,
		DisposableIncome: disposableIncome,
		DaysRemaining:    daysRemaining,
		DailyBudget:      dailyBudget,
		CCPayable:        ccPayable,
		Month:            monthData,
	}, nil
}

// calculateDaysRemaining returns the number of days from today until the end of the month
// If viewing a past month, returns 0. If viewing a future month, returns days in that month.
func (s *DashboardService) calculateDaysRemaining(year, month int) int {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// End of the target month (last day at midnight)
	endOfMonth := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.Local)

	// Start of the target month
	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)

	// If viewing a past month, return 0
	if endOfMonth.Before(today) {
		return 0
	}

	// If viewing a future month, return total days in that month
	if startOfMonth.After(today) {
		return endOfMonth.Day()
	}

	// Current month: calculate days from today to end of month (inclusive)
	daysRemaining := int(endOfMonth.Sub(today).Hours()/24) + 1
	if daysRemaining < 1 {
		daysRemaining = 1
	}

	return daysRemaining
}

// calculateTotalBalance calculates total balance from all accounts
// Total = sum of all calculated balances
// Note: For liabilities (CC), the calculated balance is negative when debt exists,
// so we simply sum all balances to get net worth
func (s *DashboardService) calculateTotalBalance(workspaceID int32) (decimal.Decimal, error) {
	balances, err := s.calcService.CalculateAccountBalances(workspaceID)
	if err != nil {
		return decimal.Zero, err
	}

	total := decimal.Zero
	for _, balance := range balances {
		total = total.Add(balance.CalculatedBalance)
	}

	return total, nil
}

// GetCCPayable calculates the CC payable summary for a workspace
func (s *DashboardService) GetCCPayable(workspaceID int32) (*domain.CCPayableSummary, error) {
	rows, err := s.transactionRepo.GetCCPayableSummary(workspaceID)
	if err != nil {
		return nil, err
	}

	summary := &domain.CCPayableSummary{
		ThisMonth: decimal.Zero,
		NextMonth: decimal.Zero,
	}

	for _, row := range rows {
		switch row.SettlementIntent {
		case domain.CCSettlementThisMonth:
			summary.ThisMonth = row.Total
		case domain.CCSettlementNextMonth:
			summary.NextMonth = row.Total
		}
	}

	summary.Total = summary.ThisMonth.Add(summary.NextMonth)
	return summary, nil
}
