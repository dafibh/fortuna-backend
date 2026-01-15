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

// LoanHandler handles loan-related HTTP requests
type LoanHandler struct {
	loanService *service.LoanService
}

// NewLoanHandler creates a new LoanHandler
func NewLoanHandler(loanService *service.LoanService) *LoanHandler {
	return &LoanHandler{loanService: loanService}
}

// UpdateLoanRequest represents the update loan request body
// Only itemName and notes are editable; other fields are locked after creation
type UpdateLoanRequest struct {
	ItemName string  `json:"itemName"`
	Notes    *string `json:"notes,omitempty"`
}

// CreateLoanRequest represents the create loan request body
type CreateLoanRequest struct {
	ProviderID     int32    `json:"providerId"`
	ItemName       string   `json:"itemName"`
	TotalAmount    string   `json:"totalAmount"`
	NumMonths      int32    `json:"numMonths"`
	PurchaseDate   string   `json:"purchaseDate"`
	InterestRate   *string  `json:"interestRate,omitempty"`
	Notes          *string  `json:"notes,omitempty"`
	PaymentAmounts []string `json:"paymentAmounts,omitempty"` // Optional custom amounts for each payment
}

// PreviewLoanRequest represents the preview loan request body
type PreviewLoanRequest struct {
	ProviderID   int32   `json:"providerId"`
	TotalAmount  string  `json:"totalAmount"`
	NumMonths    int32   `json:"numMonths"`
	PurchaseDate string  `json:"purchaseDate"`
	InterestRate *string `json:"interestRate,omitempty"`
}

// LoanResponse represents a loan in API responses
type LoanResponse struct {
	ID                int32   `json:"id"`
	WorkspaceID       int32   `json:"workspaceId"`
	ProviderID        int32   `json:"providerId"`
	ItemName          string  `json:"itemName"`
	TotalAmount       string  `json:"totalAmount"`
	NumMonths         int32   `json:"numMonths"`
	PurchaseDate      string  `json:"purchaseDate"`
	InterestRate      string  `json:"interestRate"`
	MonthlyPayment    string  `json:"monthlyPayment"`
	FirstPaymentYear  int32   `json:"firstPaymentYear"`
	FirstPaymentMonth int32   `json:"firstPaymentMonth"`
	LastPaymentYear   int     `json:"lastPaymentYear"`
	LastPaymentMonth  int     `json:"lastPaymentMonth"`
	Notes             *string `json:"notes,omitempty"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
	DeletedAt         *string `json:"deletedAt,omitempty"`
}

// PreviewLoanResponse represents the preview loan calculation result
type PreviewLoanResponse struct {
	MonthlyPayment    string `json:"monthlyPayment"`
	FirstPaymentYear  int    `json:"firstPaymentYear"`
	FirstPaymentMonth int    `json:"firstPaymentMonth"`
	InterestRate      string `json:"interestRate"`
}

// LoanWithStatsResponse represents a loan with payment statistics in API responses
type LoanWithStatsResponse struct {
	ID                int32   `json:"id"`
	WorkspaceID       int32   `json:"workspaceId"`
	ProviderID        int32   `json:"providerId"`
	ItemName          string  `json:"itemName"`
	TotalAmount       string  `json:"totalAmount"`
	NumMonths         int32   `json:"numMonths"`
	PurchaseDate      string  `json:"purchaseDate"`
	InterestRate      string  `json:"interestRate"`
	MonthlyPayment    string  `json:"monthlyPayment"`
	FirstPaymentYear  int32   `json:"firstPaymentYear"`
	FirstPaymentMonth int32   `json:"firstPaymentMonth"`
	LastPaymentYear   int32   `json:"lastPaymentYear"`
	LastPaymentMonth  int32   `json:"lastPaymentMonth"`
	Notes             *string `json:"notes,omitempty"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
	DeletedAt         *string `json:"deletedAt,omitempty"`
	// Stats fields
	TotalCount       int32   `json:"totalCount"`
	PaidCount        int32   `json:"paidCount"`
	RemainingBalance string  `json:"remainingBalance"`
	Progress         float64 `json:"progress"`
}

