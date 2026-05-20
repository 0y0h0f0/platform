# 校招项目详细生成计划

## 1. 项目定位

### 1.1 项目名称
团队任务协作平台后端

### 1.2 项目目标
- 面向校招面试，做一个可以完整演示业务流程、服务拆分和工程能力的后端项目。
- 对外提供 HTTP/JSON 接口，对内通过 gRPC 做服务通信。
- 项目规模控制在 1 个 API Gateway + 2 个核心服务，保证实现难度可控、讲解逻辑清晰。

### 1.3 项目价值
- 能体现 Go 基础能力：项目结构、并发、错误处理、接口设计。
- 能体现常用后端能力：认证、权限、缓存、数据库设计、日志、监控。
- 能体现工程能力：配置管理、容器部署、统一中间件、分层设计、测试。

## 2. 技术选型

### 2.1 核心技术栈
- 语言：Go 1.23+
- Web 网关：Gin
- RPC 通信：gRPC + Protocol Buffers(proto3)
- Proto 工具链：`buf`（lint + breaking-change 检测 + 代码生成）
- 数据库：PostgreSQL 16
- 缓存：Redis 7
- ORM：GORM + `gorm.io/driver/postgres`
- 驱动：pgx/v5
- 数据库迁移：`golang-migrate`
- 配置：Viper（YAML 文件 + 环境变量覆盖，多环境分目录）
- 鉴权：JWT（HS256，仅 access token，过期 2h；`user-service` 签发，仅 `api-gateway` 做验签；内部服务不再二次校验 JWT，只校验网关注入的 `x-internal-token`、`x-request-id`，以及匿名 RPC 之外必填的 `x-user-id`、`x-username`）
- 密码哈希：bcrypt (cost=10)
- 日志：Zap（JSON 编码，按 request_id 串联）
- 可观测性：Prometheus + OpenTelemetry
- 容器编排：Docker Compose
- 测试：`go test` + `testify` + `httptest` + `testcontainers-go`（集成测试拉真实 PG/Redis）
- 静态检查：`golangci-lint`
- CI：GitHub Actions（lint + 单测 + 集成测试 + 覆盖率门槛 + `buf lint`）

### 2.2 选型理由
- Gin 学习曲线低，适合承接 HTTP API、参数绑定、中间件链路。
- gRPC 能体现内部服务调用、接口契约、状态码和超时控制。
- PostgreSQL 适合展示关系建模、事务、索引、`jsonb`、`timestamptz`。
- Redis 可用于登录态、热点缓存和简单限流，能补足工程亮点。
- Docker Compose 适合本地一键启动，便于项目交付和面试展示。

## 3. 项目范围

### 3.1 服务拆分
- `api-gateway`
  - 对外暴露 HTTP 接口
  - 参数校验、鉴权、统一错误响应
  - 调用 `user-service` 和 `task-service`
- `user-service`
  - 用户注册、登录、查询个人信息
  - 签发 JWT；仅网关验签，内部服务认其注入的可信身份
  - 提供用户存在性校验和公开资料查询
- `task-service`
  - 项目管理
  - 任务管理
  - 评论管理
  - 项目成员身份校验（`project_members` 表归属于本服务，对外提供 `CheckProjectMember`）
  - 新增成员、指派任务前调用 `user-service` 校验目标用户存在
  - 操作日志记录

### 3.2 功能边界
- 必做功能
  - 注册、登录、登出、获取当前用户信息
  - 创建项目、修改项目、归档项目、邀请成员、修改成员角色、移除成员、成员列表
  - 创建任务、编辑任务、删除任务、任务详情
  - 任务指派、状态流转、优先级、截止时间
  - 任务评论、删除评论
  - 项目/任务操作日志查询
  - 分页查询、条件筛选

#### 注册与密码规则
- 用户名：`^[a-zA-Z0-9_]{3,32}$`
- 邮箱：标准 RFC 5322 校验（依赖 `mail.ParseAddress`）
- 密码：长度 8–64，至少包含字母与数字各一个；不在仓库内置弱密码词表中（词表文件随仓库提交，如 `configs/security/weak_passwords.txt`，运行时本地读取，不依赖联网下载）
- 校验失败统一返回 `INVALID_ARGUMENT` 错误码与字段级 message

- 加分功能
  - Redis 缓存用户信息和项目详情（cache-aside：读穿透，写时失效，TTL 兜底）
  - 基于 Redis 的 token bucket 限流，分两层：
    - 匿名接口（`/auth/register`、`/auth/login`）按 IP 单独限流，阈值更严，防爆破
    - 已登录接口按 `user_id` 限流；同一 IP 同时维护一层兜底 IP 限流，阻止单 IP 多账号刷量
  - 关键写接口 `Idempotency-Key` 头 + Redis 去重，防止重复提交
  - Prometheus 指标
  - OpenTelemetry Trace
  - 操作日志异步写入（有界 channel + 单 goroutine 消费，避免拖慢主事务）
  - 乐观锁防止任务、项目覆盖更新

### 3.3 关键业务规则

#### 权限矩阵
- `owner`
  - 拥有项目完全控制权
  - 可新增/移除任意角色成员（含 `admin`）
  - 可修改任意成员角色
  - 可修改项目基本信息、归档/取消归档项目
  - 可创建、编辑、删除、分配任意任务
  - 可转让 owner（`TransferOwnership`），转让后自身降级为 `admin`
  - **不可移除自己**；如需退出必须先转让 owner 或归档项目
- `admin`
  - 可创建任务、编辑任务、分配任务、修改任务状态、发表评论
  - 可邀请新成员加入项目，但只能授予 `member` 角色
  - 可移除 `member`，**不可移除 `owner` 或其他 `admin`**
  - 不可归档项目、不可修改项目基本信息、不可提升其他成员为 `admin`
