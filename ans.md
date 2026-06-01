# Task Platform 面试问题参考答案

> 说明：这份答案对应 `ms.md` 的问题清单，不是逐题死记硬背版，而是按面试追问链路整理的“可展开回答”。面试时可以先用每节的“主答”作 30-60 秒回答，再根据追问展开“深挖点”和“风险与改进”。

## 批次一：项目整体、架构、分层与 Gateway

### 1. 项目整体与业务定位

**主答：**

这个项目是一个团队任务协作平台，核心业务包括用户注册登录、项目管理、项目成员权限、任务看板、任务状态流转、评论和操作日志。工程上采用 `api-gateway + user-service + task-service` 的克制型微服务架构，外部 HTTP/JSON，内部 gRPC，数据层使用 PostgreSQL 多 schema，Redis 承担缓存、限流、幂等和 JWT 黑名单，并接入 Prometheus、Grafana、Jaeger、结构化日志和测试体系。

如果 1 分钟介绍，我会突出三点：

1. 业务不是简单 CRUD：项目里有 owner/admin/member 权限矩阵、归档只读、任务状态机、乐观锁、操作日志、游标分页。
2. 工程链路完整：从 gateway、gRPC、数据库、Redis 到前端和测试，覆盖了真实后端服务常见问题。
3. 设计有取舍：没有为了“微服务”而过度拆分，而是把用户域和任务协作域拆开，数据库用多 schema 逻辑隔离，避免校招项目过早陷入分布式事务。

**核心数据对象：**

- `users`：用户账号、密码 hash、状态、昵称头像。
- `projects`：项目基础信息、owner、归档状态、乐观锁版本。
- `project_members`：项目成员关系和角色，承担 RBAC 权限基础。
- `tasks`：任务标题、内容、状态、优先级、creator、assignee、version。
- `task_comments`：任务评论。
- `operation_logs`：项目和任务维度的审计记录。

它们的关系是：用户创建项目并成为 owner；项目有多个成员；项目下有多个任务；任务有评论和操作日志；日志记录项目或任务维度的关键操作。

**为什么不是普通 CRUD：**

普通 CRUD 主要是增删改查。这个项目在 CRUD 之上处理了真实系统才会遇到的问题：

- 并发更新：`projects.version`、`tasks.version` 乐观锁。
- 重复提交：写接口支持 `Idempotency-Key`。
- 越权访问：非成员返回 404，避免资源探测。
- 业务状态约束：任务状态机禁止非法流转。
- 读性能与一致性：Redis cache-aside 和写后失效。
- 审计与排障：操作日志、request_id、trace_id、metrics。
- 工程质量：单元测试、集成测试、前端测试、E2E、压测。

**写请求链路：**

典型写请求链路是：

`Client -> api-gateway middleware -> handler bind/validate -> idempotency SETNX -> gRPC client metadata injection -> task-service interceptor -> service -> biz 权限/状态/事务 -> data repository -> PostgreSQL -> async operation log -> HTTP envelope response`

关键点是参数校验和幂等在 gateway handler，业务规则在 biz，持久化在 data，日志异步写入不阻塞主链路。

**读请求链路：**

典型读请求是：

`Client -> api-gateway -> task-service -> PostgreSQL/Redis -> gateway enrichment -> response`

例如任务列表由 task-service 返回任务本身，gateway 再用 user-service 的 `BatchGetUsers` 补充展示所需的用户信息，避免 task-service 存储昵称头像，也避免 N+1 RPC。

**正确性、性能、可用性的优先级：**

- 正确性优先：权限矩阵、owner 转让事务、状态机、乐观锁、非成员 404、归档只读。
- 性能优先：用户/项目缓存、BatchGetUsers、游标分页、异步操作日志、连接池、压测调优。
- 可用性优先：Redis 缓存失败降级为查 DB，限流失败 fail-open，trace exporter 失败不阻止服务启动。

**上线真实生产前的缺口：**

当前项目是面试型工程项目，如果上线真实生产，还需要补：

- HTTPS/mTLS、Secret 管理、生产级内部服务认证。
- refresh token、多端登录、密码修改后 token 失效。
- 更严格的幂等语义：pending 请求等待、绑定 method/path/body hash、缓存 status/header。
- 告警规则、日志采集、链路采样策略、SLO。
- CI/CD、Kubernetes 部署、滚动升级、灰度发布。
- 数据备份、迁移审查、审计日志可靠投递。
- 前后端契约自动生成或契约测试。

**如果被问“项目最大的亮点是什么”：**

我会选三个：

1. **权限 + 状态机：** 不是接口堆砌，而是把协作系统的核心规则落在 biz 层。
2. **并发与幂等：** 乐观锁解决并发更新，Redis `SETNX` 降低重复提交风险。
3. **可观测性与测试：** metrics、trace、日志、Grafana 和 testcontainers 让系统可验证、可排障。

### 2. 架构拆分与微服务边界

**为什么不是单体：**

这个项目拆分成 gateway、user-service、task-service，主要是为了展示服务边界和内部 RPC，而不是为了追求服务数量。用户认证和任务协作是两个相对清晰的业务域：用户域负责账号、密码、JWT 签发和用户查询；任务域负责项目、成员、任务、评论和日志。拆开后可以明确职责，也能演示 gRPC metadata、内部认证、错误映射和跨服务查询。

但它不是“完全生产级”的微服务，因为两个业务服务仍共用一个 PostgreSQL 实例，只通过 schema 隔离。这是有意取舍：在当前规模下避免跨库事务、部署复杂度和分布式一致性成本。

**为什么只拆 2 个业务服务：**

comment、member、operation log 都围绕 task/project 协作域，独立拆服务会带来额外 RPC、事务边界和一致性问题，收益不大。服务拆分不应该按表拆，而应该按业务边界、数据所有权、变更频率和团队边界拆。

如果未来通知、搜索、文件上传成为独立能力，就可以考虑单独服务，因为它们有独立数据模型、伸缩模式和故障隔离需求。

**gateway 的职责：**

gateway 负责：

- HTTP 路由和参数绑定。
- JWT 验签和当前用户注入。
- request_id、trace、metrics、access log、CORS、安全头、限流。
- HTTP/gRPC 错误转换和统一 envelope。
- 写接口幂等处理。
- 展示层需要的用户信息 enrichment。

gateway 不应该负责：

- 项目/任务权限核心判断。
- 状态机校验。
- 数据持久化。
- 业务事务协调。

这些应该在 task-service 的 biz 层。

**user-service 和 task-service 的边界：**

user-service 持有用户密码 hash 和 JWT secret，负责注册、登录、用户查询、批量用户查询。task-service 不解析 JWT，也不持有 JWT secret，只信任 gateway 注入的 `x-user-id`，并通过 `x-internal-token` 校验调用来源。

这样做的好处是：

- JWT 验签逻辑集中在 gateway。
- task-service 不接触外部认证细节。
- 内部服务只处理业务身份，不处理 HTTP 认证语义。

风险是：内部 metadata 一旦被伪造，task-service 可能错误信任用户身份。因此需要内部认证令牌，生产中最好升级为 mTLS、服务网格或 SPIFFE/SPIRE 这类工作负载身份。

**gRPC metadata 的作用：**

- `x-internal-token`：内部 RPC 静态认证，防止绕过 gateway。
- `x-user-id`：当前登录用户 ID，供 biz 权限判断。
- `x-username`：日志或展示辅助信息。
- `x-request-id`：跨 gateway 和服务串联日志。
- trace context：由 OpenTelemetry gRPC instrumentation 传播，用于 Jaeger 链路追踪。

**为什么内部用 gRPC unary：**

gRPC 的优势是强契约、二进制协议、代码生成、状态码体系和 metadata 支持。当前接口都是请求/响应模型，unary 足够。消息队列适合异步事件，不适合作为同步查询和权限判断链路；REST 用于外部兼容更好，内部则 gRPC 更适合服务间调用。

**proto 升级兼容原则：**

- 不要删除或复用已有 field number。
- 新增字段要使用新的编号。
- 字段语义不能随意改变。
- 对可选字段要谨慎处理 proto3 默认值。
- 服务方法变更要考虑旧客户端兼容。

项目里 `ListTasksRequest.status` 是一个典型点：proto3 的 int32 默认是 0，而 0 又代表 todo。当前 gateway 用 `-1` 作为“未传 status”的哨兵值，再在 service 层转成 `nil` filter。这个方案能工作，但不是最优；更稳妥的是 proto3 `optional int32 status` 或 wrapper type。

**单 PostgreSQL 多 schema 的取舍：**

优点：

- 用户域和任务域逻辑隔离。
- 迁移目录和 schema 独立。
- 避免过早引入跨库事务。
- 本地部署和测试成本低。

缺点：

- 数据库实例仍是共享故障域。
- 服务不能完全独立扩缩容数据库。
- 容易诱惑开发者跨 schema join。
- 将来拆库需要处理跨服务一致性、历史数据迁移和展示信息冗余。

**跨服务一致性：**

添加成员、指派任务时，task-service 会调用 user-service 校验用户存在且 active。这不是强一致，因为用户可能在校验后被禁用。当前项目接受这个短暂窗口。生产上可以用事件同步用户状态快照、补偿任务、或在关键写操作上增加二次校验。

**gRPC 超时、重试和熔断：**

当前回答可以说：项目已经通过 context 传播和 gateway 请求上下文具备基本取消能力，但更完整的生产方案应该在 gRPC client 层设置明确 deadline，例如 2s；对幂等读请求可以配置有限重试；对写请求要谨慎重试，必须结合幂等 key；熔断可用客户端中间件或服务网格实现。

`grpc.NewClient` 创建连接时通常不会立即完成真实网络连接，RPC 发起时才会连接。因此 gateway 启动成功不代表 user-service/task-service 一定可用。更严格的 ready 检查应主动调用 gRPC health check 或执行短超时探测。

**如果被质疑“过度微服务”：**

可以回答：我没有把每张表都拆服务，而是只拆两个高内聚业务域；数据库也用单实例多 schema 控制复杂度。这个项目的目的不是证明“服务越多越好”，而是展示在有限规模内如何处理服务边界、内部调用、安全、错误映射和可观测性。

### 3. 分层设计与代码组织

**四层结构的职责：**

- handler：HTTP 参数绑定、幂等检查、调用 gRPC client、统一响应。
- service：gRPC 方法实现、基础身份检查、proto 和 biz 参数转换。
- biz：权限、状态机、业务规则、事务协调、缓存策略、日志触发。
- data：GORM model、repository、SQL 查询和数据库错误转换。

这套分层的关键价值是隔离变化：HTTP 改动不影响业务规则，数据库查询优化不影响接口契约，biz 层可以脱离 Gin/gRPC 做单元测试。

**为什么 handler 不做权限判断：**

权限属于业务语义。例如 member 是否能删除某个任务，取决于角色、creator、任务状态、项目归档状态，这些都需要访问业务数据。如果放在 gateway，就会导致 gateway 知道 task-service 的领域规则，破坏服务边界，也容易出现不同入口权限不一致。

**service 和 biz 的边界：**

service 做“协议层校验”和转换，例如检查用户身份是否存在、把 proto request 转成 biz 参数。biz 做“业务规则校验”，例如项目是否归档、状态能否流转、角色是否有权限、目标用户是否为成员。

**data 层为什么不做业务判断：**

repository 应该只负责数据读写。如果把权限判断塞进 repository，后续权限变化会影响数据层，测试也会变得脆弱。repository 可以把数据库唯一冲突、记录不存在、乐观锁 RowsAffected=0 转成领域可理解的错误，但不应该决定 owner/admin/member 能不能操作。

**biz 层是否被框架污染：**

目前 biz 层主要依赖 repository 接口、用户服务接口、GORM 事务对象、Redis client 和 log writer。理想情况下，biz 不应该直接依赖 Gin/gRPC，这一点项目做到了。GORM 和 Redis 出现在 biz 中是为了事务和缓存，属于可以接受但仍有优化空间的取舍。进一步抽象可以用 transaction manager、cache repository decorator 来降低基础设施耦合。

**`pkg/x*` 的定位：**

`xerr`、`xjwt`、`xredis`、`xpgsql`、`xratelimit`、`xgrpc`、`xhttp`、`xtrace` 是跨服务基础设施封装。它们适合放在 `pkg`，因为多个服务复用。需要警惕的是不要为了“看起来高级”过早抽象；只有被多个模块稳定复用、边界清晰的能力才适合下沉。

**领域常量放在 data 包是否合理：**

当前角色、状态、优先级常量放在 `internal/task/data`，好处是 model、repository 和 biz 都能直接使用，简单。缺点是领域语义和持久化层耦合。更严格的 DDD 会把这些常量放到 domain 包，data model 做映射。面试中可以承认这是项目规模下的实用选择，后续领域复杂后再拆。

**GORM model 和业务实体是否应该分离：**

分离的好处是领域模型更纯净，不受 GORM tag、DeletedAt 等影响；缺点是映射代码增加。当前项目规模适中，GORM model 直接作为内部实体能减少样板代码。若业务复杂到需要聚合根、不变量和复杂领域行为，就应该分离。

**如何保持 proto、HTTP DTO、GORM model 一致：**

当前主要靠测试和手写转换函数，例如 `toProtoTask`。更进一步可以：

- 使用 buf lint 和 breaking check。
- 用 OpenAPI 或 proto 生成前端类型。
- 增加契约测试。
- 将枚举集中定义，前端自动生成常量。

**重复代码与抽象：**

`TaskBiz` 和 `ProjectBiz` 都有类似 `getProjectAndMember`，这是轻微重复。是否抽象取决于重复是否稳定。如果抽象会让权限逻辑变得隐晦，不如保持局部清晰。面试里可以说：我会先观察重复是否继续扩散，若多个 biz 都需要，就抽成 `PermissionService` 或 `ProjectAccessChecker`。

**缓存逻辑放 biz 的评价：**

项目缓存读写在 biz 层，因为缓存策略本身和业务语义有关，例如先校验成员再读项目缓存，写操作后失效缓存。缺点是 biz 承担了基础设施细节。可以改成 repository decorator，但要注意权限校验顺序不能被缓存层绕过。

