-- name: AddSpaceMember :one
INSERT INTO space_members (space_id, user_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (space_id, user_id) DO UPDATE SET role = EXCLUDED.role
RETURNING *;

-- name: RemoveSpaceMember :exec
DELETE FROM space_members
WHERE space_id = $1 AND user_id = $2;

-- name: ListSpaceMembers :many
SELECT
    sm.*,
    u.email AS user_email,
    u.display_name AS user_display_name
FROM space_members sm
JOIN users u ON u.id = sm.user_id
WHERE sm.space_id = $1
ORDER BY u.display_name;

-- name: GetSpaceMember :one
SELECT * FROM space_members
WHERE space_id = $1 AND user_id = $2;

-- name: ListSpacesForUser :many
SELECT s.*
FROM spaces s
JOIN space_members sm ON sm.space_id = s.id
WHERE sm.user_id = $1
ORDER BY s.name;
