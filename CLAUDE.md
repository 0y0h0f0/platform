# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Status

This repo is **planning-stage**. The only file is `plan.md`, a detailed Chinese-language spec for a campus-recruitment ("校招") portfolio backend called **团队任务协作平台 / Team Task Collaboration Platform**. No Go module, no source code, no `docker-compose.yml`, no protos exist yet — they will be created following `plan.md`.

When starting implementation, follow the execution order in `plan.md` §17:
1. Initialize repo and directory structure (Go Module + `.github/workflows/` + `.env.example`)
2. Write `deploy/docker-compose.yml` to bring up PostgreSQL + Redis
3. Configure `buf.yaml` + `buf.gen.yaml`; define `user.proto` and `task.proto` skeletons and generate code
4. Scaffold empty `api-gateway`, `user-service`, `task-service` (compiles + starts)
5. Land `pkg/xerr` error-code table, `golangci-lint` config, GitHub Actions workflow
6. Wire `/metrics` (dummy counter), `/healthz`, `/readyz`, graceful shutdown in each service
7. Implement `Register → Login → Logout → Me` (JWT, token blacklist, `BatchGetUsers`)
8. Implement `CreateProject → AddMember → ArchiveProject → TransferOwnership` (full permission matrix)
9. Implement `CreateTask → AssignTask → ChangeTaskStatus → ListTasks` (optimistic lock, cursor pagination)
10. Add comments, operation logs (async worker), cache invalidation, rate limiting, idempotency
11. Enrich metrics + trace, load test, Grafana screenshots
12. Finalize README, Postman Collection, interview script

## Planned Architecture

External clients speak HTTP/JSON; internal services speak gRPC. The shape is fixed in `plan.md` §4 and should be preserved:

```
client → api-gateway (Gin, HTTP)
            │  gRPC (unary, with deadline + request_id)
            ├─→ user-service  → PostgreSQL schema user_svc
            └─→ task-service  → PostgreSQL schema task_svc
                                ↘ Redis (cache + simple rate limit)
```

Key architectural rules from `plan.md`:
- **One** PostgreSQL instance, logically split into two schemas: `user_svc` and `task_svc`. Do not spin up two database instances.
- **One** Redis instance shared by both services.
- Service count is intentionally capped at **api-gateway + 2 core services**. Resist further service splits — `plan.md` §15 lists "拆分过细" (over-splitting) as the top risk.
- All RPCs are **unary**, must carry `request_id`, and must set a deadline (default 2s, per-method override allowed).
- gRPC `status` codes are the source of truth for errors; the gateway is the only layer that translates them into the unified HTTP response envelope `{ code, message, request_id, data? }`.
- The gateway owns: parameter binding, JWT auth, request-id injection, panic recovery, unified error → HTTP mapping. Business services must not duplicate this.

## Authentication & Cross-Service Contract (`plan.md` §4.4)

- External auth: `Authorization: Bearer <token>` (JWT HS256, access token only, 2h TTL).
- JWT claims: `sub(user_id)`, `username`, `jti`, `iat`, `exp`.
- **JWT is verified only at api-gateway** — signature, expiry, Redis blacklist. Internal services never hold the JWT secret.
- `POST /auth/logout` is a **gateway-local operation**: parse `jti` from the current token, write to Redis blacklist (TTL = remaining token lifetime). No RPC to `user-service`.
- `api-gateway → gRPC` calls must inject four metadata headers: `x-user-id`, `x-username`, `x-request-id`, `x-internal-token`.
- `x-internal-token` is a static shared secret from env var; **every** internal RPC interceptor must validate it.
- Anonymous RPCs (`Register`, `Login`) may omit `x-user-id` / `x-username`. All other RPCs require them; interceptor returns `UNAUTHENTICATED` if missing.
- `task-service` must never store user nicknames/avatars. For list/detail endpoints needing display names, `api-gateway` calls `BatchGetUsers` (max 100 IDs, Redis-cached TTL 5 min) to enrich the response after fetching the list.

## Middleware Chain Order (`plan.md` §9.1)

```
Recovery → RequestID → AccessLog → CORS → RateLimit(IP) → JWT → RateLimit(user) → Handler
```

Parameter binding/validation happens inside handlers, not as a standalone Gin middleware. Inside a write handler: bind/validate first, then idempotency check — prevents invalid requests from consuming `Idempotency-Key` slots.

## Planned Layout (`plan.md` §5)