### 4. API Gateway 与 HTTP 设计

**中间件顺序：**

当前顺序大体是：

`MaxBodySize -> SecurityHeaders -> RequestID -> HTTPTrace -> HTTPMetrics -> AccessLog -> CORS -> RateLimit(IP) -> Auth -> RateLimit(user) -> Handler`

这个顺序的理由：

- body limit 尽早拦截大请求。
- security headers 应尽早加到响应。
- request_id 要早于日志、metrics、错误响应。
- trace/metrics/log 包裹后续链路。
- IP 限流放 Auth 前，保护登录/注册和 JWT 验签成本。
- Auth 后才能做 user 级限流。
- Handler 最后处理业务。

如果 Auth 放在 IP 限流前，攻击者可以用大量无效 token 消耗 JWT 验签和 Redis 黑名单查询资源。RequestID 如果在 AccessLog 后，日志和错误响应就无法稳定带 request_id。

**统一 envelope 的优缺点：**

优点：

- 前端处理简单，所有响应都有 `code/message/request_id/data`。
- gRPC 错误可以统一映射成 HTTP 错误。
- request_id 对排障友好。
- 业务错误和 HTTP 状态能同时表达。

缺点：

- 与纯 REST 风格不完全一致。
- 需要前后端都遵守 envelope。
- 文件下载、流式响应、大响应不适用。
- 重放幂等响应时要注意 status/header。

**为什么状态和指派用独立接口：**

`POST /tasks/:id/status` 和 `POST /tasks/:id/assign` 是领域命令，背后有专门权限和状态机。若允许 `PUT /tasks/:id` 一次性修改所有字段，就容易绕过状态机或权限矩阵。独立端点让接口语义更清晰，也方便前端基于不同权限展示按钮。

**幂等为什么在 handler：**

幂等依赖 HTTP header、响应体捕获和具体写接口语义，所以放在 gateway handler 比统一中间件更容易控制。它必须在参数绑定和基础校验之后执行，避免无效请求占用 `Idempotency-Key`。如果在中间件阶段就 SETNX，错误请求也会污染幂等缓存。

**不传 Idempotency-Key 怎么办：**

当前系统允许不传，直接正常执行，只是没有重复提交保护。是否强制取决于产品和客户端能力。对创建任务、创建项目、转让所有权这类重要写操作，生产上可以要求强制传；对一些天然幂等或数据库唯一约束保护的操作，可以不强制。

**当前幂等实现的边界：**

`bodyCaptureWriter` 捕获响应体并缓存，但没有缓存 HTTP status 和 headers。重复请求统一返回 200，这对 201 Created、204 No Content 或特殊 header 并不严格。同时，pending 状态下重复请求会立即返回“request already processed”，严格语义上并不等于第一个请求已经成功。更完整方案应该让后续请求等待短时间、轮询结果或返回 409/202，并绑定 method/path/body hash。

**HTTP 状态语义：**

- 400：参数非法或前置条件失败。
- 401：未认证或 token 无效。
- 403：已认证但无权限，除资源隐藏场景外。
- 404：资源不存在或非成员访问需要隐藏资源存在性。
- 409：乐观锁冲突或资源已存在。
- 429：限流。
- 500：内部错误，对外隐藏细节。
- 503/504：上游不可用或超时。

**request_id 设计：**

前端会生成 `X-Request-Id`，服务端也可以生成或接收。实际生产中可以接受客户端 request_id，但应保证格式和长度，避免日志污染。响应返回 request_id 能让用户报错时提供定位线索，服务端用它串联 gateway 和服务日志。

**context cancellation：**

Gin 的 request context 会传到 gRPC client，再传到服务端和 GORM/Redis 调用。如果客户端断开，理论上 context 会取消，下游应尽早停止。但前提是每层都使用 `ctx`，并且 DB/Redis/gRPC 调用尊重 context。生产还应设置明确 timeout，而不只依赖客户端断开。

**enrichment 为什么在 gateway：**

task-service 只保存用户 ID，不保存昵称头像，避免复制用户展示信息。gateway 面向前端展示，可以调用 user-service 的 `BatchGetUsers` 一次性补齐 creator/assignee/comment author 信息，避免 N+1。

## 批次二：认证安全、权限、状态机与数据库

### 5. 认证、JWT 与安全

**JWT 签发与验签：**

注册和登录直接返回 JWT，是为了降低使用成本，注册后无需再登录。JWT 使用 HS256，优点是实现简单、性能好、只需要共享 secret；缺点是签发方和验签方都持有同一个 secret，泄露后风险大。若改成 RS256，user-service 持有私钥签发，gateway 持有公钥验签，密钥暴露面更小，但需要密钥轮换和公钥分发机制。

**claims 设计：**

- `sub`：用户 ID，是权限判断的主体。
- `jti`：token 唯一 ID，用于 logout 黑名单。
- `iat`：签发时间。
- `exp`：过期时间。
- `username`：辅助展示和日志。

生产中还可以增加 `iss/aud`，明确签发方和受众，避免 token 被其他系统误用。

**logout 为什么需要黑名单：**

JWT 本身是无状态的，签发后在过期前都有效。logout 想立即撤销 token，就需要在 gateway 验签后再检查 Redis 黑名单。黑名单 key 使用 `blacklist:<jti>`，TTL 应等于 token 剩余有效期，避免永久占用 Redis。

**Redis 黑名单失败策略：**

当前 auth 中间件查黑名单失败返回 500，属于 fail-close。理由是安全优先：如果 Redis 不可用，无法确认 token 是否被撤销，不放行更保守。缓存读取失败可以降级，因为只是性能问题；黑名单失败是安全语义问题，所以策略不同。

**密码安全：**

密码用 bcrypt，默认 cost=10。bcrypt 是慢 hash，能提高离线破解成本。cost 太低不安全，太高会拖慢登录甚至被攻击者利用造成 CPU 压力，所以项目允许 `BCRYPT_COST` 配置，并限制在 4 到 14。登录时如果发现旧 hash cost 低，会尝试 rehash，新 hash 写入失败只记日志，不影响本次登录成功。

**登录错误消息：**

登录失败统一返回“invalid account or password”，不区分用户不存在、密码错误、用户禁用，避免被枚举账号状态。

**token 生命周期：**

当前 access token TTL 为 2 小时，适合面试项目。生产中通常会使用短 access token + refresh token。引入 refresh token 后需要新增 token 表或 Redis 存储、轮换、撤销、多端管理、设备信息和异常登录检测。

**用户禁用后的旧 token：**

当前 gateway 主要校验 JWT 和黑名单，如果用户被禁用后旧 token 是否立即失效，取决于 gateway 是否每次查询用户状态。当前没有每次查状态，所以会有窗口。生产可以在禁用用户时把该用户所有 token 加入黑名单，或在 token claims 中加入 token_version，用户状态变化后递增版本。

**前端 token 存储：**

localStorage 使用简单，但有 XSS 风险。Bearer token 不走 cookie 时 CSRF 风险较低，但 XSS 更关键。生产可考虑 httpOnly secure cookie + SameSite + CSRF token，或严格 CSP、输入输出转义、最小 token TTL。

**内部服务安全：**

`x-internal-token` 是静态共享密钥，只能算基础保护。生产中更推荐 mTLS、服务网格、工作负载身份和网络策略。gRPC 当前使用 insecure transport credentials，适合本地开发，不适合跨不可信网络。

### 6. 权限模型与业务规则

**角色权限：**

- owner：项目最高权限，能归档/取消归档、转让所有权、编辑项目、管理成员、管理任务。
- admin：能协助管理任务和部分成员，但不能管理 owner/admin，不能归档或转让。
- member：能参与项目，创建任务，只能编辑/删除/变更自己有权限的任务和评论。

owner 不能直接退出项目，因为必须保证项目至少有一个 owner。退出前要先转让所有权。

**非成员返回 404：**

这是为了防止资源探测。若非成员访问某项目返回 403，就说明项目存在；返回 404 能隐藏资源存在性。缺点是排障和前端提示不够精确，所以内部日志应记录真实原因，对外统一隐藏。

**归档只读：**

项目归档后所有写操作都应拒绝，包括项目更新、成员变更、任务创建/更新/删除、评论创建/删除、状态变更、指派等。读操作仍允许成员查看历史数据和日志。返回 `FAILED_PRECONDITION` 比 `PERMISSION_DENIED` 更准确，因为不是用户角色不够，而是项目状态不允许写。

**owner 转让：**

项目 owner 的“真相来源”是 `projects.owner_id`，`project_members.role=owner` 是权限和展示辅助。转让时必须在一个事务内：

1. 更新 `projects.owner_id` 和 version。
2. 把旧 owner 的 member role 改为 admin。
3. 把目标成员 role 改为 owner。

同时数据库用 `project_members(project_id) WHERE role=0` 部分唯一索引保证一个项目最多一个 owner。它能防止多个 owner，但不能保证至少一个 owner，所以业务上禁止 owner 直接退出，并用事务维护一致性。

**成员和任务权限细节：**

当前 member 对任务的权限主要按 `creator_id` 判断，即“自己创建的任务”。这比按 assignee 更保守，但也带来产品语义问题：如果一个任务被指派给 member，但不是他创建的，他可能无法变更状态。面试中可以主动指出这是一个可讨论的业务取舍，真实产品可能会改成 creator 或 assignee 都有部分权限。

如果 creator 离开项目，任务仍保留 creator_id。owner/admin 应能继续维护；普通 member 不应自动继承。若 assignee 被移出项目，生产中可以在移除成员时清空相关任务 assignee，或保留历史但禁止后续按该 assignee 查询活动任务。

**添加成员和指派任务：**

task-service 通过 user-service 校验用户存在且 active，并检查目标用户是否是项目成员。这里存在校验后用户被禁用的竞态，当前项目接受。更强方案是用户状态变更发布事件，task-service 维护用户状态快照，或关键操作二次校验。

**前端权限控制：**

前端隐藏按钮只是用户体验，不能替代后端校验。真正的权限必须在 biz 层执行，因为请求可以绕过前端直接调用 API。

**是否需要 policy engine：**

当前权限矩阵不复杂，硬编码在 biz 层更直观、测试成本低。若角色、资源、动作和条件继续增多，可以抽象为 `PermissionService` 或引入策略表，但不建议一开始就上复杂 policy engine。

### 7. 任务状态机与领域约束

**状态流转：**

当前状态机是：

- `todo -> doing/done/cancelled`
- `doing -> done/cancelled/todo`
- `done -> doing`
- `cancelled -> todo`

它表达的是一个轻量协作系统：完成的任务如果发现问题可以回到 doing；取消的任务只能重新打开到 todo，避免直接进入 doing/done 跳过确认。

**为什么状态接口独立：**

状态变更有专门权限、版本控制和状态机校验，所以不能混在普通 `UpdateTask` 里。普通更新只处理标题、内容、优先级、截止时间等字段。

**version 设计：**

`ChangeTaskStatus` 要求客户端传 version，能防止用户基于旧状态做变更。`AssignTask` 当前使用读取到的 task.Version 做更新，不要求客户端传 version，这在一致性上比 ChangeStatus 弱。更统一的设计应该让 AssignTask 也传 version，避免并发指派覆盖。

**前后端状态机一致性：**

前端 `useStatusTransitions` 用于 UI 限制和交互提示，后端状态机才是最终约束。为了防止漂移，可以把状态机规则从后端生成到前端，或者用契约测试校验前后端枚举和转换规则一致。

**proto3 默认值问题：**

任务状态 `todo=0`，而 proto3 int32 默认也是 0。列表筛选时如果直接使用 `status=0`，就无法区分“未传 status”和“筛选 todo”。当前 gateway 把未传 status 设置为 `-1`，service 中 `req.Status != -1` 才设置 filter，解决了这个问题。但这依赖约定，最好改为 `optional int32 status`。

**字段更新语义：**

`UpdateTask` 中 title 为空表示不更新标题，因为业务也不允许空标题；content 每次都写入，可以清空内容。due_time 为空表示不更新，因此目前没有“清空 due_time”的能力。生产中可引入 explicit null、field mask 或单独字段 `clear_due_time`。

**due_time 类型：**

当前 proto 用 string 表示 due_time，简单但不严格。更好的方式是 `google.protobuf.Timestamp`，能统一时间语义。若继续用 string，需要明确 RFC3339 格式、时区、解析层和错误处理。

**priority 和 extra：**

priority 应在 service 或 biz 层校验合法枚举值，避免前端传 999。`extra` 使用 JSONB 适合 labels/checklist/attachments 这种扩展字段，但如果字段要频繁查询或参与权限判断，就应该拆成结构化表或明确索引。

### 8. 数据库设计与事务一致性

**为什么 PostgreSQL：**

PostgreSQL 支持事务、部分唯一索引、JSONB、丰富索引、强一致性和成熟生态，适合这种协作型业务。项目用到了多 schema、部分唯一索引、JSONB、事务和复合索引。

**多 schema 与无跨 schema 外键：**

`user_svc` 和 `task_svc` 逻辑隔离，迁移也分开。没有跨 schema 外键是为了保持服务边界，避免 task-service 数据库强依赖 user-service 表。代价是引用完整性要由业务层保证，例如添加成员和指派任务时调用 user-service 校验。

**软删除与物理删除：**

`users`、`projects`、`tasks` 使用软删除，因为它们是核心业务对象，需要保留历史、支持恢复和审计。`comments`、`project_members` 物理删除，因为它们更像关系或附属数据，当前产品不要求恢复。真实生产中，如果要审计成员历史或评论删除内容，可以改成软删除或把删除前内容写入操作日志。

**部分唯一索引：**

软删除配合唯一约束时，如果直接对 username/email 建唯一索引，软删除后仍不能复用。因此使用 `WHERE deleted_at IS NULL` 的部分唯一索引，只约束未删除数据。

**关键索引：**

- users：username/email 部分唯一索引。
- projects：`(owner_id, name)` 部分唯一，防止同 owner 下重名。
- project_members：`(project_id,user_id)` 唯一，防重复成员；`(project_id) WHERE role=owner` 保证最多一个 owner。
- tasks：`(project_id, created_at DESC, id DESC)` 支持默认游标分页；按 status、assignee 组合建立索引。
- operation_logs：按 project_id/task_id + created_at/id 支持日志分页。

