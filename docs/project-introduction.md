# 团队任务协作平台 — 项目介绍

## 项目定位

**团队任务协作平台** 是一个面向校招面试的 Go 微服务后端项目，完整演示了从用户认证、项目管理、任务协作到可观测性运维的分布式系统开发能力。

项目规模控制在 **1 个 API Gateway + 2 个核心微服务**，采用 Gin + gRPC + PostgreSQL + Redis 技术栈，配有 Prometheus + Grafana + Jaeger 可观测性体系，单机验证 1000 QPS，测试覆盖率 80%+。

---

## 业务功能

### 用户系统
- 注册（用户名/邮箱/密码校验，弱密码黑名单）
- 登录（用户名或邮箱 + 密码，JWT HS256，2 小时过期）
- 登出（Redis 令牌黑名单）
- 用户信息查询与批量查询（Redis 缓存，TTL 5 分钟）

### 项目管理
- 项目 CRUD，支持乐观锁并发控制
- 归档/取消归档（归档后全员只读）
- 所有权转让（事务原子更新 `projects.owner_id` 和 `project_members.role`）
- 成员管理：添加、移除、修改角色、退出、列表
- 三级角色权限：Owner / Admin / Member

### 任务管理
- 任务 CRUD，乐观锁并发控制
- 游标分页（base64url 编码，防跨筛选器不一致的 `filter_hash` 校验）
- 任务指派（校验用户存在性与项目成员身份）
- 状态机流转：`todo ↔ doing ↔ done`，`todo ↔ cancelled`
- 指派与状态变更通过独立端点操作，防止越权修改

### 评论与操作日志
- 任务评论创建/删除/列表
- 项目和任务维度的操作日志查询
- 异步日志写入：缓冲 channel（容量 1024），worker 数可配置（默认 1，最大 16），批量写入（batch 64，flush 100ms），优雅降级与关闭

### 工程基础设施
- **统一错误码：** 12 个错误码，gRPC status → HTTP `{ code, message, request_id, data? }` 统一信封
- **幂等性：** 所有写端点支持 `Idempotency-Key` 请求头，Redis `SETNX` 实现，24h TTL
- **限流：** 三级 Redis 令牌桶（Lua 脚本）—— 认证接口 5/s，IP 级 60/s，用户级 100/s
- **超时控制：** HTTP server 读写/空闲超时可配置，内部 gRPC client/server 默认 deadline 可配置
- **结构化日志：** Zap JSON 格式，`request_id` + `trace_id` 串联全链路
- **多环境配置：** Viper YAML 配置 + 环境变量覆盖，local/dev/docker 三套环境

---

## 架构设计

### 整体拓扑

```
Web/Postman (HTTP/JSON)
        │
        v
  api-gateway (Gin, :8080)
  鉴权 · 参数绑定 · 统一错误 · request_id
        │
  ┌─────┴────── gRPC (unary, 2s deadline) ──────┐
  │                                              │
  v                                              v
user-service (:9091)                     task-service (:9092)
Register · Login · GetUser               Project CRUD · Task CRUD
BatchGetUsers · Token签发                 Member管理 · Comment · 操作日志
  │                                              │
  └──────────┬───────────────────────────────────┘
             │
    ┌────────┴────────┐
    │                 │
    v                 v
PostgreSQL 16        Redis 7
schema user_svc      · 缓存（user info · project detail）
schema task_svc      · 限流（IP · user 令牌桶）
                     · 幂等键（SETNX）
                     · JWT 黑名单
```

### 关键架构决策

| 决策 | 说明 |
|------|------|
| 单体数据库 + 逻辑隔离 | 一个 PG 实例下 `user_svc` / `task_svc` 两个 schema，避免过早引入分布式事务 |
| Gateway 独掌 JWT 验签 | 内部服务只认网关注入的 `x-user-id` / `x-username`，不持有 JWT secret |
| 内部 RPC 认证 | 所有 gRPC 调用携带静态 `x-internal-token`，服务端 interceptor 强制校验 |
| 乐观锁代替悲观锁 | `projects.version` / `tasks.version` 字段，`WHERE version = ?` 自增 |
| 非成员返回 404 | 对非成员访问资源返回 `NOT_FOUND` 而非 `FORBIDDEN`，避免泄露资源存在性 |
| 操作日志异步化 | 缓冲 channel + 批量写入，不阻塞业务响应链路 |

### 四层分层

每个微服务内部遵循 **handler → service(gRPC) → biz(domain) → data(repository)** 分层：

| 层 | 职责 | 关键约束 |
|---|------|---------|
| handler（gateway） | HTTP 参数绑定、幂等性检查、调用 gRPC 客户端 | 不做业务逻辑 |
| service | gRPC 方法实现、参数校验、调用 biz 层 | 不持有 HTTP 依赖 |
| biz | 业务逻辑、权限判断、状态机、缓存策略 | 不依赖 gRPC/HTTP 库 |
| data | GORM 模型、Repository 接口、SQL 查询 | 不包含业务判断 |

### 中间件链路（固定顺序）

