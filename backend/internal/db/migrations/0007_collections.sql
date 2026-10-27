CREATE TABLE collections (
    id          uuid PRIMARY KEY,
    owner_id    text        NOT NULL,
    name        text        NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX collections_owner_idx ON collections (owner_id, created_at DESC);

CREATE TABLE collection_documents (
    collection_id uuid NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    document_id   uuid NOT NULL REFERENCES documents(id),
    added_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (collection_id, document_id)
);

CREATE INDEX collection_documents_document_idx ON collection_documents (document_id);
