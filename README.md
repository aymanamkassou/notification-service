# Notifications Service (Go) — Web Push First

Standalone service for web push notifications using VAPID, Redis (asynq), and Postgres.

Quick start (local):
- Copy .env.example to .env and set VAPID_PUBLIC_KEY/VAPID_PRIVATE_KEY and HMAC_SECRET.
- docker compose up --build

Endpoints:
- GET /healthz
- GET /v1/push/public-key
- (stubs) POST /v1/subscriptions, DELETE /v1/subscriptions, POST /v1/notifications

Phase 0 (Prereqs) — step-by-step
- Copy env: cp .env.example .env (why: services read configuration from .env).
- Generate secrets: ./scripts/generate_secrets.sh then paste VAPID_PUBLIC_KEY, VAPID_PRIVATE_KEY, HMAC_SECRET into .env (why: VAPID identifies your push sender; HMAC authenticates producers calling POST /v1/notifications).
- Start infra only: docker compose up -d postgres redis (why: spin up Postgres/Redis; we’ll add the API in Phase 1).
- Verify containers: docker compose ps (both should be running).
- Verify Redis: docker compose exec redis redis-cli PING → PONG.
- Verify Postgres: docker compose exec postgres psql -U postgres -c 'SELECT 1' → returns 1.

Notes
- Keep .env out of git; .gitignore already excludes it.
- You can regenerate keys any time; updating the public key requires clients to resubscribe.

Phase 2 (Config & auth)
- .env autoload for dev: the API loads .env automatically (godotenv) when you use `go run`; in Docker, compose passes .env via env_file.
- Public key endpoint: returns 503 with { error } if VAPID_PUBLIC_KEY is not set to avoid silent empty values.
- HMAC util: internal/auth/hmac.go exposes Sign/Verify and a middleware you can attach to POST /v1/notifications later.
- VAPID keygen (Go): `go run ./cmd/vapidgen` prints { publicKey, privateKey } you can copy into .env.
