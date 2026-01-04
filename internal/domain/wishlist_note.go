package domain

import (
	"errors"
	"time"
)

var (
	ErrNoteNotFound     = errors.New("note not found")
	ErrNoteContentEmpty = errors.New("note content is required")
)

// WishlistItemNote represents a timestamped research note on a wishlist item
type WishlistItemNote struct {
	ID        int32     `json:"id"`
	ItemID    int32     `json:"itemId"`
	Content   string    `json:"content"`
	ImagePath *string   `json:"imagePath,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// WishlistNoteRepository defines the interface for wishlist note data access
type WishlistNoteRepository interface {
	Create(note *WishlistItemNote) (*WishlistItemNote, error)
	GetByID(workspaceID int32, id int32) (*WishlistItemNote, error)
	ListByItem(workspaceID int32, itemID int32, sortAsc bool) ([]*WishlistItemNote, error)
	CountByItem(workspaceID int32, itemID int32) (int64, error)
	Update(workspaceID int32, id int32, content string, imagePath *string) (*WishlistItemNote, error)
	Delete(workspaceID int32, id int32) error
}
