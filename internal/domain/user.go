package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID         uuid.UUID `json:"id"`
	Auth0ID    string    `json:"auth0Id"`
	Email      string    `json:"email"`
	Name       *string   `json:"name"`
	PictureURL *string   `json:"pictureUrl"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// UserRepository defines the interface for user persistence operations
type UserRepository interface {
	GetByID(id uuid.UUID) (*User, error)
	GetByAuth0ID(auth0ID string) (*User, error)
	Create(user *User) (*User, error)
	Update(user *User) (*User, error)
	UpdateName(auth0ID string, name string) (*User, error)
	CreateOrGetByAuth0ID(auth0ID, email string, name, pictureURL *string) (*User, error)
}
