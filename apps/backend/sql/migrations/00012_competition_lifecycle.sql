-- +goose Up
ALTER TABLE workflow_events DROP CONSTRAINT workflow_events_action_check;
ALTER TABLE workflow_events ADD CONSTRAINT workflow_events_action_check CHECK(action IN (
 'convocatoria.creada','convocatoria.configurada','fase.publicada','fase.habilitada','fase.cerrada','elegibilidad.verificada',
 'meritos.revisados','meritos.calificados','evaluacion.iniciada','evaluacion.finalizada',
 'reconsideracion.presentada','reconsideracion.resuelta','resultado.publicado'));
ALTER TABLE submission_events DROP CONSTRAINT submission_events_event_type_check;
ALTER TABLE submission_events ADD CONSTRAINT submission_events_event_type_check CHECK(event_type IN (
 'guardada','enviada','consultada','eliminada','documento.cargado','documento.eliminado','postulacion.cerrada_elegibilidad'));
CREATE TABLE competition_controls (
 competition_id UUID PRIMARY KEY REFERENCES competitions(id),
 version BIGINT NOT NULL DEFAULT 1 CHECK(version>0),
 approved BOOLEAN NOT NULL DEFAULT false,
 published BOOLEAN NOT NULL DEFAULT false,
 form_enabled BOOLEAN NOT NULL DEFAULT false,
 sectors TEXT[] NOT NULL DEFAULT '{}',
 phases JSONB NOT NULL DEFAULT '[]',
 approved_at TIMESTAMPTZ,
 approved_by UUID,
 CHECK(NOT published OR approved),
 CHECK(NOT form_enabled OR (approved AND published))
);
-- No habilitar automáticamente períodos existentes ni reasignar expedientes legados.
INSERT INTO competition_controls(competition_id) SELECT id FROM competitions;
CREATE TABLE competition_versions (
 id UUID PRIMARY KEY DEFAULT uuidv7(), competition_id UUID NOT NULL,
 version BIGINT NOT NULL, configuration JSONB NOT NULL, actor_id UUID NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(competition_id,version)
);
CREATE TRIGGER competition_versions_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON competition_versions
 FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();

ALTER TABLE submissions ADD COLUMN eligibility JSONB;
ALTER TABLE submissions DROP CONSTRAINT submissions_status_check;
ALTER TABLE submissions ADD CONSTRAINT submissions_status_check CHECK(status IN ('draft','in_review','submitted','closed'));
ALTER TABLE submissions ADD CONSTRAINT submission_closure_evidence CHECK(status<>'closed' OR (eligibility IS NOT NULL AND eligibility->>'elegible'='false'));
CREATE TABLE eligibility_checks (
 id UUID PRIMARY KEY, submission_id UUID NOT NULL, applicant_id UUID NOT NULL,
 competition_id UUID NOT NULL, sector TEXT NOT NULL, result BOOLEAN NOT NULL,
 source TEXT NOT NULL, checked_at TIMESTAMPTZ NOT NULL, assessment JSONB NOT NULL
);
CREATE INDEX eligibility_checks_submission_idx ON eligibility_checks(submission_id,checked_at DESC,id DESC);
CREATE TRIGGER eligibility_checks_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON eligibility_checks
 FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();

-- +goose StatementBegin
CREATE FUNCTION assert_competition_writable(period_id UUID, selected_sector TEXT) RETURNS void LANGUAGE plpgsql AS $$
DECLARE p competitions; c competition_controls; instant TIMESTAMPTZ; phase JSONB;
BEGIN
 SELECT * INTO p FROM competitions WHERE id=period_id FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'Seleccione una convocatoria' USING ERRCODE='23514',CONSTRAINT='competition_open'; END IF;
 SELECT * INTO c FROM competition_controls WHERE competition_id=period_id FOR SHARE;
 instant:=clock_timestamp();
 IF NOT FOUND OR NOT(c.approved AND c.published AND c.form_enabled) OR instant<p.opens_at OR instant>=p.closes_at THEN
  RAISE EXCEPTION 'La convocatoria no admite escrituras' USING ERRCODE='23514',CONSTRAINT='competition_open';
 END IF;
 IF coalesce(selected_sector,'')<>'' AND NOT(selected_sector=ANY(c.sectors)) THEN
  RAISE EXCEPTION 'El sector no está habilitado' USING ERRCODE='23514',CONSTRAINT='competition_open';
 END IF;
 SELECT value INTO phase FROM jsonb_array_elements(c.phases) WHERE value->>'codigo'='postulacion';
 IF phase IS NULL OR NOT coalesce((phase->>'publicada')::boolean,false) OR NOT coalesce((phase->>'habilitada')::boolean,false)
  OR instant<(phase->>'apertura')::timestamptz OR instant>=(phase->>'cierre')::timestamptz THEN
  RAISE EXCEPTION 'El EF-12 no está habilitado' USING ERRCODE='23514',CONSTRAINT='competition_open';
 END IF;