- `member`
  - 可查看项目和任务
  - 可创建任务、发表评论
  - 仅可编辑自己创建的任务
  - 仅可修改自己负责的任务状态
  - 不可邀请/移除其他成员，不可删除他人任务
  - 可主动退出项目（`LeaveProject`）；owner 不能用该接口退出

#### 项目归档语义
- 归档后**只读**：成员仍可 `GET` 项目、任务、评论、操作日志；非成员仍按非成员权限处理
- 归档后**全员禁写**：禁止新增/编辑/删除/分配任务、修改状态、新增/删除评论、新增/移除成员、修改成员角色
- 仅 owner 可对归档项目执行"取消归档"

#### 任务状态流转
- `todo -> doing`
- `todo -> done`（小任务直接完成）
- `todo -> cancelled`
- `doing -> done`
- `doing -> cancelled`
- `doing -> todo`
- `done -> doing`（重新打开）
- `cancelled -> todo`
- 未列出的状态转换一律视为非法请求，返回 `FAILED_PRECONDITION`

#### 任务删除权限
- `owner` 可删除**未归档项目**下的任意任务（归档项目下全员禁写，见"项目归档语义"）
- `admin` 可删除未归档项目下的任意任务
- `member` 仅可删除自己创建且状态为 `todo` 的任务

#### 评论规则
- 评论创建后不可编辑
- 评论允许作者本人删除
- `owner/admin` 可删除任意评论

#### 读权限规则
- `GET /api/v1/projects` 仅返回“我参与的项目”，不会暴露非成员项目
- `GET /api/v1/projects/:id`、`GET /api/v1/projects/:id/members`、`GET /api/v1/projects/:id/operation-logs` 仅项目成员可访问
- `GET /api/v1/tasks`、`GET /api/v1/tasks/:id`、`GET /api/v1/tasks/:id/comments`、`GET /api/v1/tasks/:id/operation-logs` 仅项目成员可访问
- 对非成员访问项目/任务/评论/操作日志类资源时，统一返回 `NOT_FOUND`，避免泄露资源是否存在
- 项目归档不会放宽可见性；归档项目仍然只允许成员读取

#### 任务指派规则
- `AssignTask` 的目标 `assignee_id` 必须同时满足：
  - 用户存在且 `status = active`
  - 当前仍是该项目成员（`project_members` 中存在记录）
- 不允许将任务指派给非成员、已被移除成员或 disabled 用户；否则返回 `FAILED_PRECONDITION`
- V1 不提供单独“取消指派”接口；如需更换负责人，必须重新指定新的有效成员
- 若负责人后续退出项目、被移除或被禁用，任务保留原 `assignee_id` 作为历史记录，但该用户不再拥有任何任务操作权限；`owner/admin` 应手动重新分配

## 4. 总体架构

```text
                +-----------------------+
                |   Web / Postman / UI  |
                +-----------+-----------+
                            |
                       HTTP / JSON
                            |
                            v
                +-----------------------+
                |      api-gateway      |
                |         Gin           |
                | auth / bind / log     |
                | request_id / recovery |
                +-----------+-----------+
                            |
                        gRPC Client
                  +---------+---------+
                  |                   |
                  v                   v
         +----------------+   +----------------+
         |  user-service  |   |  task-service  |
         |      gRPC      |   |      gRPC      |
         +--------+-------+   +--------+-------+
                  |                    |
                  |                    |
                  v                    v
           +-------------+      +-------------+
           | PostgreSQL  |      | PostgreSQL  |
           | user schema |      | task schema |
           +------+------+      +------+------+
                  \                    /
                   \                  /
                    +--------+-------+
                             |
                           Redis

               +-----------------------------+
               | Prometheus / OpenTelemetry  |
               +-----------------------------+
```

### 4.1 数据组织建议
- 使用 1 个 PostgreSQL 实例。
- 逻辑上拆分为两个 schema：
  - `user_svc`
  - `task_svc`
- 使用 1 个 Redis 实例。

### 4.2 调用链路
1. 客户端请求到 `api-gateway`
2. 网关完成参数校验、JWT 校验、日志注入
3. 网关通过 gRPC 调用内部服务
4. 服务访问 PostgreSQL/Redis
5. 服务返回 gRPC status，网关转换为统一 HTTP 响应

### 4.3 request_id 与 trace 透传
- 网关入口：若请求带 `X-Request-Id` 则沿用，否则生成 UUID
- 注入 `gin.Context` 与 Zap logger 字段
- 调 gRPC 时通过 metadata `x-request-id` 透传
- 服务端 interceptor 取出后写入 ctx 与日志字段
- 响应头回写 `X-Request-Id`，便于客户端报障对账
- OpenTelemetry span 通过 `otelgrpc` 拦截器自动透传 `traceparent`

### 4.4 认证与跨服务契约
- 外部用户认证只接受 `Authorization: Bearer <token>`
- JWT claim 固定包含：`sub(user_id)`、`username`、`jti`、`iat`、`exp`
- **JWT 仅在 `api-gateway` 验签**：校验签名、过期、Redis 黑名单；内部服务不再持有 JWT 密钥、不再二次验签
- 本项目 V1 不做 refresh token；只做 access token
- `POST /auth/logout` 是 **gateway 本地操作**：从当前 access token 解析 `jti` 后直接写入 Redis 黑名单，TTL = token 剩余有效期；**不经过 `user-service` RPC**
- V1 不支持“全端强制下线”和“修改密码后历史 token 全失效”；这两项明确列为后续扩展，不在当前范围内
- `api-gateway -> gRPC` 调用时必须注入 `x-user-id`、`x-username`、`x-request-id`、`x-internal-token`
- `x-internal-token` 是内部服务调用口令，配置在环境变量中；**所有内部 RPC** 的 gRPC interceptor 都必须校验，拒绝绕过网关的直接请求
- `x-internal-token` 作为 V1 简化方案，**静态共享密钥风险已知**；V2 可演进到 mTLS 或短期签发的内部 token，作为面试讲解点
- `x-request-id` 对所有 RPC 必填
- 匿名 RPC 仅限 `Register`、`Login`；这两个 RPC 允许缺省 `x-user-id`、`x-username`
- 除 `Register`、`Login` 外，其他 RPC 都必须带 `x-user-id`、`x-username`；缺失时由 interceptor 直接返回 `UNAUTHENTICATED`
- `task-service` 只拥有项目、成员、任务、评论、操作日志数据；绝不落库用户昵称、头像等主数据
- `user-service` 拥有用户主数据；`task-service` 在 `AddProjectMember`、`AssignTask` 等写路径上调用 `user-service` 校验目标用户存在
- 列表/详情接口需要展示用户昵称、头像时，由 `api-gateway` 调用 `BatchGetUsers` 进行批量补全，避免 `task-service` 在读路径产生级联 RPC
- `BatchGetUsers` 单次最多 100 个 `user_id`；`user-service` 端结果优先走 Redis 缓存（TTL 5 分钟），缓存键 `user:{id}`
- `BatchGetUsers` 降级策略：若某 `user_id` 对应用户已软删除，返回占位对象 `{ id, nickname: "已注销", avatar_url: "" }`，而不是将整个批次请求失败；调用方可对此占位对象做友好展示

