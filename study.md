# 团队任务协作平台 - Go 后端学习路线

> 适用对象：Go 新手，已有基本编程经验（了解变量、函数、循环、条件判断等概念），希望
> 通过一个完整的生产级后端项目系统学习 Go 语言和工程实践。

---

## 第一部分：Go 语言基础速览（1-2 天）

在深入项目之前，先建立 Go 语言的基本认知。以下概念会贯穿整个项目。

### 1.1 必知概念

| 概念 | 一句话解释 | 项目中的体现 |
|------|-----------|-------------|
| **包 (package)** | Go 的代码组织单元，一个目录一个包 | `internal/user/biz`、`pkg/xerr` 等 |
| **接口 (interface)** | 定义行为契约，由实现者隐式满足 | `UserRepository` 接口定义在 `data` 包，实现在同包 |
| **结构体 (struct)** | 数据的组合，类似其他语言的"对象"但更轻量 | `UserBiz`、`TaskBiz`、`AuthHandler` 等 |
| **方法** | 绑定在类型上的函数 | `func (b *UserBiz) Register(...)` |
| **指针 (`*T`)** | 引用传递，Go 中 struct 默认传指针 | 几乎所有 struct 方法接收者都是指针 |
| **错误处理** | 通过返回值而非异常，`if err != nil` 无处不在 | 每个函数几乎都返回 `error` |
| **零值** | 变量声明后自动初始化为"空"值（int=0, string="", pointer=nil） | GORM 的 `DeletedAt` 用零值表示未删除 |
| **`context.Context`** | 携带请求级元数据（超时、取消信号、值）的"背包" | 每个业务方法第一个参数都是 `ctx context.Context` |
| **`defer`** | 延迟执行，用于资源清理 | `defer cleanup()`、`defer logger.Sync()` |
| **goroutine** | Go 的轻量级并发单元 | 操作日志异步写入 worker |
| **channel** | goroutine 间通信的管道 | `make(chan *data.OperationLog, 1024)` |
| **`_` 导入** | 仅执行包的 `init()` 函数，不使用包内导出符号 | `import _ "gorm.io/driver/postgres"` 注册 PG 驱动 |

### 1.2 推荐入门资源

1. **Go 官方 Tour**（`go.dev/tour`）：2-3 小时交互式教程
2. **Effective Go**（`go.dev/doc/effective_go`）：Go 编程风格指南
3. **Go by Example**（`gobyexample.com`）：常用模式速查

### 1.3 先读懂这 10 行代码

打开 `cmd/api-gateway/main.go`，逐行理解：

```go
package main          // ✅ Go 的入口包必须是 main

import (...)          // ✅ 导入依赖

func main() {         // ✅ 程序入口函数
    if err := run(); err != nil {
        panic(err)    // ✅ 启动阶段用 panic 快速失败
    }
}

func run() error {    // ✅ 真正的逻辑在 run() 中，返回 error
    // ...
}
```

**关键点：**
- Go 程序的入口永远是 `package main` 中的 `func main()`
- 惯用模式：`main()` 只做简单的错误处理包装，实际逻辑放 `run() error` 中
- `panic` 仅用于不可恢复的启动错误

---

## 第二部分：项目总览 - 一张图看懂架构（半天）

### 2.1 物理架构

```
浏览器/Postman
     │  HTTP/JSON (Gin)
     ▼
┌─────────────────┐
│  api-gateway    │  ← 对外的唯一入口，约 12 个 handler
│  (Gin 框架)     │
└───┬─────────┬───┘
    │  gRPC   │  gRPC
    ▼         ▼
┌────────┐ ┌────────┐
│ user   │ │ task   │  ← 内部微服务，外界不可直接访问
│ service│ │ service│
└───┬────┘ └───┬────┘
    │          │
    ▼          ▼
┌───────────────────┐
│   PostgreSQL 16   │  ← 一个实例，两张 schema
│ user_svc│task_svc │
└───────────────────┘
┌───────────────────┐
│     Redis 7       │  ← 缓存 + 限流 + 黑名单 + 幂等
└───────────────────┘
```

### 2.2 目录结构与职责

阅读路线按**依赖从底向上**的顺序，这也是学习的推荐顺序：

