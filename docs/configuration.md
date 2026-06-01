# 配置说明

项目配置分为两类：服务启动配置和运行期环境变量。启动配置来自 `configs/<env>/<service>.yaml`，运行期依赖、secret、连接地址和调优参数来自环境变量或 `.env`。

## 配置加载顺序

每个服务入口都使用 Viper 加载配置：

1. 如果设置 `CONFIG_FILE`，优先读取该文件。
2. 否则读取 `configs/${APP_ENV}/${service}.yaml`。
3. `APP_ENV` 未设置时默认为 `local`。
4. Viper 启用 `AutomaticEnv()`，同名环境变量可以覆盖配置项。

示例：

```bash
APP_ENV=local make run/api-gateway
CONFIG_FILE=configs/docker/task-service.yaml make run/task-service
```

## 服务配置文件

### `api-gateway`

| 字段 | 默认 | 说明 |
|------|------|------|
| `service_name` | `api-gateway` | 日志、指标和 trace 服务名 |
| `env` | `local` | 运行环境 |
| `http_addr` | `:8080` | HTTP 监听地址 |
| `shutdown_timeout_seconds` | `10` | 优雅关闭超时 |
| `read_header_timeout_seconds` | `5` | 读取 HTTP header 超时 |
| `read_timeout_seconds` | `10` | 读取完整 HTTP 请求超时 |
| `write_timeout_seconds` | `15` | 写 HTTP 响应超时 |
| `idle_timeout_seconds` | `60` | keep-alive 空闲连接超时 |

### `user-service`

| 字段 | 默认 | 说明 |
|------|------|------|
| `service_name` | `user-service` | 服务名 |
| `env` | `local` | 运行环境 |
| `grpc_addr` | `:9091` | gRPC 监听地址 |
| `admin_addr` | `:8081` | admin HTTP 地址，暴露 `/healthz`、`/readyz`、`/metrics` |
| `reflection_enabled` | `true` in local config | 是否启用 gRPC reflection |
| `shutdown_timeout_seconds` | `10` | 优雅关闭超时 |
| `read_header_timeout_seconds` | `5` | admin HTTP 读取 header 超时 |
| `read_timeout_seconds` | `10` | admin HTTP 读取完整请求超时 |
| `write_timeout_seconds` | `15` | admin HTTP 写响应超时 |
| `idle_timeout_seconds` | `60` | admin HTTP keep-alive 空闲连接超时 |

### `task-service`

| 字段 | 默认 | 说明 |
|------|------|------|
| `service_name` | `task-service` | 服务名 |
| `env` | `local` | 运行环境 |
| `grpc_addr` | `:9092` | gRPC 监听地址 |
| `admin_addr` | `:8082` | admin HTTP 地址 |
| `reflection_enabled` | `true` in local config | 是否启用 gRPC reflection |
| `shutdown_timeout_seconds` | `10` | 优雅关闭超时 |
| `read_header_timeout_seconds` | `5` | admin HTTP 读取 header 超时 |
| `read_timeout_seconds` | `10` | admin HTTP 读取完整请求超时 |
| `write_timeout_seconds` | `15` | admin HTTP 写响应超时 |
| `idle_timeout_seconds` | `60` | admin HTTP keep-alive 空闲连接超时 |

## 环境变量

根目录 `.env.example` 给出了本地模板。

