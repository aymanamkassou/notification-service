-- name: CreateRecipient :exec
INSERT INTO notification_recipients (
  notification_id,
  user_id
) VALUES (
  $1, $2
)
ON CONFLICT (notification_id, user_id) DO NOTHING;

-- name: CreateRecipientsBatch :copyfrom
INSERT INTO notification_recipients (
  notification_id,
  user_id
) VALUES (
  $1, $2
);

-- name: GetRecipientsByNotification :many
SELECT * FROM notification_recipients
WHERE notification_id = $1
ORDER BY user_id;

-- name: GetRecipientsByUser :many
SELECT * FROM notification_recipients
WHERE user_id = $1
ORDER BY notification_id;

-- name: DeleteRecipient :exec
DELETE FROM notification_recipients
WHERE notification_id = $1 AND user_id = $2;

-- name: DeleteRecipientsByNotification :exec
DELETE FROM notification_recipients
WHERE notification_id = $1;

-- name: CountRecipientsByNotification :one
SELECT COUNT(*) FROM notification_recipients
WHERE notification_id = $1;

-- name: CheckRecipientExists :one
SELECT EXISTS(
  SELECT 1 FROM notification_recipients
  WHERE notification_id = $1 AND user_id = $2
);
