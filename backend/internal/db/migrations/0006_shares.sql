CREATE TABLE shares (
    id           uuid PRIMARY KEY,
    owner_id     text        NOT NULL,
    document_id  uuid        NOT NULL REFERENCES documents(id),
    version_id   uuid        NOT NULL REFERENCES document_versions(id),
    token        text        NOT NULL UNIQUE,
    expires_at   timestamptz NOT NULL,
    revoked_at   timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX shares_document_idx ON shares (document_id, created_at DESC);
CREATE INDEX shares_token_idx ON shares (token) WHERE revoked_at IS NULL;
