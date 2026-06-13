# 可观测性运维

项目内置健康检查、Prometheus 指标、OpenTelemetry trace、Zap JSON 日志、Loki 日志聚合、Alertmanager 告警和 Grafana 仪表盘。可观测性目标是快速回答四类问题：服务是否可用、请求是否变慢、错误发生在哪里、业务链路是否完整。

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
alertmanager: alertmanager:9093
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

数据源通过 `deploy/grafana/datasources.yaml` 自动 provision：

| 数据源 | 用途 |
|--------|------|
| Prometheus | 指标查询（默认数据源） |
| Jaeger | 分布式追踪 |
| Loki | 日志聚合 |

仪表盘文件：

```text
deploy/grafana/dashboards/task-platform.json   # 核心指标仪表盘
deploy/grafana/dashboards/logs-dashboard.json   # 日志聚合仪表盘
```

预置视图覆盖：

- HTTP 总览：QPS、错误率、延迟分位数
- gRPC 调用：客户端和服务端状态码、耗时
- DB/Redis：操作耗时和错误
- 限流：IP、用户、认证接口限流命中
- 系统状态：in-flight 请求、服务启动、trace 状态
- 业务指标：项目、任务、评论、操作日志相关计数
- 日志总览：按 service/level 的日志量、实时日志查看、HTTP access log 分析、p95 延迟

截图位于 `deploy/grafana/screenshots/`，可用于文档或演示。

## Trace

OpenTelemetry 初始化位于 `pkg/xtrace`。默认行为：

| 配置 | 默认 |
|------|------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `127.0.0.1:4317` |
| `OTEL_EXPORTER_OTLP_TIMEOUT` | `5` 秒 |
| Sampler | AlwaysSample |
| Propagator | W3C TraceContext |

如果 exporter 初始化失败，服务不会启动失败，而是将 `tracing_enabled` 置为 `0` 并继续运行。此时 `TracingDisabled` 告警会触发。

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

## 日志聚合 (Loki)

Loki 提供集中式日志查询和存储。Promtail 作为日志采集 agent，自动发现并收集所有 Docker 容器的 stdout/stderr 日志。

### 架构

```text
Container stdout/stderr -> Promtail (Docker socket) -> Loki -> Grafana
```

**Go 应用代码无需修改** — Zap 已输出 JSON 格式日志，`trace_id` 和 `span_id` 已在 access log 中，Promtail 自动解析 JSON 字段。

### 组件地址

| 组件 | 地址 | 说明 |
|------|------|------|
| Loki HTTP | `http://127.0.0.1:3100` | 日志写入与查询 |
| Loki gRPC | `http://127.0.0.1:9096` | 内部 gRPC |
| Promtail HTTP | `http://127.0.0.1:9080` | 健康检查 |

### 配置文件

| 文件 | 说明 |
|------|------|
| `deploy/loki-config.yaml` | Loki 服务端配置：30 天保留，filesystem 存储，TSDB schema v13，ruler 规则引擎 |
| `deploy/promtail-config.yaml` | Promtail 采集配置：Docker socket 自动发现，按容器名打 service label |
| `deploy/loki-rules.yml` | Loki ruler 告警规则：ERROR 日志速率 |

### Grafana 中使用 Loki

1. 打开 Grafana → Explore → 数据源选择 "Loki"
2. 查询所有 API Gateway 日志：
   ```
   {service="api-gateway"}
   ```
3. 查询错误日志：
   ```
   {service=~"api-gateway|user-service|task-service"} | json | level =~ "error|ERROR"
   ```
4. 按 request_id 串联日志：
   ```
   {service=~"api-gateway|user-service|task-service"} | json | request_id = "xxxxx"
   ```

### Trace-to-Log 关联

Grafana 中已配置 Jaeger trace 到 Loki 日志的自动关联：
- 在 Jaeger 中查看某个 trace span → 点击 "Logs for this span" → 自动跳转到 Loki 查询对应 `trace_id` 的日志
- 反向：在 Loki 中查看日志 → 点击 `trace_id` 字段 → 跳转到 Jaeger 查看完整 trace

### 数据持久化

Loki 数据存储在命名 volume `loki-data`（Docker Compose）或 PVC（Kubernetes），停止服务不会删除。

## 告警

告警系统通过 **Prometheus 规则 → Alertmanager → Webhook** 管道实现自动化通知。Loki ruler 作为补充，基于日志量触发告警。

