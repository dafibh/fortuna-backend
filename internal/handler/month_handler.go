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

// MonthHandler handles month-related HTTP requests
type MonthHandler struct {
	monthService *service.MonthService
}

// NewMonthHandler creates a new MonthHandler
func NewMonthHandler(monthService *service.MonthService) *MonthHandler {
	return &MonthHandler{
		monthService: monthService,
	}
}

// MonthResponse represents a month in API responses
type MonthResponse struct {
	ID              int32  `json:"id"`
	Year            int    `json:"year"`
	Month           int    `json:"month"`
	StartDate       string `json:"startDate"`
	EndDate         string `json:"endDate"`
	StartingBalance string `json:"startingBalance"`
	TotalIncome     string `json:"totalIncome"`
	TotalExpenses   string `json:"totalExpenses"`
	ClosingBalance  string `json:"closingBalance"`
	CreatedAt       string `json:"createdAt"`
}

// GetCurrent handles GET /api/v1/months/current
func (h *MonthHandler) GetCurrent(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	now := time.Now()
	month, err := h.monthService.GetOrCreateMonth(workspaceID, now.Year(), int(now.Month()))
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return NewValidationError(c, "Invalid month or year", nil)
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get current month")
		return NewInternalError(c, "Failed to get current month")
	}

	return c.JSON(http.StatusOK, toMonthResponse(month))
}

// GetByYearMonth handles GET /api/v1/months/:year/:month
func (h *MonthHandler) GetByYearMonth(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	year, err := strconv.Atoi(c.Param("year"))
	if err != nil || year < 2000 || year > 2100 {
		return NewValidationError(c, "Invalid year", []ValidationError{
			{Field: "year", Message: "Year must be between 2000 and 2100"},
		})
	}

	monthNum, err := strconv.Atoi(c.Param("month"))
	if err != nil || monthNum < 1 || monthNum > 12 {
		return NewValidationError(c, "Invalid month", []ValidationError{
			{Field: "month", Message: "Month must be between 1 and 12"},
		})
	}

	month, err := h.monthService.GetOrCreateMonth(workspaceID, year, monthNum)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return NewValidationError(c, "Invalid month or year", nil)
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("year", year).Int("month", monthNum).Msg("Failed to get month")
		return NewInternalError(c, "Failed to get month")
	}

	return c.JSON(http.StatusOK, toMonthResponse(month))
}

// GetAllMonths handles GET /api/v1/months
func (h *MonthHandler) GetAllMonths(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	months, err := h.monthService.GetAllMonths(workspaceID)
	if err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to get months")
		return NewInternalError(c, "Failed to get months")
	}

	response := make([]MonthResponse, len(months))
	for i, month := range months {
		response[i] = toMonthResponse(month)
	}

	return c.JSON(http.StatusOK, response)
}

// Helper function to convert domain.CalculatedMonth to MonthResponse
func toMonthResponse(m *domain.CalculatedMonth) MonthResponse {
	return MonthResponse{
		ID:              m.ID,
		Year:            m.Month.Year,
		Month:           m.Month.Month,
		StartDate:       m.StartDate.Format("2006-01-02"),
		EndDate:         m.EndDate.Format("2006-01-02"),
		StartingBalance: m.StartingBalance.StringFixed(2),
		TotalIncome:     m.TotalIncome.StringFixed(2),
		TotalExpenses:   m.TotalExpenses.StringFixed(2),
		ClosingBalance:  m.ClosingBalance.StringFixed(2),
		CreatedAt:       m.CreatedAt.Format(time.RFC3339),
	}
}
