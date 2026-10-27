package collections

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound     = errors.New("collection not found")
	ErrInvalidInput = errors.New("invalid input")
)

type Collection struct {
	ID        uuid.UUID `json:"id"`
	OwnerID   string    `json:"owner_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListItem is a collection row with membership count for GET /collections.
type ListItem struct {
	Collection
	DocumentCount int `json:"document_count"`
}

// Detail is a collection with member document IDs.
type Detail struct {
	Collection
	DocumentIDs   []uuid.UUID `json:"document_ids"`
	DocumentCount int         `json:"document_count"`
}
