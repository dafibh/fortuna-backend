package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// SettlementInput represents the input for settling CC transactions
type SettlementInput struct {
	TransactionIDs    []int32 `json:"transactionIds"`
	SourceAccountID   int32   `json:"sourceAccountId"`
	TargetCCAccountID int32   `json:"targetCcAccountId"`
}

// SettlementResult represents the result of a successful settlement
type SettlementResult struct {
	TransferID   int32           `json:"transferId"`
	SettledCount int             `json:"settledCount"`
	TotalAmount  decimal.Decimal `json:"totalAmount"`
	SettledAt    time.Time       `json:"settledAt"`
}