// CreateLoan handles POST /api/v1/loans
func (h *LoanHandler) CreateLoan(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	var req CreateLoanRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	// Parse total amount
	totalAmount, err := decimal.NewFromString(req.TotalAmount)
	if err != nil {
		return NewValidationError(c, "Invalid total amount", []ValidationError{
			{Field: "totalAmount", Message: "Must be a valid decimal number"},
		})
	}

	// Parse purchase date
	purchaseDate, err := time.Parse("2006-01-02", req.PurchaseDate)
	if err != nil {
		return NewValidationError(c, "Invalid purchase date", []ValidationError{
			{Field: "purchaseDate", Message: "Must be in YYYY-MM-DD format"},
		})
	}

	// Parse optional interest rate override
	var interestRate *decimal.Decimal
	if req.InterestRate != nil && *req.InterestRate != "" {
		rate, err := decimal.NewFromString(*req.InterestRate)
		if err != nil {
			return NewValidationError(c, "Invalid interest rate", []ValidationError{
				{Field: "interestRate", Message: "Must be a valid decimal number"},
			})
		}
		interestRate = &rate
	}

	// Parse optional custom payment amounts
	var paymentAmounts []decimal.Decimal
	if len(req.PaymentAmounts) > 0 {
		if len(req.PaymentAmounts) != int(req.NumMonths) {
			return NewValidationError(c, "Invalid payment amounts", []ValidationError{
				{Field: "paymentAmounts", Message: "Must have exactly numMonths amounts"},
			})
		}
		paymentAmounts = make([]decimal.Decimal, len(req.PaymentAmounts))
		for i, amtStr := range req.PaymentAmounts {
			amt, err := decimal.NewFromString(amtStr)
			if err != nil {
				return NewValidationError(c, "Invalid payment amount", []ValidationError{
					{Field: "paymentAmounts", Message: "All amounts must be valid decimal numbers"},
				})
			}
			if amt.LessThanOrEqual(decimal.Zero) {
				return NewValidationError(c, "Invalid payment amount", []ValidationError{
					{Field: "paymentAmounts", Message: "All amounts must be positive"},
				})
			}
			paymentAmounts[i] = amt
		}
	}

	input := service.CreateLoanInput{
		ProviderID:     req.ProviderID,
		ItemName:       req.ItemName,
		TotalAmount:    totalAmount,
		NumMonths:      req.NumMonths,
		PurchaseDate:   purchaseDate,
		InterestRate:   interestRate,
		Notes:          req.Notes,
		PaymentAmounts: paymentAmounts,
	}

	loan, err := h.loanService.CreateLoan(workspaceID, input)
	if err != nil {
		if errors.Is(err, domain.ErrLoanItemNameEmpty) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "itemName", Message: "Item name is required"},
			})
		}
		if errors.Is(err, domain.ErrLoanItemNameTooLong) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "itemName", Message: "Item name must be 200 characters or less"},
			})
		}
		if errors.Is(err, domain.ErrLoanAmountInvalid) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "totalAmount", Message: "Amount must be positive"},
			})
		}
		if errors.Is(err, domain.ErrLoanMonthsInvalid) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "numMonths", Message: "Number of months must be at least 1"},
			})
		}
		if errors.Is(err, domain.ErrLoanProviderInvalid) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "providerId", Message: "Invalid loan provider"},
			})
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to create loan")
		return NewInternalError(c, "Failed to create loan")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("loan_id", loan.ID).Str("item", loan.ItemName).Msg("Loan created")

	return c.JSON(http.StatusCreated, toLoanResponse(loan))
}

// GetLoans godoc
// @Summary List loans
// @Description Get all loans/installments for the authenticated workspace
// @Tags loans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status: active, completed, all" default(all)
// @Success 200 {array} LoanWithStatsResponse
// @Failure 401 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /loans [get]
func (h *LoanHandler) GetLoans(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	// Parse status filter (defaults to "all")
	statusParam := c.QueryParam("status")
	var filter domain.LoanFilter
	switch statusParam {
	case "active":
		filter = domain.LoanFilterActive
	case "completed":
		filter = domain.LoanFilterCompleted
	case "all", "":
		filter = domain.LoanFilterAll
	default:
		return NewValidationError(c, "Invalid status parameter", []ValidationError{
			{Field: "status", Message: "Must be 'all', 'active', or 'completed'"},
		})
	}

	loans, err := h.loanService.GetLoansWithStats(workspaceID, filter)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get loans")
		return NewInternalError(c, "Failed to get loans")
	}

	response := make([]LoanWithStatsResponse, len(loans))
	for i, loan := range loans {
		response[i] = toLoanWithStatsResponse(loan)
	}

	return c.JSON(http.StatusOK, response)
}

