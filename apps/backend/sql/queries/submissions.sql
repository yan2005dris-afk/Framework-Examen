-- name: CreateSubmission :one
INSERT INTO submissions (id, applicant_id, code, status, content, created_at, updated_at, competition_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, sqlc.narg(competition_id))
RETURNING *;

-- name: GetSubmissionByID :one
SELECT * FROM submissions WHERE id = $1;

-- name: GetSubmissionForUpdate :one
SELECT * FROM submissions WHERE id = $1 FOR UPDATE;

-- name: ListSubmissionsByApplicant :many
SELECT * FROM submissions WHERE applicant_id = $1 ORDER BY updated_at DESC LIMIT $2 OFFSET $3;

-- name: CountSubmissionsByApplicant :one
SELECT count(*) FROM submissions WHERE applicant_id = $1;

-- name: ListSubmissionsByApplicantCursorFirst :many
SELECT * FROM submissions
WHERE applicant_id = $1
ORDER BY updated_at DESC, id DESC
LIMIT $2;

-- name: ListSubmissionsByApplicantCursor :many
SELECT * FROM submissions
WHERE applicant_id = $1
  AND (updated_at < $2 OR (updated_at = $2 AND id < $3))
ORDER BY updated_at DESC, id DESC
LIMIT $4;

-- name: UpdateSubmission :one
UPDATE submissions
SET content = $2, status = $3, hash = $4, submitted_at = $5, updated_at = $6
WHERE id = $1 AND status <> 'submitted'
RETURNING *;

-- name: DeleteSubmission :execrows
DELETE FROM submissions WHERE id = $1 AND status <> 'submitted';
