-- name: CreateDeviceSubscription :one
INSERT INTO device_subscriptions (
  user_id,
  endpoint,
  p256dh,
  auth,
  device_id,
  user_agent,
  locale,
  timezone
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetDeviceSubscription :one
SELECT * FROM device_subscriptions
WHERE id = $1 LIMIT 1;

-- name: GetDeviceSubscriptionByEndpoint :one
SELECT * FROM device_subscriptions
WHERE endpoint = $1 LIMIT 1;

-- name: ListDeviceSubscriptionsByUser :many
SELECT * FROM device_subscriptions
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: ListActiveDeviceSubscriptionsByUser :many
SELECT * FROM device_subscriptions
WHERE user_id = $1 AND is_active = true
ORDER BY created_at DESC;

-- name: UpdateDeviceSubscription :one
UPDATE device_subscriptions
SET
  p256dh = COALESCE(sqlc.narg('p256dh'), p256dh),
  auth = COALESCE(sqlc.narg('auth'), auth),
  device_id = COALESCE(sqlc.narg('device_id'), device_id),
  user_agent = COALESCE(sqlc.narg('user_agent'), user_agent),
  locale = COALESCE(sqlc.narg('locale'), locale),
  timezone = COALESCE(sqlc.narg('timezone'), timezone),
  is_active = COALESCE(sqlc.narg('is_active'), is_active),
  updated_at = now()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeactivateDeviceSubscription :exec
UPDATE device_subscriptions
SET is_active = false, updated_at = now()
WHERE id = $1;

-- name: DeleteDeviceSubscription :exec
DELETE FROM device_subscriptions
WHERE id = $1;

-- name: DeleteDeviceSubscriptionByEndpoint :exec
DELETE FROM device_subscriptions
WHERE endpoint = $1;

-- name: CountActiveSubscriptionsByUser :one
SELECT COUNT(*) FROM device_subscriptions
WHERE user_id = $1 AND is_active = true;

-- name: FindStaleSubscriptions :many
SELECT * FROM device_subscriptions
WHERE is_active = true
  AND updated_at < $1
ORDER BY updated_at ASC
LIMIT $2;
