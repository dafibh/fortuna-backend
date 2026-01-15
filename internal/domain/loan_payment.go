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
	ErrProviderNotConsolidated   = errors.New("provider does not use consolidated monthly payment mode")
	ErrPaymentIDsInvalid         = errors.New("one or more payment IDs are invalid or do not belong to the specified month")
	ErrNoUnpaidMonths            = errors.New("no unpaid months found for this provider")
)

// ErrMustPayEarlierMonth indicates sequential enforcement violation
type ErrMustPayEarlierMonth struct {
	Expected  string // The earliest unpaid month that should be paid first
	Requested string // The month the user tried to pay
}

func (e ErrMustPayEarlierMonth) Error() string {
	return fmt.Sprintf("must pay %s before %s (sequential enforcement)", e.Expected, e.Requested)
}

// ErrCannotSkipMonth indicates a gap in the consecutive month sequence
type ErrCannotSkipMonth struct {
	Skipped string // The month that was skipped
}

func (e ErrCannotSkipMonth) Error() string {
	return fmt.Sprintf("cannot skip %s. All months must be consecutive", e.Skipped)
}

// ErrEndMonthBeforeStart indicates the end month is before or equal to start month
var ErrEndMonthBeforeStart = errors.New("end month must be after start month")

// ErrCannotUnpayEarlierMonth indicates reverse sequential enforcement violation
type ErrCannotUnpayEarlierMonth struct {
	Latest    string // The latest paid month that can be unpaid
	Requested string // The month the user tried to unpay
}

func (e ErrCannotUnpayEarlierMonth) Error() string {
	return fmt.Sprintf("cannot unpay %s while later months are paid. Unpay %s first", e.Requested, e.Latest)
}

// ErrNoPaidMonths indicates there are no paid months to unpay
var ErrNoPaidMonths = errors.New("no paid months found for this provider")

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

// PayMonthResult contains the result of a batch pay month operation
type PayMonthResult struct {
	Month            string          `json:"month"`            // Format: "YYYY-MM"
	PaidCount        int             `json:"paidCount"`        // Number of payments marked paid
	TotalAmount      decimal.Decimal `json:"totalAmount"`      // Sum of all payment amounts
	PaidAt           time.Time       `json:"paidAt"`           // Timestamp when marked paid
	NextPayableMonth *string         `json:"nextPayableMonth"` // Next month that can be paid (nil if none)
}

// PayRangeResult contains the result of a multi-month batch pay operation
type PayRangeResult struct {
	MonthsPaid       []string        `json:"monthsPaid"`       // List of months paid (e.g., ["2026-02", "2026-03"])
	PaidCount        int             `json:"paidCount"`        // Total number of payments marked paid
	TotalAmount      decimal.Decimal `json:"totalAmount"`      // Sum of all payment amounts
	PaidAt           time.Time       `json:"paidAt"`           // Timestamp when marked paid
	NextPayableMonth *string         `json:"nextPayableMonth"` // Next month that can be paid (nil if none)
}

// UnpayMonthResult contains the result of an unpay month operation
type UnpayMonthResult struct {
	Month           string          `json:"month"`           // Format: "YYYY-MM"
	UnpaidCount     int             `json:"unpaidCount"`     // Number of payments marked unpaid
	TotalAmount     decimal.Decimal `json:"totalAmount"`     // Sum of all payment amounts
	PreviousPayable *string         `json:"previousPayable"` // The month that is now payable again
}

// LatestPaidMonth represents the latest paid month for a provider
type LatestPaidMonth struct {
	Year  int32
	Month int32
}

// EarliestUnpaidMonth represents the earliest unpaid month for a provider
type EarliestUnpaidMonth struct {
	Year  int32
	Month int32
}

type LoanPaymentRepository interface {
	Create(payment *LoanPayment) (*LoanPayment, error)
	CreateBatch(payments []*LoanPayment) error
	CreateBatchTx(tx any, payments []*LoanPayment) error // Transactional batch create
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

	// Consolidated payment methods
	GetEarliestUnpaidMonth(workspaceID int32, providerID int32) (*EarliestUnpaidMonth, error)
	GetUnpaidPaymentsByProviderMonth(workspaceID int32, providerID int32, year int32, month int32) ([]*LoanPayment, error)
	BatchUpdatePaidTx(tx any, paymentIDs []int32, workspaceID int32) (int, decimal.Decimal, error)

	// Consolidated unpay methods
	GetLatestPaidMonth(workspaceID int32, providerID int32) (*LatestPaidMonth, error)
	GetPaidPaymentsByProviderMonth(workspaceID int32, providerID int32, year int32, month int32) ([]*LoanPayment, error)
	BatchUpdateUnpaidTx(tx any, paymentIDs []int32) (int, error)

	// Trend aggregation methods
	GetTrendRaw(workspaceID int32, startYear int32, startMonth int32) ([]*TrendRawRow, error)
}