```
pkg/xerr    →  错误码定义（全项目的基础依赖）
pkg/xjwt    →  JWT 工具
pkg/xlog    →  日志封装
pkg/xredis  →  Redis 客户端封装
pkg/xpgsql  →  PostgreSQL 客户端封装
pkg/xgrpc   →  gRPC 服务端/客户端通用拦截器
pkg/xcursor →  游标分页编码
pkg/xratelimit →  令牌桶限流

internal/user/data/   →  GORM 数据模型 + 数据库访问接口
internal/user/biz/    →  核心业务逻辑（注册、登录、密码校验）
internal/user/service/→  gRPC 服务实现（把 biz 结果转成 proto 结构）
internal/user/server/ →  依赖注入 + gRPC 服务启动

internal/task/data/   →  GORM 模型（Project/Task/Comment/Log）
internal/task/biz/    →  业务逻辑（权限、状态机、操作日志异步写）
internal/task/service/→  gRPC 服务实现
internal/task/server/ →  依赖注入 + gRPC 服务启动

internal/gateway/rpc/      →  gRPC 客户端工厂（含 metadata 拦截器）
internal/gateway/middleware/→  Gin 中间件链（认证、限流、日志等）
internal/gateway/handler/  →  HTTP handler（参数绑定 → 调 RPC → 统一响应）
internal/gateway/server/   →  路由注册 + 依赖注入 + 中间件编排

cmd/api-gateway/main.go    →  网关启动入口（配置加载、优雅关闭）
cmd/user-service/main.go   →  user-service 启动入口
cmd/task-service/main.go   →  task-service 启动入口
```

### 2.3 各服务的四层架构

每个业务服务遵循统一的四层模式：

```
server ─→ service ─→ biz ─→ data
(启动)     (gRPC)    (业务)  (数据库)
```

| 层 | 职责 | 关键 Go 模式 |
|----|------|-------------|
| `data/` | 定义 GORM 模型 + Repository 接口 + 实现 | **接口隐式实现**：`userRepo` 不声明 `implements`，只要方法签名匹配就算实现了 `UserRepository` |
| `biz/` | 纯业务逻辑，依赖 `data` 层的**接口**（不依赖具体实现） | **依赖注入**：通过构造函数接收接口，方便单元测试 mock |
| `service/` | gRPC 方法实现，`biz → proto` 的数据转换层 | **嵌入未实现的服务**：`userv1.UnimplementedUserServiceServer` 保证向前兼容 |
| `server/` | 依赖注入的"组装车间"，启动 gRPC 服务器 | **配置校验 fail-fast**：启动时检查所有必需配置和密钥 |

---

## 第三部分：Phase 0 学习 - 项目骨架（2-3 天）

这是理解 Go 项目工程结构的最佳入口点。**不要跳过，不要急着写业务代码。**

### 3.1 学习目标

- 理解 Go Module 和依赖管理
- 理解项目目录约定（`cmd/`、`internal/`、`pkg/`）
- 理解配置管理（Viper + 环境变量覆盖）
- 理解优雅关闭（graceful shutdown）

### 3.2 精读文件清单

**从入口开始（按启动顺序读）：**

1. **`cmd/api-gateway/main.go`**（146 行）
   - 重点理解：
     - `func run() error` 惯用模式
     - `signal.NotifyContext` 监听系统信号
     - `atomic.Bool` 就绪标志位
     - `http.Server.Shutdown` 优雅关闭
     - Viper 配置加载（YAML 文件 + `AutomaticEnv` 环境变量覆盖）

2. **`cmd/user-service/main.go`**（178 行）
   - 额外重点：
     - gRPC 的 `GracefulStop()` vs `Stop()`
     - `select` 多路选择（等待信号或 goroutine 结果）
     - 同时启动 gRPC + Admin HTTP 两个服务器的模式

3. **`pkg/xerr/codes.go`**（199 行）
   - 重点理解：
     - 自定义 `Error` 类型（`Code` + `Message`）
     - 实现 `GRPCStatus()` 方法使 `xerr.Error` 可以被 gRPC 框架识别
     - gRPC status code ↔ HTTP status code 的映射表
     - `errors.As` 错误类型断言

4. **`pkg/xhttp/server.go`** 和 **`pkg/xgrpc/server.go`**
   - 健康检查 `/healthz`、`/readyz`
   - Prometheus `/metrics`
   - gRPC reflection 和 health check 注册

5. **`Makefile`**（80 行）
   - `make run/<service>` 的模式
   - 通过环境变量控制构建行为

