-- name: CreateDownloadGrant :exec
INSERT INTO download_grants(token_hash,actor_id,session_hash,storage_key,actor_role,policy_version,expires_at)
VALUES($1,$2,$3,$4,$5,$6,clock_timestamp()+interval '5 minutes');

-- name: GetLiveDownloadGrant :one
SELECT * FROM download_grants WHERE token_hash=$1 AND expires_at>clock_timestamp();

-- name: GetDownloadSession :one
SELECT a.id,a.role FROM sessions s JOIN applicants a ON a.id=s.applicant_id
WHERE s.token_hash=$1 AND s.expires_at>clock_timestamp();

-- name: GetDownloadTarget :one
SELECT d.id,d.submission_id,s.applicant_id,s.status,d.name,d.sha256,d.size_bytes,'documento'::text AS kind
FROM documents d JOIN submissions s ON s.id=d.submission_id WHERE d.storage_key=$1
UNION ALL
SELECT e.id,e.submission_id,s.applicant_id,s.status,e.name,e.sha256,e.size_bytes,'evaluacion'::text AS kind
FROM evaluation_evidence e JOIN submissions s ON s.id=e.submission_id WHERE e.storage_key=$1;

-- name: PurgeExpiredDownloadGrants :execrows
DELETE FROM download_grants WHERE token_hash IN (SELECT token_hash FROM download_grants WHERE expires_at<=clock_timestamp() ORDER BY expires_at LIMIT 1000 FOR UPDATE SKIP LOCKED);

-- name: GetDownloadPrivacyVersion :one
SELECT COALESCE((SELECT c.version FROM privacy_controls c JOIN submissions s ON s.competition_id=c.competition_id WHERE s.id=$1),0)::bigint AS version;

-- name: CreateDownloadAccess :exec
INSERT INTO download_access_events(actor_id,actor_role,submission_id,resource_id,request_id)
VALUES(sqlc.arg(actor_id),sqlc.arg(actor_role),sqlc.arg(submission_id),sqlc.arg(resource_id),sqlc.narg(request_id));
