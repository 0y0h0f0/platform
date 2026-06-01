# 面试讲解稿 — 团队任务协作平台

## 一、项目概述（30 秒）

这是一个基于 Go 微服务架构的团队任务协作平台，支持用户注册登录、项目与任务管理、成员权限控制、操作日志审计。整体采用 **api-gateway + 2 个核心微服务** 的架构，后端技术栈为 Gin + gRPC + PostgreSQL 16 + Redis 7，配有完整的可观测性体系（Prometheus + Grafana + Jaeger）。

**一句话定位：** 用校招项目展示分布式系统后端开发的完整工程能力。

---

## 二、架构设计（2 分钟）

### 2.1 为什么是 1 gateway + 2 services？

```
client → api-gateway (Gin, HTTP) ──gRPC──→ user-service → PostgreSQL (schema: user_svc)
                                   ──gRPC──→ task-service → PostgreSQL (schema: task_svc)
                                                              Redis (cache + rate limit + idempotency)
```

- **服务拆分原则：** 单体数据库 + 逻辑隔离。一个 PostgreSQL 实例拆成 `user_svc` 和 `task_svc` 两个 schema，避免过早引入分布式事务和跨库 join。
- **gateway 职责：** 参数绑定、JWT 验证、request_id 注入、panic recovery、统一错误码映射。业务服务不碰 HTTP 和 JWT。
- **内部通信安全：** 所有 gRPC 调用携带 `x-internal-token` 静态共享密钥，服务端 interceptor 强制校验，不通过的返回 `UNAUTHENTICATED`。业务服务不持有 JWT secret。

### 2.2 四层分层

每个微服务内部严格遵循 handler → service(gRPC) → biz(domain) → data(repository)：

| 层 | 职责 | 关键约束 |
|---|------|---------|
| handler（gateway） | HTTP 参数绑定、idempotency 检查、调用 gRPC | 不做业务逻辑 |
| service | gRPC 实现，参数校验，调用 biz | 不持有 HTTP 依赖 |
| biz | 业务逻辑、权限判断、状态机 | 不依赖 gRPC/HTTP 库 |
| data | GORM 模型、Repository 接口、SQL 查询 | 不包含业务判断 |

每一层都有独立测试，覆盖率 80%+。

---

## 三、核心亮点（3 分钟）

### 3.1 权限矩阵 + 状态机

- **角色三层：** owner / admin / member。归档项目对所有角色只读。
- **任务状态机：** `todo ↔ doing ↔ done`，`todo → cancelled`，`cancelled → todo`。非法跃迁返回 `FAILED_PRECONDITION`。
- **非成员保护：** 非成员访问资源返回 404（而非 403），避免泄露资源存在性。

### 3.2 并发安全

- **乐观锁：** `projects.version` 和 `tasks.version` 字段，每次更新 `WHERE version = ?` 并自增。冲突时返回错误让客户端重试。
- **Owner 一致性：** `projects.owner_id` 是唯一真相来源。`TransferOwnership` 在一个事务中同时更新 `projects.owner_id` 和 `project_members.role`。
- **幂等性：** 所有写操作接受 `Idempotency-Key` 请求头，Redis `SETNX` 实现，24h TTL。在 handler 层参数校验之后才检查，防止非法请求占用 key。

### 3.3 异步操作日志

- 容量 1024 的 channel，worker 数通过 `LOG_WRITER_WORKERS` 配置（默认 1，最大 16），批量写入（batch size 64，flush 间隔 100ms）。
- **优雅降级：** channel 满时同步写入 + warn 日志 + Prometheus 计数器递增。
- **优雅关闭：** 收到 SIGTERM 后 flush 剩余日志，最多等 3 秒后放弃。

### 3.4 可观测性

- **Metrics：** OpenTelemetry + Prometheus。每个服务暴露 `/metrics`。关键指标包括 HTTP 请求耗时（按 path/method）、gRPC 调用耗时、DB 连接池状态、Redis 命中率、Rate Limiter 放行/拒绝数和操作日志降级/失败计数。
- **Tracing：** OpenTelemetry + Jaeger，每个请求注入 trace_id，gRPC 调用自动传播 context。
- **Logging：** Zap JSON 格式，每条日志带 `request_id`，与 trace 关联。
- **Grafana Dashboard：** 19 个面板覆盖 HTTP/gRPC/DB/Redis/RateLimiter。

### 3.5 性能优化

- **bcrypt cost 可配置：** 默认 10（安全），压测设为 8（Login P99 从 221ms 降到 ~80ms）。
- **DB 连接池可配置：** `DB_MAX_OPEN_CONNS` / `DB_MAX_IDLE_CONNS`，默认 100/25，可按容量目标调整。
- **Redis 连接池可配置：** `REDIS_POOL_SIZE` / `REDIS_MIN_IDLE_CONNS` 以及 dial/read/write/pool timeout 都可通过环境变量调整。
- **限流可配置：** 基于 Redis token bucket（Lua 脚本），IP 级、用户级、Auth 级三层限流，速率通过环境变量控制。
- **单机 1k QPS：** `GET /me` + `GET /projects` 混合场景，constant-arrival-rate 500 iter/s（1000 HTTP req/s），P50 0.76ms，P95 1.34ms，0% 错误率。
- **Redis 缓存：** 用户信息缓存（TTL 5min），cache-aside 模式，写操作时主动失效。