// GetLoan handles GET /api/v1/loans/:id
func (h *LoanHandler) GetLoan(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid loan ID", nil)
	}

	loan, err := h.loanService.GetLoanByID(workspaceID, int32(id))
	if err != nil {
		if errors.Is(err, domain.ErrLoanNotFound) {
			return NewNotFoundError(c, "Loan not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("loan_id", id).Msg("Failed to get loan")
		return NewInternalError(c, "Failed to get loan")
	}

	return c.JSON(http.StatusOK, toLoanResponse(loan))
}

// UpdateLoan handles PUT /api/v1/loans/:id
// Only updates editable fields (itemName, notes); amount/months/dates are locked
func (h *LoanHandler) UpdateLoan(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid loan ID", nil)
	}

	var req UpdateLoanRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	input := service.UpdateLoanInput{
		ItemName: req.ItemName,
		Notes:    req.Notes,
	}

	loan, err := h.loanService.UpdateLoan(workspaceID, int32(id), input)
	if err != nil {
		if errors.Is(err, domain.ErrLoanNotFound) {
			return NewNotFoundError(c, "Loan not found")
		}
		if errors.Is(err, domain.ErrLoanItemNameEmpty) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "itemName", Message: "Item name is required"},
			})
		}
		if errors.Is(err, domain.ErrLoanItemNameTooLong) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "itemName", Message: "Item name must be 200 characters or less"},
			})
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("loan_id", id).Msg("Failed to update loan")
		return NewInternalError(c, "Failed to update loan")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("loan_id", loan.ID).Str("item", loan.ItemName).Msg("Loan updated")

	return c.JSON(http.StatusOK, toLoanResponse(loan))
}

// DeleteCheckResponse represents the response for delete check endpoint
type DeleteCheckResponse struct {
	LoanID      int32  `json:"loanId"`
	ItemName    string `json:"itemName"`
	PaidCount   int32  `json:"paidCount"`
	UnpaidCount int32  `json:"unpaidCount"`
	TotalAmount string `json:"totalAmount"`
}

