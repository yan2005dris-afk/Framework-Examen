-- +goose Up
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

-- +goose Down
DROP INDEX IF EXISTS submissions_applicant_idx;
CREATE INDEX submissions_applicant_idx ON submissions (applicant_id, updated_at DESC);

ALTER TABLE applicants ALTER COLUMN id DROP DEFAULT;
ALTER TABLE submissions ALTER COLUMN id DROP DEFAULT;
