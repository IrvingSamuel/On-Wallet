package assistant

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("assist data not found")
	ErrForbidden     = errors.New("document belongs to another owner")
	ErrUnavailable   = errors.New("assistant unavailable")
	ErrInvalidInput  = errors.New("invalid input")
	ErrDocumentGone  = errors.New("document not found")
)

const (
	ExtractPending    = "pending"
	ExtractRunning    = "running"
	ExtractDone       = "done"
	ExtractUnreadable = "unreadable"
	ExtractError      = "error"

	ResultConsistent  = "consistent"
	ResultDivergent   = "divergent"
	ResultUnreadable  = "unreadable"
	ResultPending     = "pending"
	ResultError       = "error"

	SourceKnowledge = "knowledge"
	SourceDocument  = "document"
)

type Extraction struct {
	ID         uuid.UUID       `json:"id"`
	DocumentID uuid.UUID       `json:"document_id"`
	VersionID  uuid.UUID       `json:"version_id"`
	OwnerID    string          `json:"owner_id"`
	Status     string          `json:"status"`
	RawText    string          `json:"raw_text,omitempty"`
	Fields     json.RawMessage `json:"fields_json"`
	Error      string          `json:"error,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type ConsistencyCheck struct {
	ID         uuid.UUID       `json:"id"`
	DocumentID uuid.UUID       `json:"document_id"`
	VersionID  uuid.UUID       `json:"version_id"`
	OwnerID    string          `json:"owner_id"`
	Result     string          `json:"result"`
	Summary    string          `json:"summary"`
	Details    json.RawMessage `json:"details_json"`
	CreatedAt  time.Time       `json:"created_at"`
}

type AssistStatus struct {
	Extraction  *Extraction       `json:"extraction,omitempty"`
	Consistency *ConsistencyCheck `json:"consistency,omitempty"`
	Disclaimer  string            `json:"disclaimer"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Messages   []ChatMessage `json:"messages"`
	DocumentID *string       `json:"document_id,omitempty"`
}

type ChatResponse struct {
	Reply      string   `json:"reply"`
	Citations  []string `json:"citations,omitempty"`
	Disclaimer string   `json:"disclaimer"`
}

type Chunk struct {
	ID         uuid.UUID
	Source     string
	OwnerID    *string
	DocumentID *uuid.UUID
	VersionID  *uuid.UUID
	Title      string
	Body       string
}

const Disclaimer = "Coerência assistida e o chat usam OCR/IA apenas como auxílio. Integridade (SHA-256) prova que o arquivo não mudou; nada disso prova autenticidade do emissor."
