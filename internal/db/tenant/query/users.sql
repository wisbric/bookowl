-- name: UpsertUser :one
INSERT INTO users (external_id, email, display_name, role)
VALUES ($1, $2, $3, $4)
ON CONFLICT (external_id) DO UPDATE
SET email = EXCLUDED.email,
    display_name = EXCLUDED.display_name,
    updated_at = now()
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: GetUserByExternalID :one
SELECT * FROM users
WHERE external_id = $1;

-- name: ListUsers :many
SELECT * FROM users
WHERE is_active = true
ORDER BY display_name
LIMIT $1 OFFSET $2;

-- name: UpdateUserRole :one
UPDATE users
SET role = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeactivateUser :exec
UPDATE users
SET is_active = false, updated_at = now()
WHERE id = $1;
