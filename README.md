# 团队任务协作平台

面向校招面试的 Go 微服务后端项目 —— 完整的团队任务协作平台。

**状态：** 全部阶段完成。服务编译通过，测试通过（80%+ 覆盖率），Prometheus + Jaeger + Loki + Grafana 可观测性体系完备，支持 Docker Compose 与 Kubernetes 部署，已验证 1000 QPS。

## 架构

```
             +-----------------------+
             |   Web / Postman / UI  |
             +-----------+-----------+
                         |
                    HTTP / JSON
                         |
                         v
             +-----------------------+
             |      API 网关          |
             |         Gin           |
             | 鉴权 / 绑定 / 日志     |
             | request_id / 恢复     |
             +-----------+-----------+
                         |
                     gRPC 客户端
               +---------+---------+
               |                   |
               v                   v
      +----------------+   +----------------+
      |   用户服务      |   |   任务服务      |
      |      gRPC      |   |      gRPC      |
      +--------+-------+   +--------+-------+
               |                    |
               |                    |
               v                    v
        +-------------+      +-------------+
        | PostgreSQL  |      | PostgreSQL  |
        | user 模式   |      | task 模式   |
        +------+------+      +------+------+
               \                    /
                \                  /
                 +--------+-------+
                          |
                        Redis
```

**中间件链：** `Recovery → MaxBodySize(1MB) → SecurityHeaders → RequestID → HTTPTrace → HTTPMetrics → AccessLog → CORS → RateLimit(IP) → Auth(JWT) → RateLimit(user) → Handler`

**服务分层：** `handler(HTTP) → service(gRPC 实现) → biz(领域逻辑) → data(仓库/GORM)`

**可观测性：** Prometheus（指标）+ Jaeger（链路追踪）+ Loki（日志聚合）→ Grafana（统一看板）

## 功能概览

### 用户认证
- 注册（用户名/邮箱/密码校验，弱密码黑名单）
- 登录（用户名或邮箱 + 密码，JWT HS256，2 小时过期）
- 登出（Redis 令牌黑名单）
- 当前用户查询、批量用户查询（Redis 缓存，TTL 5 分钟）

### 项目管理
- CRUD，乐观锁并发控制（`version` 列）
- 归档 / 取消归档（归档后项目全员只读）
- 所有权转让（事务原子更新 `projects.owner_id` 和 `project_members.role`）
- 成员管理：添加、移除、修改角色、退出、列表
- 三级角色权限：Owner / Admin / Member

### 任务管理
- CRUD，乐观锁并发控制
- 游标分页（base64url 编码，内含 `filter_hash` 防跨筛选器不一致）
- 任务指派（校验用户存在性与项目成员身份）
- 状态机：`todo ↔ doing ↔ done`，`todo → cancelled`，`cancelled → todo`
- 状态与指派人通过专用端点（`AssignTask`、`ChangeTaskStatus`）更新，`UpdateTask` 不接受这两个字段

### 评论与操作日志
- 任务评论的创建 / 删除 / 列表
- 项目与任务维度的操作日志查询
- 异步日志写入器：缓冲 channel（容量 1024），worker 数可配置（默认 1，最大 16），批量写入（batch 64，flush 100ms），channel 满时优雅降级

### 工程基础设施
- **统一错误码：** 12 个错误码，gRPC status 映射为 HTTP 统一信封 `{ code, message, request_id, data? }`
- **结构化日志：** Zap JSON 格式，携带 `request_id` 与 `method`/`latency` 字段；两个业务服务均包含 gRPC 一元拦截器日志
- **日志聚合：** Loki + Promtail 实现集中式日志查询与存储（应用代码零改动）
- **Redis 缓存：** Cache-aside 模式缓存用户信息与项目详情，写入时主动失效（TTL 5 分钟），singleflight 合并并发缺失请求
- **限流：** Redis 令牌桶（Lua 脚本），认证接口 5/s（突发 10），常规接口 60/s（突发 100），用户级 100/s（突发 200）
- **幂等性：** 所有写端点支持 `Idempotency-Key` 请求头，Redis `SETNX` 实现，24h TTL；请求失败时释放 key 允许重试；处理中的重复请求返回 409/`ABORTED`
- **超时控制：** HTTP 读写/空闲超时可配置，内部 gRPC 默认 deadline 可配置
- **多环境配置：** Viper YAML 配置 + 环境变量覆盖，local/dev/docker 三套环境
- **软删除：** `users`、`projects`、`tasks` 使用 GORM `DeletedAt`；`comments` 和 `members` 为物理删除
- **集成测试：** 70+ 测试用例，通过 testcontainers-go 对接真实 PostgreSQL 16 与 Redis 7