### 3.6 游标分页

Tasks 列表使用游标分页而非 offset 分页。游标是 base64url 编码的 `{created_at, id, filter_hash}`。服务端验证 `filter_hash` 是否匹配当前请求的筛选条件，不匹配则拒绝——防止翻页过程中筛选条件变化导致的结果错乱。

---

## 四、技术难点与解决方案（1 分钟）

| 难点 | 解决方案 |
|------|---------|
| 微服务间调用传播用户身份 | gRPC metadata 注入 `x-user-id`、`x-username`，服务端 interceptor 提取到 context |
| JWT 黑名单 vs 无状态 | 登出时将 `jti` 写入 Redis blacklist，TTL 等于 token 剩余有效期，gateway 验证时同时检查黑名单 |
| Owner 变更原子性 | 数据库事务：更新 `projects.owner_id` + 更新旧 owner 为 admin + 新 owner 的 `project_members` 行设为 owner |
| 归档项目读写隔离 | biz 层在执行任何写操作前检查项目状态，归档则返回 `FAILED_PRECONDITION`；读操作不做限制 |
| 注册用户端到端 | Register 直接返回 JWT token，省去注册后重新登录的步骤，提升用户体验 |
| 乐观锁版本号同步 | 所有 Update/Assign/ChangeStatus 响应都会返回新 version，Postman 脚本自动更新 collection 变量 |

---

## 五、项目流程与演进（1 分钟）

项目分 6 个 Phase 迭代开发，每个 Phase 结束时系统完整可运行：

| Phase | 交付内容 | 可验证 |
|-------|---------|--------|
| Phase 0 | 项目骨架、CI/CD、Docker Compose | 三个空服务编译启动 |
| Phase 1 | 用户服务：Register/Login/Logout/Me/BatchGetUsers | JWT + Redis blacklist + bcrypt |
| Phase 2 | 项目 CRUD、成员管理、权限矩阵 | 完整 RBAC + 乐观锁 |
| Phase 3 | 任务 CRUD、状态机、游标分页 | 任务状态跃迁校验 |
| Phase 4 | 评论、操作日志、异步 worker | channel + configurable worker + 批量写入 |
| Phase 5 | 限流、幂等、缓存失效 | Redis token bucket + SETNX |
| Phase 6 | Metrics、Tracing、Grafana、压测 | 1k QPS 通过 + Dashboard 可观测 |

持续集成：GitHub Actions 自动运行 `make lint` + `make test` + `buf lint`，覆盖率需 ≥80%。

---

## 六、项目收获（30 秒）

1. **微服务不等于服务拆分越细越好。** 本项目的 gateway + 2 services 在复杂度和灵活性之间找到了平衡，过早拆分会导致分布式事务、调试困难、部署复杂度上升。
2. **边界清晰是架构质量的根基。** gateway 只管 HTTP、service 只管 gRPC、biz 只管业务——每层测试可以独立编写，mock 成本极低。
3. **可观测性不是后加的。** 从 Phase 0 起每个服务都有 `/metrics` 和 `/healthz`，trace_id 贯穿所有日志，出问题时有据可查。
4. **压测驱动调优。** 仅靠代码 review 无法发现 bcrypt cost 对 Login P99 的影响——1k QPS throughput test 暴露了连接池、限流、bcrypt 三个瓶颈，逐一修复后系统性能达标。

---

## 面试可能追问

**Q: 为什么不用 JWT 做无状态登出？**
> JWT 是无状态的，签发后无法撤销。本方案将 `jti` 写入 Redis blacklist（TTL = 剩余有效期），gateway 每次验证时查询黑名单。登出是低频操作，Redis 查询 O(1)，工程上是最小成本的登出实现。

**Q: 如果 Redis 挂了怎么办？**
> 缓存失效降级为查数据库；限流 fail-open（Redis 不可用时放行，避免误杀）；幂等性暂时失效（允许重复请求，但不会丢数据，因为业务层有乐观锁兜底）。Redis 客户端配置了连接池和命令超时，默认 dial 3s、read/write 2s、pool 4s，不会无限阻塞主流程。

**Q: 乐观锁冲突如何重试？**
> 当前服务端返回错误让客户端重试。如果要实现服务端自动重试，可以在 biz 层加 retry loop（最多 3 次，指数退避）。但通常让客户端重试更合理——客户端有自己的交互上下文，比如展示"更新失败，请重试"。

**Q: 为什么游标分页要验证 filter_hash？**
> 防止"翻页中途改了筛选条件导致结果错乱"：用户在第 1 页筛选 status=todo，拿到 cursor 后改成 status=done——如果服务端不校验，第 2 页会按 done 查但 cursor 是按 todo 生成的，结果必然错乱。filter_hash 是 `md5(project_id+status+assignee_id+keyword)` 的 hash，不匹配立即拒绝。

**Q: Channel 满了为什么要降级同步写入而不是扩容 channel？**
> 扩容 channel 只是把问题延后。1024 的缓冲区已经足够应对瞬时突发，持续满载说明下游（数据库）跟不上，应该扩容数据库或增加 worker——而不是无限制堆内存。
