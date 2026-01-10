package handler

import (
	"errors"
	"net/http"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/middleware"
	"github.com/dafibh/fortuna/fortuna-backend/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// WorkspaceHandler handles workspace-related HTTP requests
type WorkspaceHandler struct {
	workspaceService *service.WorkspaceService
}

// NewWorkspaceHandler creates a new WorkspaceHandler
func NewWorkspaceHandler(workspaceService *service.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{workspaceService: workspaceService}
}

// ClearAllData handles DELETE /workspace/clear
// This is a destructive operation that deletes all workspace data
func (h *WorkspaceHandler) ClearAllData(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	err := h.workspaceService.ClearAllData(workspaceID)
	if err != nil {
		if errors.Is(err, domain.ErrWorkspaceNotFound) {
			return NewNotFoundError(c, "Workspace not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to clear workspace data")
		return NewInternalError(c, "Failed to clear workspace data")
	}

	log.Info().Int32("workspace_id", workspaceID).Msg("Workspace data cleared")

	return c.NoContent(http.StatusNoContent)
}

