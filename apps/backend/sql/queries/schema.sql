-- Esquema que consume sqlc. Se regenera desde las migraciones con `make db-schema`.
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


