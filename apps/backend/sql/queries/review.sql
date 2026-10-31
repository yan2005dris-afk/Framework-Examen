-- name: ListReviewCompetitions :many
SELECT * FROM competitions
WHERE (NOT sqlc.arg(has_cursor)::boolean OR (created_at,id) < (sqlc.arg(before_at)::timestamptz,sqlc.arg(before_id)::uuid))
ORDER BY created_at DESC,id DESC LIMIT sqlc.arg(row_limit);

-- name: GetReviewCompetition :one
SELECT * FROM competitions WHERE id = $1;

-- name: ListReviewSubmissions :many
SELECT s.id,s.code,s.status,s.content,s.hash,s.submitted_at,s.created_at,
    COALESCE(c.code,'SIN-CONVOCATORIA')::text AS competition_code
FROM submissions s LEFT JOIN competitions c ON c.id = s.competition_id
WHERE s.status = 'submitted' AND s.hash IS NOT NULL AND s.hash <> '' AND s.submitted_at IS NOT NULL
    AND s.competition_id IS NOT DISTINCT FROM sqlc.narg(competition_id)::uuid
    AND (sqlc.arg(sector)::text = '' OR s.content->'seccion4'->>'sector' = sqlc.arg(sector))
    AND (sqlc.arg(status)::text = '' OR s.status = sqlc.arg(status))
    AND (NOT sqlc.arg(has_cursor)::boolean OR (s.created_at,s.id) < (sqlc.arg(before_at)::timestamptz,sqlc.arg(before_id)::uuid))
ORDER BY s.created_at DESC,s.id DESC LIMIT sqlc.arg(row_limit);

-- name: CountReviewSubmissions :many
SELECT COALESCE(content->'seccion4'->>'sector','sin_sector')::text AS sector,status,
    count(DISTINCT applicant_id)::bigint AS applicants,count(*)::bigint AS total
FROM submissions
WHERE status = 'submitted' AND hash IS NOT NULL AND hash <> '' AND submitted_at IS NOT NULL
    AND competition_id IS NOT DISTINCT FROM sqlc.narg(competition_id)::uuid
    AND (sqlc.arg(sector)::text = '' OR content->'seccion4'->>'sector' = sqlc.arg(sector))
    AND (sqlc.arg(status)::text = '' OR status = sqlc.arg(status))
GROUP BY content->'seccion4'->>'sector',status ORDER BY sector,status;

-- name: ReviewExportBudget :one
SELECT count(*)::bigint AS total,COALESCE(sum(octet_length(content::text)),0)::bigint AS bytes
FROM submissions
WHERE status = 'submitted' AND hash IS NOT NULL AND hash <> '' AND submitted_at IS NOT NULL
    AND competition_id IS NOT DISTINCT FROM sqlc.narg(competition_id)::uuid
    AND (sqlc.arg(sector)::text = '' OR content->'seccion4'->>'sector' = sqlc.arg(sector))
    AND (sqlc.arg(status)::text = '' OR status = sqlc.arg(status));