**乐观锁：**

核心 SQL 是：

```sql
UPDATE ...
SET ..., version = version + 1
WHERE id = $1 AND version = $2;
```

如果 RowsAffected 为 0，说明记录不存在或版本冲突。项目返回 `ABORTED` 并映射到 HTTP 409，客户端需要重新读取最新版本再提交。

**事务边界：**

CreateProject 在一个事务里创建 project 和 owner member。TransferOwnership 在一个事务里更新 project owner_id 和两条 member role，保证两份 owner 数据一致。

跨服务 user 校验不放在事务中执行，这是为了避免数据库事务持有期间等待外部 RPC，造成长事务和锁占用。缺点是存在校验和提交之间的状态变化窗口，当前通过业务可接受性处理。

**数据库错误处理：**

当前 repository 通过字符串包含 `"duplicate key"` 判断唯一冲突，能工作但不够健壮。更好的方式是识别 PostgreSQL SQLSTATE，例如 `23505` 表示 unique_violation。这样不会受错误消息语言和驱动格式影响。

**JSONB 使用边界：**

JSONB 适合扩展字段、操作日志 detail 这种结构可能变化的数据。不适合强约束、高频查询、需要外键关系的数据。高频查询 JSONB 字段时要加 GIN 或表达式索引，否则会拖慢。

**查询性能风险：**

`ILIKE '%keyword%'` 无法利用普通 BTree 索引，任务量大后会慢。优化方案可以是 pg_trgm trigram index、PostgreSQL full-text search，或者搜索独立后接入 Elasticsearch/OpenSearch。



## 批次三：并发、幂等、Redis、分页、日志与可观测性

### 9. 并发控制、幂等与重试

**乐观锁解决什么：**

乐观锁解决的是“多个客户端基于同一旧版本更新同一资源”的问题。项目和任务是协作中的核心可编辑对象，容易被并发修改，所以加 version。comment/member 当前操作更简单，主要由唯一约束或权限规则保证，不一定需要乐观锁。

当两个请求都拿到 `version=1`，第一个更新成功并把 version 改成 2；第二个请求执行 `WHERE id=? AND version=1` 时 RowsAffected=0，返回 `ABORTED`，gateway 映射为 HTTP 409。客户端应该重新拉取最新数据，提示用户冲突或重新提交。

**为什么不用悲观锁：**

协作系统读多写少，冲突概率通常不高。悲观锁会增加锁等待、事务时间和死锁风险，也会影响用户体验。乐观锁更适合 HTTP 请求这种无状态交互。

**幂等解决什么：**

幂等解决的是客户端重试、网络超时、用户双击导致同一个写操作被执行多次的问题。它和乐观锁不同：乐观锁防止旧版本覆盖新版本；幂等防止同一个请求重复执行。

创建任务没有天然唯一键，最依赖幂等。添加成员有 `(project_id,user_id)` 唯一约束，即使没有幂等也不会重复插入，但重复请求会收到 `ALREADY_EXISTS` 而不是第一次的成功响应。

**当前 Redis SETNX 流程：**

1. handler 绑定并校验请求体。
2. 读取 `Idempotency-Key`。
3. 按用户 ID scope 成 `userID:key`。
4. Redis `SETNX idempotency:<key> pending EX 24h`。
5. 成功说明本请求第一次执行，捕获响应体。
6. 业务成功则缓存响应体；业务失败则删除 key，允许客户端修正后重试。
7. 如果 `SETNX` 失败，返回缓存响应；若值还是 pending，则返回一个“正在处理/已处理”的 OK 响应。

**这个实现的不足：**

严格幂等应该让重复请求拿到和第一次请求完全一致的 status、headers、body。当前只缓存 body，重复请求返回 200；如果第一次是 201 Created，语义不完全一致。pending 请求也被直接返回 OK，但第一个请求可能最后失败，这会误导客户端。

更严谨的方案：

- key 绑定 `method + path + user_id + body_hash`。
- value 存 `{status, headers, body, state}`。
- pending 时短轮询等待完成，或返回 409/202。
- 设置 pending 超时，避免第一个请求崩溃后 key 长时间卡住。
- 前端网络重试必须复用同一个 idempotency key。

**前端 idempotency key 的问题：**

前端 mutation 当前每次调用生成一个 randomUUID，能防止同一次 mutation 请求在传输层被重放，但如果用户双击触发两次 mutation，就会生成两个 key，后端无法识别为同一个业务操作。要防双击，需要按钮 loading 禁用、表单提交锁，或者前端在同一个用户动作生命周期内复用同一个 key。

**服务端自动重试：**

乐观锁冲突通常不适合服务端自动重试，因为服务端不知道用户是否接受新版本上的变更。读请求或幂等写请求可以有限重试网络错误；非幂等写请求必须谨慎，必须结合 Idempotency-Key。

### 10. Redis：缓存、限流、黑名单与降级

**Redis 的四类职责：**

- 用户和项目缓存。
- IP/auth/user 级 token bucket 限流。
- 写接口 Idempotency-Key。
- JWT logout 黑名单。

这些职责共用一个 Redis 简化部署，但生产中要关注隔离：限流高流量不能影响黑名单，缓存大 key 不能挤压幂等 key。可以用不同 DB、不同 prefix、不同实例或 Redis Cluster 做隔离。

**降级策略：**

- 缓存失败：降级查数据库，因为缓存只是性能优化。
- 限流失败：fail-open 放行，避免 Redis 故障导致全站不可用。
- 幂等失败：继续执行，但失去重复提交保证，因为可用性优先。
- 黑名单失败：fail-close 返回 500，因为这是安全语义，无法确认 token 是否被撤销。

**cache-aside 流程：**

读：先查 Redis；命中返回并记录 cache hit；未命中查 DB；DB 结果写入 Redis，TTL 5 分钟。写：先更新 DB，成功后删除缓存，后续读请求重新加载。

项目详情缓存必须先校验成员再读缓存，不能先读缓存再判断权限，否则可能让非成员通过缓存拿到项目数据。

**缓存一致性：**

TTL 5 分钟是性能和一致性的折中。写后主动失效能减少脏读。极端情况下如果失效失败，可能短时间读到旧数据，日志会记录 warn。生产可以加入重试、延迟双删、版本号缓存或更强一致策略。

**BatchGetUsers 的顺序问题：**

当前实现先 append 缓存命中用户，再 append DB missed 用户，可能不保留请求 ID 顺序。如果前端按 map 使用影响不大；如果依赖顺序就有问题。更稳妥的做法是先去重，再按原始 ids 顺序组装返回。

**token bucket Lua：**

Lua 脚本把读取 token、计算补充、扣减 token、写回和设置过期时间放在 Redis 单次原子执行中，避免多命令并发竞态。key 过期时间设置为 `ceil(capacity/rate)+60`，表示桶自然恢复到满所需时间再加缓冲，避免长期保存冷 key。

**真实 IP 问题：**