6. **`go.mod`** - Go Module 定义文件
   - `module task-platform` 声明模块路径（代码中所有 import 都以它开头）
   - `require (...)` 声明直接依赖（Gin、gRPC、GORM、Redis 等）
   - `indirect` 标记间接依赖（由直接依赖引入的包）

### 3.3 动手练习

- [ ] 在本地运行 `make up` 启动 PG + Redis
- [ ] 运行 `make run/api-gateway` 启动网关
- [ ] 用 `curl /healthz` 验证服务存活
- [ ] 用 `curl /readyz` 验证就绪状态
- [ ] 用 `curl /metrics` 查看 Prometheus 指标
- [ ] 按 `Ctrl+C` 观察优雅关闭日志
- [ ] 阅读 `go.mod` 理解项目依赖

### 3.4 Phase 0 核心 Go 知识点

```go
// 1. atomic.Bool - 无锁的布尔标志
ready := &atomic.Bool{}
ready.Store(true)   // 写入
ready.Load()        // 读取

// 2. signal.NotifyContext - 优雅关闭的信号监听
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()
<-ctx.Done()  // 阻塞直到收到信号

// 3. errCh + goroutine + select 模式
errCh := make(chan error, 1)
go func() { errCh <- server.ListenAndServe() }()
select {
case err := <-errCh:  // 服务异常退出
case <-ctx.Done():    // 收到关闭信号
}

// 4. context.WithTimeout 超时控制
shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
server.Shutdown(shutdownCtx)

// 5. 环境变量 + 默认值 辅助函数（贯穿全项目的模式）
func envOrDefault(key, defaultVal string) string {
    if v := os.Getenv(key); v != "" { return v }
    return defaultVal
}
```

---

## 第四部分：Phase 1 学习 - 用户服务（3-4 天）

这是理解**四层架构 + 依赖注入 + 接口设计**的核心阶段。

### 4.1 学习目标

- 理解 Go 接口的设计哲学（小接口、隐式实现）
- 理解依赖注入的手动实现（不使用框架）
- 理解 GORM 的基本用法
- 理解 JWT 的签发与校验
- 理解 bcrypt 密码哈希

### 4.2 精读文件清单（按依赖顺序从底向上）

**Layer 1: data（数据访问层）**

1. **`internal/user/data/model.go`**
   - GORM 模型定义（struct tag `gorm:"column:..."`）
   - `TableName()` 方法覆盖默认表名
   - `gorm.DeletedAt` 软删除

2. **`internal/user/data/repository.go`**（89 行）
   - **接口定义** `UserRepository`（5 个方法）
   - 具体实现 `userRepo`（小写开头 = 包外不可见）
   - 构造函数 `NewUserRepository`（返回接口类型）
   - 原生 SQL 示例（`UNION ALL` 联合查询）
   - 错误处理：`gorm.ErrRecordNotFound` → 业务错误码

**Layer 2: biz（业务逻辑层）**

3. **`internal/user/biz/user.go`**（249 行）
   - 正则校验（`regexp.MustCompile` 在包级别编译一次）
   - `bcrypt.GenerateFromPassword`（cost=10，生成哈希）
   - `bcrypt.CompareHashAndPassword`（验证密码）
   - `bcrypt.Cost`（检查是否需要重哈希）
   - **cache-aside 模式**：先查 Redis，miss 再查 DB，回填缓存
   - `json.Marshal/Unmarshal` 序列化用户对象存 Redis

**Layer 3: service（gRPC 服务层）**

4. **`internal/user/service/service.go`**（90 行）
   - 嵌入 `userv1.UnimplementedUserServiceServer`（proto 向前兼容关键）
   - `biz → proto` 的数据转换函数 `toProtoUser`
   - 薄层：几乎只有参数转发，逻辑都在 biz

**Layer 4: server（组装启动层）**

5. **`internal/user/server/server.go`**（218 行）
   - **依赖注入**：`NewUserRepository → NewUserBiz → NewUserService` 的组装链
   - gRPC 拦截器链：
     - `otelgrpc.NewServerHandler()` - 分布式追踪
     - `UnaryServerMetricsInterceptor()` - Prometheus 指标
     - `loggingInterceptor` - 请求日志
     - `authInterceptor` - 内部令牌校验
   - `metadata.FromIncomingContext` 提取 gRPC 元数据
   - `subtle.ConstantTimeCompare` 防时序攻击比较内部令牌

