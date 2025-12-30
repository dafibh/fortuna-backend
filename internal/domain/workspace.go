package domain

import (
	"time"

	"github.com/google/uuid"
)

// Workspace represents a user's workspace
type Workspace struct {
	ID        int32     `json:"id"`
	UserID    uuid.UUID `json:"userId"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// WorkspaceRepository defines the interface for workspace persistence operations
type WorkspaceRepository interface {
	GetByID(id int32) (*Workspace, error)
	GetByUserID(userID uuid.UUID) (*Workspace, error)
	GetByUserAuth0ID(auth0ID string) (*Workspace, error)
	Create(workspace *Workspace) (*Workspace, error)
	Update(workspace *Workspace) (*Workspace, error)
	Delete(id int32) error
}