### 架构

```text
Metrics:  Prometheus (rule_files) --> Alertmanager --> Webhook / 日志
Logs:     Loki (ruler) -----------> Alertmanager --> Webhook / 日志
```

### 组件地址

| 组件 | 地址 | 说明 |
|------|------|------|
| Alertmanager | `http://127.0.0.1:9093` | 告警管理、分组、路由、静默 |
| Prometheus Alerts | `http://127.0.0.1:9090/alerts` | 当前触发中的告警 |
| Alertmanager UI | `http://127.0.0.1:9093/#/alerts` | 接收到的告警列表 |

### 配置文件

| 文件 | 说明 |
|------|------|
| `deploy/alertmanager.yml` | Alertmanager 路由规则 + webhook receiver |
| `deploy/prometheus-rules.yml` | Prometheus 告警规则（8 组 20 条） |
| `deploy/loki-rules.yml` | Loki ruler 告警规则（ERROR 日志速率） |

### 告警严重度分级

| 级别 | 含义 | 响应要求 |
|------|------|----------|
| **critical** | 服务宕机、数据丢失风险 | **立即处理**，1h 重复通知 |
| warning | 性能下降、异常但不致命 | 工作时间处理，4h 重复通知 |

### Prometheus 告警规则清单

#### 服务健康 (service_health)

| 告警 | 严重度 | 条件 | for |
|------|--------|------|-----|
| `ServiceDown` | **critical** | `up == 0`（服务不可达） | 1m |
| `ServiceFlapping` | warning | `rate(boot[5m]) > 0.01`（频繁重启） | 2m |

#### HTTP 错误 (http_errors)

| 告警 | 严重度 | 条件 | for |
|------|--------|------|-----|
| `HighHTTP5xxRate` | **critical** | 5xx 比例 > 5% | 2m |
| `HighHTTP4xxRate` | warning | 4xx 比例 > 20% | 2m |

#### HTTP 延迟 (http_latency)

| 告警 | 严重度 | 条件 | for |
|------|--------|------|-----|
| `HighHTTPLatency` | warning | p99 > 1s | 2m |

#### gRPC 错误 (grpc_errors)

| 告警 | 严重度 | 条件 | for |
|------|--------|------|-----|
| `HighgRPCServerErrorRate` | **critical** | 服务端错误 > 5% | 2m |
| `HighgRPCClientErrorRate` | warning | 客户端错误 > 5% | 2m |

#### gRPC 延迟 (grpc_latency)

| 告警 | 严重度 | 条件 | for |
|------|--------|------|-----|
| `HighgRPCLatency` | warning | p99 > 1s | 2m |

#### 数据库 (database)

| 告警 | 严重度 | 条件 | for |
|------|--------|------|-----|
| `HighDatabaseErrorRate` | **critical** | DB 错误 > 2% | 2m |
| `DatabaseConnectionPoolHigh` | warning | 连接池使用率 > 80% | 2m |
| `HighDatabaseLatency` | warning | p99 > 500ms | 2m |

#### Redis (redis)

| 告警 | 严重度 | 条件 | for |
|------|--------|------|-----|
| `HighRedisErrorRate` | warning | Redis 错误 > 2% | 2m |
| `LowRedisCacheHitRate` | warning | 缓存命中率 < 50% | 5m |

#### 限流器 (rate_limiter)

| 告警 | 严重度 | 条件 | for |
|------|--------|------|-----|
| `RateLimiterFailingOpen` | warning | Redis 错误 > 5次/min | 2m |
| `HighAuthRateLimitRejectionRate` | warning | auth 接口拒绝 > 50% | 2m |

#### 操作日志写入 (log_writer)

| 告警 | 严重度 | 条件 | for |
|------|--------|------|-----|
| `LogWriterChannelFull` | warning | channel 满触发降级 | 1m |
| `LogWriterBatchFailure` | **critical** | 批量写入失败（数据丢失） | 1m |
| `LogWriterWorkerPanic` | **critical** | worker goroutine panic | 1m |

#### Tracing (tracing)

| 告警 | 严重度 | 条件 | for |
|------|--------|------|-----|
| `TracingDisabled` | warning | `tracing_enabled == 0` | 5m |

### Loki 告警规则

