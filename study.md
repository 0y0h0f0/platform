# 团队任务协作平台 - Go 后端学习路线（Go 初学者详细版）

> 适用对象：Go 新手，已有基本编程经验（了解变量、函数、循环、条件判断等概念），希望
> 通过一个完整的生产级后端项目系统学习 Go 语言和工程实践。
>
> 学习目标：读完本文后，你应该能独立启动项目、看懂主要调用链、理解 Go 后端常见工程结构，
> 并能围绕注册、登录、项目管理、任务流转、评论、操作日志、限流、幂等、测试和可观测性
> 说清楚代码是如何工作的。

---

## 如何使用这份文档

这不是一份只讲语法的 Go 教程，而是一份“边学 Go，边读真实项目”的学习手册。推荐按下面节奏学习：

1. **先跑起来。** 不要先把所有代码看完。先完成附录 A 的启动命令，确认服务能跑，接口能访问。
2. **再读主链路。** 从 `POST /api/v1/auth/register` 开始，按“HTTP handler → gRPC client → user-service → biz → data → PostgreSQL”追踪一次。
3. **边读边改。** 每读完一个 Phase，至少做一个小改动，比如加一条校验、补一个测试、改一个错误消息。
4. **最后再看工程化。** 限流、幂等、trace、metrics、异步日志等内容更抽象，适合在你已经看懂业务代码后学习。

初学者最容易卡住的不是某个语法点，而是“不知道代码为什么要拆成这么多层”。读这个项目时请反复问三件事：

| 问题 | 你要找的答案 |
|------|-------------|
| 这一层负责什么？ | 比如 handler 只处理 HTTP，biz 只处理业务规则，data 只处理数据库 |
| 它依赖谁？ | 通过 import 和构造函数参数看依赖方向 |
| 谁调用它？ | 通过 `rg "NewUserBiz"`、`rg "Register("` 之类命令反向查找 |

配套文档：

- [docs/README.md](docs/README.md)：完整文档清单和阅读顺序
- [docs/api-reference.md](docs/api-reference.md)：HTTP API、错误码、幂等规则
- [docs/configuration.md](docs/configuration.md)：YAML 配置、环境变量、超时和连接池参数
- [docs/observability.md](docs/observability.md)：日志、指标、trace 和排障入口

---

## 第零部分：Go 初学者预备课（2-4 天）

如果你刚开始学 Go，建议先完整读这一部分，再进入项目代码。这里会用本项目中的真实写法解释 Go 的核心概念。

### 0.1 Go 程序的最小结构

一个 Go 文件通常长这样：

```go
package main

import "fmt"

func main() {
    fmt.Println("hello go")
}
```

逐行解释：

| 代码 | 含义 |
|------|------|
| `package main` | 当前文件属于 `main` 包。只有 `main` 包才能编译成可执行程序 |
| `import "fmt"` | 导入标准库 `fmt`，用来格式化输出 |
| `func main()` | 程序入口。运行二进制时，Go 会从这里开始执行 |
| `fmt.Println(...)` | 调用 `fmt` 包里的公开函数 `Println` |

本项目有三个入口：

| 入口文件 | 启动的服务 |
|----------|------------|
| `cmd/api-gateway/main.go` | HTTP 网关，监听 `:8080` |
| `cmd/user-service/main.go` | 用户 gRPC 服务，监听 `:9091`，管理端口 `:8081` |
| `cmd/task-service/main.go` | 任务 gRPC 服务，监听 `:9092`，管理端口 `:8082` |

Go 项目里常见的 `cmd/<name>/main.go` 约定表示：每个子目录都是一个可执行程序。

### 0.2 Go Module、包名和导入路径

本项目的 `go.mod` 开头是：

```go
module task-platform

go 1.26.3
```

这说明：

- 当前项目的模块名是 `task-platform`
- 项目内部包的导入路径都以 `task-platform/...` 开头
- 例如 `cmd/api-gateway/main.go` 里导入了 `task-platform/internal/gateway/server`

Go 里有两个容易混淆的概念：

| 概念 | 说明 |
|------|------|
| 目录 | 文件系统上的文件夹，比如 `internal/user/biz` |
| 包 | Go 编译单位，一个目录通常对应一个包，比如 `package biz` |

注意：

- 同一个目录下的 `.go` 文件必须声明同一个包名，测试文件除外
- 包名通常是目录名最后一段，比如 `internal/user/biz` 的包名是 `biz`
- 导入时用完整路径，使用时用包名，例如 `biz.NewUserBiz(...)`

### 0.3 `internal` 和 `pkg` 的区别

本项目有两个重要目录：

```text
internal/
pkg/
```

| 目录 | 用途 | 初学者理解 |
|------|------|------------|
| `internal/` | 项目内部业务代码，外部项目不能导入 | 放核心业务：用户、任务、网关 |
| `pkg/` | 可复用工具包 | 放通用能力：错误码、JWT、日志、Redis、PostgreSQL、gRPC |

Go 对 `internal` 有编译级限制：`internal` 目录外部的项目不能 import 它。这可以防止业务代码被别的项目误用。

### 0.4 标识符大小写：Go 的可见性规则

Go 没有 `public`、`private` 关键字。可见性只看首字母大小写：

```go
type UserBiz struct{}   // 大写开头：包外可访问
type userRepo struct{}  // 小写开头：只能当前包内访问
```

本项目里经常看到这种组合：

```go
type UserRepository interface {
    Create(ctx context.Context, user *User) error
}

type userRepo struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
    return &userRepo{db: db}
}
```

设计意图：

- `UserRepository` 是公开接口，其他包可以依赖它
- `userRepo` 是私有实现，其他包不需要知道细节
- `NewUserRepository` 是公开构造函数，返回接口类型

