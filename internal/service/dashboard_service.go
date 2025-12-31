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

	return &domain.DashboardSummary{
		TotalBalance:  totalBalance,
		InHandBalance: inHandBalance,
		Month:         monthData,
	}, nil
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
