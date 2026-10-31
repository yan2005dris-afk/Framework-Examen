-- name: CreateApplicant :one
INSERT INTO applicants (
    id, cedula, nombres, apellidos, correo, password_hash, metodo_validacion, is_verified, role, fecha_registro
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, now()
) RETURNING *;

-- name: GetApplicantByID :one
SELECT * FROM applicants
WHERE id = $1 LIMIT 1;

-- name: GetApplicantByCedula :one
SELECT * FROM applicants
WHERE cedula = $1 LIMIT 1;

-- name: GetApplicantByEmail :one
SELECT * FROM applicants
WHERE correo = $1 LIMIT 1;

-- name: UpdateApplicantPassword :exec
UPDATE applicants
SET password_hash = $2, updated_at = now()
WHERE id = $1;

-- name: UpdateApplicantRoleByCedula :execrows
UPDATE applicants
SET role = $2, updated_at = now()
WHERE cedula = $1;

-- name: CreateSession :one
INSERT INTO sessions (
    id, applicant_id, token_hash, expires_at, ip, user_agent, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, now()
) RETURNING *;

-- name: GetSessionWithApplicant :one
SELECT 
    s.id AS session_id,
    s.token_hash,
    s.expires_at,
    s.ip,
    s.user_agent,
    s.created_at AS session_created_at,
    a.id AS applicant_id,
    a.cedula,
    a.nombres,
    a.apellidos,
    a.correo,
    a.role,
    a.metodo_validacion,
    a.is_verified,
    a.fecha_registro
FROM sessions s
JOIN applicants a ON s.applicant_id = a.id
WHERE s.token_hash = $1 AND s.expires_at > now()
LIMIT 1;

-- name: DeleteSessionByTokenHash :exec
DELETE FROM sessions
WHERE token_hash = $1;

-- name: DeleteSessionsByApplicantID :exec
DELETE FROM sessions
WHERE applicant_id = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at <= now();

-- name: LockApplicantCredentials :one
SELECT id,password_hash FROM applicants WHERE id=$1 FOR UPDATE;

-- name: CreatePasswordReset :one
INSERT INTO password_resets (
    id, applicant_id, token_hash, expires_at, used, created_at
) VALUES (
    $1, $2, $3, $4, false, now()
) RETURNING *;

-- name: GetValidPasswordReset :one
SELECT * FROM password_resets
WHERE token_hash = $1 AND used = false AND expires_at > now()
LIMIT 1;

-- name: MarkPasswordResetUsed :exec
UPDATE password_resets
SET used = true
WHERE token_hash = $1;

