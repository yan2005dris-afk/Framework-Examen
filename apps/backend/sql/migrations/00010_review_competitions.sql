-- +goose Up
-- Registro de períodos para T9. T18 gestionará aprobación, fases y publicación.
-- Sin semilla: no inventar una convocatoria para los expedientes anteriores.
CREATE TABLE competitions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    code VARCHAR(64) NOT NULL UNIQUE CHECK (btrim(code) <> ''),
    title VARCHAR(200) NOT NULL CHECK (btrim(title) <> ''),
    opens_at TIMESTAMPTZ NOT NULL,
    closes_at TIMESTAMPTZ NOT NULL CHECK (closes_at > opens_at),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE submissions ADD COLUMN competition_id UUID REFERENCES competitions(id);
CREATE INDEX submissions_review_idx ON submissions (competition_id, created_at DESC, id DESC)
    WHERE status = 'submitted' AND hash IS NOT NULL AND submitted_at IS NOT NULL;
CREATE INDEX submissions_review_sector_idx ON submissions
    (competition_id, (content->'seccion4'->>'sector'), created_at DESC, id DESC)
    WHERE status = 'submitted' AND hash IS NOT NULL AND submitted_at IS NOT NULL;

-- +goose StatementBegin
CREATE FUNCTION protect_submission_snapshot() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE period competitions;
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.status = 'submitted' THEN
            RAISE EXCEPTION 'El expediente enviado es inmutable';
        END IF;
        RETURN OLD;
    END IF;
    IF TG_OP = 'UPDATE' THEN
        IF OLD.status = 'submitted' AND NEW IS DISTINCT FROM OLD THEN
            RAISE EXCEPTION 'El expediente enviado es inmutable';
        END IF;
        IF NEW.competition_id IS DISTINCT FROM OLD.competition_id THEN
            RAISE EXCEPTION 'La convocatoria de origen es inmutable';
        END IF;
    END IF;
    IF NEW.competition_id IS NOT NULL AND (TG_OP = 'INSERT' OR NEW.status = 'submitted') THEN
        SELECT * INTO period FROM competitions WHERE id = NEW.competition_id FOR SHARE;
        IF NOT FOUND OR statement_timestamp() < period.opens_at OR statement_timestamp() >= period.closes_at THEN
            RAISE EXCEPTION 'La convocatoria no admite postulaciones' USING ERRCODE = '23514', CONSTRAINT = 'competition_open';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER submissions_snapshot BEFORE INSERT OR UPDATE OR DELETE ON submissions
    FOR EACH ROW EXECUTE FUNCTION protect_submission_snapshot();

-- +goose StatementBegin
CREATE FUNCTION protect_sealed_documents() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE parent_id uuid; previous_id uuid;
BEGIN
    parent_id := CASE WHEN TG_OP = 'DELETE' THEN OLD.submission_id ELSE NEW.submission_id END;
    IF TG_OP = 'UPDATE' THEN previous_id := OLD.submission_id; END IF;
    PERFORM id FROM submissions WHERE id IN (parent_id,previous_id) ORDER BY id FOR SHARE;
    IF EXISTS (SELECT 1 FROM submissions WHERE id IN (parent_id,previous_id) AND status = 'submitted') THEN
        RAISE EXCEPTION 'Los certificados de un expediente enviado son inmutables';
    END IF;
    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER documents_sealed BEFORE INSERT OR UPDATE OR DELETE ON documents
    FOR EACH ROW EXECUTE FUNCTION protect_sealed_documents();
CREATE TRIGGER documents_no_truncate BEFORE TRUNCATE ON documents
    FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();

-- +goose StatementBegin
CREATE FUNCTION protect_competition_history() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    -- Código, nombre y período no cambian después de recibir postulaciones.
    IF EXISTS (SELECT 1 FROM submissions WHERE competition_id = OLD.id) THEN
        RAISE EXCEPTION 'La convocatoria con expedientes conserva su período e identidad';
    END IF;
    RETURN CASE WHEN TG_OP = 'DELETE' THEN OLD ELSE NEW END;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER competitions_history BEFORE UPDATE OR DELETE ON competitions
    FOR EACH ROW EXECUTE FUNCTION protect_competition_history();
CREATE TRIGGER submissions_no_truncate BEFORE TRUNCATE ON submissions
    FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER competitions_no_truncate BEFORE TRUNCATE ON competitions
    FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();

-- +goose Down
DROP TRIGGER documents_no_truncate ON documents;
DROP TRIGGER documents_sealed ON documents;
DROP FUNCTION protect_sealed_documents();
DROP TRIGGER competitions_no_truncate ON competitions;
DROP TRIGGER submissions_no_truncate ON submissions;
DROP TRIGGER competitions_history ON competitions;
DROP FUNCTION protect_competition_history();
DROP TRIGGER submissions_snapshot ON submissions;
DROP FUNCTION protect_submission_snapshot();
DROP INDEX submissions_review_sector_idx;
DROP INDEX submissions_review_idx;
ALTER TABLE submissions DROP COLUMN competition_id;
DROP TABLE competitions;
