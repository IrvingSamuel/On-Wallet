package auth

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidInput  = errors.New("invalid input")
	ErrConflict      = errors.New("email already registered")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrInvalidCreds  = errors.New("invalid credentials")
)

const (
	CookieName     = "carteira_session"
	SessionTTL     = 7 * 24 * time.Hour
	MinPasswordLen = 8
)

type User struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// OwnerID is the string used as documents.owner_id (user UUID).
func (u User) OwnerID() string {
	return u.ID.String()
}
