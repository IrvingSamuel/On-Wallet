ALTER TABLE documents
    ADD COLUMN revoked_reason text,
    ADD COLUMN revoked_at timestamptz;
