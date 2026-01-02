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
	Paid bool `json:"paid"`
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

	payment, err := h.paymentService.TogglePaymentPaid(workspaceID, int32(loanID), int32(paymentID), req.Paid)
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
