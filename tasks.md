# Notifications Service Implementation Plan (Go) — Web Push First

Date: 2025-10-28T16:26:33.434Z

Goal: Design and deliver a standalone Notifications service (Go) that exposes a simple TS interface sendNotification(...) for this app, initially implementing Web Push for the web app, with a roadmap to multi-channel.


1) Current repository findings (scanned)
- Email: src/lib/mail.ts uses Resend for verification, password reset, and 2FA emails.
- DB models (prisma/schema.prisma):
  - User.notifications Json? (user preferences placeholder), language (default "en"), timezone.
  - StockRequestNotification: per-request recipients and enum types: NEW_REQUEST, REQUEST_REVIEWED, REQUEST_APPROVED, PARTIALLY_APPROVED, REQUEST_REJECTED, REQUEST_FULFILLED, REQUEST_CANCELLED, STOCK_UNAVAILABLE.
  - Stock Request workflow models that generate/need notifications.
- UI mentions: in-app notifications dropdown uses mock data; settings pages reference notification preferences.
- Implication: The new service should support these stock-request events out-of-the-box and respect per-user locale; future preference sync can be added once settings UI persists real prefs.


2) Scope (Phase 1)
- Implement Web Push only (standards-based, VAPID). No vendor lock-in.
- Provide a minimal TS SDK for this repo: register/unregister push, and sendNotification(...).
- Keep in-app StockRequestNotification DB writes in the existing Node app (producer) to avoid cross-DB coupling; the service focuses on push delivery + its own audit.


3) High-level architecture (Go)
- API (chi):
  - GET /v1/push/public-key → return VAPID public key (base64url).
  - POST /v1/subscriptions → register a browser PushSubscription {endpoint, keys, userId, userAgent, locale?, tz?, deviceId?}.
  - DELETE /v1/subscriptions → unsubscribe by endpoint or deviceId.
  - POST /v1/notifications → producers publish a notification job.
  - GET /healthz, GET /metrics (Prometheus), readiness/liveness.
- Worker (asynq + Redis):
  - Queue: notifications:send, retries with backoff; dead-letter notifications:dlq.
  - Resolves recipients → active subscriptions; renders i18n template; delivers via webpush-go; prunes dead subs on 404/410.
- Storage (Postgres via pgx + sqlc):
  - device_subscriptions: stores PushSubscription per user/device.
  - notifications: logical notifications ledger (idempotency, dedupeKey, payload, type, status).
  - notification_attempts: outcome per subscription send (status, http_status, latency, error, pruned flag).
  - optional topics (future): notification_topics, topic_subscriptions.
- Config & security:
  - VAPID keys (PUBLIC/PRIVATE).
  - HMAC service-to-service auth for POST /v1/notifications.
  - TLS termination by gateway/ingress; rate limit on subscriptions.
- Observability:
  - OpenTelemetry traces; Prometheus counters/histograms; structured logs (slog).

Rationale for choices
- Go + chi: small footprint, fast, easy to containerize.
- asynq + Redis: simple queue/retries/backoff today, can migrate to SQS/Kafka later.
- webpush-go: mature VAPID/Web Push implementation.
- sqlc + pgx: type-safe queries, good performance.


4) API contracts (initial)
- GET /v1/push/public-key
  - 200: { publicKey: string }
- POST /v1/subscriptions (Bearer user-session from app or HMAC if proxied)
  - Body: { userId: string, subscription: { endpoint: string, keys: { p256dh: string, auth: string } }, deviceId?: string, userAgent?: string, locale?: string, timezone?: string }
  - 201 Created; 200 OK idempotent
- DELETE /v1/subscriptions
  - Body: { userId?: string, endpoint?: string, deviceId?: string }
  - 204 No Content
- POST /v1/notifications (HMAC X-Signature header)
  - Body: {
      idempotencyKey?: string,
      type: string,                // e.g., STOCK_REQUEST.NEW_REQUEST
      recipientUserIds?: string[], // or topics[] (future)
      locale?: string,
      data?: Record<string, any>,  // template variables
      title?: string,              // optional override
      body?: string,               // optional override
      icon?: string,
      url?: string,
      ttlSeconds?: number,
      priority?: "low"|"normal"|"high",
      dedupeKey?: string
    }
  - 202 Accepted
  - Errors: 400 (validation), 401/403 (auth), 409 (idempotency conflict)

Notes
- type drives template selection; when title/body provided, bypass template.
- idempotencyKey ensures duplicates aren’t re-enqueued.


5) Data model (service DB)
- device_subscriptions
  - id (uuid pk)
  - user_id (text, indexed)
  - endpoint (text unique)
  - p256dh (text)
  - auth (text)
  - device_id (text nullable, indexed)
  - user_agent (text)
  - locale (text)
  - timezone (text)
  - is_active (bool default true)
  - created_at timestamptz default now()
  - updated_at timestamptz default now()