## 5. 目录规划

```text
task-platform/
├── .github/
│   └── workflows/        # CI：lint + test + buf lint
├── cmd/
│   ├── api-gateway/
│   ├── user-service/
│   └── task-service/
├── api/
│   └── proto/
│       ├── user/v1/
│       └── task/v1/
├── internal/
│   ├── gateway/
│   │   ├── handler/
│   │   ├── middleware/
│   │   ├── rpc/
│   │   └── service/
│   ├── user/
│   │   ├── biz/
│   │   ├── data/
│   │   ├── service/
│   │   └── server/
│   └── task/
│       ├── biz/
│       ├── data/
│       ├── service/
│       └── server/
├── pkg/
│   ├── xerr/
│   ├── xgrpc/
│   ├── xhttp/
│   ├── xjwt/
│   ├── xlog/
│   ├── xtrace/
│   ├── xredis/
│   └── xpgsql/
├── configs/
│   ├── local/
│   ├── dev/
│   ├── docker/
│   └── security/           # 弱密码词表等静态安全配置
├── migrations/
├── scripts/              # run-migrations.sh、seed.sh 等工具脚本
├── deploy/
│   ├── docker-compose.yml
│   ├── prometheus.yml
│   └── grafana/
├── test/
├── .env.example          # 敏感配置占位；.env 列入 .gitignore
└── README.md
```

## 6. 数据库设计计划

### 6.1 `user_svc` schema

#### `users`
- `id`
- `username`
- `email`
- `password_hash`
- `nickname`
- `avatar_url`
- `status`（smallint：0=active / 1=disabled；disabled 用户登录返回 `PERMISSION_DENIED`）
- `created_at`
- `updated_at`
- `deleted_at`

### 6.2 `task_svc` schema

#### `projects`
- `id`
- `name`
- `description`
- `owner_id`
- `status`（smallint，0=active / 1=archived）
- `version`（乐观锁，bigint，默认 0）
- `created_at`
- `updated_at`
- `deleted_at`

#### `project_members`
- `id`
- `project_id`
- `user_id`
- `role`（smallint：0=owner / 1=admin / 2=member）
- `joined_at`
- `updated_at`（记录角色变更时间；由 `UpdateProjectMemberRole` 与 `TransferProjectOwnership` 更新）

#### `tasks`
- `id`
- `project_id`
- `title`（`VARCHAR(200) NOT NULL`）
- `content`（`TEXT`，可空）
- `status`（smallint：0=todo / 1=doing / 2=done / 3=cancelled）
- `priority`（smallint：0=low / 1=normal / 2=high / 3=urgent）
- `assignee_id`（可空，`NULL` 代表未指派）
- `creator_id`
- `due_time`（`timestamptz`，可空）
- `version`（乐观锁，bigint，默认 0）
- `extra jsonb`（**仅允许约定字段**：`labels` `checklist` `attachments`；其他键禁止写入，避免变成垃圾桶）
- `created_at`
- `updated_at`
- `deleted_at`

#### `task_comments`
- `id`
- `task_id`
- `user_id`
- `content`
- `created_at`
- （评论不可编辑，无需 `updated_at`；删除操作直接物理删除，不引入软删除字段）

#### `operation_logs`
- `id`
- `project_id`（可空，记录用户级动作时为 NULL）
- `task_id`（可空）
- `operator_id`
- `action`（枚举字符串，全集如下）
  - 任务类：`task.create` / `task.update` / `task.assign` / `task.status_change` / `task.delete`
  - 评论类：`comment.create` / `comment.delete`
  - 成员类：`member.add` / `member.remove` / `member.role_change` / `member.leave`
  - 项目类：`project.create` / `project.update` / `project.archive` / `project.unarchive` / `project.transfer_ownership`
- `detail jsonb`
- `created_at`
- 写入路径：业务层 → 有界 channel → 后台 goroutine 批量 `INSERT`；channel 满则降级为同步写并打 warn 日志

### 6.3 索引计划
- `users(username)` 部分唯一索引：`WHERE deleted_at IS NULL`
- `users(email)` 部分唯一索引：`WHERE deleted_at IS NULL`
- `projects(owner_id, name)` 部分唯一索引：`WHERE deleted_at IS NULL`（禁止同一用户重名项目）
- `project_members(project_id, user_id)` 唯一索引
- `project_members(project_id)` 部分唯一索引：`WHERE role = 0`，保证每个项目最多一个 owner 行
- `project_members(user_id, project_id)` 组合索引：支撑“我参与的项目”列表查询
- `tasks(project_id, created_at DESC, id DESC) WHERE deleted_at IS NULL` 部分索引：支撑 `ListTasks` 默认游标分页
- `tasks(project_id, status, created_at DESC, id DESC) WHERE deleted_at IS NULL` 部分索引：支撑按状态筛选的列表查询
- `tasks(project_id, assignee_id, created_at DESC, id DESC) WHERE deleted_at IS NULL` 部分索引：支撑按负责人筛选的列表查询
- `tasks(due_time) WHERE deleted_at IS NULL AND due_time IS NOT NULL` 部分索引：支撑"全项目逾期任务"查询
- `task_comments(task_id, id)` 组合索引：支撑评论 `after_id` 顺序翻页
- `operation_logs(project_id, created_at DESC, id DESC)` 组合索引
- `operation_logs(task_id, created_at DESC, id DESC)` 组合索引
- 时间列统一使用 `timestamptz`（UTC 存储，展示层做时区转换）

