-- +goose Up
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

-- +goose Down
DROP TABLE IF EXISTS password_resets;
DROP TABLE IF EXISTS sessions;
ALTER TABLE applicants DROP COLUMN IF EXISTS is_verified;
ALTER TABLE applicants DROP COLUMN IF EXISTS password_hash;

