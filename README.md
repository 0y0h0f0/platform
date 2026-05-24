# Task Platform

A campus-recruitment backend scaffold — team task collaboration platform built with Go microservices.

**Status:** All phases complete. Services compile, tests pass (80%+ coverage), metrics + traces + Grafana dashboard in place, 1k QPS verified.

## Architecture

```
             +-----------------------+
             |   Web / Postman / UI  |
             +-----------+-----------+
                         |
                    HTTP / JSON
                         |
                         v
             +-----------------------+
             |      api-gateway      |
             |         Gin           |
             | auth / bind / log     |
             | request_id / recovery |
             +-----------+-----------+
                         |
                     gRPC Client
               +---------+---------+
               |                   |
               v                   v
      +----------------+   +----------------+
      |  user-service  |   |  task-service  |
      |      gRPC      |   |      gRPC      |
      +--------+-------+   +--------+-------+
               |                    |
               |                    |
               v                    v
        +-------------+      +-------------+
        | PostgreSQL  |      | PostgreSQL  |
        | user schema |      | task schema |
        +------+------+      +------+------+
               \                    /
                \                  /
                 +--------+-------+
                          |
                        Redis
```

**Middleware chain:** `Recovery → RequestID → HTTPTrace → HTTPMetrics → AccessLog → CORS → RateLimit(IP) → Auth(JWT) → RateLimit(user) → Handler`

**Service layer:** `handler(HTTP) → service(gRPC impl) → biz(domain) → data(repository/GORM)`

## Features

### Authentication & Users
- Register (username/email/password with validation, weak-password blacklist)
- Login (username or email + password, JWT HS256, 2h TTL)
- Logout (Redis token blacklist)
- Get current user, batch get users (Redis-cached, TTL 5 min)

### Projects
- CRUD with optimistic locking (`version` column)
- Archive / unarchive (read-only mode for archived projects)
- Transfer ownership (atomic: updates both `projects.owner_id` and `project_members.role`)
- Member management: add, remove, change role, leave, list
- Role-based permissions: owner / admin / member

### Tasks
- CRUD with optimistic locking
- Cursor-based pagination (base64url-encoded cursor with filter hash validation)
- Task assignment with user existence and membership validation
- State machine: `todo ↔ doing ↔ done`, `todo → cancelled`, `cancelled → todo`
- Status and assignee updated through dedicated endpoints (`AssignTask`, `ChangeTaskStatus`)

### Comments & Operation Logs
- Create / delete / list task comments
- Operation logs for projects and tasks
- Async log writer: buffered channel (capacity 1024), batch write (size 64, flush 100ms), graceful degradation on channel full

### Engineering
- **Unified error codes:** 12 error codes mapped from gRPC status to HTTP responses via `{ code, message, request_id, data? }` envelope
- **Structured logging:** Zap JSON with `request_id` and `method`/`latency` fields; gRPC unary logging interceptors on both services
- **Redis caching:** Cache-aside for user info and project details with write invalidation (TTL 5 min)
- **Rate limiting:** Redis token bucket (Lua script), separate keys for auth (5/s, burst 10) and regular paths (60/s, burst 100), plus per-user rate limiting (100/s, burst 200)
- **Idempotency:** `Idempotency-Key` header on all write endpoints, Redis `SETNX` with 24h TTL, key freed on error for retry
- **Multi-env config:** YAML configs for local/dev/docker with Viper + env var override
- **Soft delete:** `users`, `projects`, `tasks` use GORM `DeletedAt`; `comments` and `members` are physically deleted
- **Integration tests:** 70+ tests against real PostgreSQL 16 and Redis 7 via testcontainers-go

## Quick Start

### Prerequisites
- Go 1.26+
- Docker (for PostgreSQL, Redis, and testcontainers)

### Setup

```bash
cp .env.example .env
make up          # Start PostgreSQL + Redis
make migrate     # Run database migrations
```

### Run Services

```bash
make run/api-gateway     # HTTP :8080
make run/user-service    # gRPC :9091, admin HTTP :8081
make run/task-service    # gRPC :9092, admin HTTP :8082
```

Or run all three in separate terminals.


### Frontend

```bash
cd web
npm install
npm run dev       # Proxy /api to the Go gateway
npm run dev:mock  # Standalone MSW mode, no backend required
npm run e2e       # Playwright E2E against MSW mode
```

See `web/README.md` for frontend quality checks and bundle stats generation.

### Test

```bash
make test                              # All unit tests
go test -tags=integration ./test/integration/ -v   # Integration tests (requires Docker)
make coverage                          # Coverage report (≥80% threshold)
make lint                              # golangci-lint
```

## API Reference

All endpoints return the envelope: `{ "code": "OK", "message": "...", "request_id": "...", "data": {...} }`

### Auth — `/api/v1/auth`

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/auth/register` | No | Register (username, email, password) |
| POST | `/auth/login` | No | Login (account, password) |
| POST | `/auth/logout` | Bearer | Logout (blacklists current token) |

### Users — `/api/v1/users`

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/users/me` | Bearer | Get current user |

