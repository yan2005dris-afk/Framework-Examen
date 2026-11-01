-- name: CreateStageNotice :exec
INSERT INTO stage_notices(id,submission_id,source_id,action,template_version) VALUES($1,$2,$3,$4,$5) ON CONFLICT(submission_id,source_id,action) DO NOTHING;
-- name: GetStageNotice :one
SELECT * FROM stage_notices WHERE submission_id=$1 AND source_id=$2 AND action=$3;
-- name: ListSubmissionNotices :many
SELECT m.id,m.kind,m.status,m.attempts,m.total_attempts,m.retry_count,m.last_error_code,m.created_at,m.updated_at,
 COALESCE(n.action,'postulacion.enviada')::text AS action,COALESCE(n.template_version,'ef12-print-v1')::text AS template_version
FROM mail_outbox m LEFT JOIN stage_notices n ON n.id=m.resource_id
WHERE (n.submission_id=sqlc.arg(submission_id) OR (m.resource_id=sqlc.arg(submission_id) AND m.kind='submission.confirmation'))
 AND (NOT sqlc.arg(has_cursor)::boolean OR (m.created_at,m.id)<(sqlc.arg(before_at)::timestamptz,sqlc.arg(before_id)::uuid))
ORDER BY m.created_at DESC,m.id DESC LIMIT sqlc.arg(row_limit);
-- name: LockSubmissionMail :one
SELECT m.* FROM mail_outbox m LEFT JOIN stage_notices n ON n.id=m.resource_id
WHERE m.id=sqlc.arg(mail_id) AND (n.submission_id=sqlc.arg(submission_id) OR (m.resource_id=sqlc.arg(submission_id) AND m.kind='submission.confirmation')) FOR UPDATE OF m;
-- name: RetryStageMail :exec
UPDATE mail_outbox SET status='pending',attempts=0,retry_count=retry_count+1,last_error_code='',next_attempt_at=clock_timestamp(),updated_at=clock_timestamp(),lease_id=NULL WHERE id=$1;
-- name: RecordMailRetry :exec
INSERT INTO mail_delivery_events(mail_id,status,error_code,attempt,cycle,actor_id,reason) VALUES($1,'pending','manual_retry',0,$2,$3,$4);
-- name: ListMailDeliveryEvents :many
SELECT id,status,error_code,attempt,cycle,created_at FROM mail_delivery_events WHERE mail_id=sqlc.arg(mail_id)
 AND (NOT sqlc.arg(has_cursor)::boolean OR (created_at,id)<(sqlc.arg(before_at)::timestamptz,sqlc.arg(before_id)::uuid))
ORDER BY created_at DESC,id DESC LIMIT sqlc.arg(row_limit);