```
task-platform/
├── .github/
│   └── workflows/            # CI: lint + test + buf lint + coverage gate
├── cmd/
│   ├── api-gateway/
│   ├── user-service/
│   └── task-service/
├── api/
│   └── proto/
│       ├── user/v1/
│       └── task/v1/
├── internal/
│   ├── gateway/{handler,middleware,rpc,service}/
│   ├── user/{biz,data,service,server}/   # biz=domain, data=repo, service=gRPC impl, server=wiring
│   └── task/{biz,data,service,server}/
├── pkg/{xerr,xgrpc,xhttp,xjwt,xlog,xtrace,xredis,xpgsql}/   # shared, prefixed with `x`
├── configs/{local,dev,docker}/
├── migrations/               # SQL migrations per schema
├── scripts/                  # run-migrations.sh, seed.sh
├── deploy/{docker-compose.yml,prometheus.yml,grafana/}
├── test/
├── .env.example              # placeholder for secrets; .env is gitignored
└── README.md
```

Each business service follows the same four-layer split: `biz` (domain logic) ← `data` (repository / GORM) ← `service` (gRPC method implementations) ← `server` (DI + lifecycle). Cross-cutting concerns belong in `pkg/x*`, never duplicated per service.

## Tech Stack (fixed by `plan.md` §2.1)

Go 1.23+ · Gin · gRPC + protobuf (proto3) · `buf` (lint + breaking-change + codegen) · PostgreSQL 16 · Redis 7 · GORM (`gorm.io/driver/postgres`) · pgx/v5 driver · `golang-migrate` · Viper (YAML + env-var override, multi-env dirs) · JWT HS256 · bcrypt (cost=10) · Zap (JSON, request_id-linked) · Prometheus + OpenTelemetry · Docker Compose · `go test` + `testify` + `httptest` + `testcontainers-go` (integration tests against real PG/Redis) · `golangci-lint` · GitHub Actions CI.

Do not introduce alternatives (e.g. Echo, sqlx, MySQL, logrus, sqlc) without an explicit reason — the stack is part of the project's interview narrative.

## Proto Design Summary (`plan.md` §7)

**`user.proto`**: `Register` / `Login` / `GetUser` / `BatchGetUsers`

`Logout` is **not** in `user.proto` — it is handled entirely inside the gateway (Redis blacklist write, no RPC).

**`task.proto`**: Projects (`CreateProject` / `UpdateProject` / `ArchiveProject` / `UnarchiveProject` / `TransferProjectOwnership` / `ListProjects` / `GetProject`) · Members (`AddProjectMember` / `RemoveProjectMember` / `UpdateProjectMemberRole` / `LeaveProject` / `ListProjectMembers` / `CheckProjectMember`) · Tasks (`CreateTask` / `UpdateTask` / `DeleteTask` / `GetTask` / `ListTasks` / `AssignTask` / `ChangeTaskStatus`) · Comments (`CreateTaskComment` / `DeleteTaskComment` / `ListTaskComments`) · Logs (`ListOperationLogs`)

`UpdateTask` must not accept `assignee_id` or `status` — those fields belong to `AssignTask` and `ChangeTaskStatus` exclusively.

## HTTP API Summary (`plan.md` §8)

- Auth: `POST /api/v1/auth/register` · `POST /api/v1/auth/login` · `POST /api/v1/auth/logout` · `GET /api/v1/users/me`
- Projects: CRUD + `POST /:id/archive` · `POST /:id/unarchive` · `POST /:id/transfer` · member endpoints + `GET /:id/operation-logs`
- Tasks: CRUD + `POST /:id/assign` · `POST /:id/status` · comment endpoints + `GET /:id/operation-logs`

Pagination: projects use offset (`limit` default 20 max 50). Tasks use **cursor pagination** — cursor is base64url-encoded JSON containing `created_at`, `id`, `filter_hash`; server rejects a cursor whose `filter_hash` doesn't match the current filter params.

## Commands (to be created during Phase 0)

`plan.md` §10 Phase 0 commits the project to a `Makefile`. Expected targets:
- `make proto` — regenerate gRPC stubs via `buf generate`
- `make run/<svc>` — run a service locally (parameterized)
- `make test` — run all tests
- `make lint` — run `golangci-lint`
- `make migrate` — run database migrations
- `docker-compose -f deploy/docker-compose.yml up` — bring up PostgreSQL + Redis (Phase 0 acceptance criterion)

Until `Makefile` exists, use the standard Go toolchain:
- Build a service: `go build ./cmd/<service>`
- Run tests: `go test ./...`
- Run a single test: `go test ./internal/user/biz -run TestRegister`
- Coverage: `go test -coverprofile=coverage.out ./internal/... ./pkg/...` (target ≥80%; `cmd/` and generated code excluded from the gate)

