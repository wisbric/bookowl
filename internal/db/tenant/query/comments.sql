-- name: CreateComment :one
INSERT INTO comments (document_id, parent_id, author_id, body, body_rendered)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetCommentByID :one
SELECT * FROM comments WHERE id = $1;

-- name: ListCommentsByDocument :many
SELECT * FROM comments
WHERE document_id = $1 AND parent_id IS NULL
ORDER BY created_at ASC;

-- name: ListReplies :many
SELECT * FROM comments
WHERE parent_id = $1
ORDER BY created_at ASC;

-- name: UpdateCommentBody :one
UPDATE comments
SET body = $2, body_rendered = $3, edited_at = now(), updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SoftDeleteComment :exec
UPDATE comments
SET body = '[deleted]', body_rendered = '[deleted]', is_deleted = true, updated_at = now()
WHERE id = $1;

-- name: ResolveComment :exec
UPDATE comments
SET is_resolved = true, resolved_by = $2, resolved_at = now(), updated_at = now()
WHERE id = $1;

-- name: UnresolveComment :exec
UPDATE comments
SET is_resolved = false, resolved_by = NULL, resolved_at = NULL, updated_at = now()
WHERE id = $1;

-- name: CountCommentsByDocument :one
SELECT COUNT(*) FROM comments
WHERE document_id = $1 AND is_deleted = false;
