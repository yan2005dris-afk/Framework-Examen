-- +goose Up
-- El sector del cargo (EF-12 1.4) pasa a ser una columna propia de la postulación.
--
-- Fase 1 de la normalización del formulario: mientras el borrador viva en
-- `content`, la columna es generada a partir de él. Así la calcula PostgreSQL
-- para cualquier escritor (API, pruebas, semillas) y agregarla reescribe la
-- tabla sin disparar triggers: los expedientes sellados no se modifican y su
-- hash no cambia. Cuando las secciones vivan en tablas propias, la expresión se
-- retira con `ALTER COLUMN sector DROP EXPRESSION` y la columna conserva los valores.

-- +goose StatementBegin
-- Única definición de dónde está el sector dentro del formulario. La usan la
-- columna generada y el trigger BEFORE, que no puede leer columnas generadas.
CREATE FUNCTION submission_sector(content JSONB) RETURNS TEXT
LANGUAGE sql IMMUTABLE PARALLEL SAFE RETURN content->'seccion4'->>'sector';
-- +goose StatementEnd

ALTER TABLE submissions
    ADD COLUMN sector TEXT GENERATED ALWAYS AS (submission_sector(content)) STORED;

-- Valores de `definition.json` (opciones.sectores); vacío mientras el borrador no lo elige.
ALTER TABLE submissions ADD CONSTRAINT submissions_sector_check
    CHECK (sector IS NULL OR sector IN ('', 'afiliados', 'jubilados', 'empleadores'));

DROP INDEX submissions_review_sector_idx;
CREATE INDEX submissions_review_sector_idx ON submissions
    (competition_id, sector, created_at DESC, id DESC)
    WHERE status = 'submitted' AND hash IS NOT NULL AND submitted_at IS NOT NULL;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION enforce_submission_window() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE item submissions;
BEGIN
 IF TG_OP='DELETE' THEN item:=OLD; ELSE item:=NEW; END IF;
 IF TG_OP<>'INSERT' AND OLD.status IN ('submitted','closed') THEN
  RAISE EXCEPTION 'El expediente no admite cambios' USING ERRCODE='23514',CONSTRAINT='submission_closed';
 END IF;
 -- En un trigger BEFORE la columna generada todavía no está calculada.
 PERFORM assert_competition_writable(item.competition_id,submission_sector(item.content));
 IF EXISTS(SELECT 1 FROM submissions WHERE applicant_id=item.applicant_id AND competition_id=item.competition_id AND status='closed' AND id<>item.id) THEN
  RAISE EXCEPTION 'Existe una postulación cerrada para esta cuenta y convocatoria' USING ERRCODE='23514',CONSTRAINT='submission_closed';
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; ELSE RETURN NEW; END IF;
END; $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION enforce_document_window() RETURNS trigger LANGUAGE plpgsql AS $$
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
 PERFORM assert_competition_writable(parent.competition_id,parent.sector);
 IF TG_OP='DELETE' THEN RETURN OLD; ELSE RETURN NEW; END IF;
END; $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION enforce_document_window() RETURNS trigger LANGUAGE plpgsql AS $$
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

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION enforce_submission_window() RETURNS trigger LANGUAGE plpgsql AS $$
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

DROP INDEX submissions_review_sector_idx;
CREATE INDEX submissions_review_sector_idx ON submissions
    (competition_id, (content->'seccion4'->>'sector'), created_at DESC, id DESC)
    WHERE status = 'submitted' AND hash IS NOT NULL AND submitted_at IS NOT NULL;

ALTER TABLE submissions DROP CONSTRAINT submissions_sector_check;
ALTER TABLE submissions DROP COLUMN sector;
DROP FUNCTION submission_sector(JSONB);
