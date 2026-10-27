package documents

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ObjectStore interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Remove(ctx context.Context, key string) error
	Get(ctx context.Context, key string) (io.ReadCloser, int64, string, error)
}

// ShareRevoker revokes active shares when a document is soft-deleted (RF-18/22).
type ShareRevoker interface {
	RevokeAllForDocument(ctx context.Context, ownerID string, docID uuid.UUID) error
}

// AssistAnalyzer kicks off OCR + coherence after upload/version (optional).
type AssistAnalyzer interface {
	EnqueueAnalyze(ownerID string, docID uuid.UUID)
}

type Service struct {
	repo         *Repository
	store        ObjectStore
	maxBytes     int64
	publicOrigin string
	shares       ShareRevoker
	assist       AssistAnalyzer
}

func NewService(repo *Repository, store ObjectStore, maxBytes int64, publicOrigin string) *Service {
	if publicOrigin == "" {
		publicOrigin = "https://carteira.eflow.space"
	}
	return &Service{repo: repo, store: store, maxBytes: maxBytes, publicOrigin: strings.TrimRight(publicOrigin, "/")}
}

func (s *Service) SetShareRevoker(r ShareRevoker) { s.shares = r }

func (s *Service) SetAssistAnalyzer(a AssistAnalyzer) { s.assist = a }

func (s *Service) MaxBytes() int64 { return s.maxBytes }

type preparedFile struct {
	mime string
	ext  string
	hash string
}

// prepareFile validates size/MIME and returns SHA-256 of the exact bytes (file left at offset 0).
func (s *Service) prepareFile(file io.ReadSeeker, size int64) (preparedFile, error) {
	if size <= 0 {
		return preparedFile{}, fmt.Errorf("%w: empty file", ErrInvalidMetadata)
	}
	if size > s.maxBytes {
		return preparedFile{}, ErrTooLarge
	}
	head := make([]byte, 512)
	n, err := io.ReadFull(file, head)
	if err != nil && err != io.ErrUnexpectedEOF {
		return preparedFile{}, err
	}
	mime := http.DetectContentType(head[:n])
	ext, ok := AllowedMIME[mime]
	if !ok {
		return preparedFile{}, fmt.Errorf("%w: %s", ErrUnsupportedType, mime)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return preparedFile{}, err
	}
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return preparedFile{}, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return preparedFile{}, err
	}
	return preparedFile{mime: mime, ext: ext, hash: hex.EncodeToString(h.Sum(nil))}, nil
}

// Upload validates the file, hashes its exact bytes, stores it in MinIO and records metadata + integrity.
func (s *Service) Upload(ctx context.Context, in UploadInput, file io.ReadSeeker) (UploadResult, error) {
	in.Type = strings.ToLower(strings.TrimSpace(in.Type))
	in.Origin = strings.TrimSpace(in.Origin)
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		in.Title = in.FileName
	}
	if !slices.Contains(Types, in.Type) {
		return UploadResult{}, fmt.Errorf("%w: type must be one of %v", ErrInvalidMetadata, Types)
	}
	if in.Origin == "" {
		return UploadResult{}, fmt.Errorf("%w: origin is required", ErrInvalidMetadata)
	}
	if in.ExpiresAt != nil && !in.ExpiresAt.After(time.Now().UTC()) {
		return UploadResult{}, fmt.Errorf("%w: expires_at must be in the future", ErrInvalidMetadata)
	}

	pf, err := s.prepareFile(file, in.Size)
	if err != nil {
		return UploadResult{}, err
	}

	docID, versionID := uuid.New(), uuid.New()
	key := fmt.Sprintf("documents/%s/%s/v1/%s.%s", in.OwnerID, docID, pf.hash, pf.ext)

	if err := s.store.Put(ctx, key, file, in.Size, pf.mime); err != nil {
		return UploadResult{}, fmt.Errorf("store object: %w", err)
	}

	res := UploadResult{
		Document: Document{
			ID: docID, OwnerID: in.OwnerID, Title: in.Title, Type: in.Type, Origin: in.Origin,
			Tags: normalizeTags(in.Tags), ExpiresAt: in.ExpiresAt,
		},
		Version: Version{
			ID: versionID, DocumentID: docID, Version: 1, ObjectKey: key, FileName: in.FileName, SizeBytes: in.Size, MimeType: pf.mime,
		},
		Integrity: IntegrityRecord{
			ID: uuid.New(), VersionID: versionID, Version: 1, Algorithm: HashAlgorithm, Hash: pf.hash,
		},
	}

	if err := s.repo.Create(ctx, &res); err != nil {
		if rmErr := s.store.Remove(context.WithoutCancel(ctx), key); rmErr != nil {
			slog.Error("orphan object after db failure", "key", key, "err", rmErr)
		}
		return UploadResult{}, fmt.Errorf("persist document: %w", err)
	}
	ext := strings.ToUpper(pf.ext)
	s.audit(ctx, in.OwnerID, in.OwnerID, AuditInclusao, &docID, in.Title,
		fmt.Sprintf("%s, %s", ext, formatBytes(in.Size)))
	if s.assist != nil {
		s.assist.EnqueueAnalyze(in.OwnerID, docID)
	}
	return res, nil
}

