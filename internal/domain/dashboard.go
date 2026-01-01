package domain

import "github.com/shopspring/decimal"

// CCPayableSummary contains credit card payable amounts by settlement intent
type CCPayableSummary struct {
	ThisMonth decimal.Decimal `json:"thisMonth"`
	NextMonth decimal.Decimal `json:"nextMonth"`
	Total     decimal.Decimal `json:"total"`
}

// DashboardSummary contains the main dashboard metrics
type DashboardSummary struct {
	TotalBalance     decimal.Decimal   `json:"totalBalance"`
	InHandBalance    decimal.Decimal   `json:"inHandBalance"`
	DisposableIncome decimal.Decimal   `json:"disposableIncome"`
	DaysRemaining    int               `json:"daysRemaining"`
	DailyBudget      decimal.Decimal   `json:"dailyBudget"`
	CCPayable        *CCPayableSummary `json:"ccPayable"`
	Month            *CalculatedMonth  `json:"month"`
}