### 6.4 操作日志可靠性约定
- `operation_logs` 的定位是“业务追踪日志”，不是强审计账本
- 可靠性目标为“正常运行和正常关闭条件下尽量不丢；进程崩溃/机器断电场景允许少量丢失”
- 写入路径：业务线程投递到有界 channel，后台 goroutine 批量写库
- 默认参数：channel 容量 `1024`、单 worker、批大小 `64`、flush 间隔 `100ms`
- channel 满时降级为同步写库，并记录 `warn` 日志与 Prometheus counter
- 批量写失败时进行有限次重试（最多 3 次，指数退避）；重试失败后记录 `error` 日志和失败指标，不阻塞主业务返回
- 服务关闭时执行 flush，最多等待 3 秒；超时后放弃剩余日志并输出告警
- 该表不作为权限判断、业务回滚、账务核对的数据源；允许极低概率重复写入，查询端按 `id/created_at` 去重容忍

### 6.5 软删除约定
- `users` / `projects` / `tasks` 使用软删除：GORM `gorm.DeletedAt` 类型，查询自动 `WHERE deleted_at IS NULL`
- `task_comments` 物理删除；`project_members` 物理删除（成员关系本身就是可逆映射）
- 删除项目时，**不级联**删除任务和评论；改为标记项目 `deleted_at`，子表通过 join 过滤
- 跨表 join 时（如 task → project）必须显式带上 `projects.deleted_at IS NULL`，避免读到孤立任务

### 6.6 Owner 一致性约定
- `projects.owner_id` 是 owner 身份的**唯一真源**
- `project_members` 中仍保留一条 `role = owner` 的冗余成员记录，仅用于成员列表展示和统一权限遍历
- 创建项目时，必须在一个事务里同时写入：
  - `projects.owner_id = creator_id`
  - `project_members(project_id, user_id=creator_id, role=owner)`
- `TransferProjectOwnership` 必须在一个事务中同时更新：
  - `projects.owner_id`
  - 原 owner 的成员角色：`owner -> admin`
  - 新 owner 的成员角色：`admin/member -> owner`
- 权限判断以 `projects.owner_id` 为准；若发现它与 `project_members.role=owner` 不一致，视为数据异常并返回 `INTERNAL`

## 7. Proto 设计计划

### 7.1 `user.proto`
- `Register`
- `Login`
- `GetUser`
- `BatchGetUsers`

> 注：`Logout` **不在 `user.proto` 中定义**。登出语义是将当前 token 的 `jti` 写入 Redis 黑名单（TTL = 剩余有效期），整个操作在 `api-gateway` 内部完成，无需调用 `user-service`。`POST /api/v1/auth/logout` 由网关 handler 直接操作 Redis，不产生 RPC 调用。

### 7.2 `task.proto`
- 项目
  - `CreateProject`
  - `UpdateProject`（改名/改描述）
  - `ArchiveProject` / `UnarchiveProject`
  - `TransferProjectOwnership`
  - `ListProjects`
  - `GetProject`
- 成员
  - `AddProjectMember`
  - `RemoveProjectMember`
  - `UpdateProjectMemberRole`
  - `LeaveProject`（成员主动退出，owner 不可用）
  - `ListProjectMembers`
  - `CheckProjectMember`（数据归属本服务，由本服务对外提供）
- 任务
  - `CreateTask`
  - `UpdateTask`（统一编辑入口，覆盖 title/content/priority/due_time/extra）
  - `DeleteTask`
  - `GetTask`
  - `ListTasks`（游标分页，避免 offset 深翻页）
  - `AssignTask`（拆出，独立审计 + 幂等）
  - `ChangeTaskStatus`（拆出，校验状态流转表）
- 评论
  - `CreateTaskComment`
  - `DeleteTaskComment`
  - `ListTaskComments`
- 操作日志
  - `ListOperationLogs`（按 project / task 维度查询）

> 注：`AssignTask` / `ChangeTaskStatus` 与 `UpdateTask` 在字段上有重叠。**约定 `UpdateTask` 不允许修改 `assignee_id` 与 `status`**；这两个字段必须走专用接口，便于权限、状态机校验和操作日志埋点单点维护。

### 7.3 RPC 设计原则
- 第一版只做 unary RPC
- 所有请求必须带 `request_id`
- 所有 RPC 调用都设置 deadline（默认 2s，可按方法覆盖）
- 失败仅做一次幂等重试（仅对 `UNAVAILABLE`/`DEADLINE_EXCEEDED`），写接口必须依赖 `Idempotency-Key`
- 统一使用 gRPC status code 表达错误语义
- 网关负责将 gRPC 错误转换为 HTTP 响应
- **所有 RPC** 都必须校验 `x-internal-token`
- `Register`、`Login` 为匿名 RPC，可不带 `x-user-id`、`x-username`
- 其他 RPC 必须带 `x-user-id`、`x-username`，业务层从 metadata 读取其作为操作者身份
- 每个服务暴露 `grpc.health.v1.Health/Check` 与 reflection（仅 dev/local 环境）

## 8. HTTP API 设计计划

### 8.1 认证接口
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `GET /api/v1/users/me`

