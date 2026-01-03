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

// WishlistNoteHandler handles wishlist note-related HTTP requests
type WishlistNoteHandler struct {
	noteService *service.WishlistNoteService
}

// NewWishlistNoteHandler creates a new WishlistNoteHandler
func NewWishlistNoteHandler(noteService *service.WishlistNoteService) *WishlistNoteHandler {
	return &WishlistNoteHandler{noteService: noteService}
}

// CreateNoteRequest represents the create note request body
type CreateNoteRequest struct {
	Content string `json:"content"`
}

// UpdateNoteRequest represents the update note request body
type UpdateNoteRequest struct {
	Content string `json:"content"`
}

// NoteResponse represents a note in API responses
type NoteResponse struct {
	ID        int32  `json:"id"`
	ItemID    int32  `json:"itemId"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// CreateNote handles POST /api/v1/wishlist-items/:id/notes
func (h *WishlistNoteHandler) CreateNote(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	itemID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid item ID", nil)
	}

	var req CreateNoteRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	input := service.CreateNoteInput{
		Content: req.Content,
	}

	note, err := h.noteService.CreateNote(workspaceID, int32(itemID), input)
	if err != nil {
		if errors.Is(err, domain.ErrWishlistItemNotFound) {
			return NewNotFoundError(c, "Wishlist item not found")
		}
		if errors.Is(err, domain.ErrNoteContentEmpty) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "content", Message: "Content is required"},
			})
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("item_id", itemID).Msg("Failed to create note")
		return NewInternalError(c, "Failed to create note")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("note_id", note.ID).Msg("Note created")

	return c.JSON(http.StatusCreated, toNoteResponse(note))
}

// ListNotes handles GET /api/v1/wishlist-items/:id/notes
func (h *WishlistNoteHandler) ListNotes(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	itemID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid item ID", nil)
	}

	// Parse sort parameter (default: desc = newest first)
	sortAsc := c.QueryParam("sort") == "asc"

	notes, err := h.noteService.ListNotesByItem(workspaceID, int32(itemID), sortAsc)
	if err != nil {
		if errors.Is(err, domain.ErrWishlistItemNotFound) {
			return NewNotFoundError(c, "Wishlist item not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("item_id", itemID).Msg("Failed to list notes")
		return NewInternalError(c, "Failed to list notes")
	}

	response := make([]NoteResponse, len(notes))
	for i, note := range notes {
		response[i] = toNoteResponse(note)
	}

	return c.JSON(http.StatusOK, response)
}

// GetNote handles GET /api/v1/wishlist-item-notes/:id
func (h *WishlistNoteHandler) GetNote(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid note ID", nil)
	}

	note, err := h.noteService.GetNoteByID(workspaceID, int32(id))
	if err != nil {
		if errors.Is(err, domain.ErrNoteNotFound) {
			return NewNotFoundError(c, "Note not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("note_id", id).Msg("Failed to get note")
		return NewInternalError(c, "Failed to get note")
	}

	return c.JSON(http.StatusOK, toNoteResponse(note))
}

// UpdateNote handles PUT /api/v1/wishlist-item-notes/:id
func (h *WishlistNoteHandler) UpdateNote(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid note ID", nil)
	}

	var req UpdateNoteRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	note, err := h.noteService.UpdateNote(workspaceID, int32(id), req.Content)
	if err != nil {
		if errors.Is(err, domain.ErrNoteNotFound) {
			return NewNotFoundError(c, "Note not found")
		}
		if errors.Is(err, domain.ErrNoteContentEmpty) {
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "content", Message: "Content is required"},
			})
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("note_id", id).Msg("Failed to update note")
		return NewInternalError(c, "Failed to update note")
	}

	log.Info().Int32("workspace_id", workspaceID).Int32("note_id", note.ID).Msg("Note updated")

	return c.JSON(http.StatusOK, toNoteResponse(note))
}

// DeleteNote handles DELETE /api/v1/wishlist-item-notes/:id
func (h *WishlistNoteHandler) DeleteNote(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return NewValidationError(c, "Invalid note ID", nil)
	}

	if err := h.noteService.DeleteNote(workspaceID, int32(id)); err != nil {
		if errors.Is(err, domain.ErrNoteNotFound) {
			return NewNotFoundError(c, "Note not found")
		}
		log.Error().Err(err).Int32("workspace_id", workspaceID).Int("note_id", id).Msg("Failed to delete note")
		return NewInternalError(c, "Failed to delete note")
	}

	log.Info().Int32("workspace_id", workspaceID).Int("note_id", id).Msg("Note deleted")

	return c.NoContent(http.StatusNoContent)
}

// Helper function to convert domain.WishlistItemNote to NoteResponse
func toNoteResponse(note *domain.WishlistItemNote) NoteResponse {
	return NoteResponse{
		ID:        note.ID,
		ItemID:    note.ItemID,
		Content:   note.Content,
		CreatedAt: note.CreatedAt.Format(time.RFC3339),
		UpdatedAt: note.UpdatedAt.Format(time.RFC3339),
	}
}
