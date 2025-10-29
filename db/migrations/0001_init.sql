-- Schema for web push notifications service
CREATE TABLE IF NOT EXISTS device_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    endpoint TEXT NOT NULL UNIQUE,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL,
    device_id TEXT,
    user_agent TEXT,
    locale TEXT,
    timezone TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_device_subscriptions_user ON device_subscriptions(user_id, is_active);

CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key TEXT UNIQUE,
    type TEXT NOT NULL,
    title TEXT,
    body TEXT,
    icon TEXT,
    url TEXT,
    locale TEXT,
    data JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'QUEUED',
    dedupe_key TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS notification_recipients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_notification_recipients_notif ON notification_recipients(notification_id);

CREATE TABLE IF NOT EXISTS notification_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    subscription_id UUID NOT NULL REFERENCES device_subscriptions(id) ON DELETE CASCADE,
    status TEXT NOT NULL,
    http_status INT,
    retry_count INT NOT NULL DEFAULT 0,
    duration_ms INT,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_notification_attempts_notif ON notification_attempts(notification_id);
