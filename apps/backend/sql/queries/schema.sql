-- Esquema que consume sqlc. Se regenera desde las migraciones con make db-schema.

CREATE TABLE applicants (
    id                UUID PRIMARY KEY,
    cedula            VARCHAR(10)  NOT NULL UNIQUE,
    nombres           VARCHAR(120) NOT NULL,
    apellidos         VARCHAR(120) NOT NULL,
    correo            VARCHAR(255) NOT NULL UNIQUE,
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

CREATE INDEX submissions_applicant_idx ON submissions (applicant_id, updated_at DESC);



-- Identificadores ordenados en el tiempo (RFC 9562, UUIDv7).
--
-- La función uuidv7() es nativa de PostgreSQL 18: el motor debe estar
-- actualizado antes de aplicar esta migración.
--
-- El índice incluye el identificador porque, al ser los UUIDv7 ordenados
-- por fecha de creación, el desempate de la paginación por cursor deja de
-- ser arbitrario y puede resolverse dentro del índice.

ALTER TABLE applicants ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE submissions ALTER COLUMN id SET DEFAULT uuidv7();

DROP INDEX IF EXISTS submissions_applicant_idx;
CREATE INDEX submissions_applicant_idx ON submissions (applicant_id, updated_at DESC, id DESC);



-- Contexto de identidad: contraseñas con Argon2id, sesiones en PostgreSQL
-- y solicitudes de restablecimiento de contraseña.

ALTER TABLE applicants ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE applicants ADD COLUMN IF NOT EXISTS is_verified BOOLEAN NOT NULL DEFAULT true;

CREATE TABLE IF NOT EXISTS sessions (
    id           UUID PRIMARY KEY DEFAULT uuidv7(),
    applicant_id UUID NOT NULL REFERENCES applicants (id) ON DELETE CASCADE,
    token_hash   CHAR(64) NOT NULL UNIQUE,
    expires_at   TIMESTAMPTZ NOT NULL,
    ip           VARCHAR(45),
    user_agent   VARCHAR(512),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS sessions_applicant_idx ON sessions (applicant_id);
CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE IF NOT EXISTS password_resets (
    id           UUID PRIMARY KEY DEFAULT uuidv7(),
    applicant_id UUID NOT NULL REFERENCES applicants (id) ON DELETE CASCADE,
    token_hash   CHAR(64) NOT NULL UNIQUE,
    expires_at   TIMESTAMPTZ NOT NULL,
    used         BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS password_resets_applicant_idx ON password_resets (applicant_id);



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



ALTER TABLE applicants
    ADD COLUMN role VARCHAR(32) NOT NULL DEFAULT 'postulante'
    CONSTRAINT applicants_role_check CHECK (role IN ('postulante', 'analista'));



CREATE TABLE submission_events (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    applicant_id UUID NOT NULL REFERENCES applicants(id),
    event_type VARCHAR(16) NOT NULL CHECK (event_type IN ('guardada', 'enviada', 'consultada')),
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX submission_events_submission_idx
    ON submission_events (submission_id, created_at, id);



ALTER TABLE applicants DROP CONSTRAINT applicants_role_check;
ALTER TABLE applicants ADD CONSTRAINT applicants_role_check
    CHECK (role IN ('postulante', 'analista', 'administrador'));

CREATE TABLE administrative_access_events (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    actor_id UUID NOT NULL,
    actor_role VARCHAR(32) NOT NULL CHECK (actor_role IN ('analista', 'administrador')),
    action VARCHAR(64) NOT NULL,
    resource_id UUID NOT NULL,
    resource_type VARCHAR(32) NOT NULL,
    result VARCHAR(16) NOT NULL CHECK (result IN ('permitido', 'rechazado')),
    created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX administrative_access_resource_idx
    ON administrative_access_events (resource_type, resource_id, created_at DESC, id DESC);

CREATE TABLE applicant_role_events (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    applicant_id UUID NOT NULL,
    actor_id UUID,
    operator TEXT NOT NULL,
    previous_role VARCHAR(32) NOT NULL,
    new_role VARCHAR(32) NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX applicant_role_events_applicant_idx
    ON applicant_role_events (applicant_id, created_at DESC, id DESC);

-- +goose StatementBegin
CREATE FUNCTION record_applicant_role_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.role IS DISTINCT FROM NEW.role THEN
        INSERT INTO applicant_role_events
            (applicant_id, actor_id, operator, previous_role, new_role, reason)
        VALUES (NEW.id, NULLIF(current_setting('app.actor_id', true), '')::uuid,
            COALESCE(NULLIF(current_setting('app.operator', true), ''), current_user),
            OLD.role, NEW.role,
            COALESCE(NULLIF(current_setting('app.role_reason', true), ''), 'Actualización SQL por operador de base de datos'));
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER applicants_role_audit AFTER UPDATE OF role ON applicants
    FOR EACH ROW EXECUTE FUNCTION record_applicant_role_change();

-- +goose StatementBegin
CREATE FUNCTION protect_authorization_history() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'El historial de autorización no admite modificaciones';
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER administrative_access_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON administrative_access_events
    FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER applicant_role_events_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON applicant_role_events
    FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
