-- +goose Up
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    section INTEGER NOT NULL CHECK (section BETWEEN 4 AND 8),
    record_id VARCHAR(255) NOT NULL DEFAULT '',
    name VARCHAR(255) NOT NULL,
    mime VARCHAR(64) NOT NULL CHECK (mime = 'application/pdf'),
    size_bytes BIGINT NOT NULL CHECK (size_bytes > 0 AND size_bytes <= CASE WHEN section = 4 THEN 4194304 ELSE 2097152 END),
    sha256 VARCHAR(64) NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    storage_key TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (submission_id, section, record_id),
    CHECK ((section = 4 AND record_id = '') OR (section <> 4 AND record_id <> ''))
);

-- Las bajas, incluidas las cascadas, se confirman junto a los metadatos.
CREATE TABLE document_deletions (
    storage_key TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose StatementBegin
CREATE FUNCTION enqueue_document_deletion() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO document_deletions (storage_key) VALUES (OLD.storage_key) ON CONFLICT DO NOTHING;
    RETURN OLD;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER document_deletion_queue AFTER DELETE ON documents
FOR EACH ROW EXECUTE FUNCTION enqueue_document_deletion();

-- +goose Down
DROP TRIGGER IF EXISTS document_deletion_queue ON documents;
DROP FUNCTION IF EXISTS enqueue_document_deletion();
DROP TABLE IF EXISTS document_deletions;
DROP TABLE IF EXISTS documents;