这是 Go 后端项目非常常见的封装方式。

### 0.5 变量、常量和零值

Go 变量声明常见写法：

```go
var name string        // 声明变量，零值是 ""
var age int = 18       // 声明并赋值
enabled := true        // 短变量声明，只能在函数内部使用
```

Go 的零值很重要：

| 类型 | 零值 |
|------|------|
| `int` / `int64` | `0` |
| `float64` | `0` |
| `bool` | `false` |
| `string` | `""` |
| 指针 | `nil` |
| slice | `nil` |
| map | `nil` |
| interface | `nil` |
| struct | 每个字段都是自己的零值 |

本项目中很多判断都依赖零值，比如：

```go
if cfg.HTTPAddr == "" {
    cfg.HTTPAddr = ":8080"
}
```

常量声明：

```go
const (
    userCacheTTL    = 5 * time.Minute
    userCachePrefix = "user:"
)
```

命名建议：

- 包内私有常量用小写开头
- 多个相关常量用 `const (...)` 分组
- 不要把魔法数字散落在代码里

### 0.6 基本类型、切片和 map

本项目常用类型：

| 类型 | 用途 |
|------|------|
| `string` | 用户名、邮箱、ID、错误消息 |
| `int32` | proto 里的枚举值、状态、角色 |
| `int64` | 时间戳、计数 |
| `bool` | 是否启用、是否存在 |
| `time.Time` | 创建时间、更新时间 |
| `[]string` | 用户 ID 列表、弱密码列表 |
| `map[string]bool` | 弱密码集合、快速判断是否存在 |

slice 是 Go 里最常用的动态数组：

```go
ids := []string{"u1", "u2", "u3"}

for _, id := range ids {
    fmt.Println(id)
}
```

`range` 返回两个值：

```go
for i, v := range ids {
    fmt.Println(i, v)
}
```

如果不需要下标，用 `_` 忽略：

```go
for _, id := range ids {
    // 只用 id
}
```

map 是键值表：

```go
weakPasswords := map[string]bool{
    "password123": true,
    "12345678":    true,
}

if weakPasswords[input] {
    return errors.New("password is too weak")
}
```

注意：

- 从不存在的 key 读取，会得到 value 类型的零值
- `nil` map 不能写入，写入前必须 `make`
- slice 可以 `append`，map 需要通过 key 赋值

### 0.7 struct、方法和指针接收者

Go 没有 class，但有 struct：

```go
type UserBiz struct {
    repo data.UserRepository
    rdb  *redis.Client
}
```

方法是绑定在类型上的函数：

```go
func (b *UserBiz) Register(ctx context.Context, username, email, password string) (*data.User, error) {
    // ...
}
```

`(b *UserBiz)` 叫接收者。它表示 `Register` 是 `*UserBiz` 的方法。

为什么大多数方法接收者用指针 `*UserBiz`？

| 原因 | 说明 |
|------|------|
| 避免拷贝 | struct 里可能有数据库连接、Redis 连接、logger 等字段，不应该复制 |
| 可以修改状态 | 指针接收者可以修改 struct 字段 |
| 保持一致 | 一个类型的方法通常统一用值接收者或指针接收者 |

调用时不需要手动解引用：

```go
biz := NewUserBiz(repo, rdb, weakPasswords)
user, err := biz.Register(ctx, username, email, password)
```

### 0.8 struct tag：数据库、JSON、配置都靠它

Go struct 字段后面经常带一段反引号：

```go
type config struct {
    ServiceName string `mapstructure:"service_name"`
    HTTPAddr    string `mapstructure:"http_addr"`
}
```

这叫 struct tag，用于告诉框架如何映射字段。

常见 tag：

| tag | 用途 |
|-----|------|
| `json:"username"` | JSON 序列化和反序列化 |
| `gorm:"column:username"` | GORM 数据库字段映射 |
| `mapstructure:"service_name"` | Viper 配置映射 |
| `binding:"required"` | Gin 参数校验 |

初学者常见问题：

- tag 必须用反引号，不是单引号或双引号
- 字段必须大写开头，否则 JSON/GORM 等包通常无法从包外反射访问它
- 多个 tag 可以写在同一个反引号里，例如 `json:"email" binding:"required"`

### 0.9 interface：Go 项目的解耦核心

Go 的 interface 是行为集合：

```go
type UserRepository interface {
    Create(ctx context.Context, user *User) error
    FindByID(ctx context.Context, id string) (*User, error)
}
```

任何类型只要实现了这些方法，就自动满足这个接口。不需要写 `implements`。

```go
type userRepo struct {
    db *gorm.DB
}

func (r *userRepo) Create(ctx context.Context, user *User) error {
    return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepo) FindByID(ctx context.Context, id string) (*User, error) {
    // ...
}
```

接口的价值：

| 场景 | 好处 |
|------|------|
| 业务层依赖 Repository 接口 | 业务层不关心数据库怎么实现 |
| 测试时写 mockRepo | 不需要真实 PostgreSQL 也能测业务逻辑 |
| 替换实现 | 以后从 GORM 换成 SQLC，biz 层可以少改 |

初学者记住一句话：**接口由使用方定义，接口越小越好。**

### 0.10 error：Go 的显式错误处理

Go 不使用异常作为主要错误处理方式。函数通常把错误作为最后一个返回值：

```go
user, err := repo.FindByID(ctx, userID)
if err != nil {
    return nil, err
}
```

常见模式：

```go
result, err := doSomething()
if err != nil {
    return nil, fmt.Errorf("do something: %w", err)
}
return result, nil
```

`%w` 表示包装错误，后续可以用 `errors.Is` 或 `errors.As` 判断。

本项目的统一错误类型在 `pkg/xerr/codes.go`：

