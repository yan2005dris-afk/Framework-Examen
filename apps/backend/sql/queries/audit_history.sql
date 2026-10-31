-- name: GetHistoryOwner :one
SELECT s.applicant_id, CASE WHEN s.status = 'submitted' AND (s.hash IS NULL OR s.hash = '' OR s.submitted_at IS NULL)
    THEN 'unsealed' ELSE s.status END::text AS status FROM submissions s WHERE s.id = sqlc.arg(target_id)
UNION ALL
SELECT e.applicant_id, 'deleted'::text AS status FROM submission_events e
    WHERE e.submission_id = sqlc.arg(target_id) AND NOT EXISTS (SELECT 1 FROM submissions s WHERE s.id = sqlc.arg(target_id))
LIMIT 1;

-- name: ListSubmissionHistory :many
SELECT id, actor_id, actor_role, action, resource_id::uuid AS resource_id, resource_type, result, created_at
FROM (
    SELECT id, applicant_id AS actor_id, se.actor_role::text AS actor_role,
        event_type::text AS action,
        CASE WHEN event_type IN ('documento.cargado','documento.eliminado') AND payload->>'documentId' ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
            THEN (payload->>'documentId')::uuid ELSE submission_id END::uuid AS resource_id,
        CASE WHEN event_type IN ('documento.cargado','documento.eliminado') THEN 'documento' ELSE 'postulacion' END::text AS resource_type,
        'permitido'::text AS result, created_at FROM submission_events se WHERE se.submission_id = sqlc.arg(target_id)
    UNION ALL
    SELECT id, actor_id, actor_role::text, action::text, resource_id, resource_type::text, result::text, created_at
        FROM administrative_access_events ae WHERE ae.submission_id = sqlc.arg(target_id)
    UNION ALL
    SELECT id, COALESCE(actor_id,'00000000-0000-0000-0000-000000000000'::uuid), actor_role::text, action::text, resource_id, resource_type::text, result::text, created_at
        FROM workflow_events we WHERE we.submission_id = sqlc.arg(target_id)
    UNION ALL
    SELECT re.id,COALESCE(re.actor_id,'00000000-0000-0000-0000-000000000000'::uuid),
        'no_registrado'::text,'rol.actualizado'::text,re.applicant_id,'cuenta'::text,'permitido'::text,re.created_at
    FROM applicant_role_events re WHERE re.applicant_id IN (
        SELECT s.applicant_id FROM submissions s WHERE s.id = sqlc.arg(target_id)
        UNION SELECT se.applicant_id FROM submission_events se WHERE se.submission_id = sqlc.arg(target_id)
    )
) AS history
WHERE (NOT sqlc.arg(has_cursor)::boolean OR (created_at, id) < (sqlc.arg(before_at)::timestamptz, sqlc.arg(before_id)::uuid))
ORDER BY created_at DESC, id DESC LIMIT sqlc.arg(row_limit);

-- name: CreateWorkflowEvent :exec
INSERT INTO workflow_events (submission_id,actor_id,actor_role,action,resource_id,resource_type,result,created_at)
VALUES (sqlc.narg(submission_id),sqlc.narg(actor_id),sqlc.arg(actor_role),sqlc.arg(action),sqlc.arg(resource_id),sqlc.arg(resource_type),sqlc.arg(result),sqlc.arg(created_at));