**横向：Gateway 侧**

6. **`internal/gateway/handler/auth.go`**（106 行）
   - Gin 的 `ShouldBindJSON` 参数绑定
   - 幂等检查（`SetupIdempotency` + `defer cleanup()`）
   - `handleGRPCError` 将 gRPC error 转为 HTTP 响应
   - Logout 直接在网关写入 Redis 黑名单，不调 RPC

7. **`internal/gateway/middleware/auth.go`**（71 行）
   - Gin 中间件标准签名：`func(c *gin.Context)`
   - `c.Next()` 放行 / `c.AbortWithStatusJSON()` 拦截
   - `context.WithValue` 注入用户信息到请求上下文
   - token 黑名单检查（`rdb.Exists`）

8. **`internal/gateway/rpc/client.go`**（83 行）
   - gRPC 客户端拦截器注入 metadata（`x-user-id`, `x-username`, `x-request-id`, `x-internal-token`）
   - `grpc.WithTransportCredentials(insecure.NewCredentials())` 本地开发明文通信
   - `otelgrpc.NewClientHandler()` 客户端侧追踪

### 4.3 动手练习

- [ ] 用 Postman/curl 注册一个用户
- [ ] 用注册的 token 获取 `/api/v1/users/me`
- [ ] 调用 `/api/v1/auth/logout`，再用旧 token 访问确认被拒绝
- [ ] 阅读 `pkg/xjwt/jwt.go`（56 行），理解 JWT 生成和验证流程
- [ ] 阅读 `internal/user/biz/user_test.go`，理解 mock 接口的测试模式
- [ ] 在 `internal/user/biz/user.go` 中跟踪一个 `Register` 的完整调用链

### 4.4 Phase 1 核心 Go 知识点

```go
// 1. 接口定义与实现（隐式满足）
type UserRepository interface {          // 定义在 data 包
    Create(ctx context.Context, user *User) error
}
type userRepo struct { db *gorm.DB }    // 私有实现
var _ UserRepository = (*userRepo)(nil)  // 编译期接口满足检查（项目在 task/server/server.go:251 使用了此模式）
func NewUserRepository(db *gorm.DB) UserRepository { return &userRepo{db: db} }

// 2. 函数选项模式（未在项目中使用但需了解）vs 直接传参
// 本项目用简单的构造函数传参：NewUserBiz(repo, rdb, passwords)

// 3. context.Context 的两种用法
ctx := context.Background()        // 无父上下文时使用
ctx = context.WithValue(ctx, key, value)  // 携带值
ctx, cancel := context.WithTimeout(ctx, 3*time.Second)  // 超时控制
defer cancel()

// 4. 错误处理的链式传播
user, err := repo.FindByAccount(ctx, account)
if err != nil {
    return nil, xerr.NewError(xerr.CodeUnauthenticated, "invalid account")
}

// 5. Gin 中间件模式
func Auth(...) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 前置处理
        if !valid {
            c.AbortWithStatusJSON(401, ...)
            return
        }
        c.Next()  // 放行到下一个中间件/handler
    }
}
```

---

## 第五部分：Phase 2 学习 - 项目管理与权限（3-4 天）

### 5.1 学习目标

- 理解 GORM 事务（多表原子写入）
- 理解 RBAC 权限模型在代码中的实现
- 理解跨服务 gRPC 调用（task-service 调 user-service 校验用户）
- 理解 Redis cache-aside 模式在项目中的应用

### 5.2 精读文件清单

1. **`internal/task/data/model.go`**（134 行）
   - 5 个 GORM 模型：`Project`、`ProjectMember`、`Task`、`TaskComment`、`OperationLog`
   - 命名常量代替魔法数字：`RoleOwner=0`, `TaskStatusTodo=0`
   - `*string` 可空字段（`AssigneeID`, `DueTime`）
   - `TableName()` 指定 schema 前缀：`task_svc.projects`

2. **`internal/task/data/repository.go`**
   - 多个 Repository 接口（ProjectRepository, MemberRepository, TaskRepository, CommentRepository, OperationLogRepository）
   - 乐观锁更新（`WHERE version = ?`）
   - 游标分页查询（`WHERE (created_at, id) < (?, ?)`）

