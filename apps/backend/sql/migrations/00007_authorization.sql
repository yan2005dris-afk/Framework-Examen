-- +goose Up
ALTER TABLE applicants DROP CONSTRAINT applicants_role_check;
ALTER TABLE applicants ADD CONSTRAINT applicants_role_check
    CHECK (role IN ('postulante', 'analista', 'administrador'));

CREATE TABLE administrative_access_events (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    actor_id UUID NOT NULL,
    actor_role VARCHAR(32) NOT NULL CHECK (actor_role IN ('analista', 'administrador')),
    action VARCHAR(64) NOT NULL,
    resource_id UUID NOT NULL,
    resource_type VARCHAR(32) NOT NULL,
    result VARCHAR(16) NOT NULL CHECK (result IN ('permitido', 'rechazado')),
    created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX administrative_access_resource_idx
    ON administrative_access_events (resource_type, resource_id, created_at DESC, id DESC);

CREATE TABLE applicant_role_events (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    applicant_id UUID NOT NULL,
    actor_id UUID,
    operator TEXT NOT NULL,
    previous_role VARCHAR(32) NOT NULL,
    new_role VARCHAR(32) NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX applicant_role_events_applicant_idx
    ON applicant_role_events (applicant_id, created_at DESC, id DESC);

-- +goose StatementBegin
CREATE FUNCTION record_applicant_role_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.role IS DISTINCT FROM NEW.role THEN
        INSERT INTO applicant_role_events
            (applicant_id, actor_id, operator, previous_role, new_role, reason)
        VALUES (NEW.id, NULLIF(current_setting('app.actor_id', true), '')::uuid,
            COALESCE(NULLIF(current_setting('app.operator', true), ''), current_user),
            OLD.role, NEW.role,
            COALESCE(NULLIF(current_setting('app.role_reason', true), ''), 'Actualización SQL por operador de base de datos'));
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER applicants_role_audit AFTER UPDATE OF role ON applicants
    FOR EACH ROW EXECUTE FUNCTION record_applicant_role_change();

-- +goose StatementBegin
CREATE FUNCTION protect_authorization_history() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'El historial de autorización no admite modificaciones';
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER administrative_access_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON administrative_access_events
    FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER applicant_role_events_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON applicant_role_events
    FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();

-- +goose Down
DROP TRIGGER applicants_role_audit ON applicants;
DROP FUNCTION record_applicant_role_change();
DROP TABLE applicant_role_events;
DROP TABLE administrative_access_events;
DROP FUNCTION protect_authorization_history();
UPDATE applicants SET role = 'postulante' WHERE role = 'administrador';
ALTER TABLE applicants DROP CONSTRAINT applicants_role_check;
ALTER TABLE applicants ADD CONSTRAINT applicants_role_check CHECK (role IN ('postulante', 'analista'));
