-- +goose Up
ALTER TABLE applicants
    ADD COLUMN role VARCHAR(32) NOT NULL DEFAULT 'postulante'
    CONSTRAINT applicants_role_check CHECK (role IN ('postulante', 'analista'));

-- +goose Down
ALTER TABLE applicants DROP COLUMN role;
