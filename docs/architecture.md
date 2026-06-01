# 架构设计

Task Platform 是一个团队任务协作平台，采用「1 个 API Gateway + 2 个核心微服务」的克制型微服务架构。外部使用 HTTP/JSON，内部使用 gRPC unary 调用，数据层使用单 PostgreSQL 实例内的多 schema 逻辑隔离，并通过 Redis 承担缓存、限流、幂等和 JWT 黑名单。

## 总体拓扑

```text
Web / Postman / API Client
        |
        | HTTP JSON
        v
api-gateway (:8080, Gin)
  - 参数绑定
  - JWT 鉴权
  - 统一错误响应
  - 幂等处理
  - 用户信息聚合
        |
        | gRPC unary + metadata + deadline
        +----------------------+----------------------+
                               |
                               v
           +-------------------+-------------------+
           |                                       |
           v                                       v
user-service (:9091)                    task-service (:9092)
  - Register/Login/GetUser                - Project/Member
  - BatchGetUsers                         - Task/Comment
  - JWT 签发                              - OperationLog
           |                                       |
           +-------------------+-------------------+
                               |
                               v
PostgreSQL 16
  - schema user_svc
  - schema task_svc

Redis 7
  - user/project cache
  - rate limit token bucket
  - idempotency result cache
  - JWT blacklist
```

## 服务边界

| 服务 | 对外协议 | 主要职责 | 不负责 |
|------|----------|----------|--------|
| `api-gateway` | HTTP/JSON | 路由、参数绑定、JWT 验签、统一响应、幂等、限流、用户展示信息聚合 | 持久化业务数据、执行任务权限核心逻辑 |
| `user-service` | gRPC | 注册、登录、用户查询、批量用户查询、密码与 JWT 签发 | HTTP 语义、任务协作逻辑 |
| `task-service` | gRPC | 项目、成员、任务、评论、操作日志、权限矩阵、状态机 | JWT 验签、存储用户昵称头像 |

服务数量刻意保持较少。用户域和任务协作域是当前项目的核心边界，继续拆分通知、搜索、文件等服务会增加部署和一致性成本，不符合该项目的面试型工程目标。

## 调用链路

典型写请求链路：

```text
Client
  -> api-gateway middleware
  -> handler bind/validate
  -> idempotency SETNX
  -> gRPC client metadata injection
  -> task-service interceptor
  -> service
  -> biz permission/state validation
  -> data repository
  -> PostgreSQL
  -> async operation log writer
  -> unified HTTP response
```

典型读请求链路：

```text
Client
  -> api-gateway
  -> task-service list/detail RPC
  -> api-gateway BatchGetUsers enrichment when display user data is needed
  -> unified HTTP response
```

## 内部 RPC 契约

所有内部 gRPC 调用都使用 unary RPC。Gateway 到服务的调用必须携带以下 metadata：

| Metadata | 说明 |
|----------|------|
| `x-internal-token` | 内部静态认证令牌，服务端 interceptor 强制校验 |
| `x-user-id` | 当前登录用户 ID，匿名注册/登录 RPC 可省略 |
| `x-username` | 当前登录用户名，匿名注册/登录 RPC 可省略 |
| `x-request-id` | 请求链路 ID，用于日志、错误响应和追踪 |

JWT 只在 gateway 侧验签。`task-service` 不持有 JWT secret，也不直接解析外部 token。`user-service` 持有 JWT secret 是为了注册/登录时签发 token。

## HTTP 中间件顺序

Gateway 的中间件顺序如下：

```text
Recovery -> MaxBodySize(1MB) -> SecurityHeaders -> RequestID
-> HTTPTrace -> HTTPMetrics -> AccessLog -> CORS
-> RateLimit(IP) -> Auth(JWT) -> RateLimit(user) -> Handler
```

绑定和业务参数校验在 handler 内执行。写请求的幂等检查发生在 bind/validate 之后，这样无效请求不会消耗 `Idempotency-Key`。

## 分层约定

后端保持四层结构：

| 层 | 位置 | 职责 |
|----|------|------|
| HTTP handler | `internal/gateway/handler` | 请求绑定、幂等处理、HTTP/gRPC 转换、响应聚合 |
| gRPC service | `internal/*/service` | RPC 方法实现、基础参数校验、调用 biz |
| biz | `internal/*/biz` | 业务规则、权限矩阵、状态机、缓存策略、事务协调 |
| data | `internal/*/data` | GORM model、Repository、SQL 查询 |

跨服务通用能力放在 `pkg/x*` 下，例如 `xerr`、`xgrpc`、`xhttp`、`xjwt`、`xredis`、`xpgsql`、`xratelimit`、`xtrace`。

## 关键设计决策

| 决策 | 说明 |
|------|------|
| 单 PostgreSQL 实例，多 schema | `user_svc` 与 `task_svc` 逻辑隔离，避免过早引入分布式事务 |
| Gateway 统一 HTTP 响应 | gRPC status 是服务内部错误源，gateway 转换为 `{ code, message, request_id, data? }` |
| 内部静态令牌 | 每个内部 RPC 都校验 `x-internal-token`，防止绕过 gateway 直接调用服务 |
| 默认 deadline | 内部 gRPC client/server 拦截器在调用方未设置 deadline 时补默认超时，避免请求无限挂起 |
| HTTP server timeout | gateway 和 admin HTTP server 均设置 read header/read/write/idle timeout，避免慢连接占用资源 |
| 非成员返回 `NOT_FOUND` | 防止通过状态码探测项目、任务或评论是否存在 |
| 乐观锁 | `projects.version` 和 `tasks.version` 保护并发更新 |
| 独立状态和指派端点 | `UpdateTask` 不接受 `status`、`assignee_id`，避免普通编辑绕过权限和状态机 |
| 操作日志异步写入 | 写业务响应不等待日志批量落库，channel 满时降级为同步写并记录告警 |
| Redis 作为共享基础设施 | 同时用于缓存、限流、幂等和 token 黑名单，降低组件复杂度 |
| singleflight 防击穿 | 用户和项目详情缓存 miss 时合并同 key 并发查询，减少热点 key 过期瞬间的 DB 压力 |

## 数据一致性边界

- 用户和任务数据在一个 PostgreSQL 实例内，但分别处于 `user_svc`、`task_svc` schema。
- 项目所有权以 `projects.owner_id` 为准，`project_members.role = owner` 是展示和权限辅助数据，转让所有权时必须在同一事务内更新两边。
- 添加项目成员、指派任务时，`task-service` 通过 `user-service` 校验用户存在且启用。
- 操作日志是审计数据，不阻塞主业务链路；日志写入失败需要暴露指标和日志，但不回滚主业务结果。