### 8.2 项目接口
- `POST /api/v1/projects`
- `GET /api/v1/projects`（返回“我参与的项目”列表，默认 `include_archived=false`）
- `GET /api/v1/projects/:id`
- `PUT /api/v1/projects/:id`（改名/改描述）
- `POST /api/v1/projects/:id/archive`
- `POST /api/v1/projects/:id/unarchive`
- `POST /api/v1/projects/:id/transfer`（owner 转让）
- `POST /api/v1/projects/:id/members`
- `GET /api/v1/projects/:id/members`
- `PUT /api/v1/projects/:id/members/:userId`（改角色）
- `DELETE /api/v1/projects/:id/members/:userId`
- `POST /api/v1/projects/:id/members/me/leave`（成员主动退出）
- `GET /api/v1/projects/:id/operation-logs`

### 8.3 任务接口
- `POST /api/v1/tasks`
- `GET /api/v1/tasks`（**`project_id` 为必填查询参数**；V1 不做跨项目聚合，缺失时返回 `INVALID_ARGUMENT`）
- `GET /api/v1/tasks/:id`
- `PUT /api/v1/tasks/:id`（仅可改 title/content/priority/due_time/extra）
- `DELETE /api/v1/tasks/:id`
- `POST /api/v1/tasks/:id/assign`
- `POST /api/v1/tasks/:id/status`
- `POST /api/v1/tasks/:id/comments`
- `GET /api/v1/tasks/:id/comments`
- `DELETE /api/v1/tasks/:id/comments/:commentId`
- `GET /api/v1/tasks/:id/operation-logs`

### 8.4 分页与排序约定
- `GET /api/v1/projects` 返回“我参与的项目”列表，包含 `owner/admin/member` 三种成员关系；默认 `include_archived=false`
- `GET /api/v1/projects` 使用 offset 分页（项目量少，每用户预期 <100）：`limit`（默认 20，最大 50）+ `offset`；按 `created_at DESC, id DESC` 排序
- `GET /api/v1/tasks` 使用游标分页，不使用 offset
- 排序固定为 `created_at DESC, id DESC`
- `limit` 默认 `20`，最大 `50`
- `cursor` 使用 base64url 编码的 JSON，包含：`created_at`、`id`、`filter_hash`
- `filter_hash` 由 `project_id/status/assignee_id/keyword` 等筛选条件计算得到；翻页请求的筛选条件变化时，服务端拒绝旧 cursor
- V1 不支持上一页/反向翻页
- `GET /api/v1/tasks/:id/comments` 按 `id ASC` 返回，评论量预期较小，V1 使用 `limit + after_id` 简化实现
- `GET /api/v1/projects/:id/operation-logs` 与 `GET /api/v1/tasks/:id/operation-logs` 使用游标分页：`limit`（默认 20，最大 100）+ `cursor`
- 操作日志固定按 `created_at DESC, id DESC` 排序；`cursor` 编码字段为 `created_at`、`id`

## 9. 中间件与通用组件计划

### 9.1 Gin 中间件
- Panic Recovery
- Request ID 注入
- 请求日志
- CORS（dev/local 放开，prod 白名单；白名单域名通过 Viper `cors.allowed_origins` 配置读取）
- IP 兜底限流（匿名 + 已登录都受约束）
- JWT 鉴权（仅保护接口）
- 用户级限流（依赖 user_id，挂在 JWT 之后）
- 参数校验错误转换（由 handler 产生后统一收敛，**不是单独的 bind/validate 中间件**）
- 统一响应封装
- 幂等组件（`Idempotency-Key` + Redis SETNX，仅写接口；**在 handler 完成 bind/validate 成功后调用**，不做全局中间件）

#### 中间件链固定顺序
`Recovery → RequestID → AccessLog → CORS → RateLimit(IP) → JWT → RateLimit(user) → Handler`

参数绑定与业务校验统一放在 Handler 内部完成；中间件只负责通用横切能力和错误收敛。写接口进入 Handler 后，先完成 bind/validate，再执行幂等检查，避免非法请求先占用 `Idempotency-Key`。

错序会导致 panic 不带 request_id、限流被认证绕过、未鉴权请求消耗用户配额，或无效请求污染幂等键等问题。

### 9.2 gRPC Interceptor
- Unary 日志拦截
- Trace 上下文透传（`otelgrpc`）
- Deadline 控制
- 错误码收敛
- metadata 提取与校验（`x-user-id`、`x-request-id`、`x-internal-token`）
- 健康检查与 reflection 注册（dev/local）

#### metadata 校验规则
- 对所有 RPC 强制校验：`x-internal-token`、`x-request-id`
- 对 `Register`、`Login` 放宽：允许缺省 `x-user-id`、`x-username`
- 对其他 RPC 强制要求：`x-user-id`、`x-username`
- interceptor 只做“字段存在性和内部口令”校验；业务权限判断仍在 service 层完成

### 9.3 公共基础组件
- 配置加载
- 日志初始化
- PostgreSQL 初始化（连接池默认：`MaxOpenConns=20` / `MaxIdleConns=5` / `ConnMaxLifetime=30m`，可配置覆盖）
- Redis 初始化
- JWT 工具
- bcrypt 密码哈希工具
- 错误码定义（Phase 0 即落地 `pkg/xerr`，禁止散落 `errors.New`）
- 响应结构定义
- Trace/Metric 初始化
- Graceful shutdown 工具（监听 SIGINT/SIGTERM → `http.Server.Shutdown` + `grpc.Server.GracefulStop`，超时 `10s` 后强杀）
- 健康检查 handler（HTTP `/healthz` + `/readyz`，gRPC `health.v1`）
- 迁移执行器（封装 `golang-migrate`，启动时可选自动迁移）

### 9.4 配置与密钥管理
- 配置来源：YAML 文件提供默认值 + 环境变量覆盖；环境变量优先级最高
- 必须通过环境变量注入的敏感项：`JWT_SECRET`、`INTERNAL_TOKEN`、`POSTGRES_PASSWORD`、`REDIS_PASSWORD`
- 仓库提供 `.env.example` 占位文件；`.env` 必须列入 `.gitignore`
- 启动时校验必填配置缺失立即 fail-fast，不允许带空 secret 起服务

