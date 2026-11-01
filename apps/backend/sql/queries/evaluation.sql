-- name: EnsureEvaluationPolicyStream :exec
INSERT INTO evaluation_policy_streams(competition_id,sector,kind) VALUES($1,$2,$3) ON CONFLICT DO NOTHING;
-- name: LockEvaluationPolicyStream :one
SELECT * FROM evaluation_policy_streams WHERE competition_id=$1 AND sector=$2 AND kind=$3 FOR UPDATE;
-- name: GetEvaluationPolicyStream :one
SELECT * FROM evaluation_policy_streams WHERE competition_id=$1 AND sector=$2 AND kind=$3 FOR SHARE;
-- name: SetEvaluationPolicyStream :exec
UPDATE evaluation_policy_streams SET version=$4,active_id=$5 WHERE competition_id=$1 AND sector=$2 AND kind=$3;
-- name: CreateEvaluationPolicy :exec
INSERT INTO evaluation_policies(id,competition_id,sector,kind,version,definition,definition_hash,actor_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9);
-- name: GetEvaluationPolicy :one
SELECT * FROM evaluation_policies WHERE id=$1;
-- name: ApproveEvaluationPolicy :exec
UPDATE evaluation_policies SET approved=true,approval_reference=$2,approved_by=$3,approved_at=clock_timestamp() WHERE id=$1 AND NOT approved;
-- name: ListEvaluationPolicies :many
SELECT * FROM evaluation_policies WHERE competition_id=sqlc.arg(competition_id) AND sector=sqlc.arg(sector) AND kind=sqlc.arg(kind) AND (sqlc.arg(before_version)::bigint=0 OR version<sqlc.arg(before_version)) ORDER BY version DESC LIMIT sqlc.arg(row_limit);
-- name: EnsureEvaluationHead :exec
INSERT INTO evaluation_heads(submission_id) VALUES($1) ON CONFLICT DO NOTHING;
-- name: LockEvaluationHead :one
SELECT * FROM evaluation_heads WHERE submission_id=$1 FOR UPDATE;
-- name: GetEvaluationHead :one
SELECT * FROM evaluation_heads WHERE submission_id=$1 FOR SHARE;
-- name: SetEvaluationHead :exec
UPDATE evaluation_heads SET version=version+1,state=$2 WHERE submission_id=$1;
-- name: EvaluationNow :one
SELECT clock_timestamp()::timestamptz;
-- name: CreateEvaluationRecord :exec
INSERT INTO evaluation_records(id,submission_id,action,phase,payload,created_at) VALUES($1,$2,$3,$4,$5,$6);
-- name: GetEvaluationRecord :one
SELECT * FROM evaluation_records WHERE id=$1 AND submission_id=$2;
-- name: GetFinalTheoryResult :one
SELECT payload FROM evaluation_records WHERE submission_id=$1 AND action='teorica.finalizada' ORDER BY ((payload->>'version')::bigint) DESC LIMIT 1;
-- name: GetEvaluationCommand :one
SELECT * FROM evaluation_commands WHERE id=$1;
-- name: CreateEvaluationCommand :exec
INSERT INTO evaluation_commands(id,actor_id,submission_id,operation,resource_id,request_hash,response) VALUES($1,$2,$3,$4,$5,$6,$7);
-- name: GetReconsideration :one
SELECT payload FROM reconsiderations WHERE id=$1 AND submission_id=$2;
-- name: CreateReconsideration :exec
INSERT INTO reconsiderations(id,submission_id,phase,result_id,policy_id,status,version,payload,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9);
-- name: UpdateReconsideration :exec
UPDATE reconsiderations SET status=$3,version=$4,payload=$5 WHERE id=$1 AND submission_id=$2;
-- name: ListReconsiderations :many
SELECT payload FROM reconsiderations WHERE submission_id=sqlc.arg(submission_id) AND (NOT sqlc.arg(has_cursor)::boolean OR (created_at,id)<(sqlc.arg(before_at)::timestamptz,sqlc.arg(before_id)::uuid)) ORDER BY created_at DESC,id DESC LIMIT sqlc.arg(row_limit);
-- name: ListEvaluationHistory :many
SELECT * FROM evaluation_records WHERE submission_id=sqlc.arg(submission_id) AND (NOT sqlc.arg(has_cursor)::boolean OR (created_at,id)<(sqlc.arg(before_at)::timestamptz,sqlc.arg(before_id)::uuid)) ORDER BY created_at DESC,id DESC LIMIT sqlc.arg(row_limit);
-- name: CreateEvaluationEvidence :exec
INSERT INTO evaluation_evidence(id,submission_id,applicant_id,actor_id,applicant_upload,name,size_bytes,sha256,storage_key,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10);
-- name: GetEvaluationEvidence :one
SELECT e.*,EXISTS(SELECT 1 FROM evaluation_records p JOIN evaluation_records r ON r.id::text=p.payload->>'fuenteId' AND r.submission_id=p.submission_id WHERE p.submission_id=e.submission_id AND p.action='psicotecnica.publicada' AND r.payload->'respaldo'->>'id'=e.id::text)::boolean AS published FROM evaluation_evidence e WHERE e.id=$1;
-- name: BeginEvaluationUpload :exec
INSERT INTO evaluation_upload_intents(id,submission_id,storage_key,expires_at) VALUES($1,$2,$3,clock_timestamp()+interval '1 hour');
-- name: CompleteEvaluationUpload :exec
DELETE FROM evaluation_upload_intents WHERE id=$1;
-- name: ExpiredEvaluationUploads :many
SELECT * FROM evaluation_upload_intents WHERE expires_at<clock_timestamp() ORDER BY expires_at LIMIT 100;