// AddVersion replaces the file by appending a new immutable version (RF-14). Only active documents.
func (s *Service) AddVersion(ctx context.Context, ownerID string, docID uuid.UUID, fileName string, size int64, file io.ReadSeeker) (UploadResult, error) {
	_ = s.ExpireDue(ctx)
	detail, err := s.repo.GetDetail(ctx, ownerID, docID)
	if err != nil {
		return UploadResult{}, err
	}
	if detail.Document.Status != StatusActive {
		return UploadResult{}, fmt.Errorf("%w: can only replace active documents", ErrInvalidTransition)
	}

	pf, err := s.prepareFile(file, size)
	if err != nil {
		return UploadResult{}, err
	}

	next, err := s.repo.NextVersionNumber(ctx, ownerID, docID)
	if err != nil {
		return UploadResult{}, err
	}

	versionID := uuid.New()
	key := fmt.Sprintf("documents/%s/%s/v%d/%s.%s", ownerID, docID, next, pf.hash, pf.ext)
	if err := s.store.Put(ctx, key, file, size, pf.mime); err != nil {
		return UploadResult{}, fmt.Errorf("store object: %w", err)
	}

	res := UploadResult{
		Document: detail.Document,
		Version: Version{
			ID: versionID, DocumentID: docID, Version: next, ObjectKey: key,
			FileName: fileName, SizeBytes: size, MimeType: pf.mime,
		},
		Integrity: IntegrityRecord{
			ID: uuid.New(), VersionID: versionID, Version: next, Algorithm: HashAlgorithm, Hash: pf.hash,
		},
	}
	if err := s.repo.AddVersion(ctx, &res); err != nil {
		if rmErr := s.store.Remove(context.WithoutCancel(ctx), key); rmErr != nil {
			slog.Error("orphan object after db failure", "key", key, "err", rmErr)
		}
		return UploadResult{}, fmt.Errorf("persist version: %w", err)
	}
	s.audit(ctx, ownerID, ownerID, AuditSubstituicao, &docID, detail.Document.Title,
		fmt.Sprintf("v%d → v%d", next-1, next))
	if s.assist != nil {
		s.assist.EnqueueAnalyze(ownerID, docID)
	}
	return res, nil
}

func (s *Service) ListVersions(ctx context.Context, ownerID string, docID uuid.UUID) ([]VersionSummary, error) {
	return s.repo.ListVersions(ctx, ownerID, docID)
}

func (s *Service) Integrity(ctx context.Context, ownerID string, docID uuid.UUID) (IntegrityRecord, error) {
	return s.repo.LatestIntegrity(ctx, ownerID, docID)
}

func (s *Service) List(ctx context.Context, ownerID string, f ListFilter) ([]ListItem, error) {
	_ = s.ExpireDue(ctx)
	return s.repo.List(ctx, ownerID, f)
}

func (s *Service) Get(ctx context.Context, ownerID string, docID uuid.UUID) (Detail, error) {
	_ = s.ExpireDue(ctx)
	return s.repo.GetDetail(ctx, ownerID, docID)
}

