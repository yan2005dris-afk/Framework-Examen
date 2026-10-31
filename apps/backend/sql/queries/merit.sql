-- name: EnsureMeritRuleStream :exec
INSERT INTO merit_rule_streams(competition_id,sector) VALUES($1,$2) ON CONFLICT DO NOTHING;
-- name: LockMeritRuleStream :one
SELECT * FROM merit_rule_streams WHERE competition_id=$1 AND sector=$2 FOR UPDATE;
-- name: GetMeritRuleStream :one
SELECT * FROM merit_rule_streams WHERE competition_id=$1 AND sector=$2 FOR SHARE;
-- name: SetMeritRuleStream :exec
UPDATE merit_rule_streams SET version=$3,active_rule_id=$4 WHERE competition_id=$1 AND sector=$2;
-- name: CreateMeritRules :exec
INSERT INTO merit_rule_sets(id,competition_id,sector,version,definition,definition_hash,actor_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8);
-- name: GetMeritRules :one
SELECT * FROM merit_rule_sets WHERE id=$1;
-- name: ApproveMeritRules :exec
UPDATE merit_rule_sets SET approved=true,approval_reference=$2,approved_by=$3,approved_at=now() WHERE id=$1 AND NOT approved;
-- name: ListMeritRules :many
SELECT * FROM merit_rule_sets WHERE competition_id=sqlc.arg(competition_id) AND sector=sqlc.arg(sector) AND (sqlc.arg(before_version)::bigint=0 OR version<sqlc.arg(before_version)) ORDER BY version DESC LIMIT sqlc.arg(row_limit);
-- name: EnsureMeritCase :exec
INSERT INTO merit_case_streams(submission_id) VALUES($1) ON CONFLICT DO NOTHING;
-- name: LockMeritCase :one
SELECT * FROM merit_case_streams WHERE submission_id=$1 FOR UPDATE;
-- name: GetMeritCase :one
SELECT * FROM merit_case_streams WHERE submission_id=$1 FOR SHARE;
-- name: SetMeritCase :exec
UPDATE merit_case_streams SET review_id=$2,calculation_id=$3,publication_id=$4 WHERE submission_id=$1;
-- name: CreateMeritReview :exec
INSERT INTO merit_reviews(id,submission_id,version,rule_set_id,payload) VALUES($1,$2,$3,$4,$5);
-- name: GetMeritReview :one
SELECT payload FROM merit_reviews WHERE id=$1 AND submission_id=$2;
-- name: CreateMeritCalculation :exec
INSERT INTO merit_calculations(id,submission_id,version,review_id,rule_set_id,payload) VALUES($1,$2,$3,$4,$5,$6);
-- name: GetMeritCalculation :one
SELECT payload FROM merit_calculations WHERE id=$1 AND submission_id=$2;
-- name: CreateMeritPublication :exec
INSERT INTO merit_publications(id,submission_id,version,calculation_id,supersedes_id,decision,payload) VALUES($1,$2,$3,$4,$5,$6,$7);
-- name: GetMeritPublication :one
SELECT payload FROM merit_publications WHERE id=$1 AND submission_id=$2;
-- name: GetMeritCalendar :one
SELECT c.opens_at,c.closes_at,s.approved,s.published,s.sectors,s.phases,clock_timestamp()::timestamptz AS checked_at
FROM competitions c JOIN competition_controls s ON s.competition_id=c.id WHERE c.id=$1 FOR SHARE OF c,s;