```
Recovery → MaxBodySize(1MB) → SecurityHeaders → RequestID
→ HTTPTrace → HTTPMetrics → AccessLog → CORS
→ RateLimit(IP) → Auth(JWT) → RateLimit(user) → Handler
```

---

## 数据库设计

### 逻辑模型（6 张表）

```
user_svc.users ────────┐
  id, username, email,  │   task_svc.projects ─── task_svc.project_members
  password_hash,        │     id, name, owner_id,     id, project_id, user_id,
  nickname, status      │     status, version         role, joined_at
                        │          │
                        │          ├── task_svc.tasks
                        │          │     id, project_id, title, content,
                        │          │     status, priority, assignee_id,
                        │          │     creator_id, version, extra(jsonb)
                        │          │       │
                        │          │       ├── task_svc.task_comments
                        │          │       │     id, task_id, user_id, content
                        │          │       │
                        │          └───────┴── task_svc.operation_logs
                        │                       id, project_id, task_id,
                        │                       operator_id, action, detail(jsonb)
                        │
                        └── (跨服务校验: 加成员/指派任务时校验用户存在)
```

### 关键约束与索引

- **软删除：** `users`、`projects`、`tasks` 使用 GORM `DeletedAt`，搭配 `WHERE deleted_at IS NULL` 的部分唯一索引
- **物理删除：** `task_comments`、`project_members`
- **乐观锁：** `projects.version`、`tasks.version`（bigint，默认 0）
- **Owner 唯一：** `project_members(project_id) WHERE role = 0` 部分唯一索引，确保每个项目至多一个 owner
- **游标分页索引：** `tasks` 表按 `(project_id, status, assignee_id)` 组合维度建立专用索引
- **JSONB 字段：** `tasks.extra`（限 labels/checklist/attachments 键）、`operation_logs.detail`

---

## 权限矩阵

| 操作 | Owner | Admin | Member |
|------|:-----:|:-----:|:------:|
| 归档/取消归档项目 | ✓ | ✗ | ✗ |
| 转让所有权 | ✓ | ✗ | ✗ |
| 编辑项目信息 | ✓ | ✗ | ✗ |
| 邀请成员（任意角色） | ✓ | 仅 member | ✗ |
| 移除成员 | ✓（不能移除自己） | 仅 member | ✗ |
| 修改成员角色 | ✓ | ✗ | ✗ |
| 创建/编辑/删除任意任务 | ✓ | ✓ | 仅自己的任务（删除仅限 todo 状态） |
| 指派任务/变更状态 | ✓ | ✓ | 仅自己的任务 |
| 删除任意评论 | ✓ | ✓ | 仅自己的评论 |
| 退出项目 | ✗（必须先转让） | ✓ | ✓ |

**归档项目对所有角色只读** —— 所有写操作返回 `FAILED_PRECONDITION`。

---

## 可观测性

### 三件套

| 组件 | 用途 |
|------|------|
| **Prometheus** | 指标采集：HTTP 请求（QPS/延迟/状态码）、gRPC 调用、DB/Redis 操作、限流计数 |
| **Grafana** | 19 面板仪表盘：API 概览、服务健康、业务指标、资源使用 |
| **Jaeger** | 分布式链路追踪：`gateway → gRPC → DB` 全链路 trace 可视化 |

### 关键指标

| 指标 | 含义 |
|------|------|
| `http_requests_total{method, path, status}` | HTTP 请求计数 |
| `http_request_duration_seconds` | 请求延迟直方图 |
| `grpc_client_requests_total{grpc_method, grpc_code}` | gRPC 客户端调用计数 |
| `grpc_server_requests_total{grpc_method, grpc_code}` | gRPC 服务端调用计数 |
| `gateway_rate_limit_allowed_total{scope}` | 限流放行数（scope=auth/ip/user） |
| `gateway_rate_limit_rejected_total{scope}` | 限流拒绝数（scope=auth/ip/user） |
| `task_platform_log_writer_channel_full_total` | 操作日志 channel 满次数（降级告警） |
| `task_platform_log_writer_batch_failure_total` | 操作日志批量写入最终失败次数 |

---

## 测试体系

| 层级 | 框架 | 数量 | 说明 |
|------|------|:---:|------|
| 单元测试 | go test + testify | 全覆盖 | biz/data/service/handler 四层独立测试 |
| 集成测试 | testcontainers-go | 74 个 | 拉真实 PG16 + Redis7 容器，覆盖全部 API 场景 |
| 前端测试 | Vitest + Playwright | 全覆盖 | 组件单元测试 + E2E 端到端测试 |
| 压力测试 | k6 + vegeta | — | 验证 1000 QPS 读、50 QPS 写场景 |

**覆盖率基线：** `internal/` + `pkg/` ≥ 80%，CI 门禁强制。

---

## 部署与运维

### 本地一键启动

```bash
cp .env.example .env
make up          # Docker Compose 启动 PG + Redis + Prometheus + Jaeger + Grafana
make migrate     # 数据库迁移
make run/api-gateway   # :8080
make run/user-service  # :9091 (gRPC) :8081 (admin HTTP)
make run/task-service  # :9092 (gRPC) :8082 (admin HTTP)
```

