package documents

import (
	"errors"
	"io"
	"time"

	"github.com/google/uuid"
)

const HashAlgorithm = "SHA-256"

// Types are the predefined document types (RF-05). "outro" covers anything else.
var Types = []string{"rg", "cpf", "cnh", "certidao", "certificado", "diploma", "comprovante", "outro"}

// AllowedMIME maps accepted MIME types (RF-03) to the extension used in the object key.
var AllowedMIME = map[string]string{
	"application/pdf": "pdf",
	"image/jpeg":      "jpg",
	"image/png":       "png",
}

var (
	ErrNotFound          = errors.New("document not found")
	ErrForbidden         = errors.New("document belongs to another owner")
	ErrUnsupportedType   = errors.New("unsupported file type")
	ErrTooLarge          = errors.New("file too large")
	ErrInvalidMetadata   = errors.New("invalid metadata")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrMissingIntegrity  = errors.New("integrity record missing")
)

const (
	StatusActive   = "active"
	StatusReplaced = "replaced"
	StatusRevoked  = "revoked"
	StatusExpired  = "expired"
)

type Document struct {
	ID            uuid.UUID  `json:"id"`
	OwnerID       string     `json:"owner_id"`
	Title         string     `json:"title"`
	Type          string     `json:"type"`
	Origin        string     `json:"origin"`
	Tags          []string   `json:"tags"`
	Status        string     `json:"status"`
	RevokedReason *string    `json:"revoked_reason,omitempty"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type Version struct {
	ID         uuid.UUID `json:"id"`
	DocumentID uuid.UUID `json:"document_id"`
	Version    int       `json:"version"`
	ObjectKey  string    `json:"object_key"`
	FileName   string    `json:"file_name"`
	SizeBytes  int64     `json:"size_bytes"`
	MimeType   string    `json:"mime_type"`
	CreatedAt  time.Time `json:"created_at"`
}

type IntegrityRecord struct {
	ID        uuid.UUID `json:"id"`
	VersionID uuid.UUID `json:"version_id"`
	Version   int       `json:"version"`
	Algorithm string    `json:"algorithm"`
	Hash      string    `json:"hash"`
	CreatedAt time.Time `json:"created_at"`
}

type UploadResult struct {
	Document  Document        `json:"document"`
	Version   Version         `json:"version"`
	Integrity IntegrityRecord `json:"integrity"`
}

type UploadInput struct {
	OwnerID   string
	Title     string
	Type      string
	Origin    string
	Tags      []string
	FileName  string
	Size      int64
	ExpiresAt *time.Time
}

// ListFilter holds query params for GET /documents (RF-07 + RF-06 collection).
type ListFilter struct {
	Q          string
	Type       string
	Status     string
	Tag        string
	Collection string // uuid of a collection owned by the titular
}

// ListItem is a document row for the wallet listing (latest version summary).
type ListItem struct {
	Document
	FileName  string `json:"file_name"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	Version   int    `json:"version"`
}

// Detail is the full document payload for GET /documents/{id}.
type Detail struct {
	Document  Document         `json:"document"`
	Version   Version          `json:"version"`
	Integrity *IntegrityRecord `json:"integrity,omitempty"`
}

// PublicIntegrity is the anonymous payload for receipt QR verification (RF-12).
type PublicIntegrity struct {
	Title          string    `json:"title"`
	DocumentID     uuid.UUID `json:"document_id"`
	Version        int       `json:"version"`
	MimeType       string    `json:"mime_type"`
	FileName       string    `json:"file_name"`
	Algorithm      string    `json:"algorithm"`
	Hash           string    `json:"hash"`
	RegisteredAt   time.Time `json:"registered_at"`
	Status         string    `json:"status"`
	Verification   string    `json:"verification"` // ok | divergent | unverifiable | missing_record
}

// VersionSummary is a version row with integrity hash for GET /versions (RF-14).
// Flat fields (no Version embed) so Scan can target Version int without clashing with type Version.
type VersionSummary struct {
	ID         uuid.UUID `json:"id"`
	DocumentID uuid.UUID `json:"document_id"`
	Version    int       `json:"version"`
	ObjectKey  string    `json:"object_key"`
	FileName   string    `json:"file_name"`
	SizeBytes  int64     `json:"size_bytes"`
	MimeType   string    `json:"mime_type"`
	CreatedAt  time.Time `json:"created_at"`
	Algorithm  string    `json:"algorithm,omitempty"`
	Hash       string    `json:"hash,omitempty"`
	Current    bool      `json:"current"`
}

// FilePayload is the object stream for download.
type FilePayload struct {
	Reader      io.ReadCloser
	Size        int64
	ContentType string
	FileName    string
}

// Verification result codes (RF-10).
const (
	VerifyOK            = "ok"
	VerifyDivergent     = "divergent"
	VerifyUnverifiable  = "unverifiable"
	VerifyMissingRecord = "missing_record"
)

// Verification is an evidence row for RF-10/11.
type Verification struct {
	ID           uuid.UUID `json:"id"`
	DocumentID   uuid.UUID `json:"document_id"`
	VersionID    uuid.UUID `json:"version_id"`
	Version      int       `json:"version"`
	Result       string    `json:"result"`
	Algorithm    string    `json:"algorithm"`
	ExpectedHash string    `json:"expected_hash,omitempty"`
	ComputedHash string    `json:"computed_hash,omitempty"`
	Context      string    `json:"context"`
	CreatedAt    time.Time `json:"created_at"`
}

// Audit action codes (RF-21).
const (
	AuditInclusao           = "inclusao"
	AuditSubstituicao       = "substituicao"
	AuditRevogacao          = "revogacao"
	AuditVerificacao        = "verificacao"
	AuditCorrecaoMetadados  = "correcao_metadados"
	AuditExclusao           = "exclusao"
)

// MetadataPatch is the JSON body for PATCH /documents/{id} (RF-17).
type MetadataPatch struct {
	Title     string   `json:"title"`
	Type      string   `json:"type"`
	Origin    string   `json:"origin"`
	Tags      []string `json:"tags"`
	ExpiresAt *string  `json:"expires_at"` // null/empty clears; date string sets
}

// AuditEvent is a trail row for RF-21 (never stores document file content).
type AuditEvent struct {
	ID            uuid.UUID  `json:"id"`
	OwnerID       string     `json:"owner_id"`
	Actor         string     `json:"actor"`
	Action        string     `json:"action"`
	DocumentID    *uuid.UUID `json:"document_id,omitempty"`
	DocumentTitle string     `json:"document_title"`
	Context       string     `json:"context"`
	CreatedAt     time.Time  `json:"created_at"`
}
