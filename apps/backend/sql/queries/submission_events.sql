-- name: CreateSubmissionEvent :exec
INSERT INTO submission_events (id, submission_id, applicant_id, event_type, created_at)
VALUES ($1, $2, $3, $4, $5);