## Domain Model Anchors (`plan.md` §6)

**Critical columns:**
- `tasks.version` — optimistic-lock (bigint, default 0); every update must `WHERE version = ?` and increment it.
- `projects.version` — same optimistic-lock pattern.
- `tasks.extra jsonb` and `operation_logs.detail jsonb` — use Postgres `jsonb`, not text. `extra` only allows keys `labels`, `checklist`, `attachments`.
- `tasks.status` smallint: 0=todo / 1=doing / 2=done / 3=cancelled.
- `project_members.role` smallint: 0=owner / 1=admin / 2=member.
- `users.status` smallint: 0=active / 1=disabled; disabled users receive `PERMISSION_DENIED` on login.

**Uniqueness constraints:**
- `users(username)` and `users(email)` — partial unique index `WHERE deleted_at IS NULL`.
- `projects(owner_id, name)` — partial unique index `WHERE deleted_at IS NULL`.
- `project_members(project_id, user_id)` — unique index.
- `project_members(project_id) WHERE role = 0` — partial unique index (at most one owner row per project).

**Key indexes for query patterns:**
- `tasks(project_id, created_at DESC, id DESC) WHERE deleted_at IS NULL` — default cursor pagination.
- `tasks(project_id, status, created_at DESC, id DESC) WHERE deleted_at IS NULL` — filtered list.
- `tasks(project_id, assignee_id, created_at DESC, id DESC) WHERE deleted_at IS NULL` — assignee filter.
- `task_comments(task_id, id)` — comment `after_id` pagination.
- `operation_logs(project_id, created_at DESC, id DESC)` and `(task_id, created_at DESC, id DESC)`.

**Soft delete:** `users`, `projects`, `tasks` use GORM `DeletedAt`. `task_comments` and `project_members` use physical delete. Cross-table joins must explicitly add `deleted_at IS NULL` for parent tables.

**Owner consistency (`plan.md` §6.6):** `projects.owner_id` is the single source of truth. `project_members` keeps a redundant `role=owner` row for list display. Project creation and `TransferProjectOwnership` must update both in a single transaction.

## Task State Machine (`plan.md` §3.3)

Valid transitions only — any other transition returns `FAILED_PRECONDITION`:
```
todo   → doing | done | cancelled
doing  → done | cancelled | todo
done   → doing
cancelled → todo
```

## Permission Matrix Summary (`plan.md` §3.3)

| Operation | owner | admin | member |
|-----------|-------|-------|--------|
| Archive/unarchive project | ✓ | ✗ | ✗ |
| Transfer ownership | ✓ | ✗ | ✗ |
| Edit project info | ✓ | ✗ | ✗ |
| Invite member (any role) | ✓ | member only | ✗ |
| Remove member | ✓ (not self) | member only | ✗ |
| Change member role | ✓ | ✗ | ✗ |
| Create/edit/delete any task | ✓ | ✓ | own tasks only (delete: todo status only) |
| Assign task / change status | ✓ | ✓ | own tasks only |
| Delete any comment | ✓ | ✓ | own only |
| Leave project | ✗ (must transfer first) | ✓ | ✓ |

Archived projects are **read-only for all roles** — all write operations return `FAILED_PRECONDITION`.

Non-member access to project/task/comment/log resources returns `NOT_FOUND` (not `FORBIDDEN`) to avoid leaking resource existence.

## Operation Log Async Writer (`plan.md` §6.4)

- Channel capacity 1024, single worker, batch size 64, flush interval 100ms.
- On channel full: degrade to synchronous write + emit `warn` log + increment Prometheus counter.
- On batch write failure: retry up to 3 times (exponential backoff); log `error` and increment failure metric; do not block the main business response.
- On service shutdown: flush remaining entries, wait up to 3 seconds, then abandon and emit a warning.

## Phase Discipline

`plan.md` defines six phases (0–6) with explicit acceptance criteria. Each phase must end with a runnable, verifiable system before moving on (`plan.md` §15.2). When asked to "add X", first check which phase X belongs to and whether earlier phases are complete; do not pull observability or rate-limiting work forward of the core flow.

## Working With `plan.md`

`plan.md` is the source of truth for scope, structure, and rationale — treat it as the PRD. If a task contradicts it, flag the contradiction before implementing. Section numbers in this file (e.g. §4, §6.2) refer to `plan.md` and are stable.
