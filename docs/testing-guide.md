# 测试指南

项目测试分为 Go 单元测试、Go 集成测试、前端单元测试、浏览器 E2E、覆盖率门禁和压测。

## 后端单元测试

运行全部 Go 测试：

```bash
make test
```

或直接：

```bash
go test ./...
```

运行指定包：

```bash
go test ./internal/task/biz -run TestChangeTaskStatus
go test ./pkg/xjwt -run TestManager
```

覆盖范围：

| 目录 | 重点 |
|------|------|
| `internal/user/data` | GORM repository |
| `internal/user/biz` | 注册、登录、缓存、弱密码 |
| `internal/user/service` | gRPC 方法 |
| `internal/user/server` | server wiring、interceptor |
| `internal/task/biz` | 项目权限、任务状态机、评论、操作日志 |
| `internal/task/data` | repository 查询 |
| `internal/gateway/handler` | HTTP handler、错误响应、幂等 |
| `internal/gateway/middleware` | 鉴权、限流、CORS、安全头 |
| `pkg/x*` | 通用库：错误码、JWT、Redis/PG 客户端、HTTP/gRPC timeout、限流、游标等 |

补充关注点：

- `internal/gateway/handler/idempotency_test.go` 覆盖 pending key 返回 409/`ABORTED`、错误响应释放 key。
- `internal/task/biz/log_writer_test.go` 覆盖 `LOG_WRITER_WORKERS`、channel 满降级和 worker panic recovery。
- `pkg/xredis/client_test.go` 覆盖 Redis 连接池和超时环境变量解析。
- `pkg/xhttp`、`pkg/xgrpc` 相关测试应覆盖默认超时和已有 deadline 不被覆盖的场景。

## 后端集成测试

集成测试位于 `test/integration/`，使用 testcontainers-go 拉起真实 PostgreSQL 16 和 Redis 7。

```bash
go test -tags=integration ./test/integration/ -v
```

覆盖场景包括：

- 注册、登录、登出、用户查询
- 项目 CRUD、归档、转让所有权
- 成员添加、移除、角色变更、退出
- 任务 CRUD、指派、状态流转
- 评论和操作日志
- Gateway 到 gRPC 服务的端到端链路
- 权限矩阵和非成员访问隐藏

运行前确认 Docker daemon 可用。

## 覆盖率

```bash
make coverage
```

覆盖率脚本统计 `internal/` 和 `pkg/`，排除 `cmd/` 与生成代码。项目基线为 80%+。

常用直接命令：

```bash
go test -coverprofile=coverage.out ./internal/... ./pkg/...
```

## Proto 检查

```bash
make proto-lint
make proto
```

修改 proto 后必须重新生成 `gen/go/`，并运行相关服务和 gateway 测试。

## 静态检查

```bash
make lint
```

Makefile 会优先使用本地 `golangci-lint`，不存在时使用 `golangci/golangci-lint:v2.12.2` Docker 镜像。

## 前端质量检查

```bash
cd web
npm run typecheck
npm run lint
npm run test
npm run build
```

前端测试使用 Vitest、Testing Library 和 jsdom。测试目录位于 `web/__tests__/`。

## 前端 E2E

Playwright 在 MSW mock 模式下运行，不依赖后端服务：

```bash
cd web
npx playwright install chromium
npm run e2e
```

E2E 覆盖注册、登录、项目创建、成员管理、任务创建、看板拖拽、评论、归档行为和权限矩阵。

## 压测

Makefile 提供 k6 压测入口：

```bash
make loadtest-baseline
make loadtest-stress
make loadtest-endurance
make loadtest-throughput
make loadtest-login
```

现有报告位于 `test/load/loadtest-report.md`，关键结论：

| 场景 | 结果 |
|------|------|
| 登录 4 QPS | 成功率 100%，但 bcrypt cost=10 下 P99 超过 100ms |
| `GET /users/me` 50 QPS | 成功率 100%，P95 约 2ms |
| `GET /projects` 50 QPS | 成功率 100%，P95 约 2ms |
| `POST /tasks` 50 QPS | 成功率 100%，P95 约 6ms |

压测前需要先启动后端基础设施、迁移数据库、启动三个服务，并准备可用 token 和项目数据。

## 提交前建议检查

```bash
make proto-lint
make test
make coverage
make lint
cd web && npm run typecheck && npm run lint && npm run test && npm run build
```

如果改动涉及 HTTP 流程、权限、状态机或跨服务调用，再补跑：

```bash
go test -tags=integration ./test/integration/ -v
cd web && npm run e2e
```