END; $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION enforce_submission_window() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE item submissions;
BEGIN
 IF TG_OP='DELETE' THEN item:=OLD; ELSE item:=NEW; END IF;
 IF TG_OP<>'INSERT' AND OLD.status IN ('submitted','closed') THEN
  RAISE EXCEPTION 'El expediente no admite cambios' USING ERRCODE='23514',CONSTRAINT='submission_closed';
 END IF;
 PERFORM assert_competition_writable(item.competition_id,item.content->'seccion4'->>'sector');
 IF EXISTS(SELECT 1 FROM submissions WHERE applicant_id=item.applicant_id AND competition_id=item.competition_id AND status='closed' AND id<>item.id) THEN
  RAISE EXCEPTION 'Existe una postulación cerrada para esta cuenta y convocatoria' USING ERRCODE='23514',CONSTRAINT='submission_closed';
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; ELSE RETURN NEW; END IF;
END; $$;
-- +goose StatementEnd
CREATE TRIGGER submissions_window BEFORE INSERT OR UPDATE OR DELETE ON submissions
 FOR EACH ROW EXECUTE FUNCTION enforce_submission_window();

-- +goose StatementBegin
CREATE FUNCTION enforce_document_window() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE parent submissions; parent_id UUID;
BEGIN
 parent_id:=CASE WHEN TG_OP='DELETE' THEN OLD.submission_id ELSE NEW.submission_id END;
 SELECT * INTO parent FROM submissions WHERE id=parent_id FOR SHARE;
 -- La cascada del padre ya pasó por su propio control de escritura.
 IF NOT FOUND AND TG_OP='DELETE' THEN RETURN OLD; END IF;
 IF NOT FOUND OR parent.status NOT IN ('draft','in_review') THEN
  RAISE EXCEPTION 'El expediente no admite adjuntos' USING ERRCODE='23514',CONSTRAINT='submission_closed';
 END IF;
 IF TG_OP='UPDATE' AND NEW.submission_id<>OLD.submission_id THEN RAISE EXCEPTION 'No se cambia la titularidad de un adjunto'; END IF;
 PERFORM assert_competition_writable(parent.competition_id,parent.content->'seccion4'->>'sector');
 IF TG_OP='DELETE' THEN RETURN OLD; ELSE RETURN NEW; END IF;
END; $$;
-- +goose StatementEnd
CREATE TRIGGER documents_window BEFORE INSERT OR UPDATE OR DELETE ON documents
 FOR EACH ROW EXECUTE FUNCTION enforce_document_window();

-- +goose Down
-- No revertir un cambio que ya contiene cierres: evita perder su semántica y evidencia.
ALTER TABLE submissions ADD CONSTRAINT no_closed_downgrade CHECK(status<>'closed');
ALTER TABLE submissions DROP CONSTRAINT no_closed_downgrade;
DROP TRIGGER documents_window ON documents;
DROP FUNCTION enforce_document_window();
DROP TRIGGER submissions_window ON submissions;
DROP FUNCTION enforce_submission_window();
DROP FUNCTION assert_competition_writable(UUID,TEXT);
DROP TABLE eligibility_checks;
ALTER TABLE submissions DROP COLUMN eligibility;
ALTER TABLE submissions DROP CONSTRAINT submissions_status_check;
ALTER TABLE submissions ADD CONSTRAINT submissions_status_check CHECK(status IN ('draft','in_review','submitted'));
DROP TABLE competition_versions;
DROP TABLE competition_controls;
