# 可观测性运维

项目内置健康检查、Prometheus 指标、OpenTelemetry trace、Zap JSON 日志和 Grafana 仪表盘。可观测性目标是快速回答四类问题：服务是否可用、请求是否变慢、错误发生在哪里、业务链路是否完整。

## 健康检查

所有 HTTP/admin 服务都暴露：

| 路径 | 说明 |
|------|------|
| `/healthz` | 进程存活 |
| `/readyz` | 服务是否已进入 ready 状态 |
| `/metrics` | Prometheus 指标 |

默认地址：

| 服务 | 地址 |
|------|------|
| api-gateway | `http://127.0.0.1:8080` |
| user-service admin | `http://127.0.0.1:8081` |
| task-service admin | `http://127.0.0.1:8082` |

## 指标

Prometheus 配置位于 `deploy/prometheus.yml`，默认每 15 秒采集：

```text
api-gateway: host.docker.internal:8080
user-service-admin: host.docker.internal:8081
task-service-admin: host.docker.internal:8082
```

核心指标类型：

| 指标 | 说明 |
|------|------|
| `task_platform_boot_total{service}` | 服务启动计数 |
| `http_requests_total{method,path,status}` | HTTP 请求总数 |
| `http_request_duration_seconds` | HTTP 请求延迟直方图 |
| `grpc_client_requests_total{grpc_method,grpc_code}` | gRPC 客户端调用结果 |
| `grpc_server_requests_total{grpc_method,grpc_code}` | gRPC 服务端调用结果 |
| `grpc_client_request_duration_seconds` | gRPC 客户端调用延迟直方图 |
| `grpc_server_request_duration_seconds` | gRPC 服务端处理延迟直方图 |
| `gateway_rate_limit_allowed_total{scope}` | 限流放行数 |
| `gateway_rate_limit_rejected_total{scope}` | 限流拒绝数 |
| `gateway_rate_limiter_errors_total` | 限流 Redis 错误导致 fail-open 的次数 |
| `task_platform_log_writer_channel_full_total` | 操作日志 channel 满并降级同步写的次数 |
| `task_platform_log_writer_batch_failure_total` | 操作日志批量写入重试后仍失败的次数 |
| `task_platform_log_writer_worker_panic_total` | 操作日志 worker panic 并重启的次数 |
| `tracing_enabled` | trace exporter 是否启用 |
| DB/Redis 指标 | 由 `pkg/xpgsql`、`pkg/xredis` 封装采集 |

指标命名以代码实现为准。新增指标时需要注意 label 基数，路由 label 应使用模板路径，不要使用原始 URL。

## Grafana

Grafana 默认地址：

```text
http://127.0.0.1:3000
```

仪表盘文件：

```text
deploy/grafana/dashboards/task-platform.json
```

预置视图覆盖：

- HTTP 总览：QPS、错误率、延迟分位数
- gRPC 调用：客户端和服务端状态码、耗时
- DB/Redis：操作耗时和错误
- 限流：IP、用户、认证接口限流命中
- 系统状态：in-flight 请求、服务启动、trace 状态
- 业务指标：项目、任务、评论、操作日志相关计数

截图位于 `deploy/grafana/screenshots/`，可用于文档或演示。

## Trace

OpenTelemetry 初始化位于 `pkg/xtrace`。默认行为：

| 配置 | 默认 |
|------|------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `127.0.0.1:4317` |
| `OTEL_EXPORTER_OTLP_TIMEOUT` | `5` 秒 |
| Sampler | AlwaysSample |
| Propagator | W3C TraceContext |

如果 exporter 初始化失败，服务不会启动失败，而是将 `tracing_enabled` 置为 `0` 并继续运行。

查看 trace：

```text
http://127.0.0.1:16686
```

典型链路：

```text
api-gateway HTTP span
  -> gRPC client span
  -> user-service/task-service gRPC server span
  -> DB/Redis 操作 span
```

## 日志

日志使用 Zap JSON 输出，关键字段：

| 字段 | 说明 |
|------|------|
| `service` | 服务名 |
| `env` | 环境 |
| `method` | HTTP 或 gRPC 方法 |
| `path` | HTTP 路径 |
| `status` | HTTP 状态码 |
| `latency` | 请求耗时 |
| `request_id` | 请求 ID |
| `trace_id` | trace ID |
| `span_id` | span ID |
| `error` | 错误信息 |

排障时优先使用 `request_id` 在 gateway 和两个服务之间串联日志；有 trace 时再用 `trace_id` 跳转到 Jaeger。

## 常见排障入口

### 接口返回 401

检查：

- `Authorization` header 是否存在且格式为 `Bearer <token>`。
- token 是否过期。
- 是否已调用 logout，导致 `jti` 进入 Redis 黑名单。
- Gateway 的 `JWT_SECRET` 是否与 user-service 签发 token 使用的 secret 一致。

### 接口返回 429

检查：

- 是否命中 auth/IP/user 级限流。
- Grafana 中 `gateway_rate_limit_rejected_total{scope}` 是否突增，同时结合 `gateway_rate_limit_allowed_total{scope}` 判断整体流量。
- 压测是否超过 `.env` 中的 `RATELIMIT_*` 配置。

### Gateway 返回 503 或 504

检查：

- `USER_SERVICE_ADDR`、`TASK_SERVICE_ADDR` 是否正确。
- user-service、task-service 是否 ready。
- 内部 gRPC 调用是否超时。
- 服务日志中是否有 `UNAVAILABLE` 或 deadline exceeded。

### Trace 不出现

检查：

- Jaeger 是否启动并监听 `4317`。
- `OTEL_EXPORTER_OTLP_ENDPOINT` 是否指向正确地址。
- `/metrics` 中 `tracing_enabled` 是否为 `1`。

### 操作日志缺失

检查：

- task-service 日志中是否有操作日志批量写入失败。
- `operation_logs` 表是否有对应 `project_id` 或 `task_id` 数据。
- 是否出现 log writer channel 满降级告警，或 `task_platform_log_writer_batch_failure_total`、`task_platform_log_writer_worker_panic_total` 增长。
