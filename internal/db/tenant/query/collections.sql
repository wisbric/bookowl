-- name: CreateCollection :one
INSERT INTO collections (space_id, parent_id, name, slug, icon, position, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetCollectionByID :one
SELECT * FROM collections
WHERE id = $1;

-- name: ListCollectionsBySpace :many
SELECT * FROM collections
WHERE space_id = $1
ORDER BY position;

-- name: UpdateCollection :one
UPDATE collections
SET name = $2, slug = $3, icon = $4, position = $5, parent_id = $6, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteCollection :exec
DELETE FROM collections
WHERE id = $1;