// GetDeleteCheck handles GET /api/v1/loans/:id/delete-check
// Returns payment statistics for delete confirmation dialog
func (h *LoanHandler) GetDeleteCheck(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid loan ID", nil)
	}

	loan, stats, err := h.loanService.GetDeleteStats(workspaceID, int32(id))
	if err != nil {
		if errors.Is(err, domain.ErrLoanNotFound) {
			return NewNotFoundError(c, "Loan not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("loan_id", id).Msg("Failed to get delete check stats")
		return NewInternalError(c, "Failed to get delete check stats")
	}

	return c.JSON(http.StatusOK, DeleteCheckResponse{
		LoanID:      loan.ID,
		ItemName:    loan.ItemName,
		PaidCount:   stats.PaidCount,
		UnpaidCount: stats.UnpaidCount,
		TotalAmount: stats.TotalAmount.StringFixed(2),
	})
}

// DeleteLoan handles DELETE /api/v1/loans/:id
func (h *LoanHandler) DeleteLoan(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid loan ID", nil)
	}

	if err := h.loanService.DeleteLoan(workspaceID, int32(id)); err != nil {
		if errors.Is(err, domain.ErrLoanNotFound) {
			return NewNotFoundError(c, "Loan not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("loan_id", id).Msg("Failed to delete loan")
		return NewInternalError(c, "Failed to delete loan")
	}

	log.Info().Int32("workspace_id", workspaceID).Int("loan_id", id).Msg("Loan deleted (soft)")
	return c.NoContent(http.StatusNoContent)
}

// CommitmentsResponse represents the monthly loan commitments aggregation
type CommitmentsResponse struct {
	Year        int                 `json:"year"`
	Month       int                 `json:"month"`
	TotalUnpaid string              `json:"totalUnpaid"`
	TotalPaid   string              `json:"totalPaid"`
	Payments    []CommitmentPayment `json:"payments"`
}

// CommitmentPayment represents a single payment in the monthly commitments
type CommitmentPayment struct {
	LoanID        int32  `json:"loanId"`
	ItemName      string `json:"itemName"`
	PaymentNumber int32  `json:"paymentNumber"`
	TotalPayments int32  `json:"totalPayments"`
	Amount        string `json:"amount"`
	Paid          bool   `json:"paid"`
}

// GetMonthlyCommitments handles GET /api/v1/loans/commitments/:year/:month
func (h *LoanHandler) GetMonthlyCommitments(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	year, err := strconv.Atoi(c.Param("year"))
	if err != nil || year < 2000 || year > 2100 {
		return NewValidationError(c, "Invalid year", nil)
	}

	month, err := strconv.Atoi(c.Param("month"))
	if err != nil || month < 1 || month > 12 {
		return NewValidationError(c, "Invalid month", nil)
	}

	result, err := h.loanService.GetMonthlyCommitments(workspaceID, year, month)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("year", year).Int("month", month).Msg("Failed to get monthly commitments")
		return NewInternalError(c, "Failed to get monthly commitments")
	}

	payments := make([]CommitmentPayment, len(result.Payments))
	for i, p := range result.Payments {
		payments[i] = CommitmentPayment{
			LoanID:        p.LoanID,
			ItemName:      p.ItemName,
			PaymentNumber: p.PaymentNumber,
			TotalPayments: p.TotalPayments,
			Amount:        p.Amount.StringFixed(2),
			Paid:          p.Paid,
		}
	}

	return c.JSON(http.StatusOK, CommitmentsResponse{
		Year:        result.Year,
		Month:       result.Month,
		TotalUnpaid: result.TotalUnpaid.StringFixed(2),
		TotalPaid:   result.TotalPaid.StringFixed(2),
		Payments:    payments,
	})
}

// PreviewLoan handles POST /api/v1/loans/preview
func (h *LoanHandler) PreviewLoan(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	var req PreviewLoanRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	// Parse total amount
	totalAmount, err := decimal.NewFromString(req.TotalAmount)
	if err != nil {
		return NewValidationError(c, "Invalid total amount", []ValidationError{
			{Field: "totalAmount", Message: "Must be a valid decimal number"},
		})
	}

	// Parse purchase date
	purchaseDate, err := time.Parse("2006-01-02", req.PurchaseDate)
	if err != nil {
		return NewValidationError(c, "Invalid purchase date", []ValidationError{
			{Field: "purchaseDate", Message: "Must be in YYYY-MM-DD format"},
		})
	}

	// Parse optional interest rate override
	var interestRate *decimal.Decimal
	if req.InterestRate != nil && *req.InterestRate != "" {
		rate, err := decimal.NewFromString(*req.InterestRate)
		if err != nil {
			return NewValidationError(c, "Invalid interest rate", []ValidationError{
				{Field: "interestRate", Message: "Must be a valid decimal number"},
			})
		}
		interestRate = &rate
	}

	input := service.PreviewLoanInput{
		ProviderID:   req.ProviderID,
		TotalAmount:  totalAmount,
		NumMonths:    req.NumMonths,
		PurchaseDate: purchaseDate,
		InterestRate: interestRate,
	}

	result, err := h.loanService.PreviewLoan(workspaceID, input)
	if err != nil {
		if errors.Is(err, domain.ErrLoanProviderInvalid) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "providerId", Message: "Invalid loan provider"},
			})
		}
		if errors.Is(err, domain.ErrLoanAmountInvalid) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "totalAmount", Message: "Amount must be positive"},
			})
		}
		if errors.Is(err, domain.ErrLoanMonthsInvalid) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "numMonths", Message: "Number of months must be at least 1"},
			})
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to preview loan")
		return NewInternalError(c, "Failed to preview loan")
	}

	return c.JSON(http.StatusOK, PreviewLoanResponse{
		MonthlyPayment:    result.MonthlyPayment.StringFixed(2),
		FirstPaymentYear:  result.FirstPaymentYear,
		FirstPaymentMonth: result.FirstPaymentMonth,
		InterestRate:      result.InterestRate.StringFixed(2),
	})
}

// TrendProviderResponse represents a provider's breakdown in the trend response
type TrendProviderResponse struct {
	ID     int32  `json:"id"`
	Name   string `json:"name"`
	Amount string `json:"amount"`
}

// TrendMonthResponse represents a single month in the trend response
type TrendMonthResponse struct {
	Month     string                  `json:"month"`
	Total     string                  `json:"total"`
	IsPaid    bool                    `json:"isPaid"`
	Providers []TrendProviderResponse `json:"providers"`
}

// TrendResponse represents the complete trend API response
type TrendAPIResponse struct {
	Months []TrendMonthResponse `json:"months"`
}