## 快速开始

### 环境要求
- Go 1.26+
- Docker（用于 PostgreSQL、Redis 以及 testcontainers）

### 初始化

```bash
cp .env.example .env
make up          # 启动 PostgreSQL + Redis + Prometheus + Jaeger + Loki + Grafana
make migrate     # 执行数据库迁移
```

### 启动服务

```bash
make run/api-gateway     # HTTP :8080
make run/user-service    # gRPC :9091，管理 HTTP :8081
make run/task-service    # gRPC :9092，管理 HTTP :8082
```

或在三个终端中分别启动。

### 可观测性

| 组件 | 地址 | 用途 |
|-----------|---------|---------|
| Grafana | http://127.0.0.1:3000 | 统一看板（Prometheus + Jaeger + Loki） |
| Jaeger | http://127.0.0.1:16686 | 分布式链路追踪 |
| Prometheus | http://127.0.0.1:9090 | 指标查询 |
| Loki | http://127.0.0.1:3100 | 日志聚合（通过 Grafana 查询） |

### Kubernetes 部署

**前置条件：** Kubernetes v1.28+、kubectl 已配置、Docker（如需本地构建）。

**构建镜像：**

```bash
docker build --build-arg SERVICE=api-gateway -t task-platform/api-gateway:latest .
docker build --build-arg SERVICE=user-service -t task-platform/user-service:latest .
docker build --build-arg SERVICE=task-service -t task-platform/task-service:latest .
docker build -t task-platform/web:latest web/
```

**一键部署（推荐）：**

`deploy/k8s/deploy-all.sh` 提供完整部署流程：

```bash
./deploy/k8s/deploy-all.sh dev              # 部署开发环境（单副本）
./deploy/k8s/deploy-all.sh dev --build      # 构建镜像 + 部署
./deploy/k8s/deploy-all.sh dev --skip-infra # 使用外部 PG/Redis，仅部署应用与可观测性
./deploy/k8s/deploy-all.sh dev --dry-run    # 预览生成的 YAML，不实际部署
./deploy/k8s/deploy-all.sh prod             # 部署生产环境（3 副本，更高资源限制）
```

**直接使用 kubectl + Kustomize：**

```bash
kubectl apply -k deploy/k8s/overlays/dev/     # 开发环境
kubectl apply -k deploy/k8s/overlays/prod/    # 生产环境
kubectl get pods -n task-platform -w          # 等待 Pod 就绪
```

**访问服务：**

```bash
kubectl port-forward svc/web 8080:80 -n task-platform        # 前端 SPA → http://localhost:8080
kubectl port-forward svc/grafana 3000:3000 -n task-platform   # Grafana → http://localhost:3000
kubectl port-forward svc/jaeger 16686:16686 -n task-platform  # Jaeger → http://localhost:16686
```

如果使用 minikube：`minikube service web -n task-platform`

**dev vs prod overlay 差异：**

| 项目 | dev | prod |
|------|-----|------|
| 副本数 | 1 | 3 |
| imagePullPolicy | Never（本地镜像） | 默认 |
| CPU requests | 默认 | 200m |
| CPU limits | 默认 | 1000m |

**清理：** `kubectl delete -k deploy/k8s/overlays/dev/` 或 `kubectl delete namespace task-platform`

K8s 清单位于 `deploy/k8s/`，部署了 api-gateway、user-service、task-service、web、PostgreSQL、Redis、Prometheus、Jaeger、Grafana、Loki、Promtail 全套组件。详见 `docs/deployment-guide.md`。

### 前端

```bash
cd web
npm install
npm run dev       # 代理 /api 到 Go 网关
npm run dev:mock  # 独立 MSW 模式，无需后端
npm run e2e       # Playwright E2E 测试（MSW 模式）
```

详见 `web/README.md`，包含前端质量检查与构建产物分析。

### 测试

```bash
make test                              # 所有单元测试
go test -tags=integration ./test/integration/ -v   # 集成测试（需要 Docker）
make coverage                          # 覆盖率报告（≥80% 阈值）
make lint                              # golangci-lint
```

## API 参考

所有端点返回统一信封：`{ "code": "OK", "message": "...", "request_id": "...", "data": {...} }`

### 认证 — `/api/v1/auth`

| 方法 | 路径 | 认证 | 说明 |
|--------|------|------|-------------|
| POST | `/auth/register` | 无 | 注册（username, email, password） |
| POST | `/auth/login` | 无 | 登录（account, password） |
| POST | `/auth/logout` | Bearer | 登出（将当前 token 加入黑名单） |