### 9.5 错误响应结构
- HTTP 响应统一为 `{ code: string, message: string, request_id: string, data?: any }`
- `code` 使用英文常量，与 gRPC status 一一对应（如 `INVALID_ARGUMENT` / `NOT_FOUND` / `FAILED_PRECONDITION`）
- `message` 面向终端用户，V1 统一中文
- 校验错误额外携带 `details: [{ field, reason }]`

## 10. 详细开发阶段

### Phase 0: 项目初始化

#### 目标
把工程骨架、依赖、基础运行环境、CI 与可观测性底座一次性立起来，后续每个 Phase 不再回头补。

#### 任务
- 初始化 Go Module
- 创建目录结构
- 接入 Gin、gRPC、GORM、pgx、Redis、Zap
- 编写配置文件（local/dev/docker 三套）
- 编写 Makefile（`make proto` / `make run/<svc>` / `make test` / `make lint` / `make migrate`）
- 配置 `buf.yaml` + `buf.gen.yaml`，proto 用 buf 生成
- 编写 Docker Compose
- 落地 `pkg/xerr` 错误码骨架（先定义错误码常量表）
- 接入 `golangci-lint` 配置 + GitHub Actions workflow
- 接入覆盖率统计：通过脚本筛出纳入口径的业务包后执行 `go test -coverprofile=coverage.out ...`，CI 阈值先设为 `>= 80%`
- 各服务暴露 `/metrics`（先挂一个 dummy counter，把链路打通）
- 各服务实现 graceful shutdown 与 `/healthz` `/readyz`

#### 产出
- 可执行空服务
- `docker-compose up` 能启动 PostgreSQL 和 Redis
- `make proto`、`make run`、`make test`、`make lint`、`make migrate` 可用
- GitHub Actions 在 push 时跑通 lint + 测试
- `coverage.out` 可生成，CI 会对覆盖率阈值做失败拦截
- `curl /metrics` 能看到指标输出

### Phase 1: 用户服务

#### 目标
完成注册、登录、获取用户信息。

#### 任务
- 设计 `users` 表
- 使用 bcrypt(cost=10) 哈希存储密码，禁止明文或弱哈希
- 落地仓库内置弱密码词表并接入注册校验；词表随仓库提交，不依赖联网下载
- 实现 `Register` / `Login` / `GetUser` / `BatchGetUsers`（最多 100 个 user_id，结果命中 Redis 缓存）
- 实现 `POST /api/v1/auth/logout`：在网关 handler 内直接将 `jti` 写入 Redis 黑名单（TTL = token 剩余有效期），无需调用 `user-service`
- 实现 JWT 生成和校验（HS256，密钥从配置/环境变量读取）
- 实现 Redis token 黑名单，支持 logout 失效当前 access token
- 网关接入认证路由
- 补充基础单元测试（含 bcrypt、JWT、参数校验三类工具的单测）

#### 验收标准
- 能注册和登录
- 登录后能获取当前用户信息
- logout 后当前 token 立即失效
- 无 token 请求受保护接口会被拒绝

### Phase 2: 项目管理

#### 目标
完成项目创建、成员管理、归档与完整权限模型。

#### 任务
- 设计 `projects`、`project_members` 表
- 实现创建/修改/归档/取消归档/owner 转让
- 项目创建时默认写入 owner 成员关系
- 实现邀请成员（入参 `user_id`；写库前先调 `user-service.GetUser` 校验目标用户存在且 `status=active`）、移除成员、修改成员角色、成员主动退出
- 实现成员列表查询与“我参与的项目”列表查询（默认不含 archived，可通过 `include_archived=true` 打开）
- 实现 §3.3 完整权限矩阵校验（含 admin/member 的细分能力与边界）
- 实现项目归档的全员禁写与只读语义

#### 验收标准
- 登录用户可以创建项目
- 权限矩阵端到端覆盖：3 角色 × 关键操作集全部按 §3.3 通过/拒绝
- owner 可以转让，转让后自身降级为 admin
- 归档项目下所有写操作被拒绝且能正常读取

### Phase 3: 任务管理

#### 目标
完成核心任务流转闭环。

#### 任务
- 设计 `tasks` 表
- 实现创建、编辑（`UpdateTask`，禁改 status/assignee）、删除、详情、列表
- 支持状态、优先级、负责人、截止时间
- 实现游标分页（§8.4）与条件筛选
- 引入乐观锁字段 `version`，更新走 `WHERE version = ?` 并自增
- 实现 `AssignTask` 与 `ChangeTaskStatus`，独立审计日志与幂等
- `AssignTask` 前校验目标用户存在、active 且为当前项目成员；否则返回 `FAILED_PRECONDITION`
- 按 §3.3 状态流转表校验，非法转移返回 `FAILED_PRECONDITION`

#### 验收标准
- 能创建并查询任务
- 能按项目、状态、负责人筛选；游标翻页稳定
- 多次并发更新只有一次成功（乐观锁集成测试通过）
- 非法状态流转一律被拒绝并返回正确错误码
- 归档项目下所有任务写接口被拒绝

### Phase 4: 评论与操作日志

#### 目标
补齐业务真实感和可追溯性。

#### 任务
- 设计 `task_comments`、`operation_logs` 表
- 实现评论新增、删除、列表（评论按 §3.3 规则物理删除）
- 实现 §6.4 异步写入 worker（容量 1024 / 批 64 / flush 100ms / 单 worker）
- 对关键操作记录日志，**必须与 `operation_logs.action` 枚举一一对应**：
  - 任务类：`task.create` / `task.update` / `task.assign` / `task.status_change` / `task.delete`
  - 评论类：`comment.create` / `comment.delete`
  - 成员类：`member.add` / `member.remove` / `member.role_change` / `member.leave`
  - 项目类：`project.create` / `project.update` / `project.archive` / `project.unarchive` / `project.transfer_ownership`
