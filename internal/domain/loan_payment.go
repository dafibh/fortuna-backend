package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

var (
	ErrLoanPaymentNotFound       = errors.New("loan payment not found")
	ErrLoanPaymentNumberInvalid  = errors.New("payment number must be at least 1")
	ErrLoanPaymentMonthInvalid   = errors.New("due month must be between 1 and 12")
	ErrLoanPaymentAmountInvalid  = errors.New("payment amount must be positive")
	ErrLoanPaymentLoanIDRequired = errors.New("loan ID is required")
)

type LoanPayment struct {
	ID            int32           `json:"id"`
	LoanID        int32           `json:"loanId"`
	PaymentNumber int32           `json:"paymentNumber"`
	Amount        decimal.Decimal `json:"amount"`
	DueYear       int32           `json:"dueYear"`
	DueMonth      int32           `json:"dueMonth"`
	Paid          bool            `json:"paid"`
	PaidDate      *time.Time      `json:"paidDate,omitempty"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

func (lp *LoanPayment) Validate() error {
	if lp.LoanID <= 0 {
		return ErrLoanPaymentLoanIDRequired
	}
	if lp.PaymentNumber < 1 {
		return ErrLoanPaymentNumberInvalid
	}
	if lp.DueMonth < 1 || lp.DueMonth > 12 {
		return ErrLoanPaymentMonthInvalid
	}
	if lp.Amount.LessThanOrEqual(decimal.Zero) {
		return ErrLoanPaymentAmountInvalid
	}
	return nil
}

// FormatPaymentLabel returns a formatted label like "1/6" for payment 1 of 6
func (lp *LoanPayment) FormatPaymentLabel(totalPayments int32) string {
	return fmt.Sprintf("%d/%d", lp.PaymentNumber, totalPayments)
}

// LoanDeleteStats contains payment statistics for delete confirmation
type LoanDeleteStats struct {
	TotalCount  int32           `json:"totalCount"`
	PaidCount   int32           `json:"paidCount"`
	UnpaidCount int32           `json:"unpaidCount"`
	TotalAmount decimal.Decimal `json:"totalAmount"`
}

// MonthlyPaymentDetail contains payment details with loan info for monthly aggregation
type MonthlyPaymentDetail struct {
	ID            int32           `json:"id"`
	LoanID        int32           `json:"loanId"`
	ItemName      string          `json:"itemName"`
	PaymentNumber int32           `json:"paymentNumber"`
	TotalPayments int32           `json:"totalPayments"`
	Amount        decimal.Decimal `json:"amount"`
	Paid          bool            `json:"paid"`
}

type LoanPaymentRepository interface {
	Create(payment *LoanPayment) (*LoanPayment, error)
	CreateBatch(payments []*LoanPayment) error
	CreateBatchTx(tx interface{}, payments []*LoanPayment) error // Transactional batch create
	GetByID(id int32) (*LoanPayment, error)
	GetByLoanID(loanID int32) ([]*LoanPayment, error)
	GetByLoanAndNumber(loanID int32, paymentNumber int32) (*LoanPayment, error)
	UpdateAmount(id int32, amount decimal.Decimal) (*LoanPayment, error)
	TogglePaid(id int32, paid bool, paidDate *time.Time) (*LoanPayment, error)
	GetByMonth(workspaceID int32, year, month int) ([]*LoanPayment, error)
	GetUnpaidByMonth(workspaceID int32, year, month int) ([]*LoanPayment, error)
	GetDeleteStats(loanID int32) (*LoanDeleteStats, error)
	GetPaymentsWithDetailsByMonth(workspaceID int32, year, month int) ([]*MonthlyPaymentDetail, error)
	SumUnpaidByMonth(workspaceID int32, year, month int) (decimal.Decimal, error)
}