- 业务代码返回 `xerr.NewError(...)`
- gRPC 层把它转成 gRPC status
- gateway 再把 gRPC status 转成 HTTP 响应

因此你会看到很多这样的代码：

```go
return nil, xerr.NewError(xerr.CodeInvalidArgument, "invalid email")
```

这比直接 `errors.New("invalid email")` 更适合后端服务，因为前端需要稳定的错误码。

### 0.11 context.Context：请求生命周期的主线

本项目几乎所有业务方法第一个参数都是 `ctx context.Context`：

```go
func (b *UserBiz) Register(ctx context.Context, username, email, password string) (*data.User, error)
```

`context.Context` 用来传递：

| 内容 | 说明 |
|------|------|
| 取消信号 | 客户端断开、服务关闭时，中断下游操作 |
| 超时时间 | 避免一个请求无限等待 |
| 请求级值 | request_id、user_id、trace 信息 |

常见写法：

```go
ctx := context.Background()

ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
defer cancel()

result, err := client.Call(ctx, req)
```

重要习惯：

- 不要把 `context.Context` 存进 struct 字段
- 函数需要上下文时，把 `ctx` 放在第一个参数
- 创建了 `cancel` 就要 `defer cancel()`
- 数据库、Redis、gRPC 调用都应该传递 `ctx`

### 0.12 defer：资源清理和收尾逻辑

`defer` 表示函数返回前执行：

```go
func run() error {
    logger, err := xlog.New(...)
    if err != nil {
        return err
    }
    defer syncLogger(logger)

    // ...
    return nil
}
```

常见用途：

| 用途 | 示例 |
|------|------|
| 关闭资源 | `defer rows.Close()` |
| 释放 cancel | `defer cancel()` |
| 刷新日志 | `defer logger.Sync()` |
| panic 恢复 | `defer func(){ recover() }()` |

注意：

- `defer` 在当前函数结束时执行，不是在当前代码块结束时执行
- 循环里大量 `defer` 可能导致资源迟迟不释放
- 多个 `defer` 按后进先出顺序执行

### 0.13 goroutine、channel 和 select

Go 的并发由 goroutine 和 channel 组成：

```go
go func() {
    fmt.Println("run in background")
}()
```

channel 是 goroutine 之间传递数据的管道：

```go
ch := make(chan string, 10)
ch <- "hello"
msg := <-ch
```

`select` 可以同时等待多个 channel：

```go
select {
case msg := <-ch:
    fmt.Println(msg)
case <-ctx.Done():
    return ctx.Err()
}
```

本项目最值得精读的并发代码是 `internal/task/biz/log_writer.go`。它展示了：

- 后台 worker 如何启动
- 有界 channel 如何防止无限占内存
- 定时 flush 如何用 `time.Ticker`
- 服务关闭时如何等待 goroutine 退出
- worker panic 后如何恢复

### 0.14 泛型：本项目用得很少，但要认识

Go 支持泛型，写法类似：

```go
func First[T any](items []T) (T, bool) {
    var zero T
    if len(items) == 0 {
        return zero, false
    }
    return items[0], true
}
```

本项目主要是业务后端，不需要大量泛型。初学阶段你只要能看懂 `[]T`、`map[K]V`、`any` 这些写法即可。

### 0.15 Go 代码阅读方法

读 Go 项目时，不要从所有文件随机看。推荐顺序：

1. 找入口：`cmd/<service>/main.go`
2. 看构造：`server.New...`
3. 看路由或服务注册：HTTP 路由、gRPC 注册
4. 看 handler/service：请求参数如何进入
5. 看 biz：业务规则在哪里
6. 看 data：数据库如何读写
7. 看测试：预期行为是什么

常用搜索命令：

```bash
rg "func New" internal/user
rg "Register" internal/user internal/gateway
rg "type UserRepository" -n
rg "CodeInvalidArgument" -n
```

Go 初学者读代码时可以先忽略这些内容：

- 复杂的 Prometheus 指标注册细节
- OpenTelemetry 的具体 SDK 参数
- gRPC 生成代码 `gen/go/...`
- 前端 `web/` 目录

先把“请求如何从入口走到数据库，再把结果返回给客户端”看懂，收益最大。

### 0.16 本项目的第一条完整调用链

建议初学者从注册接口开始：

```text
POST /api/v1/auth/register
  ↓
internal/gateway/handler/auth.go
  ↓
internal/gateway/rpc/client.go
  ↓
user-service gRPC
  ↓
internal/user/service/service.go
  ↓
internal/user/biz/user.go
  ↓
internal/user/data/repository.go
  ↓
PostgreSQL users 表
```

你需要在每一层回答：

| 层 | 问题 |
|----|------|
| handler | HTTP JSON 参数如何绑定？错误如何返回？ |
| rpc client | 如何连到 user-service？metadata 如何传？ |
| service | proto request 如何转成 biz 参数？ |
| biz | 用户名、邮箱、密码如何校验？密码如何哈希？ |
| data | 用户如何插入数据库？唯一键冲突如何处理？ |

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
pkg/xredis  →  Redis 客户端封装（连接池、命令超时、指标）
pkg/xpgsql  →  PostgreSQL 客户端封装
pkg/xgrpc   →  gRPC 服务端/客户端通用拦截器（认证、指标、默认 deadline）
pkg/xhttp   →  HTTP engine、健康检查、metrics、server timeout
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

1. **`cmd/api-gateway/main.go`**（约 156 行）
   - 重点理解：
     - `func run() error` 惯用模式
     - `signal.NotifyContext` 监听系统信号
     - `atomic.Bool` 就绪标志位
     - `http.Server.Shutdown` 优雅关闭
     - `xhttp.NewServer` 设置 `ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout`
     - Viper 配置加载（YAML 文件 + `AutomaticEnv` 环境变量覆盖）

