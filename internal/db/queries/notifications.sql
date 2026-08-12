-- name: CreateWebhookEvent :one
INSERT INTO webhook_events (event_type, payload)
VALUES ($1, $2)
RETURNING *;

-- name: CreateNotification :one
INSERT INTO notifications (title, message, event_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetNotifications :many
SELECT * FROM notifications
ORDER BY created_at DESC
LIMIT $1;

-- name: MarkNotificationRead :exec
UPDATE notifications SET read = true WHERE id = $1;