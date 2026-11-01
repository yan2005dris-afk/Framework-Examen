-- name: MailOutboxReady :one
SELECT EXISTS(SELECT 1 FROM mail_outbox WHERE false)::boolean AS ready;

-- name: EnqueueMail :exec
INSERT INTO mail_outbox(id,dedup_key,kind,resource_id,ciphertext,expires_at)
SELECT sqlc.arg(id),sqlc.arg(dedup_key),sqlc.arg(kind)::varchar,sqlc.narg(resource_id),sqlc.arg(ciphertext),sqlc.arg(expires_at)
WHERE sqlc.narg(resource_id)::uuid IS NULL OR NOT EXISTS (
 SELECT 1 FROM mail_outbox WHERE resource_id=sqlc.narg(resource_id)::uuid AND kind=sqlc.arg(kind)::varchar
)
ON CONFLICT(dedup_key) DO NOTHING;

-- name: ClaimMail :many
WITH candidates AS (
    SELECT id FROM mail_outbox
    WHERE status IN ('pending','processing') AND next_attempt_at<=now()
        AND expires_at>now() AND attempts<5
    ORDER BY created_at,id LIMIT 10 FOR UPDATE SKIP LOCKED
), claimed AS (
UPDATE mail_outbox m SET status='processing',attempts=m.attempts+1,
    total_attempts=m.total_attempts+1,next_attempt_at=now()+interval '5 minutes',lease_id=sqlc.arg(lease_id),updated_at=now()
FROM candidates c WHERE m.id=c.id RETURNING m.*
), logged AS (INSERT INTO mail_delivery_events(mail_id,status,error_code,attempt,cycle) SELECT id,'processing','',attempts,retry_count FROM claimed)
SELECT * FROM claimed;

-- name: FinishMail :execrows
WITH finished AS (
UPDATE mail_outbox m SET status=sqlc.arg(status)::varchar,last_error_code=sqlc.arg(error_code),
    next_attempt_at=sqlc.arg(next_attempt_at),updated_at=now(),
    ciphertext=CASE WHEN sqlc.arg(status)::varchar IN ('pending','failed') AND expires_at>now() THEN ciphertext ELSE ''::bytea END,
    lease_id=NULL
WHERE m.id=sqlc.arg(id) AND m.lease_id=sqlc.arg(lease_id) AND m.status='processing' RETURNING m.*
) INSERT INTO mail_delivery_events(mail_id,status,error_code,attempt,cycle) SELECT f.id,f.status,f.last_error_code,f.attempts,f.retry_count FROM finished f;

-- name: ExpireMail :exec
WITH finished AS (
UPDATE mail_outbox SET status=CASE WHEN expires_at<=now() THEN 'expired' ELSE 'failed' END,
    ciphertext=CASE WHEN expires_at<=now() THEN ''::bytea ELSE ciphertext END,last_error_code=CASE WHEN expires_at<=now() THEN 'expired' ELSE 'attempts_exhausted' END,updated_at=now(),lease_id=NULL
WHERE (status IN ('pending','processing') AND (expires_at<=now() OR (attempts>=5 AND next_attempt_at<=now()))) OR (status='failed' AND expires_at<=now())
RETURNING *
) INSERT INTO mail_delivery_events(mail_id,status,error_code,attempt,cycle) SELECT f.id,f.status,f.last_error_code,f.attempts,f.retry_count FROM finished f;

-- name: MailStatusByResource :one
SELECT status,total_attempts AS attempts,last_error_code,updated_at FROM mail_outbox
WHERE resource_id=$1 ORDER BY created_at DESC,id DESC LIMIT 1;
