-- name: EnsurePrivacyControl :exec
INSERT INTO privacy_controls(competition_id) VALUES($1) ON CONFLICT DO NOTHING;
-- name: LockPrivacyControl :one
SELECT * FROM privacy_controls WHERE competition_id=$1 FOR UPDATE;
-- name: GetPrivacyControl :one
SELECT * FROM privacy_controls WHERE competition_id=$1 FOR SHARE;
-- name: SetPrivacyControl :exec
UPDATE privacy_controls SET version=$2,active_id=$3,blocked=$4 WHERE competition_id=$1;
-- name: CreatePrivacyPolicy :exec
INSERT INTO privacy_policies(id,competition_id,version,definition,hash,actor_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7);
-- name: GetPrivacyPolicy :one
SELECT * FROM privacy_policies WHERE id=$1;
-- name: ListPrivacyPolicies :many
SELECT * FROM privacy_policies WHERE competition_id=$1 AND (sqlc.arg(before_version)::bigint=0 OR version<sqlc.arg(before_version)) ORDER BY version DESC LIMIT sqlc.arg(row_limit);
-- name: BindPrivacy :exec
INSERT INTO privacy_bindings(submission_id,policy_id,policy_hash,accepted_at,retention_review_at) VALUES($1,$2,$3,$4,$5);
-- name: GetBoundPrivacy :one
SELECT b.*,p.definition,s.competition_id,pc.blocked FROM privacy_bindings b JOIN privacy_policies p ON p.id=b.policy_id JOIN submissions s ON s.id=b.submission_id JOIN privacy_controls pc ON pc.competition_id=p.competition_id WHERE b.submission_id=$1;
-- name: CreatePrivacyDecision :exec
INSERT INTO privacy_decisions(id,competition_id,blocked,reason,actor_id,created_at) VALUES($1,$2,$3,$4,$5,$6);
-- name: PrivacyReviewDue :many
SELECT b.submission_id,b.policy_id,b.retention_review_at FROM privacy_bindings b JOIN submissions s ON s.id=b.submission_id
WHERE s.competition_id=sqlc.arg(competition_id) AND b.retention_review_at<=clock_timestamp()
 AND (NOT sqlc.arg(has_cursor)::boolean OR (b.retention_review_at,b.submission_id)>(sqlc.arg(after_at)::timestamptz,sqlc.arg(after_id)::uuid))
ORDER BY b.retention_review_at,b.submission_id LIMIT sqlc.arg(row_limit);
-- name: PrivacyPeriodSubmissions :many
SELECT id FROM submissions WHERE competition_id=sqlc.arg(competition_id)::uuid AND status='submitted' ORDER BY id LIMIT 10001;
-- name: MailSubmission :one
SELECT s.id FROM submissions s WHERE s.id=$1
UNION ALL SELECT n.submission_id FROM stage_notices n WHERE n.id=$1 LIMIT 1;
-- name: DocumentSubmissionForPrivacy :one
SELECT submission_id FROM documents WHERE id=$1;

-- name: PrivacyApplicantSubmissions :many
SELECT id FROM submissions WHERE applicant_id=$1 AND status='submitted' ORDER BY id LIMIT 10001;
