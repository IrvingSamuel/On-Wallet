package collections

import (
	"context"
	"errors"

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

func (r *Repository) Create(ctx context.Context, c *Collection) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO collections (id, owner_id, name)
		VALUES ($1, $2, $3)
		RETURNING created_at, updated_at`,
		c.ID, c.OwnerID, c.Name,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
}

func (r *Repository) List(ctx context.Context, ownerID string) ([]ListItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.owner_id, c.name, c.created_at, c.updated_at,
		       COUNT(cd.document_id)::int
		FROM collections c
		LEFT JOIN collection_documents cd ON cd.collection_id = c.id
		WHERE c.owner_id = $1
		GROUP BY c.id
		ORDER BY c.created_at DESC`,
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ListItem{}
	for rows.Next() {
		var item ListItem
		if err := rows.Scan(
			&item.ID, &item.OwnerID, &item.Name, &item.CreatedAt, &item.UpdatedAt,
			&item.DocumentCount,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, ownerID string, id uuid.UUID) (Detail, error) {
	var d Detail
	err := r.pool.QueryRow(ctx, `
		SELECT id, owner_id, name, created_at, updated_at
		FROM collections
		WHERE id = $1 AND owner_id = $2`,
		id, ownerID,
	).Scan(&d.ID, &d.OwnerID, &d.Name, &d.CreatedAt, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrNotFound
	}
	if err != nil {
		return d, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT cd.document_id
		FROM collection_documents cd
		JOIN documents doc ON doc.id = cd.document_id
		WHERE cd.collection_id = $1 AND doc.owner_id = $2 AND doc.deleted_at IS NULL
		ORDER BY cd.added_at DESC`,
		id, ownerID,
	)
	if err != nil {
		return d, err
	}
	defer rows.Close()

	d.DocumentIDs = []uuid.UUID{}
	for rows.Next() {
		var docID uuid.UUID
		if err := rows.Scan(&docID); err != nil {
			return d, err
		}
		d.DocumentIDs = append(d.DocumentIDs, docID)
	}
	if err := rows.Err(); err != nil {
		return d, err
	}
	d.DocumentCount = len(d.DocumentIDs)
	return d, nil
}

func (r *Repository) Rename(ctx context.Context, ownerID string, id uuid.UUID, name string) (Collection, error) {
	var c Collection
	err := r.pool.QueryRow(ctx, `
		UPDATE collections
		SET name = $3, updated_at = now()
		WHERE id = $1 AND owner_id = $2
		RETURNING id, owner_id, name, created_at, updated_at`,
		id, ownerID, name,
	).Scan(&c.ID, &c.OwnerID, &c.Name, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	return c, err
}

func (r *Repository) Delete(ctx context.Context, ownerID string, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM collections WHERE id = $1 AND owner_id = $2`,
		id, ownerID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// OwnedDocumentIDs returns the subset of ids that belong to the owner and are not soft-deleted.
func (r *Repository) OwnedDocumentIDs(ctx context.Context, ownerID string, ids []uuid.UUID) ([]uuid.UUID, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id FROM documents
		WHERE owner_id = $1 AND deleted_at IS NULL AND id = ANY($2)`,
		ownerID, ids,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *Repository) OwnedCollection(ctx context.Context, ownerID string, id uuid.UUID) error {
	var found uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT id FROM collections WHERE id = $1 AND owner_id = $2`,
		id, ownerID,
	).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (r *Repository) AddDocuments(ctx context.Context, collectionID uuid.UUID, docIDs []uuid.UUID) error {
	if len(docIDs) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, docID := range docIDs {
		batch.Queue(`
			INSERT INTO collection_documents (collection_id, document_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING`,
			collectionID, docID,
		)
	}
	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()
	for range docIDs {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	_, _ = r.pool.Exec(ctx, `UPDATE collections SET updated_at = now() WHERE id = $1`, collectionID)
	return nil
}

func (r *Repository) RemoveDocument(ctx context.Context, ownerID string, collectionID, docID uuid.UUID) error {
	if err := r.OwnedCollection(ctx, ownerID, collectionID); err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM collection_documents
		WHERE collection_id = $1 AND document_id = $2`,
		collectionID, docID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	_, _ = r.pool.Exec(ctx, `UPDATE collections SET updated_at = now() WHERE id = $1`, collectionID)
	return nil
}
