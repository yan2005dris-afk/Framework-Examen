-- +goose Up
-- Fase 5 de la normalización del formulario: `content` queda reservado para el
-- snapshot de los expedientes que ya no se editan (enviados y cerrados). Los
-- borradores viven sólo en las tablas de 00024; 00027 vacía su `content`.

-- El sector deja de calcularse desde `content`: lo escribe el adaptador. La
-- columna conserva los valores ya calculados.
ALTER TABLE submissions ALTER COLUMN sector DROP EXPRESSION;

-- +goose StatementBegin
-- Quien inserte un snapshot sin sector (semillas, pruebas, legado) lo recibe del
-- propio snapshot. El nombre ordena este trigger antes que la ventana y el
-- snapshot, que ya leen la columna.
CREATE FUNCTION derive_submission_sector() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.sector IS NULL AND NEW.content IS NOT NULL THEN
  NEW.sector := submission_sector(NEW.content);
 END IF;
 RETURN NEW;
END; $$;
-- +goose StatementEnd
CREATE TRIGGER submissions_derive_sector BEFORE INSERT OR UPDATE ON submissions
    FOR EACH ROW EXECUTE FUNCTION derive_submission_sector();

-- Un snapshot y su sector nunca discrepan.
ALTER TABLE submissions ADD CONSTRAINT submissions_sector_snapshot
    CHECK (content IS NULL OR sector IS NOT DISTINCT FROM submission_sector(content));

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION enforce_submission_window() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE item submissions;
BEGIN
 IF TG_OP='DELETE' THEN item:=OLD; ELSE item:=NEW; END IF;
 IF TG_OP<>'INSERT' AND OLD.status IN ('submitted','closed') THEN
  RAISE EXCEPTION 'El expediente no admite cambios' USING ERRCODE='23514',CONSTRAINT='submission_closed';
 END IF;
 PERFORM assert_competition_writable(item.competition_id,item.sector);
 IF EXISTS(SELECT 1 FROM submissions WHERE applicant_id=item.applicant_id AND competition_id=item.competition_id AND status='closed' AND id<>item.id) THEN
  RAISE EXCEPTION 'Existe una postulación cerrada para esta cuenta y convocatoria' USING ERRCODE='23514',CONSTRAINT='submission_closed';
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; ELSE RETURN NEW; END IF;
END; $$;
-- +goose StatementEnd

ALTER TABLE submissions ALTER COLUMN content DROP NOT NULL;
ALTER TABLE submissions ALTER COLUMN content DROP DEFAULT;

-- Todo expediente que ya no se edita conserva su snapshot.
ALTER TABLE submissions ADD CONSTRAINT submissions_sealed_content
    CHECK (status NOT IN ('submitted', 'closed') OR content IS NOT NULL);

