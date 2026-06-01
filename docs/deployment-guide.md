# 部署指南

当前仓库提供 Docker Compose 形式的本地基础设施部署：PostgreSQL、Redis、Prometheus、Jaeger 和 Grafana。三个 Go 服务通过本地进程运行，适合开发、演示和面试验收。

## 部署组件

| 组件 | 镜像/来源 | 端口 | 说明 |
|------|-----------|------|------|
| PostgreSQL | `postgres:16` | host `5433` -> container `5432` | 主数据库 |
| Redis | `redis:7` | host `6380` -> container `6379` | 缓存、限流、幂等、黑名单 |
| Prometheus | `prom/prometheus:v2.54.0` | `9090` | 指标采集 |
| Jaeger | `jaegertracing/all-in-one:1.69.0` | `16686`、`4317`、`4318` | Trace UI 和 OTLP collector |
| Grafana | `grafana/grafana:11.1.0` | `3000` | 仪表盘 |
| api-gateway | Go 进程 | `8080` | HTTP API |
| user-service | Go 进程 | `9091`、`8081` | gRPC + admin HTTP |
| task-service | Go 进程 | `9092`、`8082` | gRPC + admin HTTP |

## 本地部署步骤

1. 准备环境变量：

```bash
cp .env.example .env
```

至少设置：

```text
POSTGRES_DSN=postgres://postgres:postgres@127.0.0.1:5433/task_platform?sslmode=disable
JWT_SECRET=<32 chars or longer>
INTERNAL_TOKEN=<16 chars or longer>
```

2. 启动基础设施：

```bash
make up
```

3. 执行迁移：

```bash
make migrate
```

4. 启动服务：

```bash
make run/user-service
make run/task-service
make run/api-gateway
```

5. 验证：

```bash
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/readyz
curl http://127.0.0.1:8081/readyz
curl http://127.0.0.1:8082/readyz
```

## 启动顺序

推荐顺序：

1. PostgreSQL 和 Redis
2. 数据库迁移
3. user-service
4. task-service
5. api-gateway
6. 前端或 Postman 调试

`task-service` 会连接 `user-service`，`api-gateway` 会连接两个 gRPC 服务，因此服务进程顺序不要反过来。

## 健康检查

每个 HTTP/admin 服务都暴露：

| 路径 | 说明 |
|------|------|
| `/` | 返回服务名和 bootstrapped 状态 |
| `/healthz` | 进程健康检查 |
| `/readyz` | 就绪状态 |
| `/metrics` | Prometheus 指标 |

Prometheus 默认采集：

| Job | Target |
|-----|--------|
| `api-gateway` | `host.docker.internal:8080` |
| `user-service-admin` | `host.docker.internal:8081` |
| `task-service-admin` | `host.docker.internal:8082` |

## 数据持久化

Docker Compose 使用命名 volume：

| Volume | 用途 |
|--------|------|
| `postgres-data` | PostgreSQL 数据 |
| `redis-data` | Redis AOF 数据 |
| `prometheus-data` | Prometheus TSDB |
| `grafana-data` | Grafana 数据 |

停止服务：

```bash
make down
```

该命令不会删除 volume。如需清空数据，应显式执行 Docker volume 删除命令，并确认不会误删有价值数据。

## Grafana

默认地址：

```text
http://127.0.0.1:3000
```

默认账号密码由环境变量控制：

| 变量 | 默认 |
|------|------|
| `GRAFANA_USER` | `admin` |
| `GRAFANA_PASSWORD` | `admin` |

仪表盘和数据源通过 `deploy/grafana/` 自动 provision。

## Jaeger

默认 UI：

```text
http://127.0.0.1:16686
```

服务默认向 `127.0.0.1:4317` 发送 OTLP gRPC trace。Docker Compose 中 Jaeger 暴露了 `4317` 和 `4318`。

## 生产化注意事项

当前 Compose 文件面向本地开发。生产部署前至少需要补齐：

- 将 Go 服务容器化，并用编排系统管理生命周期。
- PostgreSQL、Redis 使用托管服务或独立高可用部署。
- secret 通过 secret manager 注入，不使用 `.env` 文件。
- Redis 开启密码、网络隔离和持久化策略评估。
- Gateway 放在反向代理或负载均衡后，并正确处理真实客户端 IP。
- TLS 在入口层终止，必要时内部链路也启用 TLS。
- 为数据库迁移建立发布前检查和回滚流程。
- Prometheus、Grafana、Jaeger 设置持久化、鉴权和保留周期。
- 根据容量目标调整 `DB_MAX_OPEN_CONNS`、`DB_MAX_IDLE_CONNS`、Redis 连接池参数、Redis 限流参数、`GRPC_CLIENT_TIMEOUT_SECONDS`、`GRPC_SERVER_TIMEOUT_SECONDS`、`LOG_WRITER_WORKERS` 和服务副本数。

## 回滚建议

- 代码回滚：先回滚 gateway，再回滚 task-service/user-service，避免新 gateway 调用旧服务不存在的 RPC。
- 数据库回滚：只对可安全回滚的 schema 变更执行 `.down.sql`；涉及数据迁移时优先使用向前兼容策略。
- 配置回滚：确保 `JWT_SECRET` 与 `INTERNAL_TOKEN` 的变更与服务重启顺序一致，否则会出现 token 验签或内部 RPC 认证失败。