2. **`cmd/user-service/main.go`**（约 187 行）
   - 额外重点：
     - gRPC 的 `GracefulStop()` vs `Stop()`
     - `select` 多路选择（等待信号或 goroutine 结果）
     - 同时启动 gRPC + Admin HTTP 两个服务器的模式

3. **`pkg/xerr/codes.go`**
   - 重点理解：
     - 自定义 `Error` 类型（`Code` + `Message`）
     - 实现 `GRPCStatus()` 方法使 `xerr.Error` 可以被 gRPC 框架识别
     - gRPC status code ↔ HTTP status code 的映射表
     - `errors.As` 错误类型断言

4. **`pkg/xhttp/server.go`** 和 **`pkg/xgrpc/server.go`**
   - 健康检查 `/healthz`、`/readyz`
   - Prometheus `/metrics`
   - HTTP server 默认超时：header 5s、read 10s、write 15s、idle 60s
   - gRPC reflection 和 health check 注册
   - `pkg/xgrpc/timeout.go` 通过拦截器给内部 RPC 设置默认 deadline

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

3. **`internal/user/biz/user.go`**（约 260 行）
   - 正则校验（`regexp.MustCompile` 在包级别编译一次）
   - `bcrypt.GenerateFromPassword`（cost=10，生成哈希）
   - `bcrypt.CompareHashAndPassword`（验证密码）
   - `bcrypt.Cost`（检查是否需要重哈希）
   - **cache-aside 模式**：先查 Redis，miss 再查 DB，回填缓存
   - `singleflight.Group` 合并同一个 userID 的并发缓存 miss，避免瞬时击穿数据库
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
     - `UnaryServerTimeoutInterceptor()` - 没有入站 deadline 时补默认超时
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
- 理解 `singleflight` 如何合并并发缓存 miss

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
   - **singleflight 防击穿**：同一个 projectID 的并发详情查询只让一个 goroutine 查 DB
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
- 理解 worker pool 的简化版（可配置 worker + 有界 channel）
- 理解 Prometheus 指标的定义和注册

### 7.2 精读文件清单

1. **`internal/task/biz/log_writer.go`**（约 209 行）-**本项目的并发编程教科书**
   - goroutine 生命周期管理
   - channel 容量设计（1024）
   - `LOG_WRITER_WORKERS` 控制 worker 数（默认 1，最大 16）
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
        if !w.closed.Load() {
            w.wg.Add(1)
            go w.run()
        }  // 自动重启
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

1. **`internal/gateway/middleware/ratelimit.go`**（约 152 行）
   - 双层限流：IP 限流 + 用户级限流
   - 认证接口使用更严格阈值
   - Redis 错误时"fail-open"（放行不阻断）
   - Prometheus 指标记录 allowed、rejected 和 Redis error

2. **`internal/gateway/handler/idempotency.go`**（约 104 行）
   - `SetNX` 原子的"抢锁"语义
   - `bodyCaptureWriter` 拦截响应 body 缓存
   - 失败请求清理 key（不放垃圾数据）
   - 重复请求命中 `pending` 时返回 `409 Conflict` + `ABORTED`

3. **`pkg/xratelimit/token_bucket.go`**（60 行）
   - Redis Lua 脚本做原子令牌桶
   - 令牌桶算法：`tokens = min(capacity, tokens + elapsed * rate)`

4. **`internal/gateway/middleware/`** 其他中间件
   - `requestid.go` - Request ID 生成/提取/回溯
   - `accesslog.go` - Zap 结构化日志
   - `cors.go` - 跨域配置
   - `metrics.go` - HTTP 请求直方图 + 计数器
   - `security.go` / `bodylimit.go` - 安全响应头和 1MB 请求体限制

### 8.3 核心 Go 知识点

```go
// 1. Gin 中间件链是洋葱模型
// Recovery → MaxBodySize → SecurityHeaders → RequestID → Trace → Metrics
// → AccessLog → CORS → RateLimit(IP) → Auth → RateLimit(user) → Handler
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
    // 如果还是 pending，返回 409/ABORTED；如果已有结果，返回缓存响应
    return cachedResult
}
// 首次请求，执行业务逻辑，写入结果
```

`singleflight` 是本阶段新增的一个重要读优化：

```go
v, err, _ := b.sf.Do(userID, func() (any, error) {
    if u := b.getCachedUser(ctx, userID); u != nil {
        return u, nil
    }
    user, err := b.repo.FindByID(ctx, userID)
    if err != nil { return nil, err }
    b.cacheUser(ctx, user)
    return user, nil
})
```

它解决的是“同一个热点 key 过期时，很多 goroutine 同时查数据库”的问题。注意它不是缓存，只是把同 key 的并发 miss 合并成一次实际查询。

---

## 第九部分：Phase 6 学习 - 可观测性（2-3 天）

### 9.1 学习目标

- 理解 OpenTelemetry 的 Span 概念
- 理解 Prometheus 的指标类型（Counter / Histogram）
- 理解结构化日志（Zap JSON）的字段约定

### 9.2 精读文件清单

1. **`pkg/xtrace/trace.go`** - OpenTelemetry 初始化
2. **`pkg/xgrpc/metrics.go`** - gRPC 请求指标
3. **`pkg/xgrpc/timeout.go`** - gRPC client/server 默认 deadline
4. **`pkg/xpgsql/metrics.go`** - 数据库连接池指标
5. **`pkg/xredis/metrics.go`** - Redis 缓存命中率指标
6. **`internal/gateway/middleware/metrics.go`** - HTTP 请求指标

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

## 附录 A：项目命令速查