| 变量 | 默认/示例 | 使用方 | 说明 |
|------|-----------|--------|------|
| `APP_ENV` | `local` | all | 配置目录选择 |
| `CONFIG_FILE` | 空 | all | 显式配置文件路径 |
| `POSTGRES_USER` | `postgres` | migration/docker | PostgreSQL 用户 |
| `POSTGRES_PASSWORD` | `postgres` | migration/docker | PostgreSQL 密码 |
| `POSTGRES_DB` | `task_platform` | migration/docker | 数据库名 |
| `POSTGRES_HOST` | `127.0.0.1` | migration | 数据库主机 |
| `POSTGRES_PORT` | `5433` | migration | 数据库端口 |
| `POSTGRES_DSN` | 必填 | user/task | Go 服务连接数据库的 DSN |
| `REDIS_HOST` | `127.0.0.1` | all | Redis 主机 |
| `REDIS_PORT` | `6380` | all | Redis 端口 |
| `REDIS_ADDR` | 可选 | gateway/user/task | 代码默认用 `REDIS_HOST:REDIS_PORT` 拼接 |
| `REDIS_PASSWORD` | 空 | all | Redis 密码 |
| `REDIS_POOL_SIZE` | `100` | all | Redis 最大连接池大小 |
| `REDIS_MIN_IDLE_CONNS` | `10` | all | Redis 最小空闲连接数 |
| `REDIS_DIAL_TIMEOUT_SECONDS` | `3` | all | Redis 建连超时 |
| `REDIS_READ_TIMEOUT_SECONDS` | `2` | all | Redis 读超时 |
| `REDIS_WRITE_TIMEOUT_SECONDS` | `2` | all | Redis 写超时 |
| `REDIS_POOL_TIMEOUT_SECONDS` | `4` | all | Redis 从连接池取连接超时 |
| `JWT_SECRET` | 必填 | gateway/user | JWT HS256 secret，至少 32 字符，不能使用模板占位值 |
| `INTERNAL_TOKEN` | 必填 | all | 内部 RPC secret，至少 16 字符，不能使用模板占位值 |
| `USER_SERVICE_ADDR` | `127.0.0.1:9091` | gateway/task | user-service gRPC 地址 |
| `TASK_SERVICE_ADDR` | `127.0.0.1:9092` | gateway | task-service gRPC 地址 |
| `GRPC_ADDR` | 服务默认值 | user/task | 覆盖 gRPC 监听地址 |
| `ADMIN_ADDR` | 服务默认值 | user/task | 覆盖 admin HTTP 地址 |
| `GRPC_CLIENT_TIMEOUT_SECONDS` | `2` | gateway/task | 内部 gRPC client 默认 deadline；已有 deadline 时不覆盖 |
| `GRPC_SERVER_TIMEOUT_SECONDS` | `3` | user/task | gRPC server 默认处理 deadline；入站已有 deadline 时不覆盖 |
| `WEAK_PASSWORDS_PATH` | `configs/security/weak_passwords.txt` | user | 弱密码黑名单路径 |
| `BCRYPT_COST` | `10` | user | bcrypt cost，建议 10；已有低 cost hash 登录后会自动重算 |
| `DB_MAX_OPEN_CONNS` | `100` in example | user/task | DB 最大连接数 |
| `DB_MAX_IDLE_CONNS` | `25` in example | user/task | DB 最大空闲连接数 |
| `LOG_WRITER_WORKERS` | `1` | task | 操作日志异步写入 worker 数，最大 16 |
| `RATELIMIT_AUTH_RATE` | `5` | gateway | 认证接口每秒令牌数 |
| `RATELIMIT_AUTH_BURST` | `10` | gateway | 认证接口突发容量 |
| `RATELIMIT_IP_RATE` | `60` | gateway | 普通接口 IP 级每秒令牌数 |
| `RATELIMIT_IP_BURST` | `100` | gateway | 普通接口 IP 级突发容量 |
| `RATELIMIT_USER_RATE` | `100` | gateway | 用户级每秒令牌数 |
| `RATELIMIT_USER_BURST` | `200` | gateway | 用户级突发容量 |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `127.0.0.1:4317` | all | OTLP gRPC trace exporter 地址 |
| `OTEL_EXPORTER_OTLP_TIMEOUT` | `5` | all | 初始化 exporter 超时秒数 |

## 本地端口

| 服务 | 端口 |
|------|------|
| api-gateway HTTP | `8080` |
| user-service gRPC | `9091` |
| user-service admin HTTP | `8081` |
| task-service gRPC | `9092` |
| task-service admin HTTP | `8082` |
| PostgreSQL host port | `5433` |
| Redis host port | `6380` |
| Prometheus | `9090` |
| Jaeger UI | `16686` |
| Grafana | `3000` |

## 前端配置

`web/.env.example`：

| 变量 | 默认 | 说明 |
|------|------|------|
| `VITE_API_BASE_URL` | `/api/v1` | 前端 API base URL |
| `VITE_ENABLE_MSW` | `false` | 是否启用浏览器端 MSW mock |

开发时：

```bash
cd web
cp .env.example .env.local
npm run dev
```

独立 mock 模式：

```bash
cd web
VITE_ENABLE_MSW=true npm run dev
```

## Secret 要求

- `JWT_SECRET` 至少 32 字符。
- `INTERNAL_TOKEN` 至少 16 字符。
- 不允许使用 `.env.example` 中的 `replace-with-*` 占位值启动服务。
- 生产环境 secret 必须来自安全的 secret manager 或部署系统注入，不能提交到仓库。
