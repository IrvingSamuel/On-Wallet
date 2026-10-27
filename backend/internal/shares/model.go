package shares

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("share not found")
	ErrGone          = errors.New("share expired or revoked")
	ErrInvalidInput  = errors.New("invalid input")
	ErrDocumentGone  = errors.New("document not available")
)

const (
	AuditLinkGerado   = "link_gerado"
	AuditLinkRevogado = "link_revogado"
)

// AllowedDays are the validity options for creating a share (RF-19).
var AllowedDays = []int{1, 7, 30}

type Share struct {
	ID         uuid.UUID  `json:"id"`
	OwnerID    string     `json:"owner_id"`
	DocumentID uuid.UUID  `json:"document_id"`
	VersionID  uuid.UUID  `json:"version_id"`
	Token      string     `json:"token"`
	ExpiresAt  time.Time  `json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	Revoked    bool       `json:"revoked"`
}

type PublicShare struct {
	Title        string    `json:"title"`
	DocumentID   uuid.UUID `json:"document_id"`
	Version      int       `json:"version"`
	MimeType     string    `json:"mime_type"`
	FileName     string    `json:"file_name"`
	Algorithm    string    `json:"algorithm"`
	Hash         string    `json:"hash"`
	ExpiresAt    time.Time `json:"expires_at"`
	Verification string    `json:"verification"` // ok | divergent | unverifiable | missing_record
}
