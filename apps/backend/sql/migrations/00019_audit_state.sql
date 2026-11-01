-- +goose Up
-- No se rellena historial anterior: sólo captura operaciones a partir de esta migración.
CREATE TABLE audit_state_events (
 id UUID PRIMARY KEY DEFAULT uuidv7(), source_table TEXT NOT NULL,
 resource_id UUID NOT NULL, record_id UUID NOT NULL, submission_id UUID, actor_id UUID,
 actor_role TEXT NOT NULL CHECK(actor_role IN ('sistema','no_registrado','postulante','analista','administrador')),
 action TEXT NOT NULL, phase TEXT NOT NULL DEFAULT '', previous_state JSONB, next_state JSONB,
 reason TEXT NOT NULL CHECK(btrim(reason)<>''), request_id UUID,
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 CHECK((actor_role='sistema')=(actor_id IS NULL))
);
CREATE INDEX audit_state_resource ON audit_state_events(resource_id,created_at DESC,id DESC);
CREATE INDEX audit_state_submission ON audit_state_events(submission_id,created_at DESC,id DESC) WHERE submission_id IS NOT NULL;
CREATE INDEX audit_state_request ON audit_state_events(request_id) WHERE request_id IS NOT NULL;
CREATE TRIGGER audit_state_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON audit_state_events
 FOR EACH STATEMENT EXECUTE FUNCTION protect_authorization_history();

-- +goose StatementBegin
CREATE FUNCTION audit_state_projection(j JSONB) RETURNS JSONB LANGUAGE sql IMMUTABLE AS $$
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

-- +goose StatementBegin
CREATE FUNCTION capture_audit_state() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE j JSONB; old_j JSONB; before_state JSONB; after_state JSONB;
 rid UUID; record UUID; sid UUID; actor UUID; role_name TEXT; request UUID; phase_name TEXT; action_name TEXT; motive TEXT;
BEGIN
 j:=CASE WHEN TG_OP='DELETE' THEN to_jsonb(OLD) ELSE to_jsonb(NEW) END;
 old_j:=CASE WHEN TG_OP IN ('UPDATE','DELETE') THEN to_jsonb(OLD) ELSE NULL END;
 rid:=CASE WHEN TG_TABLE_NAME IN ('submissions','documents','mail_outbox') THEN (j->>'id')::uuid ELSE COALESCE(j->>'submission_id',j->>'competition_id',j->>'resource_id',j->>'attempt_id',j->>'mail_id',j->>'id')::uuid END;
 sid:=CASE WHEN TG_TABLE_NAME='submissions' THEN (j->>'id')::uuid ELSE (j->>'submission_id')::uuid END;
 IF sid IS NULL AND TG_TABLE_NAME IN ('exam_events','exam_reports') THEN SELECT submission_id INTO sid FROM exam_attempts WHERE id=rid; END IF;
 record:=COALESCE(j->>'id',j->>'attempt_id',j->>'submission_id',j->>'competition_id')::uuid;
 phase_name:=COALESCE(j->>'phase',j#>>'{payload,fase}',j->>'action','');
 action_name:=COALESCE(j->>'action',TG_TABLE_NAME||'.'||lower(TG_OP));
 actor:=COALESCE(NULLIF(current_setting('audit.actor_id',true),''),j->>'actor_id',
   CASE WHEN TG_OP='UPDATE' AND j->>'approved'='true' THEN j->>'approved_by' END,
   j#>>'{payload,responsableId}')::uuid;
 IF actor='00000000-0000-0000-0000-000000000000'::uuid THEN actor:=NULL; END IF;
 IF actor IS NULL THEN role_name:='sistema';
 ELSE role_name:=COALESCE(NULLIF(current_setting('audit.actor_role',true),''),j->>'actor_role',
   (SELECT role FROM applicants WHERE id=actor),'no_registrado'); END IF;
 request:=NULLIF(current_setting('audit.request_id',true),'')::uuid;
 motive:=COALESCE(NULLIF(j->>'reason',''),NULLIF(j#>>'{payload,motivo}',''),
   NULLIF(j#>>'{payload,revision,motivo}',''),NULLIF(j#>>'{payload,resultado,motivo}',''),NULLIF(j->>'approval_reference',''),NULLIF(j->>'last_error_code',''),
   'operacion_tecnica:'||action_name);
 before_state:=audit_state_projection(old_j);
 IF TG_OP='INSERT' THEN
  SELECT next_state INTO before_state FROM audit_state_events
   WHERE resource_id=rid AND source_table=TG_TABLE_NAME AND phase=phase_name ORDER BY created_at DESC,id DESC LIMIT 1;
 END IF;
 after_state:=CASE WHEN TG_OP='DELETE' THEN NULL ELSE audit_state_projection(j) END;
 INSERT INTO audit_state_events(source_table,resource_id,record_id,submission_id,actor_id,actor_role,action,phase,previous_state,next_state,reason,request_id)
 VALUES(TG_TABLE_NAME,rid,record,sid,actor,role_name,action_name,phase_name,before_state,after_state,left(motive,8000),request);
 RETURN NULL;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$ DECLARE tab TEXT; BEGIN
 FOREACH tab IN ARRAY ARRAY['submissions','documents','eligibility_checks','mail_outbox','workflow_events','competition_versions','merit_rule_sets','merit_reviews','merit_calculations','merit_publications',
  'evaluation_policies','evaluation_records','reconsiderations','stage_notices','mail_delivery_events','result_lists','result_list_publications',
  'exam_banks','exam_configs','exam_enrollments','exam_events','exam_reports','privacy_policies','privacy_bindings','privacy_decisions','administrative_access_events'] LOOP
 EXECUTE format('CREATE TRIGGER audit_state_capture AFTER INSERT OR UPDATE OR DELETE ON %I FOR EACH ROW EXECUTE FUNCTION capture_audit_state()',tab);
 END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$ DECLARE tab TEXT; BEGIN
 IF EXISTS(SELECT 1 FROM audit_state_events) THEN RAISE EXCEPTION 'Hay trazabilidad histórica; requiere plan de reversión'; END IF;
 FOREACH tab IN ARRAY ARRAY['submissions','documents','eligibility_checks','mail_outbox','workflow_events','competition_versions','merit_rule_sets','merit_reviews','merit_calculations','merit_publications',
  'evaluation_policies','evaluation_records','reconsiderations','stage_notices','mail_delivery_events','result_lists','result_list_publications',
  'exam_banks','exam_configs','exam_enrollments','exam_events','exam_reports','privacy_policies','privacy_bindings','privacy_decisions','administrative_access_events'] LOOP
 EXECUTE format('DROP TRIGGER audit_state_capture ON %I',tab);
 END LOOP;
END $$;
-- +goose StatementEnd
DROP FUNCTION capture_audit_state();
DROP FUNCTION audit_state_projection(JSONB);
DROP TABLE audit_state_events;
