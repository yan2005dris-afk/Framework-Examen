-- Secciones del formulario de postulación en tablas normalizadas (00024).
-- Borrar la fila de submission_sections elimina en cascada los datos de la sección.

-- name: DeleteFormSection :exec
DELETE FROM submission_sections WHERE submission_id = $1 AND section = $2;

-- name: InsertFormSection :exec
INSERT INTO submission_sections (submission_id, section, items_declared, saved_at) VALUES ($1, $2, $3, $4);

-- name: InsertFormPersonal :exec
INSERT INTO submission_personal (submission_id, first_name, last_name, gender, birth_date, nationality, marital_status, spouse_name, spouse_id_card)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: InsertFormContact :exec
INSERT INTO submission_contact (submission_id, country_id, province_id, province_name, city_id, city_name, mobile, secondary_email,
    main_street, number, secondary_street, area, reference, phone)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);

-- name: InsertFormAdditional :exec
INSERT INTO submission_additional (submission_id, has_relatives) VALUES ($1, $2);

-- name: InsertFormRelative :exec
INSERT INTO submission_relatives (submission_id, position, record_key, name, relationship, institution, area, job_title)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: InsertFormPosition :exec
INSERT INTO submission_position (submission_id, eligible, verification_detail, has_organization, organization, document_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: InsertFormDegree :exec
INSERT INTO submission_degrees (submission_id, position, record_key, level, institution, type, title, area, registry, registry_date, document_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: InsertFormExperience :exec
INSERT INTO submission_experiences (submission_id, position, record_key, type, job_title, institution, start_date, end_date, is_current,
    activities, functions, document_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: InsertFormPublicationsInfo :exec
INSERT INTO submission_publications_info (submission_id, has_publications) VALUES ($1, $2);

-- name: InsertFormPublication :exec
INSERT INTO submission_publications (submission_id, position, record_key, title, type, medium, published_on, related, document_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: InsertFormTraining :exec
INSERT INTO submission_trainings (submission_id, position, record_key, type, name, institution, start_date, end_date, hours, document_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: InsertFormDisqualifications :exec
INSERT INTO submission_disqualifications (submission_id, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);

-- name: InsertFormRemovalDetails :exec
INSERT INTO submission_removal_details (submission_id, institution, job_title, end_date, reason) VALUES ($1, $2, $3, $4, $5);

-- name: DeleteFormConsent :exec
DELETE FROM submission_consents WHERE submission_id = $1;

-- name: InsertFormConsent :exec
INSERT INTO submission_consents (submission_id, privacy_policy_id, privacy_policy_hash, accepted_at, data_processing, truthfulness,
    competition_rules, name_publication, physical_delivery, signature, signed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: FindCountryID :one
SELECT id FROM catalogs WHERE kind = 'countries' AND parent_key = '' AND value = $1;

-- name: FindChildCatalogID :one
-- La clave compuesta de submission_contact garantiza que el hijo sea del tipo correcto.
SELECT id FROM catalogs WHERE parent_id = $1 AND value = $2;

-- name: ListFormDocuments :many
SELECT id, section, record_id, name, size_bytes, sha256, created_at FROM documents WHERE submission_id = $1;

-- name: GetFormIdentity :one
SELECT a.cedula, a.correo, COALESCE(s.sector, '')::text AS sector
FROM submissions s JOIN applicants a ON a.id = s.applicant_id WHERE s.id = $1;

-- name: ListFormSections :many
SELECT section, items_declared FROM submission_sections WHERE submission_id = $1 ORDER BY section;

-- name: GetFormPersonal :one
SELECT first_name, last_name, gender, birth_date, nationality, marital_status, spouse_name, spouse_id_card
FROM submission_personal WHERE submission_id = $1;

-- name: GetFormContact :one
SELECT COALESCE(co.value, '')::text AS country, COALESCE(pr.value, c.province_name)::text AS province,
    COALESCE(ci.value, c.city_name)::text AS city, c.mobile, c.secondary_email, c.main_street, c.number,
    c.secondary_street, c.area, c.reference, c.phone
FROM submission_contact c
LEFT JOIN catalogs co ON co.id = c.country_id
LEFT JOIN catalogs pr ON pr.id = c.province_id
LEFT JOIN catalogs ci ON ci.id = c.city_id
WHERE c.submission_id = $1;

-- name: GetFormAdditional :one
SELECT has_relatives FROM submission_additional WHERE submission_id = $1;

-- name: ListFormRelatives :many
SELECT record_key, name, relationship, institution, area, job_title FROM submission_relatives WHERE submission_id = $1 ORDER BY position;

-- name: GetFormPosition :one
SELECT eligible, verification_detail, has_organization, organization, document_id FROM submission_position WHERE submission_id = $1;

-- name: ListFormDegrees :many
SELECT record_key, level, institution, type, title, area, registry, registry_date, document_id
FROM submission_degrees WHERE submission_id = $1 ORDER BY position;

-- name: ListFormExperiences :many
SELECT record_key, type, job_title, institution, start_date, end_date, is_current, activities, functions, document_id
FROM submission_experiences WHERE submission_id = $1 ORDER BY position;

-- name: GetFormPublicationsInfo :one
SELECT has_publications FROM submission_publications_info WHERE submission_id = $1;

-- name: ListFormPublications :many
SELECT record_key, title, type, medium, published_on, related, document_id
FROM submission_publications WHERE submission_id = $1 ORDER BY position;

-- name: ListFormTrainings :many
SELECT record_key, type, name, institution, start_date, end_date, hours, document_id
FROM submission_trainings WHERE submission_id = $1 ORDER BY position;

-- name: GetFormDisqualifications :one
SELECT q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13 FROM submission_disqualifications WHERE submission_id = $1;

-- name: GetFormRemovalDetails :one
SELECT institution, job_title, end_date, reason FROM submission_removal_details WHERE submission_id = $1;

-- name: GetFormConsent :one
SELECT privacy_policy_id, privacy_policy_hash, accepted_at, data_processing, truthfulness, competition_rules, name_publication,
    physical_delivery, signature, signed_at
FROM submission_consents WHERE submission_id = $1;
