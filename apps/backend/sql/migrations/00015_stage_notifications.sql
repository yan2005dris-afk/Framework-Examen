-- +goose Up
ALTER TABLE mail_outbox ADD COLUMN retry_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE mail_outbox ADD COLUMN total_attempts INTEGER NOT NULL DEFAULT 0;
UPDATE mail_outbox SET total_attempts=attempts;
CREATE TABLE stage_notices (
 id UUID PRIMARY KEY,submission_id UUID NOT NULL REFERENCES submissions(id),source_id UUID NOT NULL,
 action TEXT NOT NULL,template_version TEXT NOT NULL,created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(submission_id,source_id,action)
);
-- Vincula avisos T21/T22 ya encolados sin regenerar sus destinatarios ni cuerpos.
INSERT INTO stage_notices(id,submission_id,source_id,action,template_version,created_at)
SELECT e.id,e.submission_id,e.id,e.action,'resultado-etapa-v1',m.created_at FROM evaluation_records e
JOIN mail_outbox m ON m.resource_id=e.id WHERE m.kind IN ('stage.result','reconsideration.decision')
ON CONFLICT DO NOTHING;
CREATE INDEX stage_notices_page ON stage_notices(submission_id,created_at DESC,id DESC);
CREATE TABLE mail_delivery_events (
 id UUID PRIMARY KEY DEFAULT uuidv7(),mail_id UUID NOT NULL REFERENCES mail_outbox(id),
 status TEXT NOT NULL,error_code TEXT NOT NULL,attempt INTEGER NOT NULL,cycle INTEGER NOT NULL,
 actor_id UUID,reason TEXT,created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX mail_delivery_page ON mail_delivery_events(mail_id,created_at DESC,id DESC);
CREATE TRIGGER stage_notices_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON stage_notices FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER mail_delivery_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON mail_delivery_events FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
ALTER TABLE workflow_events DROP CONSTRAINT workflow_events_resource_type_check;
ALTER TABLE workflow_events ADD CONSTRAINT workflow_events_resource_type_check CHECK(resource_type IN ('postulacion','convocatoria','evaluacion','reconsideracion','notificacion'));
ALTER TABLE workflow_events DROP CONSTRAINT workflow_events_action_check;
ALTER TABLE workflow_events ADD CONSTRAINT workflow_events_action_check CHECK(action IN (
 'convocatoria.creada','convocatoria.configurada','fase.publicada','fase.habilitada','fase.cerrada','elegibilidad.verificada',
 'meritos.reglas_creadas','meritos.reglas_aprobadas','meritos.revisados','meritos.calificados','evaluacion.iniciada','evaluacion.finalizada',
 'reconsideracion.presentada','reconsideracion.resuelta','resultado.publicado','evaluacion.reglas_creadas','evaluacion.reglas_aprobadas',
 'psicotecnica.registrada','psicotecnica.publicada','expediente.consolidado','reconsideracion.atendida',
 'notificacion.reintentada','lista.generada','lista.publicada'));
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM stage_notices) OR EXISTS(SELECT 1 FROM mail_delivery_events) THEN RAISE EXCEPTION 'Hay historial de notificaciones; requiere plan de reversión'; END IF; END $$;
-- +goose StatementEnd
DROP TABLE mail_delivery_events;
DROP TABLE stage_notices;
ALTER TABLE mail_outbox DROP COLUMN total_attempts;
ALTER TABLE mail_outbox DROP COLUMN retry_count;
