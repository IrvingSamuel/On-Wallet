package shares

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
)

type ObjectStore interface {
	Get(ctx context.Context, key string) (io.ReadCloser, int64, string, error)
}

type Service struct {
	repo  *Repository
	store ObjectStore
}

func NewService(repo *Repository, store ObjectStore) *Service {
	return &Service{repo: repo, store: store}
}

func (s *Service) Create(ctx context.Context, ownerID string, docID uuid.UUID, days int) (Share, error) {
	if !slices.Contains(AllowedDays, days) {
		return Share{}, fmt.Errorf("%w: days must be one of %v", ErrInvalidInput, AllowedDays)
	}
	dv, err := s.repo.LatestOwnedVersion(ctx, ownerID, docID)
	if err != nil {
		return Share{}, err
	}

	token, err := randomToken()
	if err != nil {
		return Share{}, err
	}
	sh := Share{
		ID:         uuid.New(),
		OwnerID:    ownerID,
		DocumentID: docID,
		VersionID:  dv.VersionID,
		Token:      token,
		ExpiresAt:  time.Now().UTC().Add(time.Duration(days) * 24 * time.Hour),
	}
	if err := s.repo.Create(ctx, &sh); err != nil {
		return Share{}, err
	}
	_ = s.repo.InsertAudit(ctx, ownerID, ownerID, AuditLinkGerado, &docID, dv.Title,
		fmt.Sprintf("days=%d version=v%d token=%s…", days, dv.Version, token[:8]))
	return sh, nil
}

func (s *Service) List(ctx context.Context, ownerID string, docID uuid.UUID) ([]Share, error) {
	return s.repo.ListByDocument(ctx, ownerID, docID)
}

func (s *Service) Revoke(ctx context.Context, ownerID string, shareID uuid.UUID) (Share, error) {
	sh, err := s.repo.Revoke(ctx, ownerID, shareID)
	if err != nil {
		return Share{}, err
	}
	_ = s.repo.InsertAudit(ctx, ownerID, ownerID, AuditLinkRevogado, &sh.DocumentID, "",
		fmt.Sprintf("share_id=%s", shareID))
	return sh, nil
}

// RevokeAllForDocument implements documents.ShareRevoker (RF-22).
func (s *Service) RevokeAllForDocument(ctx context.Context, ownerID string, docID uuid.UUID) error {
	n, err := s.repo.RevokeAllForDocument(ctx, ownerID, docID)
	if err != nil {
		return err
	}
	if n > 0 {
		slog.Info("shares revoked on document delete", "document_id", docID, "count", n)
		_ = s.repo.InsertAudit(ctx, ownerID, ownerID, AuditLinkRevogado, &docID, "",
			fmt.Sprintf("bulk_revoke_on_delete count=%d", n))
	}
	return nil
}

func (s *Service) PublicGet(ctx context.Context, token string) (PublicShare, error) {
	row, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return PublicShare{}, err
	}
	if row.Deleted {
		return PublicShare{}, ErrDocumentGone
	}
	if row.Revoked || !row.ExpiresAt.After(time.Now().UTC()) {
		return PublicShare{}, ErrGone
	}

	out := PublicShare{
		Title:      row.Title,
		DocumentID: row.DocumentID,
		Version:    row.Version,
		MimeType:   row.MimeType,
		FileName:   row.FileName,
		Algorithm:  row.Algorithm,
		Hash:       row.Hash,
		ExpiresAt:  row.ExpiresAt,
	}
	out.Verification = s.verifyObject(ctx, row.ObjectKey, row.Hash)
	return out, nil
}

// FilePayload streams the shared version for public preview/download.
type FilePayload struct {
	Reader      io.ReadCloser
	Size        int64
	ContentType string
	FileName    string
}

func (s *Service) PublicFile(ctx context.Context, token string) (FilePayload, error) {
	row, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return FilePayload{}, err
	}
	if row.Deleted {
		return FilePayload{}, ErrDocumentGone
	}
	if row.Revoked || !row.ExpiresAt.After(time.Now().UTC()) {
		return FilePayload{}, ErrGone
	}
	rc, size, contentType, err := s.store.Get(ctx, row.ObjectKey)
	if err != nil {
		return FilePayload{}, err
	}
	if contentType == "" {
		contentType = row.MimeType
	}
	if size <= 0 {
		size = row.SizeBytes
	}
	return FilePayload{Reader: rc, Size: size, ContentType: contentType, FileName: row.FileName}, nil
}

func (s *Service) verifyObject(ctx context.Context, objectKey, expectedHash string) string {
	if expectedHash == "" {
		return "missing_record"
	}
	rc, _, _, err := s.store.Get(ctx, objectKey)
	if err != nil {
		slog.Error("public share get object failed", "err", err)
		return "unverifiable"
	}
	defer rc.Close()
	h := sha256.New()
	if _, err := io.Copy(h, rc); err != nil {
		return "unverifiable"
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got == expectedHash {
		return "ok"
	}
	return "divergent"
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
