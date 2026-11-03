-- +goose Up
-- El historial sobrevive a la baja de borradores/cuentas. No contiene sus datos personales.
ALTER TABLE submission_events DROP CONSTRAINT submission_events_submission_id_fkey;
ALTER TABLE submission_events DROP CONSTRAINT submission_events_applicant_id_fkey;
ALTER TABLE submission_events DROP CONSTRAINT submission_events_event_type_check;
ALTER TABLE submission_events ALTER COLUMN event_type TYPE VARCHAR(64);
-- No adivinar el rol de eventos antiguos a partir del perfil actual.
ALTER TABLE submission_events ADD COLUMN actor_role VARCHAR(32) NOT NULL DEFAULT 'no_registrado';
ALTER TABLE submission_events ADD CONSTRAINT submission_events_event_type_check
    CHECK (event_type IN ('guardada','enviada','consultada','eliminada','documento.cargado','documento.eliminado'));
CREATE TRIGGER submission_events_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON submission_events
    FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();

-- +goose StatementBegin
CREATE FUNCTION record_submission_deletion() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO submission_events(submission_id,applicant_id,event_type,actor_role)
        VALUES (OLD.id,OLD.applicant_id,'eliminada',
            COALESCE((SELECT role FROM applicants WHERE id=OLD.applicant_id),'no_registrado'));
    RETURN OLD;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER submissions_deletion_audit BEFORE DELETE ON submissions
    FOR EACH ROW EXECUTE FUNCTION record_submission_deletion();

-- +goose StatementBegin
CREATE FUNCTION record_document_change() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE sid uuid; owner_id uuid; document_id uuid;
BEGIN
    sid := CASE WHEN TG_OP = 'DELETE' THEN OLD.submission_id ELSE NEW.submission_id END;
    document_id := CASE WHEN TG_OP = 'DELETE' THEN OLD.id ELSE NEW.id END;
    SELECT applicant_id INTO owner_id FROM submissions WHERE id = sid;
    -- La cascada de una baja de borrador ya está cubierta por 'eliminada'.
    IF owner_id IS NOT NULL THEN
        INSERT INTO submission_events(submission_id,applicant_id,event_type,payload,actor_role)
            VALUES (sid,owner_id,CASE WHEN TG_OP = 'DELETE' THEN 'documento.eliminado' ELSE 'documento.cargado' END,
                jsonb_build_object('documentId',document_id),
                COALESCE((SELECT role FROM applicants WHERE id=owner_id),'no_registrado'));
    END IF;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER documents_change_audit AFTER INSERT OR DELETE ON documents
    FOR EACH ROW EXECUTE FUNCTION record_document_change();

ALTER TABLE administrative_access_events ADD COLUMN submission_id UUID;
-- La migración puede completar enlaces antiguos antes de volver a proteger el historial.
ALTER TABLE administrative_access_events DISABLE TRIGGER administrative_access_immutable;
UPDATE administrative_access_events SET submission_id = resource_id WHERE resource_type = 'postulacion';
UPDATE administrative_access_events a SET submission_id = d.submission_id
    FROM documents d WHERE a.resource_type = 'documento' AND a.resource_id = d.id;
ALTER TABLE administrative_access_events ENABLE TRIGGER administrative_access_immutable;
CREATE INDEX administrative_access_submission_idx
    ON administrative_access_events (submission_id, created_at DESC, id DESC);

-- Puerto para las operaciones de T17/T18/T20-T27: sólo metadatos, sin payload libre.
CREATE TABLE workflow_events (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    submission_id UUID,
    actor_id UUID,
    actor_role VARCHAR(32) NOT NULL CHECK (actor_role IN ('sistema','postulante','analista','administrador')),
    action VARCHAR(64) NOT NULL CHECK (action IN (
        'fase.publicada','fase.habilitada','fase.cerrada','elegibilidad.verificada',
        'meritos.revisados','meritos.calificados','evaluacion.iniciada','evaluacion.finalizada',
        'reconsideracion.presentada','reconsideracion.resuelta','resultado.publicado')),
    resource_id UUID NOT NULL,
    resource_type VARCHAR(32) NOT NULL CHECK (resource_type IN ('postulacion','convocatoria','evaluacion','reconsideracion')),
    result VARCHAR(16) NOT NULL CHECK (result IN ('permitido','rechazado')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((actor_role = 'sistema') = (actor_id IS NULL))
);
CREATE INDEX workflow_events_submission_idx ON workflow_events (submission_id, created_at DESC, id DESC);
CREATE TRIGGER workflow_events_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON workflow_events
    FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();

-- +goose Down
DROP TRIGGER documents_change_audit ON documents;
DROP FUNCTION record_document_change();
DROP TRIGGER submissions_deletion_audit ON submissions;
DROP FUNCTION record_submission_deletion();
DROP TABLE workflow_events;
DROP INDEX administrative_access_submission_idx;
ALTER TABLE administrative_access_events DROP COLUMN submission_id;
DROP TRIGGER submission_events_immutable ON submission_events;
-- No reinstalar cascadas: podrían destruir el historial que ya sobrevivió a una baja.