3. **`internal/task/biz/project.go`**
   - **GORM 事务**：`db.Transaction(func(tx *gorm.DB) error { ... })` 创建项目同时写入 owner 成员记录
   - **乐观锁**：更新时带 `version` 条件
   - **权限检查**：在每个操作前校验调用者角色
   - **操作日志**：`logWriter.Enqueue(ctx, log)` 异步写入

4. **`internal/task/service/service.go`**
   - gRPC metadata 提取用户身份（`extractUserID(ctx)`, `extractUsername(ctx)`）
   - 跨服务调用：`userClient.GetUser` 校验目标用户存在

### 5.3 动手练习

- [ ] 创建项目、邀请成员，验证 owner/admin/member 三个角色的权限差异
- [ ] 测试归档项目的读写行为
- [ ] 测试转让 owner 后原 owner 角色变化
- [ ] 阅读 `internal/task/biz/project.go` 中的 `TransferProjectOwnership` 理解事务用法
- [ ] 跟踪一个 `AddProjectMember` 的完整调用链（gateway → task-service → user-service）

### 5.4 Phase 2 核心 Go 知识点

```go
// 1. GORM 事务 - 通过传入事务 tx 创建新的 Repository 来复用方法
err := b.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    // 在事务内用 tx（而非原始 db）创建 Repository，操作自动走事务
    if err := data.NewProjectRepository(tx).Create(ctx, p); err != nil { return err }
    if err := data.NewMemberRepository(tx).Add(ctx, m); err != nil { return err }
    return nil  // nil = 提交，非 nil = 回滚
})

// 2. 乐观锁更新
result := db.Model(&Task{}).
    Where("id = ? AND version = ?", id, oldVersion).
    Updates(map[string]any{"title": newTitle, "version": oldVersion + 1})
if result.RowsAffected == 0 {
    return xerr.NewError(xerr.CodeAborted, "task was modified, please retry")
}

// 3. 接口用于跨包依赖解耦
// task/biz 不直接导入 user 服务的 gRPC client，而是定义自己的小接口
type UserServiceClient interface {
    GetUser(ctx context.Context, userID string) (exists bool, active bool, err error)
}
```

---

## 第六部分：Phase 3 学习 - 任务管理（3-4 天）

### 6.1 学习目标

- 理解状态机（state machine）在代码中的实现
- 理解游标分页（cursor pagination）的原理和编码
- 理解条件筛选 + 分页的组合查询

### 6.2 精读文件清单

1. **`internal/task/biz/task.go`**
   - 状态流转校验（`validTransitions` map）
   - `AssignTask` 的指派前多条件校验
   - `ChangeTaskStatus` 的状态机检查

2. **`pkg/xcursor/cursor.go`**
   - base64url 编解码
   - `filter_hash` 防跨筛选篡改
   - JSON 序列化 cursor 结构

### 6.3 Phase 3 核心 Go 知识点

```go
// 1. 状态机 = map + 查找
var validTransitions = map[int32][]int32{
    Todo:  {Doing, Done, Cancelled},
    Doing: {Done, Cancelled, Todo},
}
func isValid(from, to int32) bool {
    for _, valid := range validTransitions[from] {
        if valid == to { return true }
    }
    return false
}

// 2. 游标分页查询
// SQL: WHERE (created_at, id) < (?, ?) ORDER BY created_at DESC, id DESC LIMIT ?
// cursor = base64url(json{created_at, id, filter_hash})
```

---

## 第七部分：Phase 4 学习 - 评论与操作日志（2-3 天）

### 7.1 学习目标

- 理解 Go 的 goroutine + channel 异步处理模式
- 理解 worker pool 的简化版（单 worker + 有界 channel）
- 理解 Prometheus 指标的定义和注册

### 7.2 精读文件清单

1. **`internal/task/biz/log_writer.go`**（188 行）-**本项目的并发编程教科书**
   - goroutine 生命周期管理
   - channel 容量设计（1024）
   - `select` 多路复用（channel + ticker + done）
   - `atomic.Bool` 优雅关闭标志
   - `sync.WaitGroup` 等待 worker 退出
   - channel 满时的降级策略（同步写 + 打 warn 日志）
   - `recover()` 防止 worker panic 导致整个进程崩溃
   - Prometheus counter 注册（`sync.Once` 保证只注册一次）

### 7.3 Phase 4 核心 Go 知识点

