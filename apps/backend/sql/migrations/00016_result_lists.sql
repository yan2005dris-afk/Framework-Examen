-- +goose Up
CREATE TABLE result_list_streams (
 competition_id UUID NOT NULL REFERENCES competitions(id),sector TEXT NOT NULL CHECK(sector IN ('afiliados','jubilados','empleadores')),
 version BIGINT NOT NULL DEFAULT 0,published_version BIGINT NOT NULL DEFAULT 0,published_id UUID,
 PRIMARY KEY(competition_id,sector)
);
CREATE TABLE result_lists (
 id UUID PRIMARY KEY,competition_id UUID NOT NULL REFERENCES competitions(id),sector TEXT NOT NULL,
 version BIGINT NOT NULL,sources_hash VARCHAR(64) NOT NULL,payload JSONB NOT NULL,actor_id UUID NOT NULL,created_at TIMESTAMPTZ NOT NULL,
 UNIQUE(competition_id,sector,version),UNIQUE(id,competition_id,sector)
);
CREATE TABLE result_list_publications (
 id UUID PRIMARY KEY,list_id UUID NOT NULL UNIQUE REFERENCES result_lists(id),competition_id UUID NOT NULL,sector TEXT NOT NULL,
 version BIGINT NOT NULL,previous_id UUID REFERENCES result_list_publications(id),request_hash VARCHAR(64) NOT NULL,
 approval_reference TEXT NOT NULL,privacy_reference TEXT NOT NULL,is_public BOOLEAN NOT NULL,public_payload JSONB NOT NULL,
 actor_id UUID NOT NULL,created_at TIMESTAMPTZ NOT NULL,
 UNIQUE(competition_id,sector,version),UNIQUE(id,competition_id,sector),
 FOREIGN KEY(list_id,competition_id,sector) REFERENCES result_lists(id,competition_id,sector)
);
ALTER TABLE result_list_streams ADD FOREIGN KEY(published_id,competition_id,sector) REFERENCES result_list_publications(id,competition_id,sector);
CREATE TABLE result_list_commands (
 id UUID PRIMARY KEY,actor_id UUID NOT NULL,operation TEXT NOT NULL,scope TEXT NOT NULL,request_hash VARCHAR(64) NOT NULL,resource_id UUID NOT NULL
);
CREATE INDEX result_lists_page ON result_lists(competition_id,sector,created_at DESC,id DESC);
CREATE INDEX result_publications_page ON result_list_publications(competition_id,sector,created_at DESC,id DESC);
CREATE TRIGGER result_lists_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON result_lists FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER result_publications_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON result_list_publications FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
CREATE TRIGGER result_commands_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON result_list_commands FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM result_lists) OR EXISTS(SELECT 1 FROM result_list_publications) THEN RAISE EXCEPTION 'Hay listas históricas; requiere plan de reversión'; END IF; END $$;
-- +goose StatementEnd
DROP TABLE result_list_commands;
DROP TABLE result_list_streams;
DROP TABLE result_list_publications;
DROP TABLE result_lists;