| 告警 | 严重度 | 条件 | for |
|------|--------|------|-----|
| `HighErrorLogRate` | warning | ERROR 日志 > 2条/s | 3m |
| `VeryHighErrorLogRate` | **critical** | ERROR 日志 > 20条/s | 2m |

### SRE 平台接入

支持 Push 和 Pull 两种模式，可同时使用。

#### Push 模式（Alertmanager Webhook → SRE 平台）

Alertmanager 检测到告警后，主动推送 HTTP POST 到 SRE 平台的 webhook 端点。

**SRE 平台需要提供**：一个接收 POST 的 HTTP 端点。

**配置方式**：

```bash
# Docker Compose: 设置环境变量后启动
ALERTMANAGER_WEBHOOK_URL=https://sre-platform.example.com/api/alerts make up

# Kubernetes: 注入环境变量
kubectl set env deployment/alertmanager   ALERTMANAGER_WEBHOOK_URL=https://sre-platform.example.com/api/alerts   -n task-platform
```

**Webhook Payload 格式**（标准 Alertmanager JSON V4）：

```json
{
  "version": "4",
  "groupKey": "{}:{alertname=\"ServiceDown\",service=\"api-gateway\"}",
  "status": "firing",
  "receiver": "webhook-notifications",
  "groupLabels": {
    "alertname": "ServiceDown",
    "service": "api-gateway"
  },
  "commonLabels": {
    "alertname": "ServiceDown",
    "severity": "critical",
    "job": "api-gateway"
  },
  "commonAnnotations": {
    "summary": "Service api-gateway is down",
    "description": "api-gateway has been unreachable for more than 1 minute."
  },
  "externalURL": "http://alertmanager:9093",
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "ServiceDown",
        "severity": "critical",
        "job": "api-gateway"
      },
      "annotations": {
        "summary": "Service api-gateway is down",
        "description": "api-gateway has been unreachable for more than 1 minute."
      },
      "startsAt": "2026-06-11T08:30:00Z",
      "endsAt": "0001-01-01T00:00:00Z",
      "generatorURL": "http://prometheus:9090/graph?g0.expr=..."
    }
  ]
}
```

SRE 平台建议按以下字段做分发：

| 字段 | 用途 |
|------|------|
| `status` | `firing` = 告警触发，`resolved` = 告警恢复 |
| `commonLabels.severity` | `critical` / `warning`，决定路由优先级 |
| `commonLabels.job` | 服务名（`api-gateway` / `user-service-admin` / `task-service-admin`），用于责任分发 |
| `commonAnnotations.summary` | 告警一句话摘要 |
| `commonAnnotations.description` | 告警详细描述 |

#### Pull 模式（SRE 平台 → Alertmanager API）

SRE 平台定时拉取 Alertmanager 的 HTTP API 获取当前告警状态。无需额外配置，API 默认开启。

**Alertmanager API**（端口 9093）：

```bash
# 获取所有当前触发的告警
curl http://localhost:9093/api/v2/alerts

# 按严重度过滤
curl "http://localhost:9093/api/v2/alerts?filter=severity=critical"

# 获取已恢复的告警（最近 72 小时内）
curl "http://localhost:9093/api/v2/alerts?resolved=true"

# 获取已静默的告警
curl "http://localhost:9093/api/v2/alerts?silenced=true"
```

**Prometheus API**（端口 9090，可直接查询告警规则状态）：

```bash
# 获取所有告警规则及其当前状态
curl http://localhost:9090/api/v1/rules?type=alert

# 获取当前触发中的告警
curl http://localhost:9090/api/v1/alerts
```

**轮询建议**：

- 间隔 30s~60s 拉取一次
- 根据 `fingerprint` 字段做去重和状态变更追踪
- critical 告警可从 Prometheus 直接读取获得更低延迟

#### 两种模式对比

| 维度 | Push 模式 | Pull 模式 |
|------|-----------|-----------|
| 实时性 | 高（触发即推送） | 取决于轮询间隔 |
| 网络要求 | SRE 平台需对外暴露端点 | Alertmanager 只需内网可达 |
| 去重 | Alertmanager 分组后一并推送 | SRE 平台自行按 fingerprint 去重 |
| 复杂度 | SRE 平台实现 HTTP server | SRE 平台实现定时任务 |
| 当前状态 | `ALERTMANAGER_WEBHOOK_URL` 配置即可 | API 默认开启 |

默认情况下（未配置 webhook URL），告警仍会出现在 Alertmanager 日志中：