```go
// 1. goroutine + channel 的完整生命周期
type LogWriter struct {
    ch     chan *OperationLog     // 有界 channel
    done   chan struct{}          // 信号 channel（空 struct 不占内存）
    closed atomic.Bool            // 原子标志位
    wg     sync.WaitGroup         // 等待 goroutine 退出
}

func NewLogWriter() *LogWriter {
    w := &LogWriter{
        ch:   make(chan *OperationLog, 1024),  // 缓冲=容量
        done: make(chan struct{}),
    }
    w.wg.Add(1)
    go w.run()                    // 启动后台 goroutine
    return w
}

// 2. select 多路复用
func (w *LogWriter) run() {
    batch := make([]*OperationLog, 0, 64)
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    for {
        select {
        case <-w.done:            // 收到关闭信号
            flush(batch); return
        case log := <-w.ch:       // 收到日志
            batch = append(batch, log)
            if len(batch) >= 64 { flush(batch); batch = batch[:0] }
        case <-ticker.C:          // 定时刷新
            if len(batch) > 0 { flush(batch); batch = batch[:0] }
        }
    }
}

// 3. 非阻塞发送（select + default）
func (w *LogWriter) Enqueue(ctx context.Context, log *OperationLog) {
    select {
    case w.ch <- log:       // 正常入队
    default:                // channel 满了，降级
        logWriterChannelFull.Inc()
        w.writeSync(ctx, []*OperationLog{log})
    }
}

// 4. recover 保护 goroutine（worker panic 不会杀死进程）
defer func() {
    if r := recover(); r != nil {
        logWriterWorkerPanic.Inc()
        if !w.closed.Load() { go w.run() }  // 自动重启
    }
}()

// 5. sync.Once 保证只注册一次 Prometheus 指标
var metricsOnce sync.Once
metricsOnce.Do(func() { prometheus.MustRegister(counter) })
```

---

## 第八部分：Phase 5 学习 - 工程化增强（3-4 天）

### 8.1 学习目标

- 理解 Redis 在项目中的多种用途
- 理解中间件链的设计意图
- 理解幂等的实现原理

### 8.2 精读文件清单

1. **`internal/gateway/middleware/ratelimit.go`**（131 行）
   - 双层限流：IP 限流 + 用户级限流
   - 认证接口使用更严格阈值
   - Redis 错误时"fail-open"（放行不阻断）

2. **`internal/gateway/handler/idempotency.go`**（101 行）
   - `SetNX` 原子的"抢锁"语义
   - `bodyCaptureWriter` 拦截响应 body 缓存
   - 失败请求清理 key（不放垃圾数据）

3. **`pkg/xratelimit/token_bucket.go`**（60 行）
   - Redis Lua 脚本做原子令牌桶
   - 令牌桶算法：`tokens = min(capacity, tokens + elapsed * rate)`

4. **`internal/gateway/middleware/`** 其他中间件
   - `requestid.go` - Request ID 生成/提取/回溯
   - `accesslog.go` - Zap 结构化日志
   - `cors.go` - 跨域配置
   - `metrics.go` - HTTP 请求直方图 + 计数器

### 8.3 核心 Go 知识点

```go
// 1. Gin 中间件链是洋葱模型
// Recovery → RequestID → Log → CORS → RateLimit(IP) → Auth → RateLimit(user) → Handler
// 每个中间件通过 c.Next() 控制执行流

// 2. Redis Lua 脚本（原子操作）
var script = redis.NewScript(`
    -- 在 Redis 服务端原子执行这段 Lua
    local tokens = redis.call('HMGET', key, 'tokens', 'last_refill')
    -- ... 令牌桶计算逻辑 ...
    return allowed
`)
script.Run(ctx, rdb, []string{key}, args...)

// 3. 幂等的核心：SETNX
ok, _ := rdb.SetNX(ctx, key, "pending", 24*time.Hour).Result()
if !ok {  // key 已存在，是重复请求
    return cachedResult
}
// 首次请求，执行业务逻辑，写入结果
```

---

## 第九部分：Phase 6 学习 - 可观测性（2-3 天）

### 9.1 学习目标

- 理解 OpenTelemetry 的 Span 概念
- 理解 Prometheus 的指标类型（Counter / Histogram）
- 理解结构化日志（Zap JSON）的字段约定

### 9.2 精读文件清单

