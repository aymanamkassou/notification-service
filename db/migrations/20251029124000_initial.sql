-- Enable pgcrypto for gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- device_subscriptions: stores one PushSubscription per device
CREATE TABLE IF NOT EXISTS device_subscriptions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id text NOT NULL,
  endpoint text NOT NULL UNIQUE,
  p256dh text NOT NULL,
  auth text NOT NULL,
  device_id text,
  user_agent text,
  locale text,
  timezone text,
  is_active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_device_subscriptions_user_id ON device_subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_device_subscriptions_device_id ON device_subscriptions(device_id);
CREATE INDEX IF NOT EXISTS idx_device_subscriptions_active ON device_subscriptions(user_id, is_active);

-- notifications: logical notification records
CREATE TABLE IF NOT EXISTS notifications (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  idempotency_key text UNIQUE,
  type text NOT NULL,
  title text,
  body text,
  icon text,
  url text,
  locale text,
  data jsonb NOT NULL DEFAULT '{}'::jsonb,
  status text NOT NULL,
  dedupe_key text,
  ttl_seconds integer,
  priority text DEFAULT 'normal',
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_notifications_idempotency ON notifications(idempotency_key);
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);
CREATE INDEX IF NOT EXISTS idx_notifications_dedupe ON notifications(dedupe_key);
CREATE INDEX IF NOT EXISTS idx_notifications_created ON notifications(created_at DESC);

-- notification_recipients: who should receive a notification
CREATE TABLE IF NOT EXISTS notification_recipients (
  notification_id uuid NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
  user_id text NOT NULL,
  PRIMARY KEY (notification_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_recipients_notification ON notification_recipients(notification_id);
CREATE INDEX IF NOT EXISTS idx_recipients_user ON notification_recipients(user_id);

-- notification_attempts: delivery outcomes per subscription
CREATE TABLE IF NOT EXISTS notification_attempts (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  notification_id uuid NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
  subscription_id uuid REFERENCES device_subscriptions(id) ON DELETE SET NULL,
  user_id text NOT NULL,
  status text NOT NULL,
  http_status integer,
  latency_ms integer,
  error text,
  retry_count integer DEFAULT 0,
  pruned boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_attempts_notification ON notification_attempts(notification_id);
CREATE INDEX IF NOT EXISTS idx_attempts_subscription ON notification_attempts(subscription_id);
CREATE INDEX IF NOT EXISTS idx_attempts_user ON notification_attempts(user_id);
CREATE INDEX IF NOT EXISTS idx_attempts_status ON notification_attempts(status);
CREATE INDEX IF NOT EXISTS idx_attempts_created ON notification_attempts(created_at DESC);