```bash
# 启动基础设施
make up
make migrate

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

---

## 附录 B：Go 语法速查（结合本项目）

这一节是给初学者查漏补缺用的。你不需要一次背下来，但遇到代码看不懂时可以回到这里对照。

### B.1 函数声明和多返回值

Go 函数格式：

```go
func 函数名(参数名 参数类型) (返回值类型, error) {
    return 返回值, nil
}
```

项目中典型写法：

```go
func HashPassword(plain string) (string, error) {
    return HashPasswordWithCost(plain, bcryptCost)
}
```

多返回值是 Go 的重要特性。常见组合：

| 返回值 | 含义 |
|--------|------|
| `(T, error)` | 成功返回 T，失败返回 error |
| `(*T, error)` | 成功返回对象指针，失败返回 error |
| `(bool, error)` | 成功返回判断结果，失败返回 error |
| `(cleanup func() error, err error)` | 返回清理函数和错误，常见于初始化代码 |

初学者要注意：

```go
user, err := repo.FindByID(ctx, id)
if err != nil {
    return nil, err
}
```

这不是“啰嗦”，而是 Go 故意让错误路径显式可见。

### B.2 `:=` 和 `var` 的区别

```go
var err error       // 声明变量，可以在函数外或函数内使用
user, err := ...    // 短声明，只能在函数内使用
```

`:=` 至少要声明一个新变量：

```go
user, err := repo.FindByID(ctx, id)  // user 和 err 都是新变量
user, err = repo.FindByID(ctx, id)   // 重新赋值，不能用 :=
```

常见坑：

```go
var err error
if user, err := repo.FindByID(ctx, id); err != nil {
    return err
}
// 这里访问不到 user，因为 user 只在 if 语句内部有效
```

项目里大量使用这种 `if err := ...; err != nil` 写法：

```go
if err := v.ReadInConfig(); err != nil {
    return config{}, fmt.Errorf("read config: %w", err)
}
```

它适合只在当前判断里使用 `err` 的场景。

### B.3 指针、值和 `nil`

```go
user := data.User{}      // 值
userPtr := &user         // 指针，类型是 *data.User
```

函数参数是值时会拷贝：

```go
func UpdateUser(u data.User) {
    u.Username = "new" // 只改了副本
}
```

函数参数是指针时可以修改原对象：

```go
func UpdateUser(u *data.User) {
    u.Username = "new" // 修改调用方对象
}
```

本项目大多数实体都用指针传递：

```go
func (r *userRepo) Create(ctx context.Context, user *User) error
```

原因是：

- 避免复制大结构体
- GORM 需要通过指针写入自增字段、时间字段、ID 等
- `nil` 可以表达“没有对象”

### B.4 package import 的常见形式

普通导入：

```go
import "context"
```

分组导入：

```go
import (
    "context"
    "fmt"

    "go.uber.org/zap"

    "task-platform/pkg/xerr"
)
```

别名导入：

```go
import gwserver "task-platform/internal/gateway/server"
```

使用别名通常是为了避免包名冲突，或者让调用处更清楚。

空白导入：

```go
import _ "github.com/lib/pq"
```

`_` 表示只执行这个包的初始化逻辑，不直接使用它的导出符号。本项目较少用这种方式，但你在数据库驱动、pprof、迁移工具里会经常见到。

### B.5 数组、slice、map 的选择

| 数据结构 | 是否常用 | 适合场景 |
|----------|----------|----------|
| 数组 `[3]string` | 较少 | 固定长度，编译期确定 |
| slice `[]string` | 最常用 | 动态列表，比如用户 ID 列表 |
| map `map[string]bool` | 很常用 | 快速通过 key 查找 |

slice 初始化：

```go
users := make([]*data.User, 0, len(ids))
```

这表示：

- 当前长度是 0
- 预分配容量是 `len(ids)`
- 后续 `append` 时更少扩容

map 初始化：

```go
set := make(map[string]bool, len(weakPasswords))
for _, pw := range weakPasswords {
    set[pw] = true
}
```

这就是本项目弱密码表的写法，适合做“是否存在”的判断。

### B.6 时间处理

Go 时间类型来自标准库 `time`：

```go
5 * time.Minute
100 * time.Millisecond
time.Now()
time.Duration(seconds) * time.Second
```

项目中的典型用途：

| 用途 | 示例 |
|------|------|
| 缓存 TTL | `userCacheTTL = 5 * time.Minute` |
| 优雅关闭超时 | `context.WithTimeout(..., 10*time.Second)` |
| HTTP server 超时 | `ReadHeaderTimeout: 5 * time.Second` |
| gRPC 默认 deadline | `GRPC_CLIENT_TIMEOUT_SECONDS=2` |
| 定时 flush | `time.NewTicker(100 * time.Millisecond)` |
| JWT 过期时间 | token TTL 2 小时 |

初学者注意：`time.Duration` 本质上是纳秒数，所以整数转 duration 时要乘单位：

```go
time.Duration(cfg.ShutdownTimeoutSeconds) * time.Second
```

不要写成：

```go
time.Duration(cfg.ShutdownTimeoutSeconds) // 这只是几个纳秒
```

### B.7 `for` 是 Go 唯一的循环关键字

Go 没有 `while`，所有循环都用 `for`。

普通循环：

```go
for i := 0; i < 10; i++ {
    fmt.Println(i)
}
```

类似 while：

```go
for running {
    doWork()
}
```

无限循环：

```go
for {
    select {
    case <-ctx.Done():
        return
    }
}
```

遍历 slice：

```go
for _, id := range ids {
    fmt.Println(id)
}
```

遍历 map：

```go
for key, value := range m {
    fmt.Println(key, value)
}
```

### B.8 `switch` 和状态机

本项目的任务状态流转适合用状态机理解：

```go
var validTransitions = map[int32][]int32{
    Todo:  {Doing, Done, Cancelled},
    Doing: {Done, Cancelled, Todo},
}
```

如果用 `switch`，一般长这样：

```go
switch status {
case TaskStatusTodo:
    // todo
case TaskStatusDoing:
    // doing
case TaskStatusDone:
    // done
default:
    return xerr.NewError(xerr.CodeInvalidArgument, "invalid status")
}
```

Go 的 `switch` 默认不会自动贯穿到下一个 case，不需要手写 `break`。

### B.9 JSON 编码和解码

项目里 Redis 缓存用户对象时会用 JSON：

```go
b, err := json.Marshal(user)
if err != nil {
    return
}

