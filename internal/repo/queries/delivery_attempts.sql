-- name: CreateDeliveryAttempt :one
INSERT INTO notification_attempts (
  notification_id,
  subscription_id,
  user_id,
  status,
  http_status,
  latency_ms,
  error,
  retry_count
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetDeliveryAttempt :one
SELECT * FROM notification_attempts
WHERE id = $1 LIMIT 1;

-- name: ListDeliveryAttemptsByNotification :many
SELECT * FROM notification_attempts
WHERE notification_id = $1
ORDER BY created_at DESC;

-- name: ListDeliveryAttemptsBySubscription :many
SELECT * FROM notification_attempts
WHERE subscription_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListDeliveryAttemptsByUser :many
SELECT * FROM notification_attempts
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListDeliveryAttemptsByStatus :many
SELECT * FROM notification_attempts
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateDeliveryAttemptStatus :one
UPDATE notification_attempts
SET
  status = sqlc.arg('status'),
  http_status = COALESCE(sqlc.narg('http_status'), http_status),
  latency_ms = COALESCE(sqlc.narg('latency_ms'), latency_ms),
  error = COALESCE(sqlc.narg('error'), error),
  retry_count = COALESCE(sqlc.narg('retry_count'), retry_count)
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: MarkSubscriptionAsPruned :exec
UPDATE notification_attempts
SET pruned = true
WHERE subscription_id = $1;

-- name: CountDeliveryAttemptsByNotification :one
SELECT COUNT(*) FROM notification_attempts
WHERE notification_id = $1;

-- name: CountDeliveryAttemptsByStatus :one
SELECT COUNT(*) FROM notification_attempts
WHERE status = $1;

-- name: GetDeliveryStats :one
SELECT
  COUNT(*) as total_attempts,
  COUNT(*) FILTER (WHERE status = 'success') as success_count,
  COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
  AVG(latency_ms) FILTER (WHERE latency_ms IS NOT NULL) as avg_latency_ms
FROM notification_attempts
WHERE created_at >= $1;

-- name: FindFailedAttemptsBySubscription :many
SELECT * FROM notification_attempts
WHERE subscription_id = $1
  AND status = 'failed'
  AND created_at >= $2
ORDER BY created_at DESC;

-- name: DeleteOldAttempts :exec
DELETE FROM notification_attempts
WHERE created_at < $1;
