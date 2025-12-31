package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type Month struct {
	ID              int32           `json:"id"`
	WorkspaceID     int32           `json:"workspaceId"`
	Year            int             `json:"year"`
	Month           int             `json:"month"`
	StartDate       time.Time       `json:"startDate"`
	EndDate         time.Time       `json:"endDate"`
	StartingBalance decimal.Decimal `json:"startingBalance"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

// CalculatedMonth extends Month with calculated values
type CalculatedMonth struct {
	Month
	TotalIncome    decimal.Decimal `json:"totalIncome"`
	TotalExpenses  decimal.Decimal `json:"totalExpenses"`
	ClosingBalance decimal.Decimal `json:"closingBalance"`
}

type MonthRepository interface {
	Create(workspaceID int32, year, month int, startDate, endDate time.Time, startingBalance decimal.Decimal) (*Month, error)
	GetByYearMonth(workspaceID int32, year, month int) (*Month, error)
	GetLatest(workspaceID int32) (*Month, error)
	GetAll(workspaceID int32) ([]*Month, error)
	UpdateStartingBalance(workspaceID, id int32, balance decimal.Decimal) error
}
