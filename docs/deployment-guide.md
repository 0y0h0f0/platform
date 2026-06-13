# 部署指南

项目支持两种部署方式：Docker Compose（本地开发）和 Kubernetes（生产级）。Docker Compose 适合开发、演示和面试验收；Kubernetes 提供容器编排、自动扩缩和高可用。

## Docker Compose 部署

### 部署组件

| 组件 | 镜像/来源 | 端口 | 说明 |
|------|-----------|------|------|
| PostgreSQL | `postgres:16` | host `5433` -> container `5432` | 主数据库 |
| Redis | `redis:7` | host `6380` -> container `6379` | 缓存、限流、幂等、黑名单 |
| Prometheus | `prom/prometheus:v2.54.0` | `9090` | 指标采集 |
| Jaeger | `jaegertracing/all-in-one:1.69.0` | `16686`、`4317`、`4318` | Trace UI 和 OTLP collector |
| Loki | `grafana/loki:3.5.0` | `3100` | 日志聚合 |
| Promtail | `grafana/promtail:3.5.0` | `9080` | 日志采集 agent |
| Grafana | `grafana/grafana:11.1.0` | `3000` | 仪表盘 |
| api-gateway | Go 进程 | `8080` | HTTP API |
| user-service | Go 进程 | `9091`、`8081` | gRPC + admin HTTP |
| task-service | Go 进程 | `9092`、`8082` | gRPC + admin HTTP |

### 本地部署步骤

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

### 启动顺序

推荐顺序：

1. PostgreSQL 和 Redis
2. 数据库迁移
3. user-service
4. task-service
5. api-gateway
6. 前端或 Postman 调试

`task-service` 会连接 `user-service`，`api-gateway` 会连接两个 gRPC 服务，因此服务进程顺序不要反过来。

### 健康检查

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

### 数据持久化

Docker Compose 使用命名 volume：

| Volume | 用途 |
|--------|------|
| `postgres-data` | PostgreSQL 数据 |
| `redis-data` | Redis AOF 数据 |
| `prometheus-data` | Prometheus TSDB |
| `grafana-data` | Grafana 数据 |
| `loki-data` | Loki 日志和索引 |

停止服务：

```bash
make down
```

该命令不会删除 volume。如需清空数据，应显式执行 Docker volume 删除命令，并确认不会误删有价值数据。

### 日志聚合 (Loki)

配置文件：

| 文件 | 说明 |
|------|------|
| `deploy/loki-config.yaml` | Loki 服务端配置（30 天保留，filesystem 存储） |
| `deploy/promtail-config.yaml` | Promtail 采集配置（Docker 容器自动发现） |

Promtail 通过 Docker socket 自动发现容器并采集日志，无需修改应用代码。在 Grafana → Explore → Loki 数据源中即可查询日志。

### Grafana

默认地址：

```text
http://127.0.0.1:3000
```

默认账号密码由环境变量控制：

| 变量 | 默认 |
|------|------|
| `GRAFANA_USER` | `admin` |
| `GRAFANA_PASSWORD` | `admin` |

仪表盘和数据源通过 `deploy/grafana/` 自动 provision。预置数据源：Prometheus、Jaeger、Loki。

### Jaeger

默认 UI：

```text
http://127.0.0.1:16686
```

服务默认向 `127.0.0.1:4317` 发送 OTLP gRPC trace。Docker Compose 中 Jaeger 暴露了 `4317` 和 `4318`。

---

## Kubernetes 部署

Kubernetes 部署清单位于 `deploy/k8s/`，使用 Kustomize 管理多环境配置。

### 前置条件

- Kubernetes 集群（v1.28+）
- kubectl 已配置
- 镜像已构建并推送到 registry

### 构建镜像

```bash
# 构建三个服务镜像
docker build --build-arg SERVICE=api-gateway -t task-platform/api-gateway:latest .
docker build --build-arg SERVICE=user-service -t task-platform/user-service:latest .
docker build --build-arg SERVICE=task-service -t task-platform/task-service:latest .
```

### 部署

```bash
# 开发环境（单副本，较低资源限制）
kubectl apply -k deploy/k8s/overlays/dev/

# 生产环境（多副本，更高资源限制）
kubectl apply -k deploy/k8s/overlays/prod/
```

### 部署组件（K8s）