- 实现 `ListOperationLogs`，按 project / task 维度查询，并支持 §8.4 定义的游标分页
- 实现 `GET /projects/:id/operation-logs` 与 `GET /tasks/:id/operation-logs` HTTP 接口
- 评论列表与操作日志列表需关联 `BatchGetUsers` 补全用户昵称/头像：网关在取得列表后，收集所有 `user_id`/`operator_id`，单次调用 `BatchGetUsers` 批量补全，再合并进响应，避免 N+1 RPC

#### 验收标准
- 任务下可查看评论
- 可通过 HTTP 接口查看项目/任务操作日志
- 异步 worker 在 channel 满时降级为同步写并打 warn 指标
- 服务关闭时 flush 在 3 秒内完成

### Phase 5: 工程化增强

#### 目标
让项目从“能跑”升级为“能交付、能讲”。

#### 任务
- 统一错误码（覆盖所有业务路径，禁止裸 `errors.New`）
- 完善日志格式，确认 request_id 字段贯穿（trace_id 留到 Phase 6 配合 OpenTelemetry 落地）
- Redis 缓存用户信息和项目详情，采用 cache-aside（读穿透 + 写时失效 + TTL 兜底）
- Redis token bucket 限流（按 user_id / IP）
- 关键写接口接入 `Idempotency-Key` 幂等
- 多环境配置（local/dev/docker）落地
- 增加接口测试和集成测试（用 `testcontainers-go` 起真实 PG/Redis）
- 完善 README

#### 验收标准
- 日志可追踪请求链路
- 热点数据查询命中缓存，且写入后能正确失效
- 同一 `Idempotency-Key` 重复提交只生效一次
- README 能指导他人启动和测试

### Phase 6: 可观测性与展示能力

#### 目标
增加面试展示亮点。`/metrics`、健康检查在 Phase 0 已经就位，本阶段做指标丰富与对外展示。

#### 任务
- 补齐 HTTP/gRPC 请求耗时直方图、错误率计数器、DB/Redis 调用指标
- 接 OpenTelemetry Trace（gateway → service → DB 全链路），补齐 trace_id 字段贯穿与日志串联
- 增加 Grafana 仪表盘并截图存档（RED 指标 + DB 慢查询 + 缓存命中率）
- 用 `k6` 或 `vegeta` 编写压测脚本，定义并验证 SLO（建议：单机 1k QPS，登录接口 P99 < 100ms）
- 整理压测报告（QPS / P50 / P99 / 错误率 / 资源占用）

#### 验收标准
- 能看到 HTTP 和 gRPC 请求耗时指标
- 能展示一次完整请求链路，并通过 trace_id 串联相关日志
- 有明确 SLO 与压测数据、瓶颈分析结论

## 11. 里程碑安排

> 原则：测试随每个 Phase 同步写（TDD），不堆到最后；监控/健康检查在 Week 1 就接通。

### Week 1
- 完成项目初始化（Phase 0）
- 跑通 PostgreSQL、Redis、Gin、gRPC 基础骨架
- CI、`/metrics`、`/healthz`、graceful shutdown、错误码骨架就位

### Week 2
- 完成用户服务（Phase 1，含单元 + 集成测试）
- 完成 JWT 鉴权

### Week 3
- 完成项目管理与成员权限（Phase 2，含权限矩阵测试）

### Week 4
- 完成任务管理核心流程（Phase 3，含乐观锁并发测试、游标分页）
- 顺带接入缓存与限流（前置 Phase 5 中较轻的两项，避免 Week 5 过载）

### Week 5
- 完成评论、操作日志（Phase 4，含异步 worker 与日志查询接口）
- 完成幂等、错误码全量收敛、多环境配置、接口与集成测试（Phase 5 剩余）

### Week 6
- 丰富指标与 trace、压测、Grafana 截图（Phase 6）
- 完成 README、架构图、面试讲解稿、简历输出

## 12. 每阶段交付清单

### 代码交付
- 可运行代码
- 配置文件
- SQL migration
- proto 文件
- Docker Compose 文件

### 文档交付
- README
- 架构图
- 表结构说明
- 接口文档（Postman Collection，演示友好；也可同时导出 OpenAPI/Swagger）
- 压测报告
- 面试讲解提纲

## 13. 测试计划

### 13.1 单元测试
- JWT 工具测试
- bcrypt 密码加密和校验测试
- 弱密码词表校验测试（命中词表、大小写归一、边界长度）
- 任务状态流转规则测试（合法/非法转移矩阵）
- 权限判断测试（owner/admin/member × 各操作）
- 错误码映射（gRPC status → HTTP code）测试

### 13.2 集成测试（基于 `testcontainers-go`，每用例或每测试套件独立 PostgreSQL/Redis 测试实例）
- 生产环境仍固定使用 `user_svc` / `task_svc` 两个 schema；测试环境也保持这两个 schema 名，不再额外引入“每用例独立 schema”约定
- 推荐做法：每个集成测试用例或测试套件启动独立 PostgreSQL/Redis 容器；若为提速复用容器，则至少保证独立 database，并在用例结束后清理
- 用户注册登录流程
- 创建项目并添加成员
- 创建任务并分配负责人
- 评论和日志写入
- 乐观锁并发更新（同一任务并发 update，仅一个成功）
- 幂等：同一 `Idempotency-Key` 重复提交只生效一次
- 缓存：写入后下次读取走 DB 且回填缓存
- token 黑名单：logout 后同 token 再次访问被拒
- 归档项目：所有写接口返回 `FAILED_PRECONDITION`，读接口正常
- owner 转让：转让前后角色与权限矩阵均正确
- BatchGetUsers：边界（空入参、单次 100 上限、超额拒绝）
- 指派非成员或已退出成员应返回 `FAILED_PRECONDITION`（需覆盖：被移除成员、主动退出成员、disabled 用户三种子场景）

