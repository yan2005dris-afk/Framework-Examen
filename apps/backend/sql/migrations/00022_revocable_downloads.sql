-- +goose Up
-- Capacidades opacas: nunca se conserva el token de descarga ni la cookie de sesión en claro.
CREATE TABLE download_grants (
 token_hash CHAR(64) PRIMARY KEY,
 actor_id UUID NOT NULL,
 session_hash CHAR(64) NOT NULL,
 storage_key TEXT NOT NULL,
 actor_role TEXT NOT NULL,
 policy_version BIGINT NOT NULL,
 expires_at TIMESTAMPTZ NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX download_grants_expiry ON download_grants(expires_at);
CREATE TABLE download_access_events (
 id UUID PRIMARY KEY DEFAULT uuidv7(), actor_id UUID NOT NULL,
 actor_role TEXT NOT NULL CHECK(actor_role IN ('postulante','analista','administrador')),
 submission_id UUID NOT NULL,resource_id UUID NOT NULL,request_id UUID,
 action TEXT NOT NULL DEFAULT 'documento.descarga_verificada',created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX download_access_submission ON download_access_events(submission_id,created_at DESC,id DESC);
CREATE TRIGGER download_access_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON download_access_events FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
-- +goose Down
DROP TABLE download_access_events;
DROP TABLE download_grants;
