package documents

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

// Create persists document, first version and integrity record atomically.
func (r *Repository) Create(ctx context.Context, res *UploadResult) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	d, v, i := &res.Document, &res.Version, &res.Integrity

	if err := tx.QueryRow(ctx, `
		INSERT INTO documents (id, owner_id, title, type, origin, tags, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING status, created_at`,
		d.ID, d.OwnerID, d.Title, d.Type, d.Origin, d.Tags, d.ExpiresAt,
	).Scan(&d.Status, &d.CreatedAt); err != nil {
		return err
	}

	if err := tx.QueryRow(ctx, `
		INSERT INTO document_versions (id, document_id, version, object_key, file_name, size_bytes, mime_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at`,
		v.ID, v.DocumentID, v.Version, v.ObjectKey, v.FileName, v.SizeBytes, v.MimeType,
	).Scan(&v.CreatedAt); err != nil {
		return err
	}

	if err := tx.QueryRow(ctx, `
		INSERT INTO integrity_records (id, version_id, algorithm, hash)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at`,
		i.ID, i.VersionID, i.Algorithm, i.Hash,
	).Scan(&i.CreatedAt); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// LatestIntegrity returns the integrity record of the newest version, scoped to the owner.
func (r *Repository) LatestIntegrity(ctx context.Context, ownerID string, docID uuid.UUID) (IntegrityRecord, error) {
	var rec IntegrityRecord
	err := r.pool.QueryRow(ctx, `
		SELECT ir.id, ir.version_id, v.version, ir.algorithm, ir.hash, ir.created_at
		FROM documents d
		JOIN document_versions v ON v.document_id = d.id
		JOIN integrity_records ir ON ir.version_id = v.id
		WHERE d.id = $1 AND d.owner_id = $2 AND d.deleted_at IS NULL
		ORDER BY v.version DESC
		LIMIT 1`, docID, ownerID,
	).Scan(&rec.ID, &rec.VersionID, &rec.Version, &rec.Algorithm, &rec.Hash, &rec.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return rec, ErrNotFound
	}
	return rec, err
}