// ExpireDue applies RF-16 status transitions for documents past expires_at.
func (s *Service) ExpireDue(ctx context.Context) error {
	n, err := s.repo.ExpireDue(ctx)
	if err != nil {
		slog.Error("expire due failed", "err", err)
		return err
	}
	if n > 0 {
		slog.Info("documents expired", "count", n)
	}
	return nil
}

// StartExpiryJob runs ExpireDue on an interval until ctx is cancelled (RF-16).
func (s *Service) StartExpiryJob(ctx context.Context, every time.Duration) {
	if every <= 0 {
		every = time.Minute
	}
	go func() {
		t := time.NewTicker(every)
		defer t.Stop()
		_ = s.ExpireDue(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				_ = s.ExpireDue(ctx)
			}
		}
	}()
}

// ParseExpiresAt parses optional upload expires_at (YYYY-MM-DD or RFC3339). Empty → nil.
func ParseExpiresAt(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		u := t.UTC()
		return &u, nil
	}
	t, err := time.ParseInLocation("2006-01-02", raw, time.UTC)
	if err != nil {
		return nil, fmt.Errorf("%w: expires_at inválido", ErrInvalidMetadata)
	}
	// End of that UTC calendar day so a date-only field is usable through the day.
	u := t.Add(24*time.Hour - time.Second).UTC()
	return &u, nil
}

func (s *Service) File(ctx context.Context, ownerID string, docID uuid.UUID) (FilePayload, error) {
	v, err := s.repo.LatestVersionForFile(ctx, ownerID, docID)
	if err != nil {
		return FilePayload{}, err
	}
	rc, size, contentType, err := s.store.Get(ctx, v.ObjectKey)
	if err != nil {
		return FilePayload{}, fmt.Errorf("store get: %w", err)
	}
	if contentType == "" {
		contentType = v.MimeType
	}
	if size <= 0 {
		size = v.SizeBytes
	}
	return FilePayload{Reader: rc, Size: size, ContentType: contentType, FileName: v.FileName}, nil
}

// PublicIntegrity returns live integrity check for a document by id (receipt QR, no auth).
func (s *Service) PublicIntegrity(ctx context.Context, docID uuid.UUID) (PublicIntegrity, error) {
	row, err := s.repo.GetPublicLatest(ctx, docID)
	if err != nil {
		return PublicIntegrity{}, err
	}
	out := PublicIntegrity{
		Title:      row.Title,
		DocumentID: row.DocumentID,
		Version:    row.Version,
		MimeType:   row.MimeType,
		FileName:   row.FileName,
		Status:     row.Status,
		Algorithm:  row.Algorithm,
		Hash:       row.Hash,
	}
	if row.HasIntegrity {
		out.RegisteredAt = row.Registered
		if out.Algorithm == "" {
			out.Algorithm = HashAlgorithm
		}
	}
	out.Verification = s.verifyObject(ctx, row.ObjectKey, row.Hash, row.HasIntegrity)
	return out, nil
}

// PublicFile streams the current version for anonymous receipt preview.
func (s *Service) PublicFile(ctx context.Context, docID uuid.UUID) (FilePayload, error) {
	row, err := s.repo.GetPublicLatest(ctx, docID)
	if err != nil {
		return FilePayload{}, err
	}
	rc, size, contentType, err := s.store.Get(ctx, row.ObjectKey)
	if err != nil {
		return FilePayload{}, fmt.Errorf("store get: %w", err)
	}
	if contentType == "" {
		contentType = row.MimeType
	}
	if size <= 0 {
		size = row.SizeBytes
	}
	return FilePayload{Reader: rc, Size: size, ContentType: contentType, FileName: row.FileName}, nil
}

