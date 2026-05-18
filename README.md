# Task Platform

`task-platform` is a campus-recruitment backend scaffold built from `plan.md`.

Current status: Phase 0 bootstrap.

## Included

- Go module and planned directory layout
- `api-gateway`, `user-service`, `task-service` empty entrypoints
- HTTP `/metrics`, `/healthz`, `/readyz`
- gRPC health service and reflection for internal services
- `buf` config and proto skeletons
- Docker Compose for PostgreSQL 16 and Redis 7
- GitHub Actions CI, `golangci-lint` config, coverage script

## Quick Start

```bash
cp .env.example .env
make up
make run/api-gateway
make run/user-service
make run/task-service
```

Default ports:

- `api-gateway`: HTTP `:8080`
- `user-service`: gRPC `:9091`, admin HTTP `:8081`
- `task-service`: gRPC `:9092`, admin HTTP `:8082`
- PostgreSQL: `:5432`
- Redis: `:6379`

## Notes

- `make proto` uses local `buf` when available; otherwise it falls back to the official Docker image.
- `make lint` uses local `golangci-lint` when available; otherwise it falls back to the official Docker image.
- Service business logic, migrations, and generated proto code land in later phases.

