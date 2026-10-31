-- name: CreateDocument :exec
INSERT INTO documents (id, submission_id, section, record_id, name, mime, size_bytes, sha256, storage_key, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10);

-- name: GetDocument :one
SELECT * FROM documents WHERE id = $1;

-- name: ListDocuments :many
SELECT * FROM documents WHERE submission_id = $1 ORDER BY section, created_at, id;

-- name: TotalDocumentBytes :one
SELECT COALESCE(sum(d.size_bytes), 0)::bigint FROM documents d
JOIN submissions s ON s.id = d.submission_id WHERE s.applicant_id = $1;

-- name: DeleteDocument :execrows
DELETE FROM documents WHERE id = $1;

-- name: LockDocumentApplicant :one
SELECT id FROM applicants WHERE id = $1 FOR UPDATE;

-- name: PendingDocumentDeletions :many
SELECT storage_key FROM document_deletions ORDER BY created_at, storage_key LIMIT 100;

-- name: CompleteDocumentDeletion :exec
DELETE FROM document_deletions WHERE storage_key = $1;