- notifications
  - id (uuid pk)
  - idempotency_key (text unique null)
  - type (text)
  - title (text null)
  - body (text null)
  - icon (text null)
  - url (text null)
  - locale (text null)
  - data (jsonb)
  - status (text) // QUEUED, SENDING, PARTIAL, SENT, FAILED
  - dedupe_key (text null)
  - created_at timestamptz default now()
- notification_recipients
  - id (uuid pk)
  - notification_id (uuid fk → notifications)
  - user_id (text)
- notification_attempts
  - id (uuid pk)
  - notification_id (uuid fk)
  - user_id (text)
  - subscription_id (uuid fk → device_subscriptions)
  - status (text) // SENT, FAILED, SKIPPED, PRUNED
  - http_status (int null)
  - retry_count (int)
  - duration_ms (int)
  - error (text null)
  - created_at timestamptz default now()

Indexes to add
- device_subscriptions(user_id, is_active)
- notification_recipients(notification_id)
- notification_attempts(notification_id)


6) Template and i18n
- Convention: type namespace, e.g., STOCK_REQUEST.NEW_REQUEST, STOCK_REQUEST.APPROVED, etc.
- Template resolver selects localized strings and constructs a compact payload: title, body, url, icon.
- Source of locale: send payload locale || user locale (from subscription) || app default.
- For this phase, store templates in code with go-i18n message files; move to DB later if needed.

Initial event → template mapping (from Prisma enum StockRequestNotificationType)
- STOCK_REQUEST.NEW_REQUEST → title: "New stock request", body: "Request {{requestNumber}} from {{warehouse}} ({{totalItems}} items)"
- STOCK_REQUEST.REVIEWED → title: "Request reviewed" ...
- STOCK_REQUEST.APPROVED → title: "Request approved" ...
- STOCK_REQUEST.PARTIALLY_APPROVED → title: "Request partially approved" ...
- STOCK_REQUEST.REJECTED → title: "Request rejected" ...
- STOCK_REQUEST.FULFILLED → title: "Request fulfilled" ...
- STOCK_REQUEST.CANCELLED → title: "Request cancelled" ...
- STOCK_UNAVAILABLE → title: "Stock unavailable" ...


7) TS SDK for this repo (thin client)
- Install in-app service worker (sw.js) under /public or /src/app root as Next asset.
- SDK functions:
  - getPublicKey(): fetch from service; cache it.
  - registerPush(userId): ensures Service Worker, asks Notification permission, subscribes with pushManager, POST /v1/subscriptions.
  - unregisterPush(userId): unsubscribe locally and DELETE /v1/subscriptions.
  - sendNotification(input): POST /v1/notifications with HMAC (this call should be from server-side API routes only).
- TypeScript interface
  - type NotificationInput = { idempotencyKey?, type, recipientUserIds, locale?, data?, title?, body?, icon?, url?, ttlSeconds?, priority?, dedupeKey? }


8) Integration points in this app
- Service Worker registration trigger: after user sign-in and on settings page toggle.
- Persist subscription lifecycle on sign-out (unregister) and token refresh (handle SW updates).
- Produce notifications at existing server actions:
  - Stock Requests: src/app/.../stock-requests/new/_lib/actions.ts (creation → NEW_REQUEST) and other actions files for APPROVED/REJECTED/FULFILLED etc.
  - Auth emails remain as-is for now; optional: future push for security alerts.
- Keep writing StockRequestNotification rows in Prisma for in-app UI; producer can call both: create DB row + POST to notifications service.


9) Security
- VAPID: generate a key pair; expose public key via API; store private key in service secret store.
- HMAC auth for producers: X-Timestamp, X-Idempotency-Key, X-Signature = HMAC_SHA256(secret, method + path + body + timestamp). Reject skew > 5 min.
- CORS: restrict origins to this app domain for subscription endpoints.
- Rate limiting: IP + userId on subscription endpoints.


10) Retry, dedupe, pruning
- Retry policy: exponential backoff up to N attempts, DLQ after.
- Dedupe by idempotencyKey at API and by dedupeKey per user (skip sending if same dedupeKey recently sent within window).
- On 404/410 from push service: mark subscription inactive and delete or soft-delete.


11) Observability and ops
- Metrics: counters for notifications_enqueued, notifications_sent, notifications_failed, pruned_subscriptions; histogram for send_latency_ms.
- Logs: structured with ids (notification_id, user_id, endpoint hash, attempt_id).
- Traces: wrap enqueue→send path, annotate errors.
- Admin tools (later): list subscriptions by user, revoke, DLQ requeue.


12) Deployment
- Dockerfile (multi-stage), Docker Compose for local (service + Postgres + Redis).
- Env vars:
  - DATABASE_URL, REDIS_URL
  - VAPID_PUBLIC_KEY, VAPID_PRIVATE_KEY
  - HMAC_SECRET
  - SERVICE_BASE_URL
  - LOG_LEVEL, OTEL_EXPORTER_OTLP_ENDPOINT (optional)