var user data.User
if err := json.Unmarshal([]byte(raw), &user); err != nil {
    return nil
}
```

理解要点：

- `Marshal`：Go struct → JSON bytes
- `Unmarshal`：JSON bytes → Go struct
- `Unmarshal` 第二个参数必须传指针，否则无法写入结果

### B.10 日志字段

本项目使用 Zap 结构化日志：

```go
logger.Info("http server listening", zap.String("addr", cfg.HTTPAddr))
logger.Error("cleanup failed", zap.Error(err))
```

结构化日志的好处是字段可检索：

```json
{"level":"info","msg":"http server listening","addr":":8080"}
```

初学者不要只写：

```go
logger.Info("server started")
```

更好的写法是带关键字段：

```go
logger.Info("server started", zap.String("addr", addr), zap.String("service", serviceName))
```

---

## 附录 C：注册接口逐层精读

这一节用一个最典型的请求讲清楚项目的分层。你可以一边打开代码，一边按下面问题检查自己是否看懂。

### C.1 HTTP 请求进入网关

请求大致是：

```http
POST /api/v1/auth/register
Content-Type: application/json
Idempotency-Key: demo-register-001

{
  "username": "alice_001",
  "email": "alice@example.com",
  "password": "Passw0rd123"
}
```

你要关注 `internal/gateway/handler/auth.go`：

| 代码点 | 要理解的问题 |
|--------|--------------|
| request struct | JSON 字段如何映射到 Go struct |
| `ShouldBindJSON` | 参数绑定失败时如何返回错误 |
| idempotency | 重复请求为什么不会重复创建用户 |
| RPC 调用 | handler 为什么不直接访问数据库 |
| response envelope | 为什么所有 HTTP 响应都包一层 `{code,message,request_id,data}` |

初学者需要建立的观念：**gateway 是边界层，不写核心业务。** 它负责 HTTP、认证、限流、幂等、参数绑定、错误转换，然后把请求交给后端服务。

### C.2 gRPC client 发起内部调用

关注 `internal/gateway/rpc/client.go`。

内部服务之间不用 HTTP JSON，而是 gRPC。gRPC client 负责：

- 连接 `user-service` 和 `task-service`
- 设置超时
- 注入内部认证 token
- 传递 `x-request-id`、`x-user-id` 等 metadata
- 接入 OpenTelemetry client trace

你需要理解 metadata 类似 HTTP header，只不过它属于 gRPC 请求。

### C.3 user-service 的 service 层

关注 `internal/user/service/service.go`。

service 层负责把 proto 请求转为 biz 参数：

```text
RegisterRequest
  ↓
username, email, password
  ↓
UserBiz.Register(...)
  ↓
RegisterResponse
```

这一层应该保持很薄。判断一个 service 层是否健康，可以看它是否只做三件事：

- 取参数
- 调 biz
- 转 response

如果你在 service 层看到大量业务规则，通常说明分层开始混乱。

### C.4 biz 层做核心业务

关注 `internal/user/biz/user.go` 的 `Register`。

注册的业务规则包括：

| 规则 | 代码位置 |
|------|----------|
| 用户名只能是字母、数字、下划线，长度 3-32 | `usernameRE.MatchString` |
| 邮箱必须合法 | `mail.ParseAddress` |
| 密码长度 8-64 | `utf8.RuneCountInString` |
| 密码必须包含字母和数字 | `hasLetter`、`hasDigit` |
| 不能使用弱密码 | `weakPasswords` map |
| 密码不能明文入库 | `HashPassword` / bcrypt |
| 邮箱统一小写 | `strings.ToLower` |
| 创建用户交给 repository | `b.repo.Create(ctx, user)` |

你应该特别注意：业务层不知道 HTTP，也不知道 gRPC。它只接收普通参数，返回普通 Go 对象和 error。这让业务层很好测试。

### C.5 data 层写入数据库

关注 `internal/user/data/repository.go`。

data 层负责：

- 组织 GORM 查询
- 处理数据库错误
- 把数据库错误转换成业务错误码
- 隐藏表结构细节

典型写法：

```go
if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
    return convertError(err)
}
return nil
```

`WithContext(ctx)` 很重要，因为它让数据库操作能响应请求取消和超时。

### C.6 响应如何回到客户端

返回路径和进入路径相反：

```text
PostgreSQL
  ↑
data.User
  ↑
biz.Register 返回 user
  ↑
service 转 RegisterResponse
  ↑
gRPC 返回 gateway
  ↑
handler 组装 HTTP JSON
  ↑
浏览器/Postman
```

如果中间任意一层返回 error，会进入错误转换链：

```text
xerr.Error
  ↓
gRPC status
  ↓
