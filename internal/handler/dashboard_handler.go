package handler

import (
	"net/http"

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

// DashboardSummaryResponse represents the dashboard summary API response
type DashboardSummaryResponse struct {
	TotalBalance     string             `json:"totalBalance"`
	InHandBalance    string             `json:"inHandBalance"`
	DisposableIncome string             `json:"disposableIncome"`
	DaysRemaining    int                `json:"daysRemaining"`
	DailyBudget      string             `json:"dailyBudget"`
	CCPayable        *CCPayableResponse `json:"ccPayable"`
	Month            MonthResponse      `json:"month"`
}

// GetSummary handles GET /api/v1/dashboard/summary
func (h *DashboardHandler) GetSummary(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	summary, err := h.dashboardService.GetSummary(workspaceID)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get dashboard summary")
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

	return c.JSON(http.StatusOK, DashboardSummaryResponse{
		TotalBalance:     summary.TotalBalance.StringFixed(2),
		InHandBalance:    summary.InHandBalance.StringFixed(2),
		DisposableIncome: summary.DisposableIncome.StringFixed(2),
		DaysRemaining:    summary.DaysRemaining,
		DailyBudget:      summary.DailyBudget.StringFixed(2),
		CCPayable:        ccPayable,
		Month:            toMonthResponse(summary.Month),
	})
}
