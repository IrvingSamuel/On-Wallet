package shares

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

type docVersion struct {
	OwnerID   string
	Title     string
	VersionID uuid.UUID
	Version   int
	ObjectKey string
	Algorithm string
	Hash      string
}

// LatestOwnedVersion returns current version + integrity for a non-deleted owned document.
func (r *Repository) LatestOwnedVersion(ctx context.Context, ownerID string, docID uuid.UUID) (docVersion, error) {
	var d docVersion
	var alg, hash *string
	err := r.pool.QueryRow(ctx, `
		SELECT d.owner_id, d.title, v.id, v.version, v.object_key,
		       ir.algorithm, ir.hash
		FROM documents d
		JOIN LATERAL (
			SELECT id, version, object_key
			FROM document_versions
			WHERE document_id = d.id
			ORDER BY version DESC
			LIMIT 1
		) v ON true
		LEFT JOIN integrity_records ir ON ir.version_id = v.id
		WHERE d.id = $1 AND d.owner_id = $2 AND d.deleted_at IS NULL`,
		docID, ownerID,
	).Scan(&d.OwnerID, &d.Title, &d.VersionID, &d.Version, &d.ObjectKey, &alg, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrNotFound
	}
	if err != nil {
		return d, err
	}
	if alg != nil {
		d.Algorithm = *alg
	}
	if hash != nil {
		d.Hash = *hash
	}
	return d, nil
}

func (r *Repository) Create(ctx context.Context, s *Share) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO shares (id, owner_id, document_id, version_id, token, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at`,
		s.ID, s.OwnerID, s.DocumentID, s.VersionID, s.Token, s.ExpiresAt,
	).Scan(&s.CreatedAt)
}

func (r *Repository) ListByDocument(ctx context.Context, ownerID string, docID uuid.UUID) ([]Share, error) {
	if _, err := r.LatestOwnedVersion(ctx, ownerID, docID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, owner_id, document_id, version_id, token, expires_at, revoked_at, created_at
		FROM shares
		WHERE document_id = $1 AND owner_id = $2
		ORDER BY created_at DESC`,
		docID, ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Share{}
	for rows.Next() {
		var s Share
		if err := rows.Scan(
			&s.ID, &s.OwnerID, &s.DocumentID, &s.VersionID, &s.Token,
			&s.ExpiresAt, &s.RevokedAt, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		s.Revoked = s.RevokedAt != nil
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) Revoke(ctx context.Context, ownerID string, shareID uuid.UUID) (Share, error) {
	var s Share
	err := r.pool.QueryRow(ctx, `
		UPDATE shares
		SET revoked_at = now()
		WHERE id = $1 AND owner_id = $2 AND revoked_at IS NULL
		RETURNING id, owner_id, document_id, version_id, token, expires_at, revoked_at, created_at`,
		shareID, ownerID,
	).Scan(
		&s.ID, &s.OwnerID, &s.DocumentID, &s.VersionID, &s.Token,
		&s.ExpiresAt, &s.RevokedAt, &s.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, ErrNotFound
	}
	s.Revoked = true
	return s, err
}

// RevokeAllForDocument revokes every active share for a document (RF-22 hook).
func (r *Repository) RevokeAllForDocument(ctx context.Context, ownerID string, docID uuid.UUID) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE shares
		SET revoked_at = now()
		WHERE document_id = $1 AND owner_id = $2 AND revoked_at IS NULL`,
		docID, ownerID,
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

type publicRow struct {
	Share
	Title     string
	Version   int
	ObjectKey string
	MimeType  string
	FileName  string
	SizeBytes int64
	Algorithm string
	Hash      string
	Deleted   bool
}

func (r *Repository) GetByToken(ctx context.Context, token string) (publicRow, error) {
	var p publicRow
	var alg, hash *string
	var deletedAt *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT s.id, s.owner_id, s.document_id, s.version_id, s.token, s.expires_at, s.revoked_at, s.created_at,
		       d.title, d.deleted_at, v.version, v.object_key, v.mime_type, v.file_name, v.size_bytes,
		       ir.algorithm, ir.hash
		FROM shares s
		JOIN documents d ON d.id = s.document_id
		JOIN document_versions v ON v.id = s.version_id
		LEFT JOIN integrity_records ir ON ir.version_id = v.id
		WHERE s.token = $1`,
		token,
	).Scan(
		&p.ID, &p.OwnerID, &p.DocumentID, &p.VersionID, &p.Token, &p.ExpiresAt, &p.RevokedAt, &p.CreatedAt,
		&p.Title, &deletedAt, &p.Version, &p.ObjectKey, &p.MimeType, &p.FileName, &p.SizeBytes, &alg, &hash,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	p.Revoked = p.RevokedAt != nil
	p.Deleted = deletedAt != nil
	if alg != nil {
		p.Algorithm = *alg
	}
	if hash != nil {
		p.Hash = *hash
	}
	return p, nil
}

func (r *Repository) InsertAudit(ctx context.Context, ownerID, actor, action string, docID *uuid.UUID, title, contextMsg string) error {
	var created time.Time
	return r.pool.QueryRow(ctx, `
		INSERT INTO audit_events (id, owner_id, actor, action, document_id, document_title, context)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at`,
		uuid.New(), ownerID, actor, action, docID, title, contextMsg,
	).Scan(&created)
}
