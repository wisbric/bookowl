-- name: CreateStorageObject :one
INSERT INTO storage_objects (storage_key, filename, content_type, size_bytes, backend, uploaded_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetStorageObject :one
SELECT * FROM storage_objects WHERE id = $1;

-- name: GetStorageObjectByKey :one
SELECT * FROM storage_objects WHERE storage_key = $1;

-- name: DeleteStorageObject :exec
DELETE FROM storage_objects WHERE id = $1;

-- name: ListOrphanedStorageObjects :many
SELECT so.id, so.storage_key
FROM storage_objects so
LEFT JOIN document_images di ON di.storage_id = so.id
WHERE di.id IS NULL
  AND so.created_at < now() - @min_age::interval
LIMIT 100;
