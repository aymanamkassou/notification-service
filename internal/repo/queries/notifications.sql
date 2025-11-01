-- name: CreateNotification :one
INSERT INTO notifications (
  idempotency_key,
  type,
  title,
  body,
  icon,
  url,
  locale,
  data,
  status,
  dedupe_key,
  ttl_seconds,
  priority
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: GetNotification :one
SELECT * FROM notifications
WHERE id = $1 LIMIT 1;

-- name: GetNotificationByIdempotencyKey :one
SELECT * FROM notifications
WHERE idempotency_key = $1 LIMIT 1;

-- name: ListNotifications :many
SELECT * FROM notifications
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListNotificationsByStatus :many
SELECT * FROM notifications
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateNotificationStatus :one
UPDATE notifications
SET status = $2
WHERE id = $1
RETURNING *;

-- name: DeleteNotification :exec
DELETE FROM notifications
WHERE id = $1;

-- name: CountNotificationsByStatus :one
SELECT COUNT(*) FROM notifications
WHERE status = $1;

-- name: FindNotificationsByDedupeKey :many
SELECT * FROM notifications
WHERE dedupe_key = $1
  AND created_at > $2
ORDER BY created_at DESC;

-- name: DeleteOldNotifications :exec
DELETE FROM notifications
WHERE created_at < $1;