-- +goose StatementBegin
-- Sin `content`, guardar un borrador dejaría el diario de estados (00019) con un
-- antes y un después idénticos. Para las filas de `submissions` (la única tabla
-- con `content`) se proyectan además la fecha de modificación y el sector: no son
-- datos personales y vuelven visible cada guardado.
CREATE OR REPLACE FUNCTION audit_state_projection(j JSONB) RETURNS JSONB LANGUAGE sql IMMUTABLE AS $$
 SELECT CASE WHEN j IS NULL THEN NULL ELSE jsonb_strip_nulls(jsonb_build_object(
  'id',COALESCE(j->'id',j->'attempt_id',j->'submission_id',j->'competition_id'),
  'version',COALESCE(j->'version',j#>'{payload,version}'),
  'estado',COALESCE(j->'status',j->'decision',j#>'{payload,estado}',j#>'{payload,decision}'),
  'resultado',j->'result','intento',COALESCE(j->'attempt',j->'attempts'),'ciclo',j->'cycle','codigoError',COALESCE(j->'error_code',j->'last_error_code'),'aprobada',j->'approved','bloqueada',j->'blocked',
  'hash',COALESCE(j->'hash',j->'sha256',j->'definition_hash',j->'policy_hash',j#>'{payload,hashEnvio}'),
  'reglasId',j->'rule_set_id','fuenteId',COALESCE(j->'resource_id',j#>'{payload,fuenteId}',j->'source_id',j->'result_id',j->'calculation_id',j->'review_id',j->'policy_id',j->'config_id',j->'bank_id'),
  'configuracionHash',CASE WHEN j ? 'configuration' THEN to_jsonb(encode(sha256(convert_to((j->'configuration')::text,'UTF8')),'hex')) ELSE NULL END,
  'contenidoHash',CASE WHEN j ? 'content' THEN to_jsonb(encode(sha256(convert_to((j->'content')::text,'UTF8')),'hex')) ELSE NULL END,
  'actualizadoEn',CASE WHEN j ? 'content' THEN j->'updated_at' ELSE NULL END,
  'sector',CASE WHEN j ? 'content' THEN j->'sector' ELSE NULL END
 )) END
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION audit_state_projection(j JSONB) RETURNS JSONB LANGUAGE sql IMMUTABLE AS $$
 SELECT CASE WHEN j IS NULL THEN NULL ELSE jsonb_strip_nulls(jsonb_build_object(
  'id',COALESCE(j->'id',j->'attempt_id',j->'submission_id',j->'competition_id'),
  'version',COALESCE(j->'version',j#>'{payload,version}'),
  'estado',COALESCE(j->'status',j->'decision',j#>'{payload,estado}',j#>'{payload,decision}'),
  'resultado',j->'result','intento',COALESCE(j->'attempt',j->'attempts'),'ciclo',j->'cycle','codigoError',COALESCE(j->'error_code',j->'last_error_code'),'aprobada',j->'approved','bloqueada',j->'blocked',
  'hash',COALESCE(j->'hash',j->'sha256',j->'definition_hash',j->'policy_hash',j#>'{payload,hashEnvio}'),
  'reglasId',j->'rule_set_id','fuenteId',COALESCE(j->'resource_id',j#>'{payload,fuenteId}',j->'source_id',j->'result_id',j->'calculation_id',j->'review_id',j->'policy_id',j->'config_id',j->'bank_id'),
  'configuracionHash',CASE WHEN j ? 'configuration' THEN to_jsonb(encode(sha256(convert_to((j->'configuration')::text,'UTF8')),'hex')) ELSE NULL END,
  'contenidoHash',CASE WHEN j ? 'content' THEN to_jsonb(encode(sha256(convert_to((j->'content')::text,'UTF8')),'hex')) ELSE NULL END
 )) END
$$;
-- +goose StatementEnd

ALTER TABLE submissions DROP CONSTRAINT submissions_sealed_content;
-- 00027 reconstruye el contenido de los borradores antes de llegar aquí.
UPDATE submissions SET content = '{}'::jsonb WHERE content IS NULL;
ALTER TABLE submissions ALTER COLUMN content SET DEFAULT '{}'::jsonb;
ALTER TABLE submissions ALTER COLUMN content SET NOT NULL;

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

ALTER TABLE submissions DROP CONSTRAINT submissions_sector_snapshot;
DROP TRIGGER submissions_derive_sector ON submissions;
DROP FUNCTION derive_submission_sector();

-- Volver a la columna generada exige recrearla con su índice y su CHECK.
DROP INDEX submissions_review_sector_idx;
ALTER TABLE submissions DROP CONSTRAINT submissions_sector_check;
ALTER TABLE submissions DROP COLUMN sector;
ALTER TABLE submissions
    ADD COLUMN sector TEXT GENERATED ALWAYS AS (submission_sector(content)) STORED;
ALTER TABLE submissions ADD CONSTRAINT submissions_sector_check
    CHECK (sector IS NULL OR sector IN ('', 'afiliados', 'jubilados', 'empleadores'));
CREATE INDEX submissions_review_sector_idx ON submissions
    (competition_id, sector, created_at DESC, id DESC)
    WHERE status = 'submitted' AND hash IS NOT NULL AND submitted_at IS NOT NULL;
