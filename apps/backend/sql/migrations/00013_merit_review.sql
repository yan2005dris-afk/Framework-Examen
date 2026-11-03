-- +goose Up
CREATE TABLE merit_rule_streams (
 competition_id UUID NOT NULL REFERENCES competitions(id), sector TEXT NOT NULL CHECK(sector IN ('afiliados','jubilados','empleadores')),
 version BIGINT NOT NULL DEFAULT 0, active_rule_id UUID, PRIMARY KEY(competition_id,sector)
);
CREATE TABLE merit_rule_sets (
 id UUID PRIMARY KEY DEFAULT uuidv7(), competition_id UUID NOT NULL REFERENCES competitions(id), sector TEXT NOT NULL,
 version BIGINT NOT NULL CHECK(version>0), definition JSONB NOT NULL, definition_hash VARCHAR(64) NOT NULL,
 approved BOOLEAN NOT NULL DEFAULT false, approval_reference VARCHAR(2000), approved_by UUID, approved_at TIMESTAMPTZ,
 actor_id UUID NOT NULL, created_at TIMESTAMPTZ NOT NULL, UNIQUE(competition_id,sector,version), UNIQUE(id,competition_id,sector),
 CHECK(NOT approved OR (approval_reference IS NOT NULL AND approved_by IS NOT NULL AND approved_at IS NOT NULL))
);
ALTER TABLE merit_rule_streams ADD CONSTRAINT merit_active_rule FOREIGN KEY(active_rule_id,competition_id,sector) REFERENCES merit_rule_sets(id,competition_id,sector);
CREATE TABLE merit_case_streams (
 submission_id UUID PRIMARY KEY REFERENCES submissions(id), review_id UUID, calculation_id UUID, publication_id UUID
);
CREATE TABLE merit_reviews (
 id UUID PRIMARY KEY, submission_id UUID NOT NULL REFERENCES submissions(id), version BIGINT NOT NULL CHECK(version>0),
 rule_set_id UUID NOT NULL REFERENCES merit_rule_sets(id), payload JSONB NOT NULL, UNIQUE(submission_id,version), UNIQUE(id,submission_id)
);
CREATE TABLE merit_calculations (
 id UUID PRIMARY KEY, submission_id UUID NOT NULL REFERENCES submissions(id), version BIGINT NOT NULL CHECK(version>0),
 review_id UUID NOT NULL, rule_set_id UUID NOT NULL REFERENCES merit_rule_sets(id),
 payload JSONB NOT NULL, UNIQUE(submission_id,version), UNIQUE(id,submission_id),
 FOREIGN KEY(review_id,submission_id) REFERENCES merit_reviews(id,submission_id)
);
CREATE TABLE merit_publications (
 id UUID PRIMARY KEY, submission_id UUID NOT NULL REFERENCES submissions(id), version BIGINT NOT NULL CHECK(version>0),
 calculation_id UUID NOT NULL, supersedes_id UUID,
 decision TEXT NOT NULL CHECK(decision IN ('avanza','excluido')), payload JSONB NOT NULL, UNIQUE(submission_id,version), UNIQUE(id,submission_id),
 FOREIGN KEY(calculation_id,submission_id) REFERENCES merit_calculations(id,submission_id),
 FOREIGN KEY(supersedes_id,submission_id) REFERENCES merit_publications(id,submission_id)
);
ALTER TABLE merit_case_streams ADD FOREIGN KEY(review_id,submission_id) REFERENCES merit_reviews(id,submission_id);
ALTER TABLE merit_case_streams ADD FOREIGN KEY(calculation_id,submission_id) REFERENCES merit_calculations(id,submission_id);
ALTER TABLE merit_case_streams ADD FOREIGN KEY(publication_id,submission_id) REFERENCES merit_publications(id,submission_id);

CREATE TRIGGER merit_reviews_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON merit_reviews FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER merit_calculations_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON merit_calculations FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER merit_publications_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON merit_publications FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER merit_rules_no_delete BEFORE DELETE OR TRUNCATE ON merit_rule_sets FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
-- +goose StatementBegin
CREATE FUNCTION protect_merit_rules() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF OLD.approved OR NOT NEW.approved OR (NEW.id,NEW.competition_id,NEW.sector,NEW.version,NEW.definition,NEW.definition_hash,NEW.actor_id,NEW.created_at)
 IS DISTINCT FROM (OLD.id,OLD.competition_id,OLD.sector,OLD.version,OLD.definition,OLD.definition_hash,OLD.actor_id,OLD.created_at) THEN
  RAISE EXCEPTION 'Las reglas se versionan; una aprobación no modifica su definición';
 END IF;
 RETURN NEW;
END; $$;
-- +goose StatementEnd
CREATE TRIGGER merit_rules_immutable BEFORE UPDATE ON merit_rule_sets FOR EACH ROW EXECUTE FUNCTION protect_merit_rules();
ALTER TABLE workflow_events DROP CONSTRAINT workflow_events_action_check;
ALTER TABLE workflow_events ADD CONSTRAINT workflow_events_action_check CHECK(action IN (
 'convocatoria.creada','convocatoria.configurada','fase.publicada','fase.habilitada','fase.cerrada','elegibilidad.verificada',
 'meritos.reglas_creadas','meritos.reglas_aprobadas','meritos.revisados','meritos.calificados','evaluacion.iniciada','evaluacion.finalizada',
 'reconsideracion.presentada','reconsideracion.resuelta','resultado.publicado'));

-- +goose Down
-- Una reversión no debe eliminar resoluciones o evidencias. Requiere un plan explícito si ya hay datos.
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM merit_rule_sets) OR EXISTS(SELECT 1 FROM merit_reviews) OR EXISTS(SELECT 1 FROM merit_publications) THEN RAISE EXCEPTION 'Hay evidencia de méritos; no se elimina mediante Down'; END IF; END $$;
-- +goose StatementEnd
DROP TABLE merit_case_streams;
DROP TABLE merit_publications;
DROP TABLE merit_calculations;
DROP TABLE merit_reviews;
DROP TABLE merit_rule_streams;
DROP TABLE merit_rule_sets;
DROP FUNCTION protect_merit_rules();
