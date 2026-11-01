-- +goose Up
CREATE TABLE exam_bank_streams(competition_id UUID NOT NULL REFERENCES competitions(id),sector TEXT NOT NULL,version BIGINT NOT NULL DEFAULT 0,PRIMARY KEY(competition_id,sector));
CREATE TABLE exam_banks(id UUID PRIMARY KEY,competition_id UUID NOT NULL REFERENCES competitions(id),sector TEXT NOT NULL,version BIGINT NOT NULL,definition JSONB NOT NULL,hash VARCHAR(64) NOT NULL,actor_id UUID NOT NULL,created_at TIMESTAMPTZ NOT NULL,UNIQUE(competition_id,sector,version),UNIQUE(id,competition_id,sector));
CREATE TABLE exam_configs(id UUID PRIMARY KEY,bank_id UUID NOT NULL UNIQUE REFERENCES exam_banks(id),competition_id UUID NOT NULL,sector TEXT NOT NULL,definition JSONB NOT NULL,hash VARCHAR(64) NOT NULL,actor_id UUID NOT NULL,created_at TIMESTAMPTZ NOT NULL,FOREIGN KEY(bank_id,competition_id,sector) REFERENCES exam_banks(id,competition_id,sector));
CREATE TABLE exam_enrollments(submission_id UUID PRIMARY KEY REFERENCES submissions(id),config_id UUID NOT NULL REFERENCES exam_configs(id),merit_id UUID NOT NULL REFERENCES merit_publications(id),actor_id UUID NOT NULL,created_at TIMESTAMPTZ NOT NULL);
CREATE TABLE exam_attempts(id UUID PRIMARY KEY,submission_id UUID NOT NULL UNIQUE REFERENCES submissions(id),config_id UUID NOT NULL REFERENCES exam_configs(id),assignment_hash VARCHAR(64) NOT NULL,payload JSONB NOT NULL,version BIGINT NOT NULL,status TEXT NOT NULL CHECK(status IN ('active','finished')),expires_at TIMESTAMPTZ NOT NULL,created_at TIMESTAMPTZ NOT NULL,UNIQUE(config_id,assignment_hash));
CREATE INDEX exam_due ON exam_attempts(expires_at,id) WHERE status='active';
CREATE TABLE exam_events(id UUID PRIMARY KEY DEFAULT uuidv7(),attempt_id UUID NOT NULL REFERENCES exam_attempts(id),version BIGINT NOT NULL,payload JSONB NOT NULL,created_at TIMESTAMPTZ NOT NULL,UNIQUE(attempt_id,version));
CREATE TABLE exam_reports(attempt_id UUID PRIMARY KEY REFERENCES exam_attempts(id),submission_id UUID NOT NULL REFERENCES submissions(id),result_id UUID NOT NULL UNIQUE,pdf BYTEA NOT NULL CHECK(octet_length(pdf)<=4194304),sha256 VARCHAR(64) NOT NULL,created_at TIMESTAMPTZ NOT NULL);
CREATE TRIGGER exam_banks_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON exam_banks FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER exam_configs_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON exam_configs FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER exam_enrollments_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON exam_enrollments FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER exam_events_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON exam_events FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER exam_reports_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON exam_reports FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
-- +goose StatementBegin
CREATE FUNCTION protect_exam_attempt() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF TG_OP='DELETE' OR OLD.status='finished' OR (NEW.id,NEW.submission_id,NEW.config_id,NEW.assignment_hash,NEW.expires_at,NEW.created_at) IS DISTINCT FROM (OLD.id,OLD.submission_id,OLD.config_id,OLD.assignment_hash,OLD.expires_at,OLD.created_at) OR (NEW.payload->'matriz',NEW.payload->'configuracion',NEW.payload->'hashBanco',NEW.payload->'hashEnvio',NEW.payload->'meritosId',NEW.payload->'iniciadaEn',NEW.payload->'venceEn',NEW.payload->'postulacionId') IS DISTINCT FROM (OLD.payload->'matriz',OLD.payload->'configuracion',OLD.payload->'hashBanco',OLD.payload->'hashEnvio',OLD.payload->'meritosId',OLD.payload->'iniciadaEn',OLD.payload->'venceEn',OLD.payload->'postulacionId') OR (NEW.payload->>'version')::bigint<>NEW.version OR NEW.payload->>'estado'<>NEW.status OR NEW.version<>OLD.version+1 THEN RAISE EXCEPTION 'El intento conserva asignación, plazo e historial'; END IF;RETURN NEW;END;$$;
-- +goose StatementEnd
CREATE TRIGGER exam_attempt_protection BEFORE UPDATE OR DELETE ON exam_attempts FOR EACH ROW EXECUTE FUNCTION protect_exam_attempt();
CREATE TRIGGER exam_attempt_no_truncate BEFORE TRUNCATE ON exam_attempts FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
ALTER TABLE workflow_events DROP CONSTRAINT workflow_events_action_check;
ALTER TABLE workflow_events ADD CONSTRAINT workflow_events_action_check CHECK(action IN (
 'convocatoria.creada','convocatoria.configurada','fase.publicada','fase.habilitada','fase.cerrada','elegibilidad.verificada',
 'meritos.reglas_creadas','meritos.reglas_aprobadas','meritos.revisados','meritos.calificados','evaluacion.iniciada','evaluacion.finalizada',
 'reconsideracion.presentada','reconsideracion.resuelta','resultado.publicado','evaluacion.reglas_creadas','evaluacion.reglas_aprobadas',
 'psicotecnica.registrada','psicotecnica.publicada','expediente.consolidado','reconsideracion.atendida','notificacion.reintentada','lista.generada','lista.publicada',
 'banco.versionado','examen.configurado','evaluacion.habilitada','privacidad.publicada','privacidad.restringida'));
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM exam_banks) OR EXISTS(SELECT 1 FROM exam_attempts) THEN RAISE EXCEPTION 'Hay banco o evaluaciones históricas';END IF;END;$$;
-- +goose StatementEnd
DROP TABLE exam_reports;
DROP TABLE exam_events;
DROP TABLE exam_attempts;
DROP FUNCTION protect_exam_attempt();
DROP TABLE exam_enrollments;
DROP TABLE exam_configs;
DROP TABLE exam_banks;
DROP TABLE exam_bank_streams;
