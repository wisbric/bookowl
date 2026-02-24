-- name: UpsertDocumentImage :exec
INSERT INTO document_images (document_id, storage_id)
VALUES ($1, $2)
ON CONFLICT (document_id, storage_id) DO NOTHING;

-- name: DeleteDocumentImagesByDocument :exec
DELETE FROM document_images WHERE document_id = $1;

-- name: DeleteStaleDocumentImages :exec
DELETE FROM document_images
WHERE document_id = $1
  AND storage_id != ALL(@keep_ids::uuid[]);

-- name: ListDocumentImages :many
SELECT di.*, so.storage_key
FROM document_images di
JOIN storage_objects so ON so.id = di.storage_id
WHERE di.document_id = $1;