- K8s: Deployment with liveness/readiness, ConfigMap/Secret, HPA on CPU/RPS, Prometheus scraping.


13) Testing plan
- Go unit tests: template resolver, HMAC validator, repository layer.
- Integration: Testcontainers for Postgres + Redis + webpush mock; validate enqueue→worker→attempts.
- Contract tests for API (OpenAPI and httpexpect).
- App E2E (manual): browser receives push after permission + subscription.


14) Step-by-step delivery tasks
Phase 0 — Prereqs
- [ ] Confirm infra (Redis, Postgres) and DNS.
- [ ] Pick domains and CORS configuration.

Phase 1 — Repo bootstrap (service)
- [ ] go mod init notifications
- [ ] Add deps: chi, pgx, sqlc, asynq, webpush-go, go-i18n, slog, zap/slog, prometheus client, envconfig, testify, migrate.
- [ ] Project layout: /cmd/api, /cmd/worker, /internal/{http,queue,repo,model,config,auth,templates,i18n,log,metrics}.
- [ ] Migrations: device_subscriptions, notifications, notification_recipients, notification_attempts.

Phase 2 — Config & keys
- [ ] env loading and validation.
- [ ] VAPID key generation tool; store secrets.
- [ ] HMAC middleware and signer util.

Phase 3 — Data access
- [ ] Define SQL (sqlc) and generate code for CRUD on tables.
- [ ] Repository layer with context timeouts.

Phase 4 — API
- [ ] GET /v1/push/public-key.
- [ ] POST/DELETE /v1/subscriptions with validation and upsert; basic rate limit.
- [ ] POST /v1/notifications with validation, idempotency, enqueue.
- [ ] healthz and metrics.

Phase 5 — Worker
- [ ] Asynq server setup, handler for notifications:send.
- [ ] Recipient resolution: userIds → active device_subscriptions.
- [ ] Template resolver: map type→title/body/url/icon using go-i18n.
- [ ] Delivery via webpush-go with TTL/urgency.
- [ ] Attempts recording, pruning, status updates.

Phase 6 — TS SDK + Service Worker in this repo
- [ ] Add sw.js with push event handler to show notifications and handle click → open url.
- [ ] SDK: getPublicKey, registerPush, unregisterPush, sendNotification types.
- [ ] Settings UI hook to toggle push and display status.

Phase 7 — Integration points
- [ ] Wire calls in stock-requests actions for each event type; include URL deep-links.
- [ ] Ensure locale passed or let service infer from subscription/user.
- [ ] Continue writing StockRequestNotification rows in Prisma for in-app UI.

Phase 8 — QA & hardening
- [ ] Load tests for push send throughput.
- [ ] Failure injection (expired subs) ensures pruning works.
- [ ] Alerting on DLQ growth / error rates.

Phase 9 — Rollout
- [ ] Enable for internal users first; monitor metrics.
- [ ] Gradual rollout to all users; document opt-in/out.


15) Open questions / decisions
- Do we want the notification service to read the main Postgres directly? Proposed: no, keep boundaries; pass userIds from producer.
- Where to persist real notification preferences? Proposed: keep in main app (User.notifications JSON) and have producer enforce, or add a sync endpoint later.
- Topic-based broadcasts? Add in Phase 2+ if needed.


16) Appendix: example payloads
- Example: NEW_REQUEST
```
POST /v1/notifications
{
  "idempotencyKey": "req-2025-0001",
  "type": "STOCK_REQUEST.NEW_REQUEST",
  "recipientUserIds": ["<adminUserId>", "<managerUserId>"],
  "locale": "en",
  "data": {
    "requestNumber": "REQ-2025-001",
    "warehouse": "Central",
    "totalItems": 12,
    "requestId": "sr_abc123"
  },
  "url": "/dashboards/warehouse/stock-requests/sr_abc123"
}
```
- Service Worker push handler (concept):
```
self.addEventListener('push', (e) => {
  const data = e.data ? e.data.json() : {};
  const title = data.title || 'Notification';
  e.waitUntil(self.registration.showNotification(title, {
    body: data.body,
    icon: data.icon,
    data: { url: data.url }
  }));
});
self.addEventListener('notificationclick', (e) => {
  e.notification.close();
  const url = e.notification.data?.url || '/';
  e.waitUntil(clients.matchAll({ type: 'window' }).then(ws => {
    for (const w of ws) { if ('focus' in w) { w.navigate(url); w.focus(); return; } }
    clients.openWindow(url);
  }));
});
```


Summary
- This plan delivers a Go-based, queue-backed Web Push notification service with clear integration points, minimal coupling to the existing app, and a migration path to multi-channel later. It aligns with existing Prisma event types and uses user locale when available.