### 用户 — `/api/v1/users`

| 方法 | 路径 | 认证 | 说明 |
|--------|------|------|-------------|
| GET | `/users/me` | Bearer | 获取当前用户信息 |

### 项目 — `/api/v1/projects`

| 方法 | 路径 | 认证 | 说明 |
|--------|------|------|-------------|
| POST | `/projects` | Bearer | 创建项目 |
| GET | `/projects` | Bearer | 我的项目列表（偏移分页，`?limit=20&offset=0`） |
| GET | `/projects/:id` | Bearer | 获取项目详情 |
| PUT | `/projects/:id` | Bearer | 更新项目（需要 `version`） |
| POST | `/projects/:id/archive` | Bearer | 归档项目（仅 Owner） |
| POST | `/projects/:id/unarchive` | Bearer | 取消归档（仅 Owner） |
| POST | `/projects/:id/transfer` | Bearer | 转让所有权（`target_user_id`） |
| POST | `/projects/:id/members` | Bearer | 添加成员（`user_id`、`role`） |
| GET | `/projects/:id/members` | Bearer | 成员列表 |
| PUT | `/projects/:id/members/:userId` | Bearer | 修改成员角色 |
| DELETE | `/projects/:id/members/:userId` | Bearer | 移除成员 |
| POST | `/projects/:id/members/me/leave` | Bearer | 退出项目 |
| GET | `/projects/:id/operation-logs` | Bearer | 项目操作日志 |

### 任务 — `/api/v1/tasks`

| 方法 | 路径 | 认证 | 说明 |
|--------|------|------|-------------|
| POST | `/tasks` | Bearer | 创建任务（`project_id`、`title`、`content`） |
| GET | `/tasks` | Bearer | 任务列表（游标分页，`?project_id=&status=&assignee_id=&keyword=&cursor=&limit=`） |
| GET | `/tasks/:id` | Bearer | 获取任务详情 |
| PUT | `/tasks/:id` | Bearer | 更新任务（需要 `version`；不接受 status/assignee） |
| DELETE | `/tasks/:id` | Bearer | 删除任务 |
| POST | `/tasks/:id/assign` | Bearer | 指派任务（`assignee_id`） |
| POST | `/tasks/:id/status` | Bearer | 变更状态（`status`、`version`） |
| POST | `/tasks/:id/comments` | Bearer | 创建评论（`content`） |
| GET | `/tasks/:id/comments` | Bearer | 评论列表（`?limit=`） |
| DELETE | `/tasks/:id/comments/:commentId` | Bearer | 删除评论 |
| GET | `/tasks/:id/operation-logs` | Bearer | 任务操作日志 |

## 文档

- **[文档索引](docs/README.md)** — 完整的中文文档地图：架构、API、配置、数据库、开发、测试、部署、可观测性与前端。
- **[Go 学习路线](study.md)** — 面向 Go 初学者，按项目代码学习 Go 语法、分层架构、并发、测试与工程化实践。
- **[项目介绍](docs/project-introduction.md)** — 完整的项目定位、架构设计、功能清单、技术栈与亮点总结。
- **[Postman 集合](docs/postman/task-platform.postman_collection.json)** — 导入 Postman 即可使用，注册/登录自动设置 token，所有端点附带示例请求体与自动同步的 version 变量。
- **[面试讲解稿](docs/interview-script.md)** — 架构决策理由、核心亮点、挑战与解决方案、常见追问。

### 分页

- **项目：** 偏移分页（`limit` 默认 20，最大 50）
- **任务：** 游标分页（base64url 编码的 JSON：`{created_at, id, filter_hash}`；服务端校验 filter_hash，不匹配则拒绝）

### 幂等性

所有写端点（`POST`、`PUT`、`DELETE`）均支持 `Idempotency-Key` 请求头。24h 内相同 key 的重复请求：首个请求完成后返回缓存响应（200 OK）；首个请求处理中到达的重复请求返回 409/`ABORTED`。请求失败时自动释放 key，允许客户端重试。

## 配置

多环境配置文件位于 `configs/` 目录：

| 环境 | 目录 | 用途 |
|-----|-----------|---------|
| local | `configs/local/` | 本地开发 |
| dev | `configs/dev/` | CI / 预发布 |
| docker | `configs/docker/` | Docker Compose / Kubernetes 部署 |

每个服务有独立的 YAML 文件，通过 Viper 加载并支持环境变量覆盖。

### 关键环境变量

