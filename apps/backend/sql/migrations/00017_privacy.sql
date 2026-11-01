-- +goose Up
CREATE TABLE privacy_policies(
 id UUID PRIMARY KEY,competition_id UUID NOT NULL REFERENCES competitions(id),version BIGINT NOT NULL,
 definition JSONB NOT NULL,hash VARCHAR(64) NOT NULL,actor_id UUID NOT NULL,created_at TIMESTAMPTZ NOT NULL,
 UNIQUE(competition_id,version),UNIQUE(id,competition_id)
);
CREATE TABLE privacy_controls(competition_id UUID PRIMARY KEY REFERENCES competitions(id),version BIGINT NOT NULL DEFAULT 0,active_id UUID,blocked BOOLEAN NOT NULL DEFAULT false,
 FOREIGN KEY(active_id,competition_id) REFERENCES privacy_policies(id,competition_id));
CREATE TABLE privacy_bindings(submission_id UUID PRIMARY KEY REFERENCES submissions(id),policy_id UUID NOT NULL REFERENCES privacy_policies(id),policy_hash VARCHAR(64) NOT NULL,accepted_at TIMESTAMPTZ NOT NULL,retention_review_at TIMESTAMPTZ NOT NULL);
CREATE TABLE privacy_decisions(id UUID PRIMARY KEY,competition_id UUID NOT NULL REFERENCES competitions(id),blocked BOOLEAN NOT NULL,reason TEXT NOT NULL,actor_id UUID NOT NULL,created_at TIMESTAMPTZ NOT NULL);
CREATE TRIGGER privacy_policies_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON privacy_policies FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER privacy_bindings_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON privacy_bindings FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER privacy_decisions_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON privacy_decisions FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM privacy_policies) OR EXISTS(SELECT 1 FROM privacy_bindings) THEN RAISE EXCEPTION 'Hay políticas y consentimientos históricos'; END IF; END $$;
-- +goose StatementEnd
DROP TABLE privacy_decisions;
DROP TABLE privacy_bindings;
DROP TABLE privacy_controls;
DROP TABLE privacy_policies;
