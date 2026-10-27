CREATE TABLE documents (
    id          uuid PRIMARY KEY,
    owner_id    text        NOT NULL,
    title       text        NOT NULL,
    type        text        NOT NULL,
    origin      text        NOT NULL,
    tags        text[]      NOT NULL DEFAULT '{}',
    status      text        NOT NULL DEFAULT 'active'
                CHECK (status IN ('active', 'replaced', 'revoked', 'expired')),
    created_at  timestamptz NOT NULL DEFAULT now(),
    deleted_at  timestamptz
);
CREATE INDEX documents_owner_idx ON documents (owner_id) WHERE deleted_at IS NULL;

CREATE TABLE document_versions (
    id           uuid PRIMARY KEY,
    document_id  uuid        NOT NULL REFERENCES documents(id),
    version      int         NOT NULL,
    object_key   text        NOT NULL,
    file_name    text        NOT NULL,
    size_bytes   bigint      NOT NULL,
    mime_type    text        NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (document_id, version)
);

CREATE TABLE integrity_records (
    id          uuid PRIMARY KEY,
    version_id  uuid        NOT NULL UNIQUE REFERENCES document_versions(id),
    algorithm   text        NOT NULL,
    hash        text        NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);