`c.ClientIP()` 在反向代理后依赖 trusted proxy 配置。如果没有正确配置，攻击者可能伪造 `X-Forwarded-For` 或所有请求都被识别为代理 IP。生产应在 Gin/Ingress/Nginx 中配置可信代理，并统一真实 IP 提取策略。

**Redis key 设计：**

典型 prefix 包括 `user:<id>`、`project:<id>`、`ratelimit:ip:<ip>`、`ratelimit:ip:auth:<ip>`、`ratelimit:user:<user_id>`、`idempotency:<user_id>:<key>`、`blacklist:<jti>`。生产可加入环境和服务名前缀，例如 `task-platform:prod:blacklist:<jti>`。

### 11. 分页、查询与数据返回

**offset 与 cursor：**

项目列表使用 offset，因为项目数量相对少，需求简单。任务列表使用 cursor，因为任务数量可能较大，且看板滚动加载需要稳定分页。

offset 的问题是大 offset 需要跳过大量行，性能差；翻页期间新增或删除数据会导致重复或漏数据。cursor 基于 `(created_at,id)` 定位下一页，性能更稳定，适合无限滚动和时间线。id 用于处理 created_at 相同的稳定排序问题。

**cursor 结构：**

任务 cursor 是 base64url 编码的 JSON，包含 `created_at`、`id` 和后续补充的 `filter_hash`。base64 只是编码，不是加密。客户端可以篡改，所以服务端必须校验字段格式、时间格式和 filter_hash。

**filter_hash：**

filter_hash 解决“翻页过程中筛选条件变化”的问题。比如第一页筛选 todo，拿到 cursor 后改成 done，如果不校验，cursor 的排序锚点和查询条件不匹配，会导致结果错乱。项目用 SHA256 前 16 位，碰撞概率对业务可接受。新增排序条件或筛选条件时，也应纳入 hash。

**潜在改进：**

- cursor 可以签名，防止客户端构造任意 cursor。
- status 哨兵 `-1` 可以改为 proto optional。
- `assignee_id=""` 当前表示不过滤，因此无法表达“未指派任务”，可以增加 `unassigned=true` 或 optional assignee filter。
- keyword 当前后端只查 title，前端匹配 title/content，这存在契约不一致，应统一。

**评论分页：**

评论使用 `after_id` 正向分页，适合按时间从早到晚加载。它先查 anchor comment，再用 `(created_at,id)>anchor` 查询。数据量大时也需要 `(task_id, created_at, id)` 索引，否则 anchor 后续查询可能慢。

**返回字段不足：**

proto 中 Project/Task/Comment 没有完整返回 created_at/updated_at，这会影响前端排序、展示和冲突提示。当前可作为简化设计，生产上建议补齐时间字段。

### 12. 操作日志与异步处理

**为什么异步：**

操作日志是审计和展示辅助，不应该阻塞主业务响应。同步写日志会增加写接口延迟，并且日志库短暂变慢可能拖累业务。异步 channel + batch 能降低 DB 写放大，提高吞吐。

**当前 LogWriter 设计：**

- channel 容量 1024。
- batch size 64。
- flush interval 100ms。
- channel 满时同步降级写。
- batch 写失败最多重试 3 次，指数退避。
- Shutdown 时 flush 当前 batch 和剩余 channel。
- worker panic 会记录指标并重启。

**channel 满为什么同步写：**

直接丢日志会影响审计完整性；无限扩容 channel 会把问题转成内存风险。同步降级能保住日志，但会牺牲当前请求延迟，并通过指标暴露压力。如果持续 channel full，说明 DB 写入能力或 worker 配置跟不上，应扩 worker、优化 DB、引入队列或降级策略。

**可靠性边界：**

内存 channel 无法抵抗进程崩溃。业务成功但日志还在内存中时，实例被强杀会丢日志。当前对面试项目可接受；生产若要求审计可靠，应使用 outbox pattern 或消息队列。outbox 的思路是业务事务内写业务表和 outbox 表，后台异步投递日志，避免业务 DB 成功但消息没发出去。

**日志 detail：**

当前很多 detail 是 `{}`，更完整的审计应记录关键字段变更前后值，例如 title、status from/to、assignee from/to、member role from/to。但不能记录密码、token、敏感个人信息。删除评论是否记录原文，要看隐私和合规要求。

**多实例影响：**

多实例 task-service 下每个实例都有自己的内存 LogWriter，这没问题，但指标要按 instance 观察。幂等和限流使用 Redis，全局有效；操作日志 channel 不是全局队列，实例崩溃仍可能丢本实例未 flush 的日志。

### 13. 错误码、异常处理与可观测性

**统一错误码：**

项目定义了常见 12 个错误码，映射到 gRPC code 和 HTTP status。统一错误码的价值是前端和 gateway 不用解析错误字符串，日志和监控也能按 code 聚合。

典型映射：`INVALID_ARGUMENT -> 400`、`UNAUTHENTICATED -> 401`、`PERMISSION_DENIED -> 403`、`NOT_FOUND -> 404`、`ALREADY_EXISTS -> 409`、`ABORTED -> 409`、`RESOURCE_EXHAUSTED -> 429`、`UNAVAILABLE -> 503`、`DEADLINE_EXCEEDED -> 504`、`INTERNAL -> 500`。

`CodeInternal` 对外统一成 `internal server error`，避免泄露数据库错误、连接串、堆栈信息。真实错误应进结构化日志。

**`FAILED_PRECONDITION` 的语义：**

归档项目只读、状态非法流转、目标用户不是项目成员，这些不是“参数格式错”，也不完全是“权限不足”，而是当前业务状态不允许执行，所以用 `FAILED_PRECONDITION` 更准确。

**request_id 与 trace_id：**

request_id 是业务排障 ID，可从响应返回给用户，贯穿日志。trace_id 是分布式追踪 ID，用于 Jaeger 查看完整调用链。没有 trace 时，request_id 仍能串联 gateway 和服务日志。有 trace 时，trace_id 能看到耗时拆解。

**RED 指标：**

RED 指 Rate、Errors、Duration。项目中体现为 HTTP 请求总数和 QPS、HTTP/gRPC 错误码计数、HTTP/gRPC 延迟 histogram。再加上 DB/Redis 指标、限流命中、log writer channel full，就能回答“服务是否可用、哪里慢、哪里错、下游是否异常”。

**为什么 label 不能高基数：**

Prometheus label 如果使用原始 URL、user_id、task_id，会造成时间序列爆炸，内存和查询性能都会恶化。应该使用模板路径，例如 `/api/v1/tasks/:id`。

**trace exporter 失败不阻止启动：**

这是可用性取舍。可观测性依赖挂了不应该导致业务服务不可用。但要通过 `tracing_enabled=0` 指标和日志暴露，让运维知道 trace 没生效。

**AlwaysSample 的问题：**

开发和压测中 AlwaysSample 方便观察；生产高 QPS 下会带来存储、网络和性能成本，应改为概率采样、tail sampling 或按错误/慢请求采样。

**排障顺序示例：**

