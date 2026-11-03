-- name: EnsureExamBankStream :exec
INSERT INTO exam_bank_streams(competition_id,sector) VALUES($1,$2) ON CONFLICT DO NOTHING;
-- name: LockExamBankStream :one
SELECT * FROM exam_bank_streams WHERE competition_id=$1 AND sector=$2 FOR UPDATE;
-- name: SetExamBankStream :exec
UPDATE exam_bank_streams SET version=$3 WHERE competition_id=$1 AND sector=$2;
-- name: CreateExamBank :exec
INSERT INTO exam_banks(id,competition_id,sector,version,definition,hash,actor_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8);
-- name: GetExamBank :one
SELECT * FROM exam_banks WHERE id=$1;
-- name: ListExamBanks :many
SELECT id,version,hash,actor_id,created_at FROM exam_banks WHERE competition_id=$1 AND sector=$2 AND (sqlc.arg(before_version)::bigint=0 OR version<sqlc.arg(before_version)) ORDER BY version DESC LIMIT sqlc.arg(row_limit);
-- name: CreateExamConfig :exec
INSERT INTO exam_configs(id,bank_id,competition_id,sector,definition,hash,actor_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8);
-- name: GetExamConfig :one
SELECT * FROM exam_configs WHERE id=$1;
-- name: CreateExamEnrollment :exec
INSERT INTO exam_enrollments(submission_id,config_id,merit_id,actor_id,created_at) VALUES($1,$2,$3,$4,$5);
-- name: GetExamEnrollment :one
SELECT * FROM exam_enrollments WHERE submission_id=$1;
-- name: LockExamConfig :exec
SELECT id FROM exam_configs WHERE id=$1 FOR UPDATE;
-- name: ExamAssignments :many
SELECT assignment_hash FROM exam_attempts WHERE config_id=$1;
-- name: GetExamAttempt :one
SELECT * FROM exam_attempts WHERE submission_id=$1;
-- name: LockExamAttempt :one
SELECT * FROM exam_attempts WHERE submission_id=$1 FOR UPDATE;
-- name: CreateExamAttempt :exec
INSERT INTO exam_attempts(id,submission_id,config_id,assignment_hash,payload,version,status,expires_at,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9);
-- name: UpdateExamAttempt :exec
UPDATE exam_attempts SET payload=$2,version=$3,status=$4 WHERE id=$1;
-- name: CreateExamEvent :exec
INSERT INTO exam_events(attempt_id,version,payload,created_at) VALUES($1,$2,$3,$4);
-- name: DueExamAttempts :many
SELECT submission_id FROM exam_attempts WHERE status='active' AND expires_at<=clock_timestamp() ORDER BY expires_at,id LIMIT 100;
-- name: CreateExamReport :exec
INSERT INTO exam_reports(attempt_id,submission_id,result_id,pdf,sha256,created_at) VALUES($1,$2,$3,$4,$5,$6);
-- name: GetExamReport :one
SELECT * FROM exam_reports WHERE submission_id=$1;
