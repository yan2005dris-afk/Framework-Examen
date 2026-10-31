-- Esquema que consume sqlc. Se regenera desde las migraciones con `make db-schema`.
CREATE TABLE applicants (
    id                UUID PRIMARY KEY,
    cedula            VARCHAR(10)  NOT NULL UNIQUE,
    nombres           VARCHAR(120) NOT NULL,
    apellidos         VARCHAR(120) NOT NULL,
    correo            VARCHAR(255) NOT NULL UNIQUE,
    password_hash     VARCHAR(255) NOT NULL DEFAULT '',
    is_verified       BOOLEAN      NOT NULL DEFAULT true,
    metodo_validacion VARCHAR(32)  NOT NULL DEFAULT 'fecha_expedicion',
    fecha_registro    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE submissions (
    id           UUID PRIMARY KEY,
    applicant_id UUID        NOT NULL REFERENCES applicants (id) ON DELETE CASCADE,
    code         VARCHAR(32) NOT NULL UNIQUE,
    status       VARCHAR(16) NOT NULL DEFAULT 'draft',
    content      JSONB       NOT NULL DEFAULT '{}'::jsonb,
    hash         VARCHAR(64),
    submitted_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT submissions_status_check CHECK (status IN ('draft', 'in_review', 'submitted'))
);

CREATE INDEX submissions_applicant_idx ON submissions (applicant_id, updated_at DESC, id DESC);

CREATE TABLE sessions (
    id           UUID PRIMARY KEY DEFAULT uuidv7(),
    applicant_id UUID NOT NULL REFERENCES applicants (id) ON DELETE CASCADE,
    token_hash   CHAR(64) NOT NULL UNIQUE,
    expires_at   TIMESTAMPTZ NOT NULL,
    ip           VARCHAR(45),
    user_agent   VARCHAR(512),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX sessions_applicant_idx ON sessions (applicant_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE password_resets (
    id           UUID PRIMARY KEY DEFAULT uuidv7(),
    applicant_id UUID NOT NULL REFERENCES applicants (id) ON DELETE CASCADE,
    token_hash   CHAR(64) NOT NULL UNIQUE,
    expires_at   TIMESTAMPTZ NOT NULL,
    used         BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX password_resets_applicant_idx ON password_resets (applicant_id);

ALTER TABLE applicants ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE submissions ALTER COLUMN id SET DEFAULT uuidv7();


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