用户反馈接口偶尔 504：先查 gateway HTTP metrics，确认哪些 path 504、P95/P99 是否升高；再查 gRPC client metrics，看是 user-service 还是 task-service deadline；然后用 request_id/trace_id 查单次链路；继续看下游 gRPC server、DB、Redis 延迟；最后查日志中的 deadline exceeded、connection refused、pool exhausted。如果只有写接口慢，再看 log writer、DB lock、乐观锁冲突。

## 批次四：测试、前端、部署、性能、契约、风险与扩展

### 14. 测试体系与质量保障

**测试分层：**

- 单元测试：biz 权限、状态机、repository 行为、middleware、工具包。
- 集成测试：testcontainers 启动真实 PostgreSQL 和 Redis，验证端到端链路。
- 前端单元测试：组件、store、hooks、React Query 缓存更新。
- Playwright E2E：在 MSW mock 模式下验证用户流程。
- 压测：k6/vegeta 验证吞吐、延迟和瓶颈。

**为什么用 testcontainers：**

很多问题 mock 测不出来，例如 GORM 软删除行为、部分唯一索引、JSONB、事务回滚、Redis Lua、SETNX、真实连接池行为。testcontainers 能让集成测试更接近生产依赖，同时保持本地和 CI 可重复。

**必须真实数据库测试的场景：**

owner 唯一部分索引、`(project_id,user_id)` 并发唯一约束、乐观锁 RowsAffected、事务回滚、软删除和部分唯一索引、cursor 分页排序、GORM `Updates(map)` 零值行为。

**关键测试用例：**

非成员访问返回 404；归档项目所有写接口失败；member 不能删除别人的任务；非法状态流转失败；两个 version 相同的更新只有一个成功；同 Idempotency-Key 重复请求返回缓存；Redis 不可用时限流 fail-open、幂等降级、黑名单 fail-close；log writer channel full 同步降级；filter_hash 不匹配拒绝翻页；owner 转让事务保持 owner_id 和 role 一致。

**覆盖率怎么看：**

80% 覆盖率是门槛，不是质量证明。关键是覆盖风险高的业务规则和边界条件。生成代码、cmd 启动代码、配置装配不必追求同等覆盖率；权限、状态机、幂等、事务必须重点测。

**MSW E2E 的优缺点：**

优点是快、稳定、不依赖后端，适合前端流程测试。缺点是测不到真实 gateway、gRPC、数据库、鉴权、CORS 和前后端契约问题。因此还需要后端集成测试或契约测试补充。

### 15. 前端架构与用户体验

**技术选型：**

React + TypeScript + Vite 提供现代前端开发体验；Ant Design 适合后台协作系统；React Query 负责服务端状态、缓存和 mutation；Zustand 负责认证这种轻量客户端状态。

**状态职责：**

Zustand auth store 管 token、当前用户、认证状态。React Query 管项目、任务、成员、评论、日志等远端数据。组件本地 state 管弹窗、表单、筛选条件、临时 UI。

**Axios 客户端：**

request interceptor 注入 Bearer token、`X-Request-Id`，写请求携带 `Idempotency-Key`。response interceptor 解析统一 envelope，`code != OK` 抛 `AppError`，401 时清 token 并派发 `auth:expired`，网络超时转成 `DEADLINE_EXCEEDED`。

**React Query 乐观更新：**

任务更新、指派、状态变化会先 cancel 当前 tasks queries，snapshot detail 和所有 task list query，写入乐观 task 并 version +1；如果失败则恢复 snapshot；如果成功则用服务端返回 task 覆盖；settle 后 invalidate detail 和 list，确保最终一致。

多个列表缓存要同时更新，因为同一个任务可能出现在不同筛选条件的列表中。`taskMatchesFilters` 用来判断变更后的任务是否仍应留在当前列表。

**前端潜在问题：**

- 前端 keyword 匹配 title/content，后端只查 title，契约不一致。
- 每次 mutation 新 idempotency key，不能天然防用户双击。
- 状态机前后端各维护一份，可能漂移。
- token 放 localStorage 有 XSS 风险。
- MSW mock 可能和真实 API 行为不一致。
- antd bundle 需要通过 stats 分析和按需优化。

**权限 UI：**

前端权限 hook 只负责体验，比如隐藏按钮、禁用操作、显示只读状态。安全必须由后端 biz 层保证。

### 16. 配置、部署与运维

**多环境配置：**

local/dev/docker 三套配置用于本地开发、CI/staging 和容器环境。YAML 管默认配置，环境变量覆盖敏感或部署差异配置，例如 `JWT_SECRET`、`INTERNAL_TOKEN`、`POSTGRES_DSN`、`REDIS_ADDR`。

敏感信息不能进仓库，生产应使用 Secret Manager、Kubernetes Secret、Vault 或云厂商密钥服务。

**Docker Compose：**

Compose 编排 PostgreSQL、Redis、Prometheus、Jaeger、Grafana。Go 服务本地运行，便于调试。PostgreSQL 映射 5433、Redis 映射 6380 是为了避免和本机默认端口冲突。

**迁移策略：**

user_svc 和 task_svc 独立迁移目录、独立 migrations table，保证 schema 级隔离。新增迁移必须有 up/down。生产中 down 不一定真的执行，但它能帮助审查变更是否可逆。

**ready/health：**

`/healthz` 表示进程活着，`/readyz` 应表示依赖可用、服务可接流量。当前 ready 如果只是进程状态，不足以证明 gRPC 后端、DB、Redis 可用。生产应在 ready 检查中主动探测关键依赖，或区分强依赖和弱依赖。

**Kubernetes 改造：**

需要 Deployment/Service/Ingress、ConfigMap/Secret、readiness/liveness probes、resource requests/limits、HPA、PodDisruptionBudget、migration Job、Prometheus scrape annotations、日志采集和 trace exporter 配置、网络策略和 mTLS/Service Mesh。

多副本下 Redis 限流和幂等仍全局有效，因为共享 Redis；LogWriter 是实例内存队列，实例崩溃会丢未 flush 日志。

### 17. 性能、容量与压测

**1k QPS 怎么理解：**

要说明测试场景、机器配置、读写比例、数据规模、限流配置、连接池配置和指标结果。1k QPS 如果主要是缓存命中的读接口，不代表登录或写接口也能 1k QPS。

登录接口受 bcrypt cost 影响明显，CPU-bound，P99 容易高。读接口如 `/users/me`、`/projects` 在缓存和连接池合理时延迟较低。写接口受 DB、事务、操作日志和唯一约束影响。

**常见瓶颈：**

bcrypt CPU、DB 连接池过小或慢查询、Redis 连接池或网络延迟、gRPC deadline/连接复用、keyword `ILIKE` 大范围扫描、Prometheus label 高基数、AlwaysSample trace、前端 bundle 过大。

**调优方向：**

登录限流和 bcrypt cost 合理配置；DB max open/max idle 按压测调优；热点读用 cache-aside；列表使用 cursor 和合适索引；批量用户查询避免 N+1；操作日志异步 batch；trace 采样；前端拆包和按需加载。

**扩到更大规模：**

如果支持千万任务，需要按 project_id + created_at 优化索引，归档历史任务冷热分离，大项目任务分区，搜索能力独立，读写分离时处理 read-after-write，缓存项目权限或成员关系，并避免全局 offset。

### 18. 前后端契约与 API 演进