| 资源 | 类型 | 副本 | 说明 |
|------|------|------|------|
| api-gateway | Deployment + Service (LoadBalancer) + HPA | 2-3 | HTTP 入口，自动扩缩 |
| user-service | Deployment + Service (ClusterIP) | 2-3 | gRPC 内部服务 |
| task-service | Deployment + Service (ClusterIP) | 2-3 | gRPC 内部服务 |
| postgres | StatefulSet + Service (headless) | 1 | 数据库（生产建议 RDS/Cloud SQL） |
| redis | StatefulSet + Service (headless) | 1 | 缓存（生产建议托管 Redis） |
| prometheus | Deployment + Service + PVC | 1 | 指标采集，10Gi |
| jaeger | Deployment + Service | 1 | all-in-one 追踪 |
| grafana | Deployment + Service + PVC + Ingress | 1 | 仪表盘，2Gi |
| loki | StatefulSet + Service (headless) + PVC | 1 | 日志聚合，10Gi |
| promtail | DaemonSet + ClusterRole + ClusterRoleBinding | 每节点 1 | 日志采集 |

### K8s 目录结构

```
deploy/k8s/
├── kustomization.yaml            # 顶层入口
├── namespace.yaml                # task-platform namespace
├── secrets.yaml                  # 敏感信息
├── api-gateway/                  # Deployment + Service + HPA
├── user-service/                 # Deployment + Service
├── task-service/                 # Deployment + Service
├── postgres/                     # StatefulSet + Service + PVC
├── redis/                        # StatefulSet + Service + PVC
├── prometheus/                   # ConfigMap + Deployment + Service + PVC
├── jaeger/                       # Deployment + Service
├── grafana/                      # ConfigMap×3 + Deployment + Service + PVC + Ingress
├── loki/                         # ConfigMap + StatefulSet + Service + PVC
├── promtail/                     # ConfigMap + DaemonSet + RBAC
└── overlays/
    ├── dev/kustomization.yaml    # 开发环境覆盖
    └── prod/kustomization.yaml   # 生产环境覆盖
```

### K8s 验证

```bash
# 查看所有 Pod 状态
kubectl get pods -n task-platform -w

# 验证 API Gateway
kubectl port-forward svc/api-gateway 8080:80 -n task-platform
curl http://localhost:8080/healthz

# 验证 Grafana
kubectl port-forward svc/grafana 3000:3000 -n task-platform
# 打开 http://localhost:3000
```

---

## 生产化注意事项

Docker Compose 面向本地开发。Kubernetes 部署已提供以下生产特性：

- 容器化：多阶段 Dockerfile（`golang:1.26-alpine` → `distroless`），`CGO_ENABLED=0` 静态构建
- 健康检查：所有 Deployment/StatefulSet 配置 liveness + readiness probes
- 资源限制：所有容器配置 requests + limits
- 自动扩缩：api-gateway HPA（CPU 70%，2-10 副本）
- 配置分离：ConfigMap 管理非敏感配置，Secret 管理 JWT、密码等
- 日志采集：Promtail DaemonSet + RBAC 自动收集所有 Pod 日志到 Loki

仍需补充的生产化事项：

- PostgreSQL、Redis 使用托管服务或独立高可用部署。
- secret 通过外部 secret manager（如 Vault、Sealed Secrets）注入。
- Redis 开启密码、网络隔离和持久化策略评估。
- Gateway 放在反向代理或负载均衡后，并正确处理真实客户端 IP。
- TLS 在入口层终止（可通过 cert-manager + Let's Encrypt），必要时内部链路也启用 TLS。
- 为数据库迁移建立发布前检查和回滚流程。
- Prometheus、Grafana、Jaeger、Loki 设置持久化、鉴权和保留周期。
- 根据容量目标调整 `DB_MAX_OPEN_CONNS`、`DB_MAX_IDLE_CONNS`、Redis 连接池参数、Redis 限流参数、`GRPC_CLIENT_TIMEOUT_SECONDS`、`GRPC_SERVER_TIMEOUT_SECONDS`、`LOG_WRITER_WORKERS` 和服务副本数。

## 回滚建议

- 代码回滚：先回滚 gateway，再回滚 task-service/user-service，避免新 gateway 调用旧服务不存在的 RPC。
- 数据库回滚：只对可安全回滚的 schema 变更执行 `.down.sql`；涉及数据迁移时优先使用向前兼容策略。
- 配置回滚：确保 `JWT_SECRET` 与 `INTERNAL_TOKEN` 的变更与服务重启顺序一致，否则会出现 token 验签或内部 RPC 认证失败。
- Kubernetes 回滚：`kubectl rollout undo deployment/<name> -n task-platform`
