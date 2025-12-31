package domain

import "github.com/shopspring/decimal"

// DashboardSummary contains the main dashboard metrics
type DashboardSummary struct {
	TotalBalance  decimal.Decimal  `json:"totalBalance"`
	InHandBalance decimal.Decimal  `json:"inHandBalance"`
	Month         *CalculatedMonth `json:"month"`
}
