-- +goose Up
CREATE TABLE evaluation_policy_streams (
 competition_id UUID NOT NULL REFERENCES competitions(id),sector TEXT NOT NULL CHECK(sector IN ('afiliados','jubilados','empleadores')),
 kind TEXT NOT NULL CHECK(kind IN ('oposicion','reconsideracion')),version BIGINT NOT NULL DEFAULT 0,active_id UUID,
 PRIMARY KEY(competition_id,sector,kind)
);
CREATE TABLE evaluation_policies (
 id UUID PRIMARY KEY,competition_id UUID NOT NULL REFERENCES competitions(id),sector TEXT NOT NULL,kind TEXT NOT NULL,
 version BIGINT NOT NULL CHECK(version>0),definition JSONB NOT NULL,definition_hash VARCHAR(64) NOT NULL,approved BOOLEAN NOT NULL DEFAULT false,
 approval_reference TEXT,approved_by UUID,approved_at TIMESTAMPTZ,actor_id UUID NOT NULL,created_at TIMESTAMPTZ NOT NULL,
 UNIQUE(competition_id,sector,kind,version),UNIQUE(id,competition_id,sector,kind),
 CHECK(NOT approved OR (approval_reference IS NOT NULL AND approved_by IS NOT NULL AND approved_at IS NOT NULL))
);
ALTER TABLE evaluation_policy_streams ADD FOREIGN KEY(active_id,competition_id,sector,kind) REFERENCES evaluation_policies(id,competition_id,sector,kind);
CREATE TABLE evaluation_heads (
 submission_id UUID PRIMARY KEY REFERENCES submissions(id),version BIGINT NOT NULL DEFAULT 0,
 state JSONB NOT NULL DEFAULT '{"resultados":{}}'
);
CREATE TABLE evaluation_records (
 id UUID PRIMARY KEY,submission_id UUID NOT NULL REFERENCES submissions(id),action TEXT NOT NULL,
 phase TEXT NOT NULL CHECK(phase IN ('meritos','oposicion_teorica','oposicion_psicotecnica','resultados')),
 payload JSONB NOT NULL,created_at TIMESTAMPTZ NOT NULL,
 CHECK(action IN ('psicotecnica.registrada','psicotecnica.publicada','teorica.finalizada','expediente.consolidado','reconsideracion.presentada','reconsideracion.atendida','reconsideracion.resuelta'))
);
CREATE INDEX evaluation_history_idx ON evaluation_records(submission_id,created_at DESC,id DESC);
CREATE UNIQUE INDEX evaluation_result_version ON evaluation_records(submission_id,phase,((payload->>'version')::bigint)) WHERE action IN ('psicotecnica.publicada','teorica.finalizada','expediente.consolidado');
CREATE UNIQUE INDEX evaluation_theory_attempt ON evaluation_records(submission_id,(payload->>'fuenteId')) WHERE action='teorica.finalizada';
CREATE TABLE evaluation_commands (
 id UUID PRIMARY KEY,actor_id UUID NOT NULL,submission_id UUID NOT NULL REFERENCES submissions(id),operation TEXT NOT NULL,resource_id UUID NOT NULL,
 request_hash VARCHAR(64) NOT NULL,response JSONB NOT NULL
);
CREATE TABLE reconsiderations (
 id UUID PRIMARY KEY,submission_id UUID NOT NULL REFERENCES submissions(id),phase TEXT NOT NULL,result_id UUID NOT NULL,
 policy_id UUID NOT NULL REFERENCES evaluation_policies(id),status TEXT NOT NULL CHECK(status IN ('recibida','atendida','resuelta')),
 version BIGINT NOT NULL CHECK(version>0),payload JSONB NOT NULL,created_at TIMESTAMPTZ NOT NULL,
 UNIQUE(submission_id,phase,result_id)
);
CREATE INDEX reconsiderations_page ON reconsiderations(submission_id,created_at DESC,id DESC);
CREATE TABLE evaluation_evidence (
 id UUID PRIMARY KEY,submission_id UUID NOT NULL REFERENCES submissions(id),applicant_id UUID NOT NULL,actor_id UUID NOT NULL,
 applicant_upload BOOLEAN NOT NULL,name TEXT NOT NULL,size_bytes BIGINT NOT NULL CHECK(size_bytes>0 AND size_bytes<=2097152),
 sha256 VARCHAR(64) NOT NULL,storage_key TEXT NOT NULL UNIQUE,created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX evaluation_evidence_quota ON evaluation_evidence(applicant_id) WHERE applicant_upload;
CREATE TABLE evaluation_upload_intents (
 id UUID PRIMARY KEY,submission_id UUID NOT NULL REFERENCES submissions(id),storage_key TEXT NOT NULL UNIQUE,expires_at TIMESTAMPTZ NOT NULL
);
CREATE TRIGGER evaluation_records_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON evaluation_records FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER evaluation_commands_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON evaluation_commands FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER evaluation_evidence_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON evaluation_evidence FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER evaluation_policies_no_delete BEFORE DELETE OR TRUNCATE ON evaluation_policies FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
-- +goose StatementBegin
CREATE FUNCTION protect_evaluation_policy() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF OLD.approved OR NOT NEW.approved OR (NEW.id,NEW.competition_id,NEW.sector,NEW.kind,NEW.version,NEW.definition,NEW.definition_hash,NEW.actor_id,NEW.created_at)
 IS DISTINCT FROM (OLD.id,OLD.competition_id,OLD.sector,OLD.kind,OLD.version,OLD.definition,OLD.definition_hash,OLD.actor_id,OLD.created_at) THEN
 RAISE EXCEPTION 'Las reglas aprobadas son inmutables; cree otra versión'; END IF; RETURN NEW;
END; $$;
-- +goose StatementEnd
CREATE TRIGGER evaluation_policy_immutable BEFORE UPDATE ON evaluation_policies FOR EACH ROW EXECUTE FUNCTION protect_evaluation_policy();
ALTER TABLE workflow_events DROP CONSTRAINT workflow_events_action_check;
ALTER TABLE workflow_events ADD CONSTRAINT workflow_events_action_check CHECK(action IN (
 'convocatoria.creada','convocatoria.configurada','fase.publicada','fase.habilitada','fase.cerrada','elegibilidad.verificada',
 'meritos.reglas_creadas','meritos.reglas_aprobadas','meritos.revisados','meritos.calificados','evaluacion.iniciada','evaluacion.finalizada',
 'reconsideracion.presentada','reconsideracion.resuelta','resultado.publicado','evaluacion.reglas_creadas','evaluacion.reglas_aprobadas',
 'psicotecnica.registrada','psicotecnica.publicada','expediente.consolidado','reconsideracion.atendida'));
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM evaluation_records) OR EXISTS(SELECT 1 FROM evaluation_policies) OR EXISTS(SELECT 1 FROM evaluation_evidence) OR EXISTS(SELECT 1 FROM evaluation_upload_intents) THEN RAISE EXCEPTION 'Hay evidencia de evaluación; requiere plan de reversión'; END IF; END $$;
-- +goose StatementEnd
DROP TABLE evaluation_upload_intents;
DROP TABLE evaluation_evidence;
DROP TABLE reconsiderations;
DROP TABLE evaluation_commands;
DROP TABLE evaluation_records;
DROP TABLE evaluation_heads;
DROP TABLE evaluation_policy_streams;
DROP TABLE evaluation_policies;
DROP FUNCTION protect_evaluation_policy();
