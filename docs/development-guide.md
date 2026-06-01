# 开发指南

本文档说明如何在本地启动、开发和验证 Task Platform。

## 前置依赖

| 依赖 | 用途 |
|------|------|
| Go 1.26+ | 后端服务、测试和代码生成 |
| Docker | PostgreSQL、Redis、Prometheus、Jaeger、Grafana、testcontainers |
| Make | 常用开发命令 |
| Node.js 22+ / npm 10+ | 前端开发 |
| k6 | 可选，压测脚本 |
| golangci-lint | 可选；本地没有时 Makefile 会用 Docker 镜像 |
| buf | 可选；本地没有时 Makefile 会用 Docker 镜像 |

## 初始化

```bash
cp .env.example .env
```

编辑 `.env`，至少替换：

```text
JWT_SECRET=replace-with-a-real-secret-at-least-32-chars
INTERNAL_TOKEN=replace-with-a-real-internal-token
POSTGRES_DSN=postgres://postgres:postgres@127.0.0.1:5433/task_platform?sslmode=disable
```

`JWT_SECRET` 和 `INTERNAL_TOKEN` 不能使用模板占位值，否则服务会拒绝启动。

## 启动后端基础设施

```bash
make up
make migrate
```

`make up` 启动 PostgreSQL、Redis、Prometheus、Jaeger 和 Grafana。`make migrate` 对 `user_svc` 和 `task_svc` 分别执行数据库迁移。

## 启动服务

三个服务需要分别运行：

```bash
make run/user-service
make run/task-service
make run/api-gateway
```

默认端口：

| 服务 | 地址 |
|------|------|
| api-gateway | `http://127.0.0.1:8080` |
| user-service gRPC | `127.0.0.1:9091` |
| user-service admin | `http://127.0.0.1:8081` |
| task-service gRPC | `127.0.0.1:9092` |
| task-service admin | `http://127.0.0.1:8082` |

健康检查：

```bash
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8081/readyz
curl http://127.0.0.1:8082/metrics
```

## 常用命令

| 命令 | 说明 |
|------|------|
| `make proto` | 使用 buf 重新生成 gRPC Go 代码 |
| `make proto-lint` | lint proto 文件 |
| `make test` | 运行全部 Go 测试 |
| `make lint` | 运行 golangci-lint |
| `make coverage` | 生成覆盖率报告，目标为 `internal/` 和 `pkg/` 80%+ |
| `make migrate` | 执行数据库迁移 |
| `make seed-users` | 执行用户种子脚本 |
| `make down` | 停止 Docker Compose 基础设施 |

直接运行某个测试：

```bash
go test ./internal/user/biz -run TestRegister
go test ./internal/gateway/handler -run TestTask
```

## 代码结构

```text
api/proto/              # user/task proto 定义
cmd/                    # 三个服务入口
configs/                # local/dev/docker 多环境配置
gen/go/                 # 生成的 protobuf Go 代码
internal/gateway/       # HTTP gateway
internal/user/          # 用户服务
internal/task/          # 任务协作服务
pkg/                    # 跨服务共享包
migrations/             # schema 级 SQL 迁移
scripts/                # 迁移和覆盖率脚本
test/integration/       # testcontainers 集成测试
test/load/              # k6/vegeta 压测脚本和报告
deploy/                 # Docker Compose、Prometheus、Grafana
web/                    # React 前端
```

## 开发新接口的流程

1. 先判断是否属于 user 域或 task 域。
2. 修改 `api/proto/*`，运行 `make proto`。
3. 在对应服务中按 `data -> biz -> service` 补齐逻辑和测试。
4. 在 `internal/gateway/handler` 增加 HTTP handler。
5. 在 `internal/gateway/server/server.go` 注册路由。
6. 更新 [api-reference.md](api-reference.md) 和 Postman 集合。
7. 运行单元测试、集成测试和 lint。

## 开发约束

- Gateway 负责 HTTP 语义、JWT 验签、统一错误、限流和幂等。
- 服务层不要依赖 Gin 或 HTTP 状态码。
- `task-service` 不存用户昵称、头像；需要展示时由 gateway 批量查询并聚合。
- `UpdateTask` 不能直接修改状态或指派人，必须使用 `ChangeTaskStatus` 和 `AssignTask`。
- 归档项目全员只读，任何写操作都应返回 `FAILED_PRECONDITION`。
- 非成员访问资源返回 `NOT_FOUND`，避免泄露资源存在。
- 新增共享能力优先放在 `pkg/x*`，避免服务间复制实现。

## 前端开发

```bash
cd web
npm install
npm run dev
```

Mock 模式无需后端：

```bash
cd web
npm run dev:mock
```

详见 [frontend-guide.md](frontend-guide.md)。

## 常见问题

### 服务提示 `JWT_SECRET is required`

检查 `.env` 是否存在，或当前 shell 是否加载了 `.env`。通过 `make run/<service>` 启动会自动 source 根目录 `.env`。

### 服务提示 `POSTGRES_DSN is required`

`user-service` 和 `task-service` 必须设置 `POSTGRES_DSN`。示例：

```text
POSTGRES_DSN=postgres://postgres:postgres@127.0.0.1:5433/task_platform?sslmode=disable
```

### 集成测试无法启动容器

确认 Docker daemon 正常运行。集成测试依赖 testcontainers-go，会临时启动 PostgreSQL 和 Redis。

### 登录压测 P99 偏高

登录路径包含 bcrypt cost=10，低并发时均值可接受，但 P99 可能超过 100ms。除非明确接受安全性 tradeoff，不建议随意降低 bcrypt cost。
