CREATE TABLE verifications (
    id             uuid PRIMARY KEY,
    document_id    uuid        NOT NULL REFERENCES documents(id),
    version_id     uuid        NOT NULL REFERENCES document_versions(id),
    result         text        NOT NULL
                   CHECK (result IN ('ok', 'divergent', 'unverifiable', 'missing_record')),
    algorithm      text        NOT NULL,
    expected_hash  text,
    computed_hash  text,
    context        text        NOT NULL DEFAULT 'owner',
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX verifications_document_created_idx
    ON verifications (document_id, created_at DESC);