### Docker Compose 栈（5 容器）

| 服务 | 端口 | 说明 |
|------|:---:|------|
| PostgreSQL 16 | 5433 | 数据库 |
| Redis 7 | 6380 | 缓存/限流/幂等 |
| Prometheus | 9090 | 指标采集 |
| Jaeger | 16686 | 链路追踪 |
| Grafana | 3000 | 仪表盘（预置 19 面板） |

### CI/CD

GitHub Actions 流水线：`buf lint` → `golangci-lint` → 单元测试 → 集成测试 → 覆盖率门禁（≥80%）→ 前端 typecheck/lint/test/build/E2E

---

## 技术栈总览

| 类别 | 技术 | 版本 |
|------|------|------|
| 语言 | Go | 1.26 |
| HTTP 框架 | Gin | v1.12 |
| RPC 框架 | gRPC + Protobuf (proto3) | v1.81 |
| Proto 工具链 | buf | v1.59 |
| 数据库 | PostgreSQL | 16 |
| ORM | GORM + pgx/v5 | v1.31 |
| 缓存 | Redis | 7 |
| 迁移 | golang-migrate | — |
| 配置 | Viper | v1.21 |
| 鉴权 | JWT HS256 (golang-jwt/v5) | v5.3 |
| 密码哈希 | bcrypt (cost=10) | x/crypto |
| 日志 | Zap | v1.28 |
| 指标 | Prometheus client_golang | v1.23 |
| 追踪 | OpenTelemetry + Jaeger | v1.43 |
| 容器 | Docker Compose | — |
| 测试 | testify · httptest · testcontainers-go | — |
| 静态检查 | golangci-lint | v2.12 |
| CI | GitHub Actions | — |
| 前端 | React 19 · TypeScript 6 · Ant Design 5 · Vite 7 | — |

---

## 项目亮点总结

1. **微服务但不微碎** —— 1 Gateway + 2 Services，服务边界清晰（用户域 / 任务协作域），避免过度拆分
2. **安全纵深防御** —— JWT + 令牌黑名单 + 内部 RPC 认证 + 权限矩阵 + 非成员 404 防泄露 + 弱密码黑名单
3. **并发安全** —— 乐观锁 + Owner 事务一致性 + 幂等性保证
4. **性能工程** —— Redis cache-aside + singleflight 防击穿 + 异步操作日志 + 令牌桶限流 + 1k QPS 压测验证
5. **完整可观测性** —— Prometheus 指标 + Grafana 仪表盘 + Jaeger 分布式追踪 + 结构化日志
6. **工程质量** —— 四层分层 · CI/CD · 80%+ 测试覆盖 · protobuf 契约 · 多环境配置 · 优雅关闭
7. **异步降级** —— 操作日志异步写入，channel 满时同步写入不丢数据，关闭时 drain 3 秒

---

## 迭代历程

| 阶段 | 内容 | 状态 |
|:----:|------|:----:|
| Phase 0 | 项目初始化：目录结构、CI、Makefile、Docker Compose、错误码、健康检查 | ✓ |
| Phase 1 | 用户服务：注册/登录/登出、JWT、bcrypt、弱密码校验、批量查询缓存 | ✓ |
| Phase 2 | 项目服务：CRUD、成员管理、归档/转让、权限矩阵 | ✓ |
| Phase 3 | 任务服务：CRUD、乐观锁、指派、状态机、游标分页 | ✓ |
| Phase 4 | 评论 + 操作日志：异步写入、优雅降级、17 种操作枚举 | ✓ |
| Phase 5 | 工程增强：Redis 缓存旁路、三级限流、幂等性、多环境配置、集成测试 | ✓ |
| Phase 6 | 可观测性：Prometheus + Grafana + Jaeger + 压测 + 面试稿 | ✓ |

---

## 项目结构

```
task-platform/
├── api/proto/                  # Protobuf 定义 (user/v1, task/v1)
├── cmd/                        # 服务入口 (api-gateway, user-service, task-service)
├── configs/                    # 多环境 YAML 配置 (local, dev, docker)
├── gen/go/                     # buf generate 生成的 protobuf Go 代码
├── internal/
│   ├── gateway/                # API 网关 (handler, middleware, rpc, server)
│   ├── user/                   # 用户服务 (biz, data, service, server)
│   └── task/                   # 任务服务 (biz, data, service, server)
├── migrations/                 # SQL 迁移脚本 (user_svc, task_svc)
├── pkg/                        # 共享包 (xerr, xgrpc, xhttp, xjwt, xlog, xredis, xratelimit...)
├── scripts/                    # 构建/测试/迁移脚本
├── test/integration/           # 集成测试 (testcontainers-go, 真实 PG + Redis)
├── deploy/                     # Docker Compose, Prometheus, Grafana 配置
├── docs/                       # 文档 (面试稿, Postman Collection)
├── web/                        # React 前端 (React 19 + TypeScript 6 + Ant Design 5)
└── Makefile                    # proto, test, lint, coverage, migrate, up/down
```
