package domain

import "github.com/shopspring/decimal"

// DashboardSummary contains the main dashboard metrics
type DashboardSummary struct {
	TotalBalance     decimal.Decimal  `json:"totalBalance"`
	InHandBalance    decimal.Decimal  `json:"inHandBalance"`
	DisposableIncome decimal.Decimal  `json:"disposableIncome"`
	DaysRemaining    int              `json:"daysRemaining"`
	DailyBudget      decimal.Decimal  `json:"dailyBudget"`
	Month            *CalculatedMonth `json:"month"`
}