1. **`pkg/xtrace/trace.go`** - OpenTelemetry 初始化
2. **`pkg/xgrpc/metrics.go`** - gRPC 请求指标
3. **`pkg/xpgsql/metrics.go`** - 数据库连接池指标
4. **`pkg/xredis/metrics.go`** - Redis 缓存命中率指标
5. **`internal/gateway/middleware/metrics.go`** - HTTP 请求指标

### 9.3 核心知识点

```go
// 1. Counter（只增不减的计数器）
requestsTotal = prometheus.NewCounter(prometheus.CounterOpts{
    Name: "http_requests_total",
    Help: "Total number of HTTP requests.",
})

// 2. Histogram（分布统计）
requestDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
    Name:    "http_request_duration_seconds",
    Help:    "HTTP request duration in seconds.",
    Buckets: prometheus.DefBuckets,  // .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10
})

// 3. Span 创建
ctx, span := tracer.Start(ctx, "operation-name")
defer span.End()
span.SetAttributes(attribute.String("key", "value"))
```

---

## 第十部分：测试学习（贯穿全程）

### 10.1 测试文件命名约定

- 普通源文件：`xxx.go`（如 `user.go`、`repository.go`）
- 测试文件：`xxx_test.go`（如 `user_test.go`、`auth_test.go`），Go 编译器自动排除
- 单元测试位于与被测代码**同包同目录**（使用 `_test.go` 后缀，默认用 `package biz`），使用手写 mock 结构体（不用 mock 框架）和 Go 标准库 `testing`
- 集成测试位于 `test/integration/` 独立目录，使用 `testcontainers-go` 启动真实 PG/Redis 容器，需要 `-tags=integration` 构建标签

### 10.2 单元测试模式

```go
// 1. 接口 mock 模式（手动实现，不用 mock 框架）
type mockRepo struct {
    findByIDFn func(ctx context.Context, id string) (*User, error)
}
func (m *mockRepo) FindByID(ctx context.Context, id string) (*User, error) {
    return m.findByIDFn(ctx, id)
}

// 2. 表格驱动测试（Table-Driven Tests）
func TestRegister(t *testing.T) {
    tests := []struct {
        name     string
        username string
        wantErr  bool
    }{
        {"valid", "test_user", false},
        {"too short", "ab", true},
        {"no underscore", "test user", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := biz.Register(ctx, tt.username, ...)
            if (err != nil) != tt.wantErr {
                t.Errorf("Register() error = %v, wantErr = %v", err, tt.wantErr)
            }
        })
    }
}
```

### 10.3 集成测试

```bash
go test -tags=integration ./test/integration/ -v
```

---

## 第十一部分：学习路线时间表

```
总时长：约 4-6 周（兼职学习，每天 2-3 小时）

Week 1: Go 基础速览 + Phase 0（项目骨架）
  - Go Tour 交互式教程
  - 理解项目目录结构
  - 阅读 cmd/*/main.go 三个入口文件
  - 学习优雅关闭、配置管理

Week 2: Phase 1（用户服务）
  - 四层架构精读（data → biz → service → server）
  - 理解接口 + 依赖注入
  - 理解 JWT + bcrypt
  - 用 curl/Postman 测试所有认证接口

Week 3: Phase 2（项目管理）
  - GORM 事务 + 乐观锁
  - RBAC 权限矩阵
  - 跨服务 gRPC 调用链路
  - Redis cache-aside 模式

Week 4: Phase 3-4（任务 + 评论 + 操作日志）
  - 状态机实现
  - 游标分页
  - goroutine + channel + select 并发模型（重点！）
  - Prometheus 指标

Week 5: Phase 5（工程化增强）
  - 双层限流 + Redis Lua
  - 幂等实现（SETNX + body 缓存）
  - 中间件链设计

Week 6: Phase 6（可观测性）+ 回顾
  - OpenTelemetry Trace
  - Grafana 看板
  - 通读一遍完整调用链路
  - 尝试回答 plan.md §14.2 的面试题列表
```

---

## 第十二部分：Go 新手易犯错误与提示

### 12.1 常见误区