func (s *Service) verifyObject(ctx context.Context, objectKey, expectedHash string, hasIntegrity bool) string {
	if !hasIntegrity || expectedHash == "" {
		return VerifyMissingRecord
	}
	rc, _, _, err := s.store.Get(ctx, objectKey)
	if err != nil {
		slog.Error("public integrity get object failed", "err", err)
		return VerifyUnverifiable
	}
	defer rc.Close()
	h := sha256.New()
	if _, err := io.Copy(h, rc); err != nil {
		return VerifyUnverifiable
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got == expectedHash {
		return VerifyOK
	}
	return VerifyDivergent
}

// Verify rehashes the MinIO object and compares it to the integrity record (RF-10).
// Divergence never updates the registered hash (RN 3.3). Evidence is always persisted (RF-11).
func (s *Service) Verify(ctx context.Context, ownerID string, docID uuid.UUID) (Verification, error) {
	detail, err := s.repo.GetDetail(ctx, ownerID, docID)
	if err != nil {
		return Verification{}, err
	}

	rec := Verification{
		ID:         uuid.New(),
		DocumentID: detail.Document.ID,
		VersionID:  detail.Version.ID,
		Version:    detail.Version.Version,
		Algorithm:  HashAlgorithm,
		Context:    "owner",
	}

	finish := func() (Verification, error) {
		if err := s.repo.InsertVerification(ctx, &rec); err != nil {
			return Verification{}, err
		}
		s.audit(ctx, ownerID, ownerID, AuditVerificacao, &docID, detail.Document.Title,
			fmt.Sprintf("Resultado: %s", rec.Result))
		return rec, nil
	}

	if detail.Integrity == nil {
		rec.Result = VerifyMissingRecord
		return finish()
	}

	rec.ExpectedHash = detail.Integrity.Hash
	if detail.Integrity.Algorithm != "" {
		rec.Algorithm = detail.Integrity.Algorithm
	}

	rc, _, _, err := s.store.Get(ctx, detail.Version.ObjectKey)
	if err != nil {
		slog.Warn("verify store get failed", "doc", docID, "err", err)
		rec.Result = VerifyUnverifiable
		return finish()
	}
	defer rc.Close()

	h := sha256.New()
	if _, err := io.Copy(h, rc); err != nil {
		slog.Warn("verify hash read failed", "doc", docID, "err", err)
		rec.Result = VerifyUnverifiable
		return finish()
	}
	rec.ComputedHash = hex.EncodeToString(h.Sum(nil))

	if rec.ComputedHash == rec.ExpectedHash {
		rec.Result = VerifyOK
	} else {
		rec.Result = VerifyDivergent
	}
	return finish()
}

func (s *Service) ListVerifications(ctx context.Context, ownerID string, docID uuid.UUID) ([]Verification, error) {
	return s.repo.ListVerifications(ctx, ownerID, docID)
}

func (s *Service) ListAudit(ctx context.Context, ownerID string) ([]AuditEvent, error) {
	return s.repo.ListAuditEvents(ctx, ownerID)
}

// canRevoke reports whether status may transition to revoked (RF-13/15).
func canRevoke(status string) bool {
	return status == StatusActive || status == StatusReplaced
}

func canPatchMetadata(status string) bool {
	return status == StatusActive || status == StatusReplaced || status == StatusExpired
}

func metadataSummary(title, docType, origin string, tags []string, expiresAt *time.Time) string {
	exp := "sem validade"
	if expiresAt != nil {
		exp = expiresAt.UTC().Format("2006-01-02")
	}
	return fmt.Sprintf("title=%q type=%s origin=%q tags=%v expires_at=%s", title, docType, origin, tags, exp)
}

// UpdateMetadata corrects document metadata without touching the file hash (RF-17).
func (s *Service) UpdateMetadata(ctx context.Context, ownerID string, docID uuid.UUID, patch MetadataPatch) (Document, error) {
	_ = s.ExpireDue(ctx)
	detail, err := s.repo.GetDetail(ctx, ownerID, docID)
	if err != nil {
		return Document{}, err
	}
	if !canPatchMetadata(detail.Document.Status) {
		return Document{}, fmt.Errorf("%w: cannot patch metadata from status %s", ErrInvalidTransition, detail.Document.Status)
	}

	title := strings.TrimSpace(patch.Title)
	if title == "" {
		title = detail.Document.Title
	}
	docType := strings.ToLower(strings.TrimSpace(patch.Type))
	if docType == "" {
		docType = detail.Document.Type
	}
	if !slices.Contains(Types, docType) {
		return Document{}, fmt.Errorf("%w: type must be one of %v", ErrInvalidMetadata, Types)
	}
	origin := strings.TrimSpace(patch.Origin)
	if origin == "" {
		return Document{}, fmt.Errorf("%w: origin is required", ErrInvalidMetadata)
	}
	tags := normalizeTags(patch.Tags)

	var expiresAt *time.Time
	if patch.ExpiresAt != nil {
		expiresAt, err = ParseExpiresAt(*patch.ExpiresAt)
		if err != nil {
			return Document{}, err
		}
	}
	if expiresAt != nil && !expiresAt.After(time.Now().UTC()) {
		return Document{}, fmt.Errorf("%w: expires_at must be in the future", ErrInvalidMetadata)
	}

	status := detail.Document.Status
	if status == StatusExpired {
		status = StatusActive // cleared or future validity reactivates
	}

	before := metadataSummary(detail.Document.Title, detail.Document.Type, detail.Document.Origin, detail.Document.Tags, detail.Document.ExpiresAt)
	doc, err := s.repo.UpdateMetadata(ctx, ownerID, docID, title, docType, origin, tags, expiresAt, status)
	if err != nil {
		return Document{}, err
	}
	after := metadataSummary(doc.Title, doc.Type, doc.Origin, doc.Tags, doc.ExpiresAt)
	s.audit(ctx, ownerID, ownerID, AuditCorrecaoMetadados, &docID, doc.Title, before+" → "+after)
	return doc, nil
}

// SoftDelete marks the document as deleted (RF-22). Does not purge MinIO or integrity rows.
func (s *Service) SoftDelete(ctx context.Context, ownerID string, docID uuid.UUID) error {
	doc, err := s.repo.SoftDelete(ctx, ownerID, docID)
	if err != nil {
		return err
	}
	if s.shares != nil {
		if err := s.shares.RevokeAllForDocument(ctx, ownerID, docID); err != nil {
			slog.Error("revoke shares on delete failed", "err", err, "document_id", docID)
		}
	}
	s.audit(ctx, ownerID, ownerID, AuditExclusao, &docID, doc.Title,
		fmt.Sprintf("status=%s", doc.Status))
	return nil
}

// Revoke transitions active|replaced → revoked with a required reason (RF-15).
func (s *Service) Revoke(ctx context.Context, ownerID string, docID uuid.UUID, reason string) (Document, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return Document{}, fmt.Errorf("%w: reason is required", ErrInvalidMetadata)
	}
	_ = s.ExpireDue(ctx)
	detail, err := s.repo.GetDetail(ctx, ownerID, docID)
	if err != nil {
		return Document{}, err
	}
	if !canRevoke(detail.Document.Status) {
		return Document{}, fmt.Errorf("%w: cannot revoke from status %s", ErrInvalidTransition, detail.Document.Status)
	}
	doc, err := s.repo.Revoke(ctx, ownerID, docID, reason)
	if err != nil {
		return Document{}, err
	}
	s.audit(ctx, ownerID, ownerID, AuditRevogacao, &docID, doc.Title, reason)
	return doc, nil
}

func (s *Service) audit(ctx context.Context, ownerID, actor, action string, docID *uuid.UUID, title, contextMsg string) {
	e := AuditEvent{
		ID: uuid.New(), OwnerID: ownerID, Actor: actor, Action: action,
		DocumentID: docID, DocumentTitle: title, Context: contextMsg,
	}
	if err := s.repo.InsertAuditEvent(ctx, &e); err != nil {
		slog.Error("audit insert failed", "action", action, "err", err)
	}
}

func formatBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%d KB", (n+512)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
}

func normalizeTags(raw []string) []string {
	out := []string{}
	for _, r := range raw {
		for _, t := range strings.Split(r, ",") {
			t = strings.ToLower(strings.TrimSpace(t))
			if t != "" && !slices.Contains(out, t) {
				out = append(out, t)
			}
		}
	}
	return out
}
