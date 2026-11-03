-- name: LockResultsCalendar :one
SELECT c.*,ct.approved,ct.published,ct.sectors,ct.phases FROM competitions c JOIN competition_controls ct ON ct.competition_id=c.id WHERE c.id=$1 FOR SHARE OF c FOR UPDATE OF ct;
-- name: ResultSourceBudget :one
SELECT count(*)::bigint AS total,COALESCE(sum(octet_length(content::text)+COALESCE((SELECT sum(octet_length(payload::text)) FROM reconsiderations a WHERE a.submission_id=submissions.id),0)+COALESCE((SELECT octet_length(state::text) FROM evaluation_heads e WHERE e.submission_id=submissions.id),0)+COALESCE((SELECT sum(octet_length(payload::text)) FROM evaluation_records e WHERE e.submission_id=submissions.id AND e.action='teorica.finalizada'),0)+COALESCE((SELECT sum(octet_length(payload::text)) FROM merit_publications m WHERE m.submission_id=submissions.id),0)),0)::bigint AS bytes FROM submissions WHERE competition_id=sqlc.arg(competition_id)::uuid AND content->'seccion4'->>'sector'=sqlc.arg(sector)::text AND status='submitted';
-- name: ListResultSources :many
SELECT s.*,COALESCE(mp.payload,'{}'::jsonb)::jsonb AS merit,COALESCE(e.state,'{}'::jsonb)::jsonb AS evaluation,
 COALESCE((SELECT payload FROM evaluation_records t WHERE t.submission_id=s.id AND t.action='teorica.finalizada' ORDER BY ((t.payload->>'version')::bigint) DESC LIMIT 1),'{}'::jsonb)::jsonb AS theory,
 COALESCE((SELECT jsonb_agg(a.payload ORDER BY a.id) FROM reconsiderations a WHERE a.submission_id=s.id),'[]'::jsonb)::jsonb AS appeals
FROM submissions s LEFT JOIN merit_case_streams ms ON ms.submission_id=s.id LEFT JOIN merit_publications mp ON mp.id=ms.publication_id AND mp.submission_id=s.id
LEFT JOIN evaluation_heads e ON e.submission_id=s.id
WHERE s.competition_id=sqlc.arg(competition_id)::uuid AND s.content->'seccion4'->>'sector'=sqlc.arg(sector)::text AND s.status='submitted' ORDER BY s.id;
-- name: EnsureResultListStream :exec
INSERT INTO result_list_streams(competition_id,sector) VALUES($1,$2) ON CONFLICT DO NOTHING;
-- name: LockResultListStream :one
SELECT * FROM result_list_streams WHERE competition_id=$1 AND sector=$2 FOR UPDATE;
-- name: GetResultListStream :one
SELECT * FROM result_list_streams WHERE competition_id=$1 AND sector=$2;
-- name: SetResultListStream :exec
UPDATE result_list_streams SET version=$3,published_version=$4,published_id=$5 WHERE competition_id=$1 AND sector=$2;
-- name: CreateResultList :exec
INSERT INTO result_lists(id,competition_id,sector,version,sources_hash,payload,actor_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8);
-- name: GetResultList :one
SELECT * FROM result_lists WHERE id=$1;
-- name: ListResultLists :many
SELECT id,version,sources_hash,actor_id,created_at FROM result_lists WHERE competition_id=sqlc.arg(competition_id) AND sector=sqlc.arg(sector)
 AND (NOT sqlc.arg(has_cursor)::boolean OR (created_at,id)<(sqlc.arg(before_at)::timestamptz,sqlc.arg(before_id)::uuid)) ORDER BY created_at DESC,id DESC LIMIT sqlc.arg(row_limit);
-- name: CreateResultListPublication :exec
INSERT INTO result_list_publications(id,list_id,competition_id,sector,version,previous_id,request_hash,approval_reference,privacy_reference,is_public,public_payload,actor_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13);
-- name: GetResultListPublication :one
SELECT * FROM result_list_publications WHERE id=$1;
-- name: ListResultListPublications :many
SELECT id,list_id,version,is_public,actor_id,created_at FROM result_list_publications WHERE competition_id=sqlc.arg(competition_id) AND sector=sqlc.arg(sector)
 AND (NOT sqlc.arg(has_cursor)::boolean OR (created_at,id)<(sqlc.arg(before_at)::timestamptz,sqlc.arg(before_id)::uuid)) ORDER BY created_at DESC,id DESC LIMIT sqlc.arg(row_limit);
-- name: GetResultListCommand :one
SELECT * FROM result_list_commands WHERE id=$1;
-- name: CreateResultListCommand :exec
INSERT INTO result_list_commands(id,actor_id,operation,scope,request_hash,resource_id) VALUES($1,$2,$3,$4,$5,$6);
