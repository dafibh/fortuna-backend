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
)

// DashboardHandler handles dashboard-related HTTP requests
type DashboardHandler struct {
	dashboardService *service.DashboardService
}

// NewDashboardHandler creates a new DashboardHandler
func NewDashboardHandler(dashboardService *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: dashboardService,
	}
}

// CCPayableResponse represents the CC payable summary in API response
type CCPayableResponse struct {
	ThisMonth string `json:"thisMonth"`
	NextMonth string `json:"nextMonth"`
	Total     string `json:"total"`
}

// ProjectionResponse represents projected financial data for future months
type ProjectionResponse struct {
	RecurringIncome   string `json:"recurringIncome"`
	RecurringExpenses string `json:"recurringExpenses"`
	LoanPayments      string `json:"loanPayments"`
	Note              string `json:"note,omitempty"`
}

// DashboardSummaryResponse represents the dashboard summary API response
type DashboardSummaryResponse struct {
	IsProjection          bool                `json:"isProjection"`
	ProjectionLimitMonths int                 `json:"projectionLimitMonths,omitempty"`
	TotalBalance          string              `json:"totalBalance"`
	InHandBalance         string              `json:"inHandBalance"`
	DisposableIncome      string              `json:"disposableIncome"`
	DaysRemaining         int                 `json:"daysRemaining"`
	DailyBudget           string              `json:"dailyBudget"`
	CCPayable             *CCPayableResponse  `json:"ccPayable"`
	Month                 MonthResponse       `json:"month"`
	Projection            *ProjectionResponse `json:"projection,omitempty"`
}

// GetSummary godoc
// @Summary Get dashboard summary
// @Description Get financial summary including balances, disposable income, and projections
// @Tags dashboard
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param year query int false "Year for historical data"
// @Param month query int false "Month for historical data (1-12)"
// @Success 200 {object} DashboardSummaryResponse
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /dashboard/summary [get]
func (h *DashboardHandler) GetSummary(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	// Parse optional year/month params (default to current)
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	if yearStr := c.QueryParam("year"); yearStr != "" {
		parsedYear, err := strconv.Atoi(yearStr)
		if err != nil {
			return NewValidationError(c, "Invalid year format", []ValidationError{{Field: "year", Message: "Must be a valid integer"}})
		}
		if parsedYear < 2000 || parsedYear > 2100 {
			return NewValidationError(c, "Year must be between 2000 and 2100", []ValidationError{{Field: "year", Message: "Must be between 2000 and 2100"}})
		}
		year = parsedYear
	}
	if monthStr := c.QueryParam("month"); monthStr != "" {
		parsedMonth, err := strconv.Atoi(monthStr)
		if err != nil {
			return NewValidationError(c, "Invalid month format", []ValidationError{{Field: "month", Message: "Must be a valid integer"}})
		}
		if parsedMonth < 1 || parsedMonth > 12 {
			return NewValidationError(c, "Month must be between 1 and 12", []ValidationError{{Field: "month", Message: "Must be between 1 and 12"}})
		}
		month = parsedMonth
	}

	summary, err := h.dashboardService.GetSummaryForMonth(workspaceID, year, month)
	if err != nil {
		// Handle projection limit exceeded error
		if errors.Is(err, service.ErrProjectionLimitExceeded) {
			return NewValidationError(c, "Cannot project more than 12 months ahead", []ValidationError{
				{Field: "month", Message: "Projection limit is 12 months from current month"},
			})
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("year", year).Int("month", month).Msg("Failed to get dashboard summary")
		return NewInternalError(c, "Failed to get dashboard summary")
	}

	var ccPayable *CCPayableResponse
	if summary.CCPayable != nil {
		ccPayable = &CCPayableResponse{
			ThisMonth: summary.CCPayable.ThisMonth.StringFixed(2),
			NextMonth: summary.CCPayable.NextMonth.StringFixed(2),
			Total:     summary.CCPayable.Total.StringFixed(2),
		}
	}

	var projection *ProjectionResponse
	if summary.Projection != nil {
		projection = &ProjectionResponse{
			RecurringIncome:   summary.Projection.RecurringIncome.StringFixed(2),
			RecurringExpenses: summary.Projection.RecurringExpenses.StringFixed(2),
			LoanPayments:      summary.Projection.LoanPayments.StringFixed(2),
			Note:              summary.Projection.Note,
		}
	}

	// Set projection limit only for projections
	projectionLimit := 0
	if summary.IsProjection {
		projectionLimit = domain.MaxProjectionMonths
	}

	return c.JSON(http.StatusOK, DashboardSummaryResponse{
		IsProjection:          summary.IsProjection,
		ProjectionLimitMonths: projectionLimit,
		TotalBalance:          summary.TotalBalance.StringFixed(2),
		InHandBalance:         summary.InHandBalance.StringFixed(2),
		DisposableIncome:      summary.DisposableIncome.StringFixed(2),
		DaysRemaining:         summary.DaysRemaining,
		DailyBudget:           summary.DailyBudget.StringFixed(2),
		CCPayable:             ccPayable,
		Month:                 toMonthResponse(summary.Month),
		Projection:            projection,
	})
}
