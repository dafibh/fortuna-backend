package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/middleware"
	"github.com/dafibh/fortuna/fortuna-backend/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

// LoanPaymentHandler handles loan payment-related HTTP requests
type LoanPaymentHandler struct {
	paymentService *service.LoanPaymentService
}

// NewLoanPaymentHandler creates a new LoanPaymentHandler
func NewLoanPaymentHandler(paymentService *service.LoanPaymentService) *LoanPaymentHandler {
	return &LoanPaymentHandler{paymentService: paymentService}
}

// LoanPaymentResponse represents a loan payment in API responses
type LoanPaymentResponse struct {
	ID            int32   `json:"id"`
	LoanID        int32   `json:"loanId"`
	PaymentNumber int32   `json:"paymentNumber"`
	Amount        string  `json:"amount"`
	DueYear       int32   `json:"dueYear"`
	DueMonth      int32   `json:"dueMonth"`
	Paid          bool    `json:"paid"`
	PaidDate      *string `json:"paidDate,omitempty"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
}

// UpdatePaymentAmountRequest represents the update payment amount request body
type UpdatePaymentAmountRequest struct {
	Amount string `json:"amount"`
}

// TogglePaymentPaidRequest represents the toggle payment paid request body
type TogglePaymentPaidRequest struct {
	Paid     bool    `json:"paid"`
	PaidDate *string `json:"paidDate,omitempty"` // Optional: YYYY-MM-DD format, defaults to today when marking paid
}

// PayRangeRequest represents the pay-range request body for multi-month payment
type PayRangeRequest struct {
	StartMonth string  `json:"startMonth"` // Format: YYYY-MM
	EndMonth   string  `json:"endMonth"`   // Format: YYYY-MM
	PaymentIDs []int32 `json:"paymentIds"`
}

// PayRangeResponse represents the pay-range response
type PayRangeResponse struct {
	MonthsPaid       []string `json:"monthsPaid"`
	PaidCount        int      `json:"paidCount"`
	TotalAmount      string   `json:"totalAmount"`
	PaidAt           string   `json:"paidAt"`
	NextPayableMonth *string  `json:"nextPayableMonth,omitempty"`
}

// PayMonthRequest represents the pay-month request body for single month payment
type PayMonthRequest struct {
	Month      string  `json:"month"`      // Format: YYYY-MM
	PaymentIDs []int32 `json:"paymentIds"`
}

// PayMonthResponse represents the pay-month response
type PayMonthResponse struct {
	Month            string  `json:"month"`
	PaidCount        int     `json:"paidCount"`
	TotalAmount      string  `json:"totalAmount"`
	PaidAt           string  `json:"paidAt"`
	NextPayableMonth *string `json:"nextPayableMonth,omitempty"`
}

// UnpayMonthRequest represents the unpay-month request body
type UnpayMonthRequest struct {
	Month string `json:"month"` // Format: YYYY-MM
}

// UnpayMonthResponse represents the unpay-month response
type UnpayMonthResponse struct {
	Month           string  `json:"month"`
	UnpaidCount     int     `json:"unpaidCount"`
	TotalAmount     string  `json:"totalAmount"`
	PreviousPayable *string `json:"previousPayable,omitempty"`
}

// EarliestUnpaidMonthResponse represents the earliest unpaid month response
type EarliestUnpaidMonthResponse struct {
	Year  int32 `json:"year"`
	Month int32 `json:"month"`
}

// GetPaymentsByLoanID handles GET /api/v1/loans/:loanId/payments
func (h *LoanPaymentHandler) GetPaymentsByLoanID(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	loanID, err := strconv.Atoi(c.Param("loanId"))
	if err != nil {
		return NewValidationError(c, "Invalid loan ID", nil)
	}

	payments, err := h.paymentService.GetPaymentsByLoanID(workspaceID, int32(loanID))
	if err != nil {
		if errors.Is(err, domain.ErrLoanNotFound) {
			return NewNotFoundError(c, "Loan not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("loan_id", loanID).Msg("Failed to get loan payments")
		return NewInternalError(c, "Failed to get loan payments")
	}

	response := make([]LoanPaymentResponse, len(payments))
	for i, payment := range payments {
		response[i] = toLoanPaymentResponse(payment)
	}

	return c.JSON(http.StatusOK, response)
}

// UpdatePaymentAmount handles PATCH /api/v1/loans/:loanId/payments/:paymentId
func (h *LoanPaymentHandler) UpdatePaymentAmount(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	loanID, err := strconv.Atoi(c.Param("loanId"))
	if err != nil {
		return NewValidationError(c, "Invalid loan ID", nil)
	}

	paymentID, err := strconv.Atoi(c.Param("paymentId"))
	if err != nil {
		return NewValidationError(c, "Invalid payment ID", nil)
	}

	var req UpdatePaymentAmountRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return NewValidationError(c, "Invalid amount", []ValidationError{
			{Field: "amount", Message: "Must be a valid decimal number"},
		})
	}

	payment, err := h.paymentService.UpdatePaymentAmount(workspaceID, int32(loanID), int32(paymentID), amount)
	if err != nil {
		if errors.Is(err, domain.ErrLoanNotFound) {
			return NewNotFoundError(c, "Loan not found")
		}
		if errors.Is(err, domain.ErrLoanPaymentNotFound) {
			return NewNotFoundError(c, "Payment not found")
		}
		if errors.Is(err, domain.ErrLoanPaymentAmountInvalid) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "amount", Message: "Amount must be positive"},
			})
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("loan_id", loanID).Int("payment_id", paymentID).Msg("Failed to update payment amount")
		return NewInternalError(c, "Failed to update payment amount")
	}

	log.Info().Int32("workspace_id", workspaceID).Int("loan_id", loanID).Int("payment_id", paymentID).Msg("Payment amount updated")

	return c.JSON(http.StatusOK, toLoanPaymentResponse(payment))
}

// TogglePaymentPaid handles PUT /api/v1/loans/:loanId/payments/:paymentId/toggle-paid
func (h *LoanPaymentHandler) TogglePaymentPaid(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	loanID, err := strconv.Atoi(c.Param("loanId"))
	if err != nil {
		return NewValidationError(c, "Invalid loan ID", nil)
	}

	paymentID, err := strconv.Atoi(c.Param("paymentId"))
	if err != nil {
		return NewValidationError(c, "Invalid payment ID", nil)
	}

	var req TogglePaymentPaidRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	// Parse optional paid date
	var paidDate *time.Time
	if req.Paid && req.PaidDate != nil && *req.PaidDate != "" {
		parsed, err := time.Parse("2006-01-02", *req.PaidDate)
		if err != nil {
			return NewValidationError(c, "Invalid paid date", []ValidationError{
				{Field: "paidDate", Message: "Must be in YYYY-MM-DD format"},
			})
		}
		paidDate = &parsed
	}

	payment, err := h.paymentService.TogglePaymentPaid(workspaceID, int32(loanID), int32(paymentID), req.Paid, paidDate)
	if err != nil {
		if errors.Is(err, domain.ErrLoanNotFound) {
			return NewNotFoundError(c, "Loan not found")
		}
		if errors.Is(err, domain.ErrLoanPaymentNotFound) {
			return NewNotFoundError(c, "Payment not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("loan_id", loanID).Int("payment_id", paymentID).Msg("Failed to toggle payment paid status")
		return NewInternalError(c, "Failed to toggle payment paid status")
	}

	log.Info().Int32("workspace_id", workspaceID).Int("loan_id", loanID).Int("payment_id", paymentID).Bool("paid", req.Paid).Msg("Payment paid status toggled")

	return c.JSON(http.StatusOK, toLoanPaymentResponse(payment))
}

// PayRange handles POST /api/v1/loan-providers/:id/pay-range
func (h *LoanPaymentHandler) PayRange(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	providerID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid loan provider ID", nil)
	}

	var req PayRangeRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	// Validate required fields
	if req.StartMonth == "" {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "startMonth", Message: "Start month is required"},
		})
	}
	if req.EndMonth == "" {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "endMonth", Message: "End month is required"},
		})
	}
	if len(req.PaymentIDs) == 0 {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "paymentIds", Message: "At least one payment ID is required"},
		})
	}

	result, err := h.paymentService.PayRange(c.Request().Context(), workspaceID, int32(providerID), req.StartMonth, req.EndMonth, req.PaymentIDs)
	if err != nil {
		if errors.Is(err, domain.ErrLoanProviderNotFound) {
			return NewNotFoundError(c, "Loan provider not found")
		}
		if errors.Is(err, domain.ErrProviderNotConsolidated) {
			return NewValidationError(c, "Provider does not use consolidated monthly payment mode", nil)
		}
		if errors.Is(err, domain.ErrNoUnpaidMonths) {
			return NewValidationError(c, "No unpaid months found for this provider", nil)
		}
		if errors.Is(err, domain.ErrEndMonthBeforeStart) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "endMonth", Message: "End month must be after start month"},
			})
		}
		if errors.Is(err, domain.ErrPaymentIDsInvalid) {
			return NewValidationError(c, "One or more payment IDs are invalid or do not belong to the specified month range", nil)
		}

		// Check for ErrMustPayEarlierMonth
		var mustPayErr domain.ErrMustPayEarlierMonth
		if errors.As(err, &mustPayErr) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "startMonth", Message: mustPayErr.Error()},
			})
		}

		// Check for ErrCannotSkipMonth
		var skipErr domain.ErrCannotSkipMonth
		if errors.As(err, &skipErr) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "paymentIds", Message: skipErr.Error()},
			})
		}

		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("provider_id", providerID).Msg("Failed to pay range")
		return NewInternalError(c, "Failed to pay range")
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Int("provider_id", providerID).
		Strs("months_paid", result.MonthsPaid).
		Int("paid_count", result.PaidCount).
		Msg("Multi-month payment completed")

	response := PayRangeResponse{
		MonthsPaid:       result.MonthsPaid,
		PaidCount:        result.PaidCount,
		TotalAmount:      result.TotalAmount.StringFixed(2),
		PaidAt:           result.PaidAt.Format(time.RFC3339),
		NextPayableMonth: result.NextPayableMonth,
	}

	return c.JSON(http.StatusOK, response)
}

// PayMonth handles POST /api/v1/loan-providers/:id/pay-month
func (h *LoanPaymentHandler) PayMonth(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	providerID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid loan provider ID", nil)
	}

	var req PayMonthRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	// Validate required fields
	if req.Month == "" {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "month", Message: "Month is required"},
		})
	}
	if len(req.PaymentIDs) == 0 {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "paymentIds", Message: "At least one payment ID is required"},
		})
	}

	result, err := h.paymentService.PayMonth(c.Request().Context(), workspaceID, int32(providerID), req.Month, req.PaymentIDs)
	if err != nil {
		if errors.Is(err, domain.ErrLoanProviderNotFound) {
			return NewNotFoundError(c, "Loan provider not found")
		}
		if errors.Is(err, domain.ErrProviderNotConsolidated) {
			return NewValidationError(c, "Provider does not use consolidated monthly payment mode", nil)
		}
		if errors.Is(err, domain.ErrNoUnpaidMonths) {
			return NewValidationError(c, "No unpaid months found for this provider", nil)
		}
		if errors.Is(err, domain.ErrPaymentIDsInvalid) {
			return NewValidationError(c, "One or more payment IDs are invalid or do not belong to the specified month", nil)
		}

		// Check for ErrMustPayEarlierMonth
		var mustPayErr domain.ErrMustPayEarlierMonth
		if errors.As(err, &mustPayErr) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "month", Message: mustPayErr.Error()},
			})
		}

		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("provider_id", providerID).Str("month", req.Month).Msg("Failed to pay month")
		return NewInternalError(c, "Failed to pay month")
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Int("provider_id", providerID).
		Str("month", result.Month).
		Int("paid_count", result.PaidCount).
		Msg("Single-month payment completed")

	response := PayMonthResponse{
		Month:            result.Month,
		PaidCount:        result.PaidCount,
		TotalAmount:      result.TotalAmount.StringFixed(2),
		PaidAt:           result.PaidAt.Format(time.RFC3339),
		NextPayableMonth: result.NextPayableMonth,
	}

	return c.JSON(http.StatusOK, response)
}

// UnpayMonth handles POST /api/v1/loan-providers/:id/unpay-month
func (h *LoanPaymentHandler) UnpayMonth(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	providerID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid loan provider ID", nil)
	}

	var req UnpayMonthRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	// Validate required fields
	if req.Month == "" {
		return NewValidationError(c, "Validation failed", []ValidationError{
			{Field: "month", Message: "Month is required"},
		})
	}

	result, err := h.paymentService.UnpayMonth(c.Request().Context(), workspaceID, int32(providerID), req.Month)
	if err != nil {
		if errors.Is(err, domain.ErrLoanProviderNotFound) {
			return NewNotFoundError(c, "Loan provider not found")
		}
		if errors.Is(err, domain.ErrProviderNotConsolidated) {
			return NewValidationError(c, "Provider does not use consolidated monthly payment mode", nil)
		}
		if errors.Is(err, domain.ErrNoPaidMonths) {
			return NewValidationError(c, "No paid months found for this provider", nil)
		}

		// Check for ErrCannotUnpayEarlierMonth
		var cannotUnpayErr domain.ErrCannotUnpayEarlierMonth
		if errors.As(err, &cannotUnpayErr) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "month", Message: cannotUnpayErr.Error()},
			})
		}

		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("provider_id", providerID).Str("month", req.Month).Msg("Failed to unpay month")
		return NewInternalError(c, "Failed to unpay month")
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Int("provider_id", providerID).
		Str("month", result.Month).
		Int("unpaid_count", result.UnpaidCount).
		Msg("Single-month unpay completed")

	response := UnpayMonthResponse{
		Month:           result.Month,
		UnpaidCount:     result.UnpaidCount,
		TotalAmount:     result.TotalAmount.StringFixed(2),
		PreviousPayable: result.PreviousPayable,
	}

	return c.JSON(http.StatusOK, response)
}

// GetEarliestUnpaidMonth handles GET /api/v1/loan-providers/:id/earliest-unpaid
func (h *LoanPaymentHandler) GetEarliestUnpaidMonth(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	providerID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid loan provider ID", nil)
	}

	result, err := h.paymentService.GetEarliestUnpaidMonth(workspaceID, int32(providerID))
	if err != nil {
		if errors.Is(err, domain.ErrLoanProviderNotFound) {
			return NewNotFoundError(c, "Loan provider not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("provider_id", providerID).Msg("Failed to get earliest unpaid month")
		return NewInternalError(c, "Failed to get earliest unpaid month")
	}

	// Return null if no unpaid months
	if result == nil {
		return c.JSON(http.StatusOK, nil)
	}

	response := EarliestUnpaidMonthResponse{
		Year:  result.Year,
		Month: result.Month,
	}

	return c.JSON(http.StatusOK, response)
}

// Helper function to convert domain.LoanPayment to LoanPaymentResponse
func toLoanPaymentResponse(payment *domain.LoanPayment) LoanPaymentResponse {
	resp := LoanPaymentResponse{
		ID:            payment.ID,
		LoanID:        payment.LoanID,
		PaymentNumber: payment.PaymentNumber,
		Amount:        payment.Amount.StringFixed(2),
		DueYear:       payment.DueYear,
		DueMonth:      payment.DueMonth,
		Paid:          payment.Paid,
		CreatedAt:     payment.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     payment.UpdatedAt.Format(time.RFC3339),
	}
	if payment.PaidDate != nil {
		paidDate := payment.PaidDate.Format("2006-01-02")
		resp.PaidDate = &paidDate
	}
	return resp
}
