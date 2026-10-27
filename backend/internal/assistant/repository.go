package assistant

import (
	"context"
	"encoding/json"
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

type docVersionRow struct {
	DocumentID uuid.UUID
	VersionID  uuid.UUID
	OwnerID    string
	Title      string
	Type       string
	Origin     string
	Tags       []string
	ObjectKey  string
	FileName   string
	MimeType   string
	SizeBytes  int64
	Version    int
}

func (r *Repository) GetOwnedLatest(ctx context.Context, ownerID string, docID uuid.UUID) (docVersionRow, error) {
	var row docVersionRow
	err := r.pool.QueryRow(ctx, `
		SELECT d.id, v.id, d.owner_id, d.title, d.type, d.origin, COALESCE(d.tags, '{}'),
		       v.object_key, v.file_name, v.mime_type, v.size_bytes, v.version
		FROM documents d
		JOIN LATERAL (
			SELECT id, object_key, file_name, mime_type, size_bytes, version
			FROM document_versions WHERE document_id = d.id
			ORDER BY version DESC LIMIT 1
		) v ON true
		WHERE d.id = $1 AND d.owner_id = $2 AND d.deleted_at IS NULL`,
		docID, ownerID,
	).Scan(
		&row.DocumentID, &row.VersionID, &row.OwnerID, &row.Title, &row.Type, &row.Origin, &row.Tags,
		&row.ObjectKey, &row.FileName, &row.MimeType, &row.SizeBytes, &row.Version,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		var other bool
		_ = r.pool.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM documents WHERE id=$1 AND owner_id<>$2 AND deleted_at IS NULL)`,
			docID, ownerID,
		).Scan(&other)
		if other {
			return row, ErrForbidden
		}
		return row, ErrDocumentGone
	}
	return row, err
}

func (r *Repository) UpsertExtractionRunning(ctx context.Context, e *Extraction) error {
	now := time.Now().UTC()
	e.UpdatedAt = now
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	if e.Fields == nil {
		e.Fields = json.RawMessage(`{}`)
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO document_extractions (id, document_id, version_id, owner_id, status, raw_text, fields_json, error, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (version_id) DO UPDATE SET
			id = EXCLUDED.id,
			status = EXCLUDED.status,
			raw_text = EXCLUDED.raw_text,
			fields_json = EXCLUDED.fields_json,
			error = EXCLUDED.error,
			updated_at = EXCLUDED.updated_at`,
		e.ID, e.DocumentID, e.VersionID, e.OwnerID, e.Status, e.RawText, e.Fields, e.Error, e.CreatedAt, e.UpdatedAt,
	)
	return err
}

func (r *Repository) LatestExtraction(ctx context.Context, ownerID string, docID uuid.UUID) (*Extraction, error) {
	var e Extraction
	var fields []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, document_id, version_id, owner_id, status, raw_text, fields_json, error, created_at, updated_at
		FROM document_extractions
		WHERE document_id = $1 AND owner_id = $2
		ORDER BY updated_at DESC LIMIT 1`,
		docID, ownerID,
	).Scan(&e.ID, &e.DocumentID, &e.VersionID, &e.OwnerID, &e.Status, &e.RawText, &fields, &e.Error, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	e.Fields = fields
	return &e, nil
}

func (r *Repository) InsertConsistency(ctx context.Context, c *ConsistencyCheck) error {
	if c.Details == nil {
		c.Details = json.RawMessage(`{}`)
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO consistency_checks (id, document_id, version_id, owner_id, result, summary, details_json, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		c.ID, c.DocumentID, c.VersionID, c.OwnerID, c.Result, c.Summary, c.Details, c.CreatedAt,
	)
	return err
}

func (r *Repository) LatestConsistency(ctx context.Context, ownerID string, docID uuid.UUID) (*ConsistencyCheck, error) {
	var c ConsistencyCheck
	var details []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, document_id, version_id, owner_id, result, summary, details_json, created_at
		FROM consistency_checks
		WHERE document_id = $1 AND owner_id = $2
		ORDER BY created_at DESC LIMIT 1`,
		docID, ownerID,
	).Scan(&c.ID, &c.DocumentID, &c.VersionID, &c.OwnerID, &c.Result, &c.Summary, &details, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.Details = details
	return &c, nil
}

func (r *Repository) ReplaceDocumentChunks(ctx context.Context, ownerID string, docID, versionID uuid.UUID, title, body string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		DELETE FROM assistant_chunks WHERE source = 'document' AND document_id = $1`, docID); err != nil {
		return err
	}
	if body == "" {
		return tx.Commit(ctx)
	}
	id := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO assistant_chunks (id, source, owner_id, document_id, version_id, title, body, tsv)
		VALUES ($1,'document',$2,$3,$4,$5,$6, to_tsvector('portuguese', coalesce($5,'') || ' ' || coalesce($6,'')))`,
		id, ownerID, docID, versionID, title, body,
	)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) SearchChunks(ctx context.Context, ownerID, query string, limit int) ([]Chunk, error) {
	if limit <= 0 {
		limit = 6
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, source, owner_id, document_id, version_id, title, body
		FROM assistant_chunks
		WHERE (source = 'knowledge' OR (source = 'document' AND owner_id = $1))
		  AND tsv @@ plainto_tsquery('portuguese', $2)
		ORDER BY ts_rank(tsv, plainto_tsquery('portuguese', $2)) DESC
		LIMIT $3`,
		ownerID, query, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Chunk
	for rows.Next() {
		var c Chunk
		if err := rows.Scan(&c.ID, &c.Source, &c.OwnerID, &c.DocumentID, &c.VersionID, &c.Title, &c.Body); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if len(out) > 0 {
		return out, rows.Err()
	}
	// Fallback: recent knowledge + latest doc chunk if query matches poorly.
	rows2, err := r.pool.Query(ctx, `
		(
			SELECT id, source, owner_id, document_id, version_id, title, body
			FROM assistant_chunks WHERE source = 'knowledge' ORDER BY created_at LIMIT 3
		)
		UNION ALL
		(
			SELECT id, source, owner_id, document_id, version_id, title, body
			FROM assistant_chunks WHERE source = 'document' AND owner_id = $1
			ORDER BY created_at DESC LIMIT 2
		)`,
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var c Chunk
		if err := rows2.Scan(&c.ID, &c.Source, &c.OwnerID, &c.DocumentID, &c.VersionID, &c.Title, &c.Body); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows2.Err()
}