| 误区 | 正确做法 |
|------|---------|
| `for` 循环中捕获循环变量 | Go 1.22+ 已自动修复，本项目 Go 1.26 不受影响 |
| goroutine 泄漏（启动了没退出） | 本项目 `LogWriter.run()` 通过 `done` channel + `WaitGroup` 正确退出 |
| 传递 nil interface 而非 nil 具体类型 | 返回 error 时写 `return nil` 而非 `return (*MyError)(nil)` |
| channel 没关闭就 range | 本项目用 `select` + done channel，不 range |
| `defer` 在循环中使用导致资源堆积 | `defer` 在函数退出才执行，循环中用 `func(){defer ...}()` 包裹 |

### 12.2 Go 特有习惯

1. **错误处理不吞：** 永远不要 `_ = err`，要么处理要么向上传播
2. **小写=私有，大写=公开：** 没有 `public/private` 关键字
3. **构造函数习惯命名为 `NewXxx`：** 如 `NewUserBiz`、`NewAuthHandler`
4. **接收者用类型首字母缩写：** `func (b *UserBiz) Register(...)` 而非 `func (ub *UserBiz) Register(...)`
5. **单文件通常不超过 400 行：** 本项目严格遵守，最长文件 ~250 行

---

## 第十三部分：推荐阅读顺序（按学习效果）

### 路线 A：自底向上（推荐 - 理解依赖关系）

```
pkg/xerr → pkg/xjwt → pkg/xlog
    ↓
internal/user/data → internal/user/biz → internal/user/service → internal/user/server
    ↓
internal/task/data → internal/task/biz → internal/task/service → internal/task/server
    ↓
internal/gateway/rpc → internal/gateway/middleware → internal/gateway/handler
    ↓
cmd/*/main.go
```

### 路线 B：跟随请求链路（直观 - 从一个注册请求跟踪）

```
1. cmd/api-gateway/main.go 启动网关
2. POST /api/v1/auth/register
3. → middleware/auth.go（跳过，register 是公开路径）
4. → handler/auth.go Register()
5. → rpc/client.go 注入 metadata
6. → gRPC → user-service
7. → internal/user/server 拦截器校验
8. → internal/user/service Register()
9. → internal/user/biz Register()
10. → internal/user/data Create()
11. → PostgreSQL
```

不管选哪条路线，**每个文件读完问自己三个问题：**
1. 这个包提供了什么功能？（"是什么"）
2. 它依赖了哪些包？（"上游是谁"）
3. 谁在用它？（"下游是谁"）

---

## 第十四部分：面试准备检查清单

对应 `plan.md` §14.2 的面试重点，学完后应能回答：

- [ ] 为什么外部用 HTTP，内部用 gRPC？
- [ ] 为什么拆成 user-service 和 task-service，不拆更多？
- [ ] 单 PG 多 schema 的收益与代价？
- [ ] JWT 的签发、校验、黑名单机制？
- [ ] RBAC 权限矩阵如何实现？
- [ ] 乐观锁的原理和 SQL 写法？
- [ ] 游标分页 vs offset 分页的优劣？
- [ ] cache-aside 缓存模式（读穿透、写失效、TTL 兜底）？
- [ ] 幂等 `Idempotency-Key` 的实现原理？
- [ ] 令牌桶限流的 Redis Lua 实现？
- [ ] `operation_logs` 为什么异步写？channel 满如何降级？
- [ ] 中间件链的顺序为什么重要？
- [ ] gRPC deadline / 重试 / 错误码如何收敛？
- [ ] request_id 和 trace_id 如何串联日志？

---

## 附录：项目命令速查

```bash
# 启动基础设施
make up

# 启动服务（三个终端分别运行）
make run/api-gateway
make run/user-service
make run/task-service

# 运行测试
make test                          # 所有测试
go test ./internal/user/biz -v     # 单个包的测试
go test -tags=integration ./test/integration/ -v  # 集成测试

# 代码质量
make lint      # 静态检查
make coverage  # 覆盖率报告
make proto     # 重新生成 gRPC 代码

# 压测
make seed-users    # 先灌入测试数据
make loadtest-baseline  # SLO 验证
```

---

> **学习原则：**
> 1. 先读懂再动手 - 每个 Phase 先通读所有文件，再动手改代码
> 2. 跟着请求链路读 - 从 HTTP 入口一直追到数据库，画出调用链
> 3. 答案在代码里 - 对照 `plan.md` 的规格描述，在代码中找到对应实现
> 4. Go 的哲学是简单 - 如果一个实现让你觉得很复杂，很可能不是 Go 的风格