// List returns the owner's documents with the latest version summary, optionally filtered.
func (r *Repository) List(ctx context.Context, ownerID string, f ListFilter) ([]ListItem, error) {
	args := []any{ownerID}
	where := []string{"d.owner_id = $1", "d.deleted_at IS NULL"}

	add := func(clause string, val any) {
		args = append(args, val)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}

	if t := strings.TrimSpace(f.Type); t != "" {
		add("d.type = $%d", strings.ToLower(t))
	}
	if s := strings.TrimSpace(f.Status); s != "" {
		add("d.status = $%d", strings.ToLower(s))
	}
	if tag := strings.TrimSpace(f.Tag); tag != "" {
		add("$%d = ANY(d.tags)", strings.ToLower(tag))
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		args = append(args, "%"+strings.ToLower(q)+"%")
		n := len(args)
		where = append(where, fmt.Sprintf(
			`(lower(d.title) LIKE $%d OR EXISTS (SELECT 1 FROM unnest(d.tags) t WHERE t LIKE $%d))`, n, n,
		))
	}
	if coll := strings.TrimSpace(f.Collection); coll != "" {
		args = append(args, coll, ownerID)
		nColl, nOwner := len(args)-1, len(args)
		where = append(where, fmt.Sprintf(
			`EXISTS (
				SELECT 1 FROM collection_documents cd
				JOIN collections c ON c.id = cd.collection_id
				WHERE cd.document_id = d.id AND c.id = $%d AND c.owner_id = $%d
			)`, nColl, nOwner,
		))
	}

	sql := `
		SELECT d.id, d.owner_id, d.title, d.type, d.origin, COALESCE(d.tags, '{}'), d.status,
		       d.revoked_reason, d.revoked_at, d.expires_at, d.created_at,
		       v.file_name, v.mime_type, v.size_bytes, v.version
		FROM documents d
		JOIN LATERAL (
			SELECT file_name, mime_type, size_bytes, version
			FROM document_versions
			WHERE document_id = d.id
			ORDER BY version DESC
			LIMIT 1
		) v ON true
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY d.created_at DESC`

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ListItem{}
	for rows.Next() {
		var item ListItem
		if err := rows.Scan(
			&item.ID, &item.OwnerID, &item.Title, &item.Type, &item.Origin, &item.Tags, &item.Status,
			&item.RevokedReason, &item.RevokedAt, &item.ExpiresAt, &item.CreatedAt,
			&item.FileName, &item.MimeType, &item.SizeBytes, &item.Version,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// publicLatestRow is the latest non-deleted document version + integrity (no owner scope).
type publicLatestRow struct {
	Title      string
	DocumentID uuid.UUID
	Status     string
	VersionID  uuid.UUID
	Version    int
	ObjectKey  string
	FileName   string
	SizeBytes  int64
	MimeType   string
	Algorithm  string
	Hash       string
	Registered time.Time
	HasIntegrity bool
}

// GetPublicLatest returns the current version of a non-deleted document by id (receipt QR).
func (r *Repository) GetPublicLatest(ctx context.Context, docID uuid.UUID) (publicLatestRow, error) {
	var row publicLatestRow
	var irAlg, irHash *string
	var irAt *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT d.title, d.id, d.status,
		       v.id, v.version, v.object_key, v.file_name, v.size_bytes, v.mime_type,
		       ir.algorithm, ir.hash, ir.created_at
		FROM documents d
		JOIN LATERAL (
			SELECT id, version, object_key, file_name, size_bytes, mime_type
			FROM document_versions
			WHERE document_id = d.id
			ORDER BY version DESC
			LIMIT 1
		) v ON true
		LEFT JOIN integrity_records ir ON ir.version_id = v.id
		WHERE d.id = $1 AND d.deleted_at IS NULL`,
		docID,
	).Scan(
		&row.Title, &row.DocumentID, &row.Status,
		&row.VersionID, &row.Version, &row.ObjectKey, &row.FileName, &row.SizeBytes, &row.MimeType,
		&irAlg, &irHash, &irAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return row, ErrNotFound
	}
	if err != nil {
		return row, err
	}
	if irAlg != nil && irHash != nil && irAt != nil {
		row.Algorithm = *irAlg
		row.Hash = *irHash
		row.Registered = *irAt
		row.HasIntegrity = true
	}
	return row, nil
}

// GetDetail returns document + latest version + integrity for the owner.
func (r *Repository) GetDetail(ctx context.Context, ownerID string, docID uuid.UUID) (Detail, error) {
	var d Detail
	var irID, irVID *uuid.UUID
	var irAlg, irHash *string
	var irAt *time.Time

	err := r.pool.QueryRow(ctx, `
		SELECT d.id, d.owner_id, d.title, d.type, d.origin, COALESCE(d.tags, '{}'), d.status,
		       d.revoked_reason, d.revoked_at, d.expires_at, d.created_at,
		       v.id, v.document_id, v.version, v.object_key, v.file_name, v.size_bytes, v.mime_type, v.created_at,
		       ir.id, ir.version_id, ir.algorithm, ir.hash, ir.created_at
		FROM documents d
		JOIN LATERAL (
			SELECT id, document_id, version, object_key, file_name, size_bytes, mime_type, created_at
			FROM document_versions
			WHERE document_id = d.id
			ORDER BY version DESC
			LIMIT 1
		) v ON true
		LEFT JOIN integrity_records ir ON ir.version_id = v.id
		WHERE d.id = $1 AND d.owner_id = $2 AND d.deleted_at IS NULL`,
		docID, ownerID,
	).Scan(
		&d.Document.ID, &d.Document.OwnerID, &d.Document.Title, &d.Document.Type, &d.Document.Origin,
		&d.Document.Tags, &d.Document.Status, &d.Document.RevokedReason, &d.Document.RevokedAt,
		&d.Document.ExpiresAt, &d.Document.CreatedAt,
		&d.Version.ID, &d.Version.DocumentID, &d.Version.Version, &d.Version.ObjectKey,
		&d.Version.FileName, &d.Version.SizeBytes, &d.Version.MimeType, &d.Version.CreatedAt,
		&irID, &irVID, &irAlg, &irHash, &irAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, r.notFoundOrForbidden(ctx, ownerID, docID)
	}
	if err != nil {
		return d, err
	}
	if irID != nil && irVID != nil && irAlg != nil && irHash != nil && irAt != nil {
		d.Integrity = &IntegrityRecord{
			ID: *irID, VersionID: *irVID, Version: d.Version.Version,
			Algorithm: *irAlg, Hash: *irHash, CreatedAt: *irAt,
		}
	}
	return d, nil
}

// notFoundOrForbidden distinguishes missing docs from cross-tenant access (RF-02 → 403).
func (r *Repository) notFoundOrForbidden(ctx context.Context, ownerID string, docID uuid.UUID) error {
	var other bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM documents
			WHERE id = $1 AND owner_id <> $2 AND deleted_at IS NULL
		)`,
		docID, ownerID,
	).Scan(&other)
	if err != nil {
		return err
	}
	if other {
		return ErrForbidden
	}
	return ErrNotFound
}

// LatestVersionForFile returns the newest version metadata for download, scoped to owner.
func (r *Repository) LatestVersionForFile(ctx context.Context, ownerID string, docID uuid.UUID) (Version, error) {
	var v Version
	err := r.pool.QueryRow(ctx, `
		SELECT v.id, v.document_id, v.version, v.object_key, v.file_name, v.size_bytes, v.mime_type, v.created_at
		FROM documents d
		JOIN document_versions v ON v.document_id = d.id
		WHERE d.id = $1 AND d.owner_id = $2 AND d.deleted_at IS NULL
		ORDER BY v.version DESC
		LIMIT 1`, docID, ownerID,
	).Scan(&v.ID, &v.DocumentID, &v.Version, &v.ObjectKey, &v.FileName, &v.SizeBytes, &v.MimeType, &v.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, ErrNotFound
	}
	return v, err
}

// NextVersionNumber returns max(version)+1 for an owned document, or ErrNotFound.
func (r *Repository) NextVersionNumber(ctx context.Context, ownerID string, docID uuid.UUID) (int, error) {
	var next int
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(v.version), 0) + 1
		FROM documents d
		LEFT JOIN document_versions v ON v.document_id = d.id
		WHERE d.id = $1 AND d.owner_id = $2 AND d.deleted_at IS NULL
		GROUP BY d.id`, docID, ownerID,
	).Scan(&next)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	return next, err
}

// AddVersion inserts a new version + integrity; does not touch older versions.
func (r *Repository) AddVersion(ctx context.Context, res *UploadResult) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	v, i := &res.Version, &res.Integrity
	if err := tx.QueryRow(ctx, `
		INSERT INTO document_versions (id, document_id, version, object_key, file_name, size_bytes, mime_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at`,
		v.ID, v.DocumentID, v.Version, v.ObjectKey, v.FileName, v.SizeBytes, v.MimeType,
	).Scan(&v.CreatedAt); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO integrity_records (id, version_id, algorithm, hash)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at`,
		i.ID, i.VersionID, i.Algorithm, i.Hash,
	).Scan(&i.CreatedAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ListVersions returns all versions with integrity hash for an owned document.
func (r *Repository) ListVersions(ctx context.Context, ownerID string, docID uuid.UUID) ([]VersionSummary, error) {
	if err := r.OwnedExists(ctx, ownerID, docID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT v.id, v.document_id, v.version, v.object_key, v.file_name, v.size_bytes, v.mime_type, v.created_at,
		       COALESCE(ir.algorithm, ''), COALESCE(ir.hash, '')
		FROM document_versions v
		LEFT JOIN integrity_records ir ON ir.version_id = v.id
		WHERE v.document_id = $1
		ORDER BY v.version DESC`, docID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []VersionSummary{}
	for rows.Next() {
		var item VersionSummary
		if err := rows.Scan(
			&item.ID, &item.DocumentID, &item.Version, &item.ObjectKey, &item.FileName,
			&item.SizeBytes, &item.MimeType, &item.CreatedAt, &item.Algorithm, &item.Hash,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) > 0 {
		out[0].Current = true
	}
	return out, nil
}

// Revoke sets status=revoked with reason, scoped to owner. Caller must validate transition.
func (r *Repository) Revoke(ctx context.Context, ownerID string, docID uuid.UUID, reason string) (Document, error) {
	var d Document
	err := r.pool.QueryRow(ctx, `
		UPDATE documents
		SET status = $3, revoked_reason = $4, revoked_at = now()
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
		RETURNING id, owner_id, title, type, origin, COALESCE(tags, '{}'), status,
		          revoked_reason, revoked_at, expires_at, created_at`,
		docID, ownerID, StatusRevoked, reason,
	).Scan(
		&d.ID, &d.OwnerID, &d.Title, &d.Type, &d.Origin, &d.Tags, &d.Status,
		&d.RevokedReason, &d.RevokedAt, &d.ExpiresAt, &d.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, r.notFoundOrForbidden(ctx, ownerID, docID)
	}
	return d, err
}

// SoftDelete sets deleted_at=now for an owned document (RF-22). Returns ErrNotFound if missing/already deleted.
func (r *Repository) SoftDelete(ctx context.Context, ownerID string, docID uuid.UUID) (Document, error) {
	var d Document
	err := r.pool.QueryRow(ctx, `
		UPDATE documents
		SET deleted_at = now()
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
		RETURNING id, owner_id, title, type, origin, COALESCE(tags, '{}'), status,
		          revoked_reason, revoked_at, expires_at, created_at`,
		docID, ownerID,
	).Scan(
		&d.ID, &d.OwnerID, &d.Title, &d.Type, &d.Origin, &d.Tags, &d.Status,
		&d.RevokedReason, &d.RevokedAt, &d.ExpiresAt, &d.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, r.notFoundOrForbidden(ctx, ownerID, docID)
	}
	return d, err
}

// UpdateMetadata updates document metadata only (RF-17). Never touches versions/integrity.
func (r *Repository) UpdateMetadata(ctx context.Context, ownerID string, docID uuid.UUID, title, docType, origin string, tags []string, expiresAt *time.Time, status string) (Document, error) {
	var d Document
	err := r.pool.QueryRow(ctx, `
		UPDATE documents
		SET title = $3, type = $4, origin = $5, tags = $6, expires_at = $7, status = $8
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
		RETURNING id, owner_id, title, type, origin, COALESCE(tags, '{}'), status,
		          revoked_reason, revoked_at, expires_at, created_at`,
		docID, ownerID, title, docType, origin, tags, expiresAt, status,
	).Scan(
		&d.ID, &d.OwnerID, &d.Title, &d.Type, &d.Origin, &d.Tags, &d.Status,
		&d.RevokedReason, &d.RevokedAt, &d.ExpiresAt, &d.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, r.notFoundOrForbidden(ctx, ownerID, docID)
	}
	return d, err
}

// ExpireDue marks due active/replaced documents as expired. Does not touch integrity hashes.
func (r *Repository) ExpireDue(ctx context.Context) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE documents
		SET status = $1
		WHERE status IN ($2, $3)
		  AND expires_at IS NOT NULL
		  AND expires_at <= now()
		  AND deleted_at IS NULL`,
		StatusExpired, StatusActive, StatusReplaced,
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// OwnedExists returns ErrNotFound if the document is missing or belongs to another owner.
func (r *Repository) OwnedExists(ctx context.Context, ownerID string, docID uuid.UUID) error {
	var one int
	err := r.pool.QueryRow(ctx, `
		SELECT 1 FROM documents
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL`, docID, ownerID,
	).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// InsertAuditEvent persists an audit trail row (RF-21).
func (r *Repository) InsertAuditEvent(ctx context.Context, e *AuditEvent) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO audit_events (id, owner_id, actor, action, document_id, document_title, context)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at`,
		e.ID, e.OwnerID, e.Actor, e.Action, e.DocumentID, e.DocumentTitle, e.Context,
	).Scan(&e.CreatedAt)
}

// ListAuditEvents returns the owner's audit trail, newest first.
func (r *Repository) ListAuditEvents(ctx context.Context, ownerID string) ([]AuditEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, owner_id, actor, action, document_id, document_title, context, created_at
		FROM audit_events
		WHERE owner_id = $1
		ORDER BY created_at DESC
		LIMIT 200`, ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []AuditEvent{}
	for rows.Next() {
		var e AuditEvent
		if err := rows.Scan(
			&e.ID, &e.OwnerID, &e.Actor, &e.Action, &e.DocumentID, &e.DocumentTitle, &e.Context, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// InsertVerification persists a verification evidence row.
func (r *Repository) InsertVerification(ctx context.Context, v *Verification) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO verifications (id, document_id, version_id, result, algorithm, expected_hash, computed_hash, context)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''), $8)
		RETURNING created_at`,
		v.ID, v.DocumentID, v.VersionID, v.Result, v.Algorithm, v.ExpectedHash, v.ComputedHash, v.Context,
	).Scan(&v.CreatedAt)
}

// ListVerifications returns verification history for an owned document.
func (r *Repository) ListVerifications(ctx context.Context, ownerID string, docID uuid.UUID) ([]Verification, error) {
	if err := r.OwnedExists(ctx, ownerID, docID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT vr.id, vr.document_id, vr.version_id, v.version, vr.result, vr.algorithm,
		       COALESCE(vr.expected_hash, ''), COALESCE(vr.computed_hash, ''), vr.context, vr.created_at
		FROM verifications vr
		JOIN document_versions v ON v.id = vr.version_id
		WHERE vr.document_id = $1
		ORDER BY vr.created_at DESC`, docID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Verification{}
	for rows.Next() {
		var item Verification
		if err := rows.Scan(
			&item.ID, &item.DocumentID, &item.VersionID, &item.Version, &item.Result, &item.Algorithm,
			&item.ExpectedHash, &item.ComputedHash, &item.Context, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
