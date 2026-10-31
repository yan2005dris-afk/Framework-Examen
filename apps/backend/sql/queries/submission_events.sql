-- name: CreateSubmissionEvent :exec
INSERT INTO submission_events (id, submission_id, applicant_id, event_type, created_at, actor_role)
VALUES ($1, $2, $3, $4, $5, COALESCE((SELECT role FROM applicants WHERE id=$3),'no_registrado'));
