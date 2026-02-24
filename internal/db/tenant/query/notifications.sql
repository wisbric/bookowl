-- name: CreateNotification :one
INSERT INTO notifications (user_id, type, document_id, comment_id, actor_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CountUnreadNotifications :one
SELECT COUNT(*) FROM notifications
WHERE user_id = $1 AND is_read = false;

-- name: ListNotifications :many
SELECT * FROM notifications
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: MarkNotificationRead :exec
UPDATE notifications
SET is_read = true
WHERE id = $1 AND user_id = $2;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications
SET is_read = true
WHERE user_id = $1 AND is_read = false;