```bash
# Docker Compose
docker logs task-platform-alertmanager

# Kubernetes
kubectl logs -l app=alertmanager -n task-platform
```

### 静默告警

在 Alertmanager UI (`http://127.0.0.1:9093/#/silences`) 中可以创建静默规则，按告警名称、服务或标签过滤。适用于计划内维护期间抑制告警。

### 告警分组与通知策略

- 告警按 `alertname` + `service` 分组，同一组内的告警合并为一条通知
- 首条通知在首次触发后等待 `group_wait: 10s`（收集同组内其他告警一起发送）
- 同组告警最小发送间隔 `group_interval: 30s`
- **critical** 告警跳过 group_wait，立即发送
- **critical** 告警 1h 重复一次，warning 告警 4h 重复一次
- 告警恢复后发送 `resolved` 通知

## 常见排障入口

### 接口返回 401

检查：

- `Authorization` header 是否存在且格式为 `Bearer <token>`。
- token 是否过期。
- 是否已调用 logout，导致 `jti` 进入 Redis 黑名单。
- Gateway 的 `JWT_SECRET` 是否与 user-service 签发 token 使用的 secret 一致。

对应告警：`HighHTTP4xxRate`（401 占比升高时触发）

### 接口返回 429

检查：

- 是否命中 auth/IP/user 级限流。
- Grafana 中 `gateway_rate_limit_rejected_total{scope}` 是否突增，同时结合 `gateway_rate_limit_allowed_total{scope}` 判断整体流量。
- 压测是否超过 `.env` 中的 `RATELIMIT_*` 配置。

对应告警：`HighAuthRateLimitRejectionRate`（auth 接口拒绝率 > 50%）

### Gateway 返回 503 或 504

检查：

- `USER_SERVICE_ADDR`、`TASK_SERVICE_ADDR` 是否正确。
- user-service、task-service 是否 ready。
- 内部 gRPC 调用是否超时。
- 服务日志中是否有 `UNAVAILABLE` 或 deadline exceeded。

对应告警：`ServiceDown`（服务不可达）、`HighgRPCServerErrorRate`、`HighgRPCClientErrorRate`

### Trace 不出现

检查：

- Jaeger 是否启动并监听 `4317`。
- `OTEL_EXPORTER_OTLP_ENDPOINT` 是否指向正确地址。
- `/metrics` 中 `tracing_enabled` 是否为 `1`。

对应告警：`TracingDisabled`（tracing 关闭超过 5 分钟时触发）

### 数据库返回大量错误

检查：

- PostgreSQL 是否运行且健康（`pg_isready`）。
- 连接池是否耗尽（`DB_MAX_OPEN_CONNS` 配置）。
- 慢查询是否拖累连接池（p99 延迟）。

对应告警：`HighDatabaseErrorRate`、`DatabaseConnectionPoolHigh`、`HighDatabaseLatency`

### 操作日志缺失

检查：

- task-service 日志中是否有操作日志批量写入失败。
- `operation_logs` 表是否有对应 `project_id` 或 `task_id` 数据。
- 是否出现 log writer channel 满降级告警，或 `task_platform_log_writer_batch_failure_total`、`task_platform_log_writer_worker_panic_total` 增长。

对应告警：`LogWriterChannelFull`、`LogWriterBatchFailure`（critical）、`LogWriterWorkerPanic`（critical）

### Loki 日志不出现

检查：

- Loki 和 Promtail 容器是否健康运行。
- Promtail 是否有权限读取 Docker socket（`/var/run/docker.sock`）。
- Grafana Loki 数据源是否配置正确（`http://loki:3100`）。
- 查询时 service label 是否匹配（使用 `{service="api-gateway"}` 测试）。

对应告警：`HighErrorLogRate`、`VeryHighErrorLogRate`（ERROR 日志量异常时触发）

### 告警不触发

检查：

- Alertmanager 是否运行且健康（`http://127.0.0.1:9093/-/healthy`）。
- Prometheus 是否成功加载告警规则（`http://127.0.0.1:9090/alerts` 页面）。
- Prometheus 是否与 Alertmanager 联通（`http://127.0.0.1:9090/status#runtime` 中的 alertmanagers 列表）。
- `ALERTMANAGER_WEBHOOK_URL` 是否配置正确。
- `for` 持续时间是否满足（短暂尖峰会被 `for` 过滤）。
