-- name: CreateDocument :one
INSERT INTO documents (
    space_id, collection_id, title, slug, content, content_text,
    doc_type, status, tags, icon, position, word_count,
    nightowl_incident_id, nightowl_alert_id, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
)
RETURNING *;

-- name: GetDocumentByID :one
SELECT * FROM documents
WHERE id = $1;

-- name: GetDocumentBySlug :one
SELECT * FROM documents
WHERE space_id = $1 AND slug = $2;

-- name: ListDocuments :many
SELECT * FROM documents
WHERE status != 'archived'
ORDER BY updated_at DESC
LIMIT $1 OFFSET $2;

-- name: ListDocumentsBySpace :many
SELECT * FROM documents
WHERE space_id = $1 AND status != 'archived'
ORDER BY position, updated_at DESC
LIMIT $2 OFFSET $3;

-- name: ListDocumentsByCollection :many
SELECT * FROM documents
WHERE collection_id = $1 AND status != 'archived'
ORDER BY position
LIMIT $2 OFFSET $3;

-- name: ListDocumentsByType :many
SELECT * FROM documents
WHERE doc_type = $1 AND status != 'archived'
ORDER BY updated_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateDocument :one
UPDATE documents
SET title = $2, slug = $3, content = $4, content_text = $5,
    collection_id = $6, doc_type = $7, status = $8, tags = $9, icon = $10,
    position = $11, word_count = $12, version = version + 1,
    nightowl_incident_id = $13, nightowl_alert_id = $14,
    updated_by = $15, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteDocument :exec
DELETE FROM documents
WHERE id = $1;

-- name: ListRunbooks :many
SELECT id, title, slug, content_text, tags, updated_at
FROM documents
WHERE doc_type = 'runbook' AND status = 'published'
ORDER BY updated_at DESC
LIMIT $1 OFFSET $2;

-- name: GetRunbookByID :one
SELECT id, title, slug, content, content_text, tags, updated_at
FROM documents
WHERE id = $1 AND doc_type = 'runbook';

-- name: CountDocuments :one
SELECT count(*) FROM documents
WHERE status != 'archived';

-- name: CountDocumentsBySpace :one
SELECT count(*) FROM documents
WHERE space_id = $1 AND status != 'archived';

-- name: ListRunbooksWithSpace :many
SELECT d.id, d.title, d.slug, d.tags, d.updated_at,
       s.slug AS space_slug
FROM documents d
JOIN spaces s ON s.id = d.space_id
WHERE d.doc_type = 'runbook' AND d.status = 'published'
ORDER BY d.updated_at DESC
LIMIT $1 OFFSET $2;

-- name: GetRunbookWithSpace :one
SELECT d.id, d.title, d.slug, d.content, d.content_text, d.tags, d.updated_at,
       s.slug AS space_slug
FROM documents d
JOIN spaces s ON s.id = d.space_id
WHERE d.id = $1 AND d.doc_type = 'runbook';

-- name: CountRunbooks :one
SELECT count(*) FROM documents
WHERE doc_type = 'runbook' AND status = 'published';

-- name: GetDocumentByNightOwlAlertID :one
SELECT id FROM documents
WHERE nightowl_alert_id = $1;

-- name: SearchRunbooks :many
SELECT
    d.id, d.title, d.slug, d.tags, d.updated_at,
    s.slug AS space_slug,
    ts_rank(d.search_vector, plainto_tsquery('english', @query::text)) AS rank,
    ts_headline('english', d.content_text, plainto_tsquery('english', @query::text),
        'StartSel=<mark>, StopSel=</mark>, MaxWords=30, MinWords=15') AS excerpt
FROM documents d
JOIN spaces s ON s.id = d.space_id
WHERE d.doc_type = 'runbook'
  AND d.status = 'published'
  AND d.search_vector @@ plainto_tsquery('english', @query::text)
ORDER BY rank DESC
LIMIT @result_limit OFFSET @result_offset;