HTTP status + JSON envelope
```

这就是为什么项目要统一错误码。没有统一错误码，前端就只能解析字符串，系统会很难维护。

---

## 附录 D：调试、排错和读日志

### D.1 服务启动失败怎么查

启动失败先看错误属于哪类：

| 错误现象 | 常见原因 | 检查方式 |
|----------|----------|----------|
| `read config` 失败 | 配置文件路径不对 | 查看 `APP_ENV`、`CONFIG_FILE` |
| PostgreSQL 连接失败 | Docker 未启动、DSN 错 | `docker compose ps`，检查 `POSTGRES_DSN` |
| Redis 连接失败 | Redis 未启动、地址错 | 检查 `REDIS_ADDR` |
| JWT secret 太短 | `.env` 配置不满足安全要求 | 检查 `JWT_SECRET` |
| internal token 太短或不一致 | 服务间认证失败 | 三个服务必须使用同一个 `INTERNAL_TOKEN` |
| 端口被占用 | 本地已有进程监听 | `lsof -i :8080` 或换端口配置 |

建议启动顺序：

```bash
make up
make migrate
make run/user-service
make run/task-service
make run/api-gateway
```

### D.2 接口返回 401 怎么查

401 通常和认证有关：

1. 是否调用了登录或注册拿到 token
2. 请求头是否有 `Authorization: Bearer <token>`
3. token 是否已经 logout，被写入 Redis 黑名单
4. token 是否过期
5. gateway 的 `JWT_SECRET` 是否和 user-service 签发时一致

重点文件：

- `pkg/xjwt/jwt.go`
- `internal/gateway/middleware/auth.go`
- `internal/gateway/handler/auth.go`

### D.3 接口返回 403 怎么查

403 通常是权限不足，不是未登录。

项目权限主要在 task-service biz 层判断：

- owner 可以管理项目、转让所有权、归档项目
- admin 可以管理成员和任务，但不能转让 owner
- member 只能做较有限的任务操作

重点文件：

- `internal/task/biz/project.go`
- `internal/task/biz/task.go`
- `internal/task/data/model.go` 中的 role 常量

### D.4 接口返回 409 怎么查

409 多半来自乐观锁冲突或幂等冲突。

乐观锁场景：

```text
用户 A 读取 version=1
用户 B 读取 version=1
用户 A 更新成功，version 变成 2
用户 B 仍然拿 version=1 更新，RowsAffected=0，返回冲突
```

解决方式：客户端重新读取最新数据，再提交更新。

幂等冲突场景：

- 同一个 `Idempotency-Key` 正在处理
- 同一个 key 已经有缓存结果
- 请求体不同但 key 相同

重点文件：`internal/gateway/handler/idempotency.go`。

### D.5 单元测试失败怎么查

Go 测试命令：

```bash
go test ./internal/user/biz -v
go test ./internal/task/biz -run TestCreateTask -v
go test ./... -count=1
```

参数解释：

| 参数 | 说明 |
|------|------|
| `-v` | 输出每个测试用例名称 |
| `-run TestName` | 只运行匹配名称的测试 |
| `-count=1` | 禁用测试缓存，强制重新执行 |
| `./...` | 当前目录及所有子目录 |

排查顺序：

1. 看第一个失败测试，不要同时追多个失败
2. 看 failure message，确认是输入错、期望错还是实现错
3. 打开对应 `_test.go`，看测试构造了什么 mock
4. 用 `t.Logf` 临时打印中间值
5. 修复后跑目标包，再跑 `go test ./...`

### D.6 集成测试失败怎么查

集成测试依赖 Docker 和 testcontainers：

```bash
go test -tags=integration ./test/integration/ -v
```

常见问题：

| 问题 | 处理 |
|------|------|
| Docker daemon 未启动 | 启动 Docker |
| 拉镜像失败 | 检查网络或镜像源 |
| 端口冲突 | testcontainers 通常随机端口，少见；检查本地 Docker 状态 |
| 迁移失败 | 查看 migrations 目录和测试日志 |

### D.7 用 `rg` 快速定位代码

读项目时，`rg` 比手工翻文件高效很多：

```bash
rg "SetupIdempotency"
rg "RateLimit"
rg "TransferProjectOwnership"
rg "validTransitions"
rg "NewLogWriter"
rg "UnaryServerMetricsInterceptor"
```

推荐技巧：

- 查类型：`rg "type UserBiz"`
- 查构造函数：`rg "NewUserBiz"`
- 查接口实现：`rg "func \(.*\) Create" internal/user/data`
- 查错误码：`rg "CodeAborted"`
- 查路由：`rg "api/v1" internal/gateway`

---

## 附录 E：初学者练习清单

这些练习按难度排序。不要只看代码，真正改一次、跑一次测试，学习效果会明显更好。

### E.1 入门练习：只读不改

- [ ] 画出 `cmd/api-gateway/main.go` 的启动流程图
- [ ] 找出 `api-gateway` 注册路由的位置
- [ ] 找出注册接口对应的 HTTP handler
- [ ] 找出 `UserBiz.Register` 的全部校验规则
- [ ] 找出 `UserRepository.Create` 如何处理数据库错误
- [ ] 找出 JWT 在哪里签发、哪里校验、哪里加入黑名单
- [ ] 找出 `request_id` 是在哪里生成的
- [ ] 找出 `/healthz` 和 `/readyz` 的区别

### E.2 基础改动：适合第一次动手

- [ ] 把用户名最小长度从 3 改成 4，并补充对应测试
- [ ] 给注册密码规则增加“不能包含空格”，并补充测试
- [ ] 给登录失败日志增加 `account` 字段，但不要打印密码
- [ ] 给某个 handler 增加更明确的参数错误消息
- [ ] 给 `pkg/xcursor` 新增一个非法 cursor 的测试用例

### E.3 业务练习：理解分层

- [ ] 新增一个“获取用户公开资料”的 gRPC 方法，只返回 ID、username、status，不返回 email
- [ ] 给项目增加一个 `description` 字段，完成 migration、model、repository、biz、service、handler、测试
- [ ] 给任务增加优先级筛选，理解 cursor filter hash 为什么要变化
- [ ] 给评论增加最小长度校验，补 biz 测试和 handler 测试
- [ ] 给操作日志增加一种新的 action 类型，并在对应业务操作里写入日志

### E.4 工程练习：理解后端质量

- [ ] 给一个新的 HTTP endpoint 接入 `Idempotency-Key`
- [ ] 给一个 Redis 缓存增加命中、未命中指标
- [ ] 给一个数据库查询增加 context timeout
- [ ] 调整 `GRPC_CLIENT_TIMEOUT_SECONDS`，观察内部 RPC 超时如何变成 `DEADLINE_EXCEEDED`
- [ ] 调整 `LOG_WRITER_WORKERS`，验证操作日志 worker 数配置和上限
- [ ] 给一个中间件补充单元测试，覆盖成功和失败路径

### E.5 面试表达练习

学完后，尝试不用看代码回答这些问题：

- [ ] 一个注册请求从 HTTP 到数据库完整经历了哪些层？
- [ ] 为什么 handler 不直接操作数据库？
- [ ] 为什么 biz 依赖 repository 接口，而不是依赖具体 struct？
- [ ] 为什么密码要用 bcrypt，而不是 MD5/SHA256？
- [ ] JWT logout 为什么需要 Redis 黑名单？
- [ ] 乐观锁比悲观锁适合这里的原因是什么？
- [ ] cursor 分页为什么比 offset 分页更适合任务列表？
- [ ] 操作日志异步写有什么收益和风险？本项目如何降级？
- [ ] Redis 限流脚本为什么要用 Lua？
- [ ] 中间件顺序错了会产生什么问题？

---

## 附录 F：从新手到能独立开发的建议路线

### F.1 第一阶段：能跑、能查、能定位

目标：不要求你能写复杂功能，但要能把服务跑起来，知道问题大概在哪一层。

完成标准：

- 能独立执行 `make up`、`make migrate`、三个 `make run/...`
- 能用 Postman 或 curl 完成注册、登录、创建项目、创建任务
- 能根据 HTTP 状态码判断是认证、权限、参数还是并发冲突问题
- 能用 `rg` 找到某个接口的 handler、biz、repository

### F.2 第二阶段：能改小需求

目标：能改字段、校验、错误消息、简单查询条件。

完成标准：

- 能修改一个 request struct 并理解 JSON tag
- 能修改 biz 校验并补单元测试
- 能修改 repository 查询并知道是否需要 migration
- 能跑目标包测试和全量测试

### F.3 第三阶段：能做完整功能

目标：能从接口到数据库完整实现一个小功能。

完成标准：

- 能改 proto 并重新生成代码
- 能写 migration 和 GORM model
- 能设计 repository 接口
- 能在 biz 层实现业务规则
- 能在 service 层转换 proto
- 能在 gateway handler 暴露 HTTP endpoint
- 能补单元测试、handler 测试和必要的集成测试

### F.4 第四阶段：能考虑工程质量

目标：不仅让功能能跑，还能解释为什么这样设计。

完成标准：

- 能判断是否需要缓存，以及缓存失效策略是什么
- 能判断写接口是否需要幂等
- 能判断接口是否需要限流
- 能为关键路径补 metrics 和日志字段
- 能解释错误码、HTTP 状态码、gRPC status 的映射
- 能识别 goroutine 泄漏、事务边界、并发冲突等风险

---

## 附录 G：术语对照表

| 术语 | 初学者解释 | 项目例子 |
|------|------------|----------|
| Handler | HTTP 请求处理函数 | `internal/gateway/handler` |
| Middleware | 请求进入 handler 前后的通用逻辑 | auth、ratelimit、accesslog |
| Service | gRPC 方法实现层 | `internal/user/service` |
| Biz / Domain | 业务规则层 | 注册校验、权限校验、状态机 |
| Repository | 数据访问抽象 | `UserRepository`、`TaskRepository` |
| Model | 数据库表对应的 Go struct | `data.User`、`data.Task` |
| DTO | 网络传输对象 | HTTP request struct、proto message |
| Entity | 业务实体 | 用户、项目、任务、评论 |
| Migration | 数据库结构变更脚本 | `migrations/*/*.sql` |
| Middleware chain | 多个中间件按顺序执行 | Recovery → MaxBodySize → RequestID → Auth → Handler |
| Singleflight | 合并同 key 并发调用 | 防止热点缓存 miss 同时打到数据库 |
| Interceptor | gRPC 的中间件 | logging、metrics、auth |
| Metadata | gRPC 请求头 | `x-user-id`、`x-internal-token` |
| Context | 请求生命周期和元数据载体 | `context.Context` |
| Graceful shutdown | 优雅关闭 | 停止接收新请求，等待已有请求完成 |
| Cache-aside | 旁路缓存模式 | 先查 Redis，miss 再查 DB，回填缓存 |
| Idempotency | 幂等 | 同一个 key 的重复写请求只执行一次 |
| Optimistic lock | 乐观锁 | `WHERE version = ?` |
| Cursor pagination | 游标分页 | 任务列表按 `created_at,id` 翻页 |
| Trace | 分布式调用链 | OpenTelemetry span |
| Metric | 指标 | Prometheus counter/histogram |
| Structured log | 结构化日志 | Zap JSON 日志 |

---

> **最终学习建议：**
> 1. 初学 Go 不要追求一次看懂所有抽象，先把一条请求链路跑通、画出来、讲清楚。
> 2. 每个新概念都回到项目里找例子：接口看 repository，context 看 handler/biz/data，goroutine 看 log writer。
> 3. 每做一个改动都跑测试。Go 项目的学习闭环是“读代码 → 改代码 → 跑测试 → 解释行为”。
> 4. 当你能独立完成“加字段、加接口、加测试、讲清楚分层原因”，就已经越过 Go 后端入门门槛。

