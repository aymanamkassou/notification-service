# Notifications Service - Phase 1 Bootstrap

> Standalone Go-based Web Push notification service for affiliate-nourhijab

## Overview

This service implements Web Push notifications (VAPID standard) for the affiliate-nourhijab Next.js application. Phase 1 focuses on setting up the project structure, database schema, and development environment.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Your Next.js App                         │
│         (Web App + Server-Side API Routes)                  │
└──────────────┬──────────────────────────────────────────────┘
               │
               │ HTTP (HMAC Auth)
               │
┌──────────────▼──────────────────────────────────────────────┐
│            Notifications Service (Go)                       │
│  ┌─────────────────┐        ┌──────────────────┐            │
│  │  API (chi)      │        │  Worker (asynq)  │            │
│  │  - Subscriptions│◄──────►│  - Queue manager │            │
│  │  - Publish jobs │        │  - Send delivery │            │
│  └─────────────────┘        └──────────────────┘            │
│           │                         │                        │
│           └────────────┬────────────┘                        │
└────────────────────────┼───────────────────────────────────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
     ┌──▼──┐         ┌───▼───┐      ┌───▼──┐
     │  PG │         │ Redis │      │ FCM  │
     │  DB │         │ Queue │      │ Push │
     └─────┘         └───────┘      └──────┘
```

## Project Structure

```
notifications-service/
├── cmd/
│   ├── api/           # API server entry point
│   ├── worker/        # Background worker entry point
│   └── vapidgen/      # VAPID key generation tool
├── internal/
│   ├── auth/          # HMAC signature validation
│   ├── config/        # Configuration loading (envconfig)
│   ├── http/          # HTTP handlers and routing
│   ├── logger/        # Structured logging (zap)
│   ├── metrics/       # Prometheus metrics
│   ├── middleware/    # HTTP middleware (CORS, auth, etc)
│   ├── models/        # Data structures
│   ├── queue/         # Asynq queue setup
│   ├── repo/          # Database access layer (sqlc)
│   ├── templates/     # i18n notification templates
│   └── webpush/       # Web Push delivery
├── db/
│   └── migrations/    # Postgres migration files
├── Dockerfile         # Multi-stage build
├── docker-compose.yml # Local dev environment
├── Makefile           # Common tasks
├── go.mod / go.sum    # Go dependencies
└── .env.example       # Environment variables template
```

## Prerequisites

- Go 1.22+
- Docker & Docker Compose
- Postgres 16
- Redis 7
- Make (optional but recommended)

## Phase 1 Checklist

- [x] Go module initialized with core dependencies
- [x] Project layout created (`/cmd`, `/internal` subdirs)
- [x] Database schema and migrations
- [x] Configuration struct with env loading
- [x] Docker and docker-compose setup
- [ ] **NEXT**: go mod tidy & go.sum generation
- [ ] **NEXT**: Environment file creation
- [ ] **NEXT**: Phase 2 kickoff (VAPID keys, HMAC auth)

## Setup Instructions

### 1. Install Dependencies

```bash
cd notifications-service
go mod download
go mod tidy
```

### 2. Generate Environment File

```bash
cp .env.example .env
```

Configure required vars in `.env`:

```env
PORT=8080
DATABASE_URL=postgres://postgres:postgres@localhost:5433/notifications?sslmode=disable
REDIS_ADDR=localhost:6380
VAPID_PUBLIC_KEY=       # Will generate in Phase 2
VAPID_PRIVATE_KEY=      # Will generate in Phase 2
HMAC_SECRET=            # Will generate in Phase 2
LOG_LEVEL=info
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8000
```

### 3. Start Development Environment

```bash
docker-compose up -d

# Wait for healthy state
docker-compose ps
```

### 4. Initialize Database

```bash
# Run migrations
docker exec notifications-postgres psql -U postgres -d notifications -f /docker-entrypoint-initdb.d/20251029124000_initial.sql

# Verify tables created
docker exec notifications-postgres psql -U postgres -d notifications -c "\dt"
```

### 5. Health Check

```bash
curl http://localhost:8080/healthz
# Expected: ok
```

## Dependencies Added (go.mod)

| Package | Version | Purpose |
|---------|---------|---------|
| chi | v5.1.0 | HTTP router/middleware |
| pgx | v5.5.5 | Postgres driver |
| redis | v9.5.1 | Redis client |
| asynq | v0.24.1 | Job queue |
| webpush-go | v1.4.0 | VAPID/Web Push |
| zap | v1.26.0 | Structured logging |
| prometheus | v1.19.0 | Metrics |
| envconfig | v11.0.0 | Config from env vars |
| uuid | v1.6.0 | UUID generation |
| migrate | v4.17.0 | DB migration tool |
| testify | v1.8.4 | Testing assertions |

## Database Schema

### Tables Created

1. **device_subscriptions** - Stores browser push subscriptions per user/device
2. **notifications** - Logical notification records (idempotency, deduping)
3. **notification_recipients** - Maps notifications to users
4. **notification_attempts** - Delivery attempts tracking (for observability)

### Indexes

- `device_subscriptions(user_id, is_active)` - Fast lookup of active subs
- `notifications(status)` - Query by status (QUEUED, SENDING, etc)
- `notification_attempts(created_at DESC)` - Recent attempts first

## Local Development Workflow

### Build & Run API

```bash
go build -o bin/api ./cmd/api
./bin/api
```

### Build & Run Worker

```bash
go build -o /tmp/worker ./cmd/worker
/tmp/worker
```

### Run Tests

```bash
go test ./...
```

### Database Inspection

```bash
# Connect to Postgres
docker exec -it notifications-postgres psql -U postgres -d notifications

# List tables
\dt

# Inspect device_subscriptions
\d device_subscriptions

# Sample query
SELECT id, user_id, endpoint, is_active, created_at FROM device_subscriptions LIMIT 5;
```

### Redis Inspection

```bash
# Connect to Redis
docker exec -it notifications-redis redis-cli

# List keys
KEYS *

# Check queue
LLEN asynq:{notifications:send}
```

## Next Steps (Phase 2)

- [ ] Generate VAPID key pair (cmd/vapidgen)
- [ ] Implement HMAC signature validation
- [ ] Create Config loader with env parsing
- [ ] Implement logger setup (structured zap logging)
- [ ] Create metrics exporter (Prometheus)

## Common Issues

### Port Already in Use
```bash
# Change ports in docker-compose.yml
# or kill existing process:
lsof -i :8080 | grep LISTEN | awk '{print $2}' | xargs kill -9
```

### Postgres Connection Refused
```bash
# Ensure container is running
docker-compose ps

# Check logs
docker-compose logs postgres
```

### Redis Connection Issues
```bash
# Test connection
redis-cli -p 6380 ping
# Expected: PONG
```

## References

- [VAPID Specification](https://datatracker.ietf.org/doc/html/draft-thomson-webpush-vapid)
- [Web Push API](https://developer.mozilla.org/en-US/docs/Web/API/Push_API)
- [Asynq Documentation](https://github.com/hibiken/asynq)
- [Go pgx Driver](https://github.com/jackc/pgx)

---

**Status**: Phase 1 Bootstrap - In Progress  
**Last Updated**: Nov 1, 2025
