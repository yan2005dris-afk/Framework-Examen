-- name: CreateAdministrativeAccessEvent :exec
INSERT INTO administrative_access_events
    (actor_id, actor_role, action, resource_id, resource_type, result, created_at, submission_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, sqlc.narg(submission_id));

-- name: LockApplicantsForRoleChange :many
SELECT id, role FROM applicants WHERE id = ANY(sqlc.arg(ids)::uuid[])
ORDER BY id FOR UPDATE;

-- name: SetRoleChangeContext :exec
SELECT set_config('app.actor_id', sqlc.arg(actor_id)::text, true),
       set_config('app.operator', sqlc.arg(operator)::text, true),
       set_config('app.role_reason', sqlc.arg(reason)::text, true),
       'applicant_role_events'::regclass;

-- name: ChangeApplicantRole :one
UPDATE applicants SET role = $2, updated_at = now() WHERE id = $1 RETURNING *;
