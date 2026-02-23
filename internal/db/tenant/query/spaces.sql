-- name: CreateSpace :one
INSERT INTO spaces (name, slug, description, icon, is_private, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetSpaceByID :one
SELECT * FROM spaces
WHERE id = $1;

-- name: GetSpaceBySlug :one
SELECT * FROM spaces
WHERE slug = $1;

-- name: ListSpaces :many
SELECT * FROM spaces
ORDER BY name;

-- name: UpdateSpace :one
UPDATE spaces
SET name = $2, slug = $3, description = $4, icon = $5, is_private = $6, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteSpace :exec
DELETE FROM spaces
WHERE id = $1;
