-- name: MailOutboxReady :one
SELECT EXISTS(SELECT 1 FROM mail_outbox WHERE false)::boolean AS ready;

-- name: EnqueueMail :exec
INSERT INTO mail_outbox(id,dedup_key,kind,resource_id,ciphertext,expires_at)
VALUES(sqlc.arg(id),sqlc.arg(dedup_key),sqlc.arg(kind),sqlc.narg(resource_id),sqlc.arg(ciphertext),sqlc.arg(expires_at))
ON CONFLICT(dedup_key) DO NOTHING;

-- name: ClaimMail :many
WITH candidates AS (
    SELECT id FROM mail_outbox
    WHERE status IN ('pending','processing') AND next_attempt_at<=now()
        AND expires_at>now() AND attempts<5
    ORDER BY created_at,id LIMIT 10 FOR UPDATE SKIP LOCKED
)
UPDATE mail_outbox m SET status='processing',attempts=m.attempts+1,
    next_attempt_at=now()+interval '5 minutes',lease_id=sqlc.arg(lease_id),updated_at=now()
FROM candidates c WHERE m.id=c.id RETURNING m.*;

-- name: FinishMail :execrows
UPDATE mail_outbox SET status=sqlc.arg(status),last_error_code=sqlc.arg(error_code),
    next_attempt_at=sqlc.arg(next_attempt_at),updated_at=now(),
    ciphertext=CASE WHEN sqlc.arg(status)::text='pending' THEN ciphertext ELSE ''::bytea END,
    lease_id=NULL
WHERE id=sqlc.arg(id) AND lease_id=sqlc.arg(lease_id) AND status='processing';

-- name: ExpireMail :exec
UPDATE mail_outbox SET status=CASE WHEN expires_at<=now() THEN 'expired' ELSE 'failed' END,
    ciphertext=''::bytea,last_error_code=CASE WHEN expires_at<=now() THEN 'expired' ELSE 'attempts_exhausted' END,updated_at=now(),lease_id=NULL
WHERE status IN ('pending','processing') AND
    (expires_at<=now() OR (attempts>=5 AND next_attempt_at<=now()));

-- name: MailStatusByResource :one
SELECT status,attempts,last_error_code,updated_at FROM mail_outbox
WHERE resource_id=$1 ORDER BY created_at DESC,id DESC LIMIT 1;
