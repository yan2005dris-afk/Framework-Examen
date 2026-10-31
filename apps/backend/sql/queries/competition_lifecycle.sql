-- name: CreateCompetitionLifecycle :exec
INSERT INTO competitions(id,code,title,opens_at,closes_at) VALUES($1,$2,$3,$4,$5);
-- name: CreateCompetitionControl :exec
INSERT INTO competition_controls(competition_id,sectors,phases) VALUES($1,$2,$3);
-- name: LockCompetitionControl :exec
SELECT competition_id FROM competition_controls WHERE competition_id=$1 FOR UPDATE;
-- name: GetCompetitionLifecycle :one
SELECT c.id,c.code,c.title,c.opens_at,c.closes_at,s.version,s.approved,s.published,s.form_enabled,s.sectors,s.phases,s.approved_at,s.approved_by
FROM competitions c JOIN competition_controls s ON s.competition_id=c.id WHERE c.id=$1;
-- name: ListPublicCompetitions :many
SELECT c.id,c.code,c.title,c.opens_at,c.closes_at,s.version,s.approved,s.published,s.form_enabled,s.sectors,s.phases,s.approved_at,s.approved_by
FROM competitions c JOIN competition_controls s ON s.competition_id=c.id
WHERE s.approved AND s.published AND (sqlc.narg(before_id)::uuid IS NULL OR c.id<sqlc.narg(before_id)::uuid)
ORDER BY c.id DESC LIMIT $1;
-- name: UpdateCompetitionControl :exec
UPDATE competition_controls SET version=version+1,approved=$2,published=$3,form_enabled=$4,sectors=$5,phases=$6,
approved_at=CASE WHEN $2::boolean AND NOT approved THEN now() WHEN $2::boolean THEN approved_at ELSE NULL END,
approved_by=CASE WHEN $2::boolean AND NOT approved THEN $7 WHEN $2::boolean THEN approved_by ELSE NULL END
WHERE competition_id=$1;
-- name: RecordCompetitionVersion :exec
INSERT INTO competition_versions(competition_id,version,configuration,actor_id) VALUES($1,$2,$3,$4);
-- name: RecordEligibilityCheck :exec
INSERT INTO eligibility_checks(id,submission_id,applicant_id,competition_id,sector,result,source,checked_at,assessment)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9);
