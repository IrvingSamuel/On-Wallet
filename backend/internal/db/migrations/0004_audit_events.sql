CREATE TABLE audit_events (
    id             uuid PRIMARY KEY,
    owner_id       text        NOT NULL,
    actor          text        NOT NULL,
    action         text        NOT NULL,
    document_id    uuid        REFERENCES documents(id),
    document_title text        NOT NULL DEFAULT '',
    context        text        NOT NULL DEFAULT '',
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX audit_events_owner_created_idx
    ON audit_events (owner_id, created_at DESC);