// GetTrend handles GET /api/v1/loans/trend
// Returns monthly loan payment aggregates with provider breakdown
func (h *LoanHandler) GetTrend(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	// Parse months query parameter (default 12, max 24)
	months := 12
	monthsParam := c.QueryParam("months")
	if monthsParam != "" {
		parsed, err := strconv.Atoi(monthsParam)
		if err != nil || parsed < 1 || parsed > 24 {
			return NewValidationError(c, "Invalid months parameter", []ValidationError{
				{Field: "months", Message: "Must be a number between 1 and 24"},
			})
		}
		months = parsed
	}

	result, err := h.loanService.GetTrend(workspaceID, months)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("months", months).Msg("Failed to get loan trend")
		return NewInternalError(c, "Failed to get loan trend")
	}

	// Convert to API response format (decimal to string)
	response := TrendAPIResponse{
		Months: make([]TrendMonthResponse, len(result.Months)),
	}
	for i, m := range result.Months {
		providers := make([]TrendProviderResponse, len(m.Providers))
		for j, p := range m.Providers {
			providers[j] = TrendProviderResponse{
				ID:     p.ID,
				Name:   p.Name,
				Amount: p.Amount.StringFixed(2),
			}
		}
		response.Months[i] = TrendMonthResponse{
			Month:     m.Month,
			Total:     m.Total.StringFixed(2),
			IsPaid:    m.IsPaid,
			Providers: providers,
		}
	}

	return c.JSON(http.StatusOK, response)
}

// Helper function to convert domain.Loan to LoanResponse
func toLoanResponse(loan *domain.Loan) LoanResponse {
	lastYear, lastMonth := loan.GetLastPaymentYearMonth()
	resp := LoanResponse{
		ID:                loan.ID,
		WorkspaceID:       loan.WorkspaceID,
		ProviderID:        loan.ProviderID,
		ItemName:          loan.ItemName,
		TotalAmount:       loan.TotalAmount.StringFixed(2),
		NumMonths:         loan.NumMonths,
		PurchaseDate:      loan.PurchaseDate.Format("2006-01-02"),
		InterestRate:      loan.InterestRate.StringFixed(2),
		MonthlyPayment:    loan.MonthlyPayment.StringFixed(2),
		FirstPaymentYear:  loan.FirstPaymentYear,
		FirstPaymentMonth: loan.FirstPaymentMonth,
		LastPaymentYear:   lastYear,
		LastPaymentMonth:  lastMonth,
		Notes:             loan.Notes,
		CreatedAt:         loan.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         loan.UpdatedAt.Format(time.RFC3339),
	}
	if loan.DeletedAt != nil {
		deletedAt := loan.DeletedAt.Format(time.RFC3339)
		resp.DeletedAt = &deletedAt
	}
	return resp
}

// Helper function to convert domain.LoanWithStats to LoanWithStatsResponse
func toLoanWithStatsResponse(loanWithStats *domain.LoanWithStats) LoanWithStatsResponse {
	resp := LoanWithStatsResponse{
		ID:                loanWithStats.ID,
		WorkspaceID:       loanWithStats.WorkspaceID,
		ProviderID:        loanWithStats.ProviderID,
		ItemName:          loanWithStats.ItemName,
		TotalAmount:       loanWithStats.TotalAmount.StringFixed(2),
		NumMonths:         loanWithStats.NumMonths,
		PurchaseDate:      loanWithStats.PurchaseDate.Format("2006-01-02"),
		InterestRate:      loanWithStats.InterestRate.StringFixed(2),
		MonthlyPayment:    loanWithStats.MonthlyPayment.StringFixed(2),
		FirstPaymentYear:  loanWithStats.FirstPaymentYear,
		FirstPaymentMonth: loanWithStats.FirstPaymentMonth,
		LastPaymentYear:   loanWithStats.LastPaymentYear,
		LastPaymentMonth:  loanWithStats.LastPaymentMonth,
		Notes:             loanWithStats.Notes,
		CreatedAt:         loanWithStats.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         loanWithStats.UpdatedAt.Format(time.RFC3339),
		// Stats fields
		TotalCount:       loanWithStats.TotalCount,
		PaidCount:        loanWithStats.PaidCount,
		RemainingBalance: loanWithStats.RemainingBalance.StringFixed(2),
		Progress:         loanWithStats.Progress,
	}
	if loanWithStats.DeletedAt != nil {
		deletedAt := loanWithStats.DeletedAt.Format(time.RFC3339)
		resp.DeletedAt = &deletedAt
	}
	return resp
}
