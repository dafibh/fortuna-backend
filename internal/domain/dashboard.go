package domain

import "github.com/shopspring/decimal"

// CCPayableSummary contains credit card payable amounts by settlement intent
type CCPayableSummary struct {
	ThisMonth decimal.Decimal `json:"thisMonth"`
	NextMonth decimal.Decimal `json:"nextMonth"`
	Total     decimal.Decimal `json:"total"`
}

// FutureSpendingData contains aggregated spending data for future months
type FutureSpendingData struct {
	Months []MonthSpendingResponse `json:"months"`
}

// MonthSpending holds intermediate aggregation data during calculation
type MonthSpending struct {
	Month      string
	Total      decimal.Decimal
	ByCategory map[int32]decimal.Decimal
	ByAccount  map[int32]decimal.Decimal
}

// MonthSpendingResponse is the JSON response format for a single month
type MonthSpendingResponse struct {
	Month      string           `json:"month"`
	Total      string           `json:"total"`
	ByCategory []CategoryAmount `json:"byCategory"`
	ByAccount  []AccountAmount  `json:"byAccount"`
}

// CategoryAmount represents spending amount for a category
type CategoryAmount struct {
	ID     int32  `json:"id"`
	Name   string `json:"name"`
	Amount string `json:"amount"`
}

// AccountAmount represents spending amount for an account
type AccountAmount struct {
	ID     int32  `json:"id"`
	Name   string `json:"name"`
	Amount string `json:"amount"`
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