**当前契约来源：**

内部 RPC 契约是 proto；HTTP API 契约主要靠 docs/api-reference、handler 实现和前端手写 types。这个方式灵活但有漂移风险。

**改进：**

可以从 OpenAPI 生成前端 types 和 client，或由 proto + grpc-gateway 生成 HTTP 契约；增加契约测试，校验 envelope、字段名、枚举值；使用 buf breaking 检查 proto 兼容性。

**proto3 默认值问题：**

默认值会让“未传”和“传了 0”混淆。项目对 `ListTasks.status` 使用 `-1` 作为哨兵解决，但这不是强类型表达。更推荐使用 `optional int32 status` 或 wrapper type。

类似问题还有：`assignee_id=""` 无法表达“未指派筛选”；`title=""` 在更新里表示不更新，不能表达清空；`due_time=""` 表示不更新，不能表达清空。这些都可以通过 field mask、optional、clear flag 或 PATCH 语义解决。

### 19. 代码细节与潜在风险

**可以主动指出的风险点：**

1. 幂等 pending 直接返回 OK 不严格，应该等待结果或返回处理中。
2. 幂等缓存没有 status/header，重放语义不完整。
3. BatchGetUsers 返回顺序可能不等于请求顺序。
4. AssignTask 不要求客户端 version，和 ChangeStatus 不一致。
5. status 用 `-1` 哨兵是工程约定，optional 更好。
6. keyword 前后端搜索范围不一致。
7. due_time/title 等字段缺少清空语义。
8. 数据库 duplicate key 用字符串判断，不如 SQLSTATE。
9. JWT 没有 issuer/audience，生产可补。
10. gRPC insecure credentials 只适合本地。
11. AlwaysSample trace 不适合高 QPS 生产。
12. 操作日志内存 channel 在进程崩溃时可能丢。
13. MSW E2E 不覆盖真实后端契约。
14. CORS 和 ClientIP 生产需要严格配置。

面试时主动说出这些点是加分项。它表明你不是只会包装亮点，也知道项目边界。

### 20. 扩展设计与系统演进

**通知服务：**

任务状态变更、评论、@ 提醒可以发出领域事件。早期可在 task-service 内部异步处理；规模扩大后拆 notification-service，用 outbox 保证事件可靠。

**附件上传：**

文件不应走 1MB JSON API。可以使用对象存储，后端生成预签名 URL，上传后保存 metadata，下载时校验项目成员权限。还要考虑文件大小、类型、病毒扫描、过期清理和访问控制。

**全文搜索：**

中小规模先用 PostgreSQL trigram 或 full-text search。大规模、多字段、高亮和复杂相关性排序时再引入 Elasticsearch/OpenSearch。

**多租户：**

引入 `tenant_id` 到 users/projects/tasks/members/logs，并在所有唯一索引和查询条件中包含 tenant_id。需要防止任何查询漏 tenant 条件，最好在 repository 层或数据库 RLS 做保护。

**OAuth2/SSO：**

user-service 增加 identity provider、external subject、绑定本地用户、登录回调、state 校验、token exchange。JWT 签发仍由本系统统一完成。

**拆独立数据库：**

user-service 和 task-service 真正拆库后，不能直接 join 用户信息。task-service 只保存 user_id，展示信息通过 user-service 查询、缓存或事件同步用户快照。跨服务一致性要接受最终一致或用补偿机制。

**灰度发布：**

要求数据库迁移向前兼容：先加字段，双写/兼容读，再切流，最后删除旧字段。proto 和 HTTP API 只能新增兼容字段，不破坏旧客户端。

### 21. 项目复盘与个人贡献

**推荐回答结构：**

项目按阶段演进：Phase 0 服务骨架、Docker Compose、基础配置；Phase 1 用户注册登录、JWT、Redis 黑名单；Phase 2 项目和成员、权限矩阵、乐观锁；Phase 3 任务 CRUD、状态机、游标分页；Phase 4 评论和异步操作日志；Phase 5 限流、幂等、缓存；Phase 6 metrics、trace、Grafana、压测和前端完善。

最大风险是权限和并发，因为 bug 不一定马上报错，但会造成越权或数据覆盖。处理方式是把规则集中在 biz 层，并用单元测试和集成测试覆盖边界。

**如果被质疑过度设计：**

回答重点是“克制”。项目没有拆很多服务，也没有引入 Kafka、Elasticsearch、Kubernetes 作为必要依赖。只保留了能展示真实后端工程问题的组件：gateway、gRPC、PostgreSQL、Redis、可观测性和测试。

**如果被质疑不够生产级：**

可以承认：内部认证还只是静态 token，gRPC 未启用 TLS，幂等 pending 语义不严格，操作日志不是强可靠，前后端契约还未自动生成，ready check 还可加强。这些不是硬伤，而是项目阶段和目标下的取舍。

### 22. 高频连环追问回答模板

**幂等模板：**

写接口幂等是为了防止网络重试和重复点击造成重复创建。项目用 Redis `SETNX` 保证同一个用户的同一个 key 只有一个请求进入业务逻辑，成功后缓存响应，失败后删除 key 允许重试。它不能替代数据库唯一约束和乐观锁。当前实现的不足是 pending 直接返回 OK、没有缓存 status/header、没有绑定 body hash。生产上我会把 value 设计成状态机，pending 时等待或返回 202，并绑定 method/path/body hash。

**权限模板：**

权限放在 task-service 的 biz 层，因为权限依赖项目状态、成员角色、任务 creator、任务状态等业务数据。owner/admin/member 是协作系统的核心规则。非成员返回 404 是为了隐藏资源存在性。owner 转让通过事务同时更新 `projects.owner_id` 和成员角色，并用部分唯一索引保证最多一个 owner。前端权限只改善体验，不能替代后端校验。

**缓存模板：**

缓存采用 cache-aside。读先查 Redis，miss 查 DB 并回填；写成功后删除缓存。项目缓存必须先做成员校验再读缓存，避免越权。Redis 出错时缓存降级查 DB，限流 fail-open，黑名单 fail-close。BatchGetUsers 能减少 N+1，但当前返回顺序需要注意，生产中应按请求 ID 顺序重组。

**乐观锁模板：**

乐观锁通过 `WHERE id=? AND version=?` 防止旧版本覆盖新版本。冲突返回 `ABORTED/409`，客户端重新拉取最新数据。它适合低冲突的 HTTP 协作场景，比悲观锁更轻。前端乐观更新只是 UI 优化，后端 version 才是最终并发控制。

**可观测性模板：**

我会用 request_id 串日志，用 trace_id 看链路，用 Prometheus 看 RED 指标和 DB/Redis 指标。慢请求先看 HTTP path 的 P95/P99，再看 gRPC client/server，再看 DB/Redis latency，最后用 trace 定位具体 span。指标 label 必须低基数，不能用原始 URL 或 user_id。

**前端模板：**

React Query 管远端状态，Zustand 管认证这种客户端状态。任务 mutation 使用乐观更新：先 snapshot，再写 detail/list cache，失败回滚，成功用服务端结果覆盖，最后 invalidate 保证一致。权限和状态机在前端只是交互约束，后端仍是最终规则。