### 13.3 接口测试
- 使用 `httptest` 测试 Gin Handler
- 使用本地测试数据库执行服务层集成测试
- 关键并发路径加 `-race` 跑

### 13.4 覆盖率口径
- 覆盖率统计命令：通过脚本筛选 `internal/`、`pkg/` 等纳入口径的业务包后执行 `go test -coverprofile=coverage.out ...`
- 统计口径以 Go 原生 statement coverage 为准
- CI 最低门槛为 `80%`
- 覆盖率主要约束 `internal/` 与 `pkg/` 核心业务代码；生成代码、`cmd/` 启动胶水代码、`mocks/` 不纳入强约束

### 13.5 性能/压测
- 工具：`k6` 或 `vegeta`
- 场景：登录、ListTasks 热点查询、CreateTask 写入
- 目标 SLO：单机 1k QPS，登录 P99 < 100ms，写入 P99 < 200ms（可按真实结果调整）

## 14. 简历输出计划

### 14.1 简历描述方向
- 基于 `Gin + gRPC + PostgreSQL + Redis` 实现团队任务协作平台后端
- 采用 `API Gateway + 微服务` 架构，完成用户服务与任务服务拆分
- 使用 `JWT + RBAC` 实现项目级权限控制（3 角色 × 操作矩阵，含 token 黑名单登出）
- 设计乐观锁、游标分页、幂等写接口、cache-aside 缓存失效等工程亮点
- 使用 Prometheus 和 OpenTelemetry 完成指标监控与链路追踪
- 测试覆盖率 ≥80%，集成测试基于 `testcontainers-go` 运行真实 PG/Redis

### 14.2 面试重点准备
- 为什么外部用 HTTP，内部用 gRPC
- 为什么拆成 `user-service` 和 `task-service`（以及不拆成更多服务的取舍）
- 单 PostgreSQL 多 schema 的拆法收益与代价
- PostgreSQL schema 如何设计
- JWT、权限、缓存、乐观锁如何实现
- 缓存一致性：cache-aside vs 双写 vs 延迟双删
- 幂等 `Idempotency-Key` 的实现细节与失效边界
- 限流为什么选 Redis token bucket，与漏桶/计数器的对比
- `operation_logs` 为什么异步写，channel 满如何降级
- gRPC 调用的 deadline / 重试 / 错误码收敛策略
- request_id 与 trace_id 的串联方式
- 如何做日志、监控、压测；SLO 数字与瓶颈点
- 为什么 GORM 不用 `sqlc`（迭代速度 vs 类型安全的取舍）
- token 黑名单 vs 完全无状态 JWT 的取舍
- cursor 分页的实现细节与 filter_hash 防跨筛选篡改
- 单 worker vs 多 worker 异步写日志的取舍
- 中间件链顺序的设计意图（panic 带 request_id、限流先于鉴权）
- 静态 `x-internal-token` 的风险与 V2 演进路径（mTLS / 短期 token）

## 15. 风险与规避

### 15.1 风险点
- 一开始服务拆分过细，导致开发复杂度过高
- 微服务收益不足：单 DB 多 schema 并未带来独立部署/扩容，但增加了 gRPC 样板代码
- proto、数据库、HTTP 接口三者不同步
- 监控和测试拖到最后，最终来不及补
- 业务功能过多，影响主线交付
- 错误码后期补，散落 `errors.New` 导致返回不一致

### 15.2 规避策略
- 严格控制为 2 个核心服务
- 在面试讲解中主动承认"拆分收益有限"，把它当作判断题而非默认答案
- 先完成最小主流程，再加亮点
- 每个 Phase 结束都做一次可运行验收
- 所有新增功能优先检查是否影响主流程
- Phase 0 即落地错误码骨架，新增错误必须走 `pkg/xerr`

## 16. 最终验收标准

- 本地通过 Docker Compose 一键启动
- 完整跑通注册、登录、登出、创建/修改/归档项目、转让 owner、添加/移除/改角色/退出成员、创建/编辑/分配/删除任务、状态流转、评论与删除评论、操作日志查询
- 权限矩阵端到端可演示（3 角色 × 关键操作集全部按 §3.3 通过/拒绝）
- API Gateway、User Service、Task Service 可以独立启动
- PostgreSQL migration 清晰可复现
- 压测报告达成 §13.5 设定的 SLO（或附偏差与瓶颈分析）
- README 覆盖启动、接口、架构、测试、压测
- 项目具备可讲述的工程亮点，而不仅是 CRUD

## 17. 建议的第一步执行顺序

### Phase 0（骨架）
1. 初始化 Go Module，创建目录结构（含 `.github/workflows/`、`.env.example`）
2. 编写 `docker-compose.yml`，先启动 PostgreSQL 和 Redis
3. 配置 `buf.yaml` + `buf.gen.yaml`，定义 `user.proto` 与 `task.proto` 骨架并生成代码
4. 搭建 `api-gateway`、`user-service`、`task-service` 空服务（能编译、能启动）
5. 落地 `pkg/xerr` 错误码表、`golangci-lint` 配置、GitHub Actions workflow
6. 各服务接入 `/metrics`（dummy counter）、`/healthz`、`/readyz`、graceful shutdown

### Phase 1–3（主流程）
7. 打通 `Register → Login → Logout → Me`（含 JWT、token 黑名单、`BatchGetUsers`）
8. 打通 `CreateProject → AddMember → ArchiveProject → TransferOwnership`（含完整权限矩阵）
9. 打通 `CreateTask → AssignTask → ChangeTaskStatus → ListTasks`（含乐观锁、游标分页）

### Phase 4–6（工程亮点）
10. 补评论、操作日志（含异步 worker）、缓存失效、限流、幂等
11. 丰富指标与 trace、压测、Grafana 截图
12. 完善 README、Postman Collection、面试讲解稿

---

这个计划以“校招项目可落地、可交付、可讲解”为目标。真正实现时，优先保证主流程闭环，其次再补工程亮点。