| 变量 | 服务 | 说明 |
|----------|---------|-------------|
| `JWT_SECRET` | gateway, user | JWT 签名密钥（≥32 字符） |
| `INTERNAL_TOKEN` | 全部 | 内部 RPC 认证令牌（≥16 字符） |
| `POSTGRES_DSN` | user, task | PostgreSQL 连接字符串 |
| `REDIS_ADDR` | 全部 | Redis 地址（`host:port`） |
| `REDIS_PASSWORD` | 全部 | Redis 密码（可选） |
| `REDIS_POOL_SIZE` | 全部 | Redis 连接池大小（默认 100） |
| `GRPC_CLIENT_TIMEOUT_SECONDS` | gateway, task | 内部 gRPC 客户端默认 deadline（默认 2s） |
| `GRPC_SERVER_TIMEOUT_SECONDS` | user, task | gRPC 服务端默认 handler deadline（默认 3s） |
| `LOG_WRITER_WORKERS` | task | 异步操作日志 worker 数（默认 1，最大 16） |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | 全部 | OTLP 链路追踪端点（默认 `127.0.0.1:4317`） |

默认端口：API 网关 `:8080`，用户服务 gRPC `:9091`，任务服务 gRPC `:9092`，PostgreSQL `:5433`，Redis `:6380`。

## 关键架构决策

- **单数据库双模式：** 单个 PostgreSQL 实例，`user_svc` 与 `task_svc` 两个逻辑模式
- **共享 Redis：** 单个 Redis 实例承载缓存、限流与幂等键
- **网关集中鉴权：** JWT 仅在 API 网关校验，内部服务信任注入的 `x-user-id`/`x-username` 元数据头，不持有 JWT 密钥
- **内部 RPC 认证：** `x-internal-token` 静态共享密钥，每个 RPC 由服务端拦截器校验
- **请求生命周期收敛：** HTTP 服务器配置显式读写/空闲超时；内部 gRPC 调用均有默认 deadline
- **乐观锁：** `projects.version` 与 `tasks.version` 防止并发写入冲突
- **软删除 + 部分唯一索引：** `users` 与 `projects` 使用 GORM 软删除，配合 `WHERE deleted_at IS NULL` 部分唯一索引
- **Owner 一致性：** `projects.owner_id` 是唯一数据源，`project_members` 冗余维护 `role=owner` 行，所有权转让时在同一事务中原子更新
- **非成员即 NOT_FOUND：** 非成员访问项目资源返回 404（而非 403），防止泄露资源存在性
- **异步操作日志：** 缓冲 channel 写入器确保日志 I/O 不阻塞业务响应，背压时优雅降级
- **幂等性在 handler 而非中间件：** 在参数绑定/校验之后检查，防止非法请求消耗幂等键
- **日志聚合（Loki）：** Zap JSON → stdout → Promtail → Loki，应用代码零改动
- **多平台部署：** Docker Compose 本地开发，Kubernetes（Kustomize）生产部署

## 技术栈

Go 1.26+ · Gin · gRPC + protobuf（buf）· PostgreSQL 16 · Redis 7 · GORM · pgx/v5 · JWT HS256 · bcrypt · Zap · Viper · Prometheus · Jaeger · Loki · Promtail · Grafana · Docker Compose · Kubernetes（Kustomize）· testcontainers-go · testify · golangci-lint · GitHub Actions CI

## 项目结构

```
├── api/proto/              # Protobuf 定义（user/v1, task/v1）
├── cmd/                    # 服务入口（api-gateway, user-service, task-service）
├── configs/                # 多环境 YAML 配置（local, dev, docker）
├── gen/go/                 # 生成的 protobuf Go 代码
├── internal/
│   ├── gateway/            # API 网关（handler, middleware, rpc, server）
│   ├── user/               # 用户服务（biz, data, service, server）
│   └── task/               # 任务服务（biz, data, service, server）
├── migrations/             # 按 schema 组织的 SQL 迁移脚本
├── pkg/                    # 共享包（xerr, xgrpc, xhttp, xjwt, xlog, xpgsql, xredis, xratelimit 等）
├── scripts/                # 构建/测试/迁移脚本
├── test/integration/       # 集成测试（通过 testcontainers 对接真实 PG + Redis）
├── deploy/                 # Docker Compose、K8s 清单、Prometheus、Grafana、Loki 配置
│   ├── docker-compose.yml
│   ├── prometheus.yml
│   ├── loki-config.yaml
│   ├── promtail-config.yaml
│   ├── grafana/
│   └── k8s/                # Kubernetes 清单（Kustomize）
├── Dockerfile              # 多阶段构建（所有服务共用）
├── docs/                   # 项目文档
├── web/                    # React 前端
└── Makefile                # proto, test, lint, coverage, migrate, up/down
```