### Projects — `/api/v1/projects`

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/projects` | Bearer | Create project |
| GET | `/projects` | Bearer | List my projects (offset pagination, `?limit=20&offset=0`) |
| GET | `/projects/:id` | Bearer | Get project detail |
| PUT | `/projects/:id` | Bearer | Update project (requires `version`) |
| POST | `/projects/:id/archive` | Bearer | Archive project (owner only) |
| POST | `/projects/:id/unarchive` | Bearer | Unarchive project (owner only) |
| POST | `/projects/:id/transfer` | Bearer | Transfer ownership (`target_user_id`) |
| POST | `/projects/:id/members` | Bearer | Add member (`user_id`, `role`) |
| GET | `/projects/:id/members` | Bearer | List members |
| PUT | `/projects/:id/members/:userId` | Bearer | Update member role |
| DELETE | `/projects/:id/members/:userId` | Bearer | Remove member |
| POST | `/projects/:id/members/me/leave` | Bearer | Leave project |
| GET | `/projects/:id/operation-logs` | Bearer | List project operation logs |

### Tasks — `/api/v1/tasks`

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/tasks` | Bearer | Create task (`project_id`, `title`, `content`) |
| GET | `/tasks` | Bearer | List tasks (cursor pagination, `?project_id=&status=&assignee_id=&keyword=&cursor=&limit=`) |
| GET | `/tasks/:id` | Bearer | Get task detail |
| PUT | `/tasks/:id` | Bearer | Update task (requires `version`; status/assignee not accepted) |
| DELETE | `/tasks/:id` | Bearer | Delete task |
| POST | `/tasks/:id/assign` | Bearer | Assign task (`assignee_id`) |
| POST | `/tasks/:id/status` | Bearer | Change status (`status`, `version`) |
| POST | `/tasks/:id/comments` | Bearer | Create comment (`content`) |
| GET | `/tasks/:id/comments` | Bearer | List comments (`?limit=`) |
| DELETE | `/tasks/:id/comments/:commentId` | Bearer | Delete comment |
| GET | `/tasks/:id/operation-logs` | Bearer | List task operation logs |

## Documentation

- **[Postman Collection](docs/postman/task-platform.postman_collection.json)** — Import into Postman. Auto-sets token on Register/Login; all endpoints with example bodies and auto-synced version variables.
- **[Interview Script](docs/interview-script.md)** — 面试讲解稿 (Chinese): architecture rationale, key highlights, challenges & solutions, common follow-up questions.

### Pagination

- **Projects:** offset-based (`limit` default 20, max 50)
- **Tasks:** cursor-based (base64url-encoded JSON: `{created_at, id, filter_hash}`; server rejects mismatched filter hash)

### Idempotency

All write endpoints (`POST`, `PUT`, `DELETE`) accept `Idempotency-Key` header. Duplicate requests with the same key within 24h return the cached response (200 OK) instead of re-executing. Keys are freed on error so clients can retry.

## Configuration

Multi-environment configs in `configs/`:

| Env | Directory | Purpose |
|-----|-----------|---------|
| local | `configs/local/` | Local development |
| dev | `configs/dev/` | CI / staging |
| docker | `configs/docker/` | Docker Compose deployment |

Each service has its own YAML file. Config is loaded via Viper with env var override.

### Key Environment Variables

| Variable | Service | Description |
|----------|---------|-------------|
| `JWT_SECRET` | gateway, user | JWT signing key (≥32 chars) |
| `INTERNAL_TOKEN` | all | Internal RPC auth token (≥16 chars) |
| `POSTGRES_DSN` | user, task | PostgreSQL connection string |
| `REDIS_ADDR` | all | Redis address (`host:port`) |
| `REDIS_PASSWORD` | all | Redis password (optional) |

Default ports: API gateway `:8080`, user-service gRPC `:9091`, task-service gRPC `:9092`, PostgreSQL `:5433`, Redis `:6380`.

## Key Architectural Decisions

- **One DB, two schemas:** Single PostgreSQL instance with `user_svc` and `task_svc` schemas
- **Shared Redis:** One Redis instance for caching, rate limiting, and idempotency
- **Gateway-only JWT verification:** Internal services trust injected `x-user-id`/`x-username` metadata headers; they never hold the JWT secret
- **Internal RPC auth:** `x-internal-token` static shared secret validated by server interceptor on every RPC
- **Optimistic locking:** `projects.version` and `tasks.version` prevent concurrent write conflicts
- **Soft deletes with partial unique indexes:** `users` and `projects` use GORM soft delete with `WHERE deleted_at IS NULL` partial unique indexes
- **Owner consistency:** `projects.owner_id` is the source of truth; `project_members` keeps a redundant `role=owner` row, updated atomically in `TransferOwnership`
- **Non-member = NOT_FOUND:** Access denied for non-members returns 404 (not 403) to avoid leaking resource existence
- **Async operation logs:** Buffered channel writer prevents log I/O from blocking business responses; degrades gracefully under backpressure
- **Idempotency in handlers, not middleware:** Checked after bind/validate so invalid requests don't consume idempotency keys

## Tech Stack

Go 1.26+ · Gin · gRPC + protobuf (buf) · PostgreSQL 16 · Redis 7 · GORM · pgx/v5 · JWT HS256 · bcrypt · Zap · Viper · Docker Compose · testcontainers-go · testify · golangci-lint · GitHub Actions CI

## Project Layout

```
├── api/proto/              # Protobuf definitions (user/v1, task/v1)
├── cmd/                    # Service entrypoints (api-gateway, user-service, task-service)
├── configs/                # Multi-env YAML configs (local, dev, docker)
├── gen/go/                 # Generated protobuf Go code
├── internal/
│   ├── gateway/            # API gateway (handler, middleware, rpc, server)
│   ├── user/               # User service (biz, data, service, server)
│   └── task/               # Task service (biz, data, service, server)
├── migrations/             # SQL migrations per schema
├── pkg/                    # Shared packages (xerr, xredis, xjwt, xlog, xratelimit, etc.)
├── scripts/                # Build/test/migration scripts
├── test/integration/       # Integration tests (real PG + Redis via testcontainers)
├── deploy/                 # Docker Compose, Prometheus, Grafana configs
└── Makefile                # proto, test, lint, coverage, migrate, up/down
```
