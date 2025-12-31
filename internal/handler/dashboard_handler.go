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

// DashboardSummaryResponse represents the dashboard summary API response
type DashboardSummaryResponse struct {
	TotalBalance  string        `json:"totalBalance"`
	InHandBalance string        `json:"inHandBalance"`
	Month         MonthResponse `json:"month"`
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

	return c.JSON(http.StatusOK, DashboardSummaryResponse{
		TotalBalance:  summary.TotalBalance.StringFixed(2),
		InHandBalance: summary.InHandBalance.StringFixed(2),
		Month:         toMonthResponse(summary.Month),
	})
}
