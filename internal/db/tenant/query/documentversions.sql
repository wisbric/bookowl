-- name: CreateDocumentVersion :one
INSERT INTO document_versions (
    document_id, version, title, content, content_text,
    doc_type, tags, changed_by, change_summary
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetDocumentVersion :one
SELECT * FROM document_versions
WHERE document_id = $1 AND version = $2;

-- name: GetDocumentVersionByID :one
SELECT * FROM document_versions
WHERE id = $1;

-- name: ListDocumentVersions :many
SELECT id, document_id, version, title, doc_type, tags, changed_by, change_summary, created_at
FROM document_versions
WHERE document_id = $1
ORDER BY version DESC
LIMIT $2 OFFSET $3;
