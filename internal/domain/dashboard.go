package domain

import "github.com/shopspring/decimal"

// CCPayableSummary contains credit card payable amounts by settlement intent
type CCPayableSummary struct {
	ThisMonth decimal.Decimal `json:"thisMonth"`
	NextMonth decimal.Decimal `json:"nextMonth"`
	Total     decimal.Decimal `json:"total"`
}

// ProjectionDetails contains projected financial data for future months
type ProjectionDetails struct {
	RecurringIncome   decimal.Decimal `json:"recurringIncome"`
	RecurringExpenses decimal.Decimal `json:"recurringExpenses"`
	LoanPayments      decimal.Decimal `json:"loanPayments"`
	Note              string          `json:"note,omitempty"`
}

// MaxProjectionMonths is the maximum number of months ahead that can be projected
const MaxProjectionMonths = 12

// DashboardSummary contains the main dashboard metrics
type DashboardSummary struct {
	IsProjection          bool               `json:"isProjection"`
	ProjectionLimitMonths int                `json:"projectionLimitMonths,omitempty"`
	TotalBalance          decimal.Decimal    `json:"totalBalance"`
	InHandBalance         decimal.Decimal    `json:"inHandBalance"`
	DisposableIncome      decimal.Decimal    `json:"disposableIncome"`
	UnpaidExpenses        decimal.Decimal    `json:"unpaidExpenses"`
	UnpaidLoanPayments    decimal.Decimal    `json:"unpaidLoanPayments"`
	DaysRemaining         int                `json:"daysRemaining"`
	DailyBudget           decimal.Decimal    `json:"dailyBudget"`
	CCPayable             *CCPayableSummary  `json:"ccPayable"`
	Month                 *CalculatedMonth   `json:"month"`
	Projection            *ProjectionDetails `json:"projection,omitempty"`
}

// FutureSpendingCategory represents spending by category for a month
type FutureSpendingCategory struct {
	ID     int32           `json:"id"`
	Name   string          `json:"name"`
	Amount decimal.Decimal `json:"amount"`
}

// FutureSpendingAccount represents spending by account for a month
type FutureSpendingAccount struct {
	ID     int32           `json:"id"`
	Name   string          `json:"name"`
	Amount decimal.Decimal `json:"amount"`
}

// FutureSpendingMonth represents spending data for a single month
type FutureSpendingMonth struct {
	Month      string                   `json:"month"` // YYYY-MM format
	Total      decimal.Decimal          `json:"total"`
	ByCategory []FutureSpendingCategory `json:"byCategory"`
	ByAccount  []FutureSpendingAccount  `json:"byAccount"`
}

// FutureSpending represents aggregated future spending data
type FutureSpending struct {
	Months []FutureSpendingMonth `json:"months"`
}
