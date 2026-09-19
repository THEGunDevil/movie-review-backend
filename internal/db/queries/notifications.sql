-- name: MarkAllNotificationsRead :exec

UPDATE notifications
SET read = true
WHERE recipient_id = $1
  AND read = false;

-- name: CreateNotificationWithEvent :one

WITH new_event AS (
    INSERT INTO events (
        event_type,
        payload
    )
    VALUES (
        sqlc.arg(event_type),
        sqlc.arg(payload)::text::jsonb
    )
    RETURNING id
)
INSERT INTO notifications (
    title,
    message,
    recipient_id,
    event_id
)
SELECT
    sqlc.arg(title),
    sqlc.arg(message),
    sqlc.arg(recipient_id),
    new_event.id
FROM new_event
RETURNING
    id,
    recipient_id,
    title,
    message,
    read,
    created_at,
    event_id;

-- name: GetNotificationsByUser :many

SELECT
    n.id,
    n.title,
    n.message,
    n.read,
    n.created_at,
    n.event_id,
    e.event_type,
    e.payload
FROM notifications n
LEFT JOIN events e
    ON e.id = n.event_id
WHERE n.recipient_id = $1
ORDER BY n.created_at DESC
LIMIT $2;

-- name: MarkNotificationRead :exec

UPDATE notifications
SET read = true
WHERE id = $1
  AND recipient_id = $2;