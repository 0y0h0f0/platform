# Task Platform 面试官深挖问题清单

> 视角：如果我是面试官，我会围绕“你是否真的理解这个项目、是否能解释工程取舍、是否能定位风险并提出改进”来追问。以下问题尽量结合本项目的实际实现：Gin API Gateway、gRPC 微服务、PostgreSQL 多 schema、Redis、JWT、幂等、限流、缓存、乐观锁、异步操作日志、Prometheus/Grafana/Jaeger、React 前端与测试体系。

## 1. 项目整体与业务定位

1. 你用 1 分钟介绍这个项目时，会突出哪些业务能力和工程能力？
2. 这个项目为什么定位成“团队任务协作平台”，而不是更简单的 TODO List 或更复杂的 Jira？
3. 该项目最能体现你后端能力的 3 个点是什么？分别解决了什么实际问题？
4. 如果面试官只看 README，你希望他记住这个项目的哪几个关键词？
5. 你如何证明这个项目不是 CRUD 堆接口，而是有真实工程复杂度？
6. 项目中哪些功能是业务功能，哪些是工程基础设施？二者的边界怎么划分？
7. 如果要把这个项目上线给真实团队使用，当前最缺的 5 个能力是什么？
8. 你在项目设计中最满意的一个取舍是什么？为什么？
9. 你在项目中最不满意或最想重构的一部分是什么？为什么当时没有做？
10. 该项目的核心数据对象有哪些？用户、项目、成员、任务、评论、操作日志之间是什么关系？
11. 项目中哪些地方体现了“面向校招面试”的克制？哪些地方又超过了普通校招项目？
12. 如果让我从代码里验证你说的亮点，我应该看哪些目录或文件？
13. 这个系统的典型写请求链路是什么？从 HTTP 请求到数据库写入要经过哪些层？
14. 这个系统的典型读请求链路是什么？什么时候需要跨服务聚合用户信息？
15. 项目中哪些能力是“正确性优先”，哪些能力是“性能优先”，哪些能力是“可用性优先”？
16. 如果你要给这个项目画系统架构图，哪些组件必须画出来？哪些组件可以省略？
17. 这个项目有没有领域模型？还是只有数据模型？你如何区分？
18. 项目中的权限、状态机、幂等、限流、缓存分别属于哪一层的责任？
19. 你如何向非技术面试官解释“归档项目只读”“非成员返回 404”“乐观锁”这些设计？
20. 如果用户反馈“偶尔重复点击按钮导致创建了重复任务”，你会从哪些模块排查？

## 2. 架构拆分与微服务边界

1. 为什么采用 `api-gateway + user-service + task-service`，而不是单体应用？
2. 为什么只拆成 2 个核心业务服务，而不是继续拆出 comment-service、log-service、member-service？
3. 这个项目里的微服务拆分依据是业务域、数据归属、团队协作，还是技术栈？
4. `api-gateway` 的职责是什么？哪些逻辑明确不应该放在 gateway？
5. `user-service` 为什么负责 JWT 签发，但 JWT 验签又放在 gateway？
6. `task-service` 为什么不直接解析 JWT？这样有什么好处和风险？
7. 内部 gRPC metadata 中的 `x-user-id`、`x-username`、`x-request-id`、`x-internal-token` 分别解决什么问题？
8. 如果某个内部服务绕过 gateway 伪造 `x-user-id`，系统如何防护？
9. 当前内部 RPC 使用静态 `x-internal-token`，相比 mTLS 有哪些不足？
10. 如果项目部署到 Kubernetes，你会如何改造内部服务认证？
11. 为什么内部通信选择 gRPC unary，而不是 REST、GraphQL 或消息队列？
12. gRPC proto 契约升级时如何保证兼容性？哪些字段不能随便改？
13. `task.proto` 中为什么把项目、任务、评论、操作日志都放在一个 `TaskService`？
14. 如果未来引入通知系统，应该作为 task-service 内部模块还是独立服务？判断标准是什么？
15. 当前 `user_svc` 和 `task_svc` 在同一个 PostgreSQL 实例中，算不算真正的微服务？
16. 单数据库多 schema 的优点是什么？它牺牲了哪些微服务独立性？
17. 如果未来拆成两个独立数据库，哪些接口和事务会受影响？
18. 服务边界和数据库 schema 边界是否一致？有没有跨边界访问数据？
19. 添加成员、指派任务时，task-service 调 user-service 校验用户存在，这里有什么一致性问题？
20. 如果 user-service 临时不可用，task-service 的哪些功能会失败？哪些功能仍应可用？
21. 你如何给 gRPC 调用设置超时、重试和熔断？当前实现还有哪些缺口？
22. `grpc.NewClient` 创建连接时的行为是什么？它是否立即建立连接？这对启动探活有什么影响？
23. gateway 启动时如何确认后端服务真的可用？当前 ready 状态是否足够？
24. 如果 task-service 返回 `UNAVAILABLE`，gateway 如何映射 HTTP 状态码？
25. 你如何避免 gateway 聚合多个服务时产生“上游慢导致整体慢”的问题？
26. 如果一个 HTTP 请求需要同时查用户、项目、任务，你会串行还是并行调用？如何控制超时预算？
27. 服务之间是否需要传递 trace context？项目里是怎么做的？
28. 你如何设计内部服务的错误码，使 gateway 不依赖字符串判断？
29. 如果某个 gRPC 方法未来要变成流式接口，哪些层需要调整？
30. 你怎么判断这个项目的微服务拆分“刚好”，而不是过度设计？

## 3. 分层设计与代码组织

1. 项目为什么采用 `handler -> service -> biz -> data` 四层结构？
2. 每一层的输入输出分别是什么？是否允许跨层调用？
3. handler 层做参数绑定和 HTTP/gRPC 转换，为什么不做权限判断？
4. service 层与 biz 层的边界在哪里？哪些校验应该在 service，哪些应该在 biz？
5. data 层为什么不应该包含业务判断？如果 repository 里判断权限会带来什么问题？
6. `pkg/x*` 下的通用库有哪些？哪些是真正通用，哪些可能过早抽象？
7. 你如何避免 `biz` 层被 gRPC、Gin、GORM 等框架污染？
8. 当前 `TaskBiz` 和 `ProjectBiz` 都有 `getProjectAndMember`，这是否是重复？该不该抽象？
9. `ProjectBiz` 中既有缓存逻辑又有业务逻辑，这是否违反单一职责？
10. 如果要把缓存从业务层挪走，你会放在哪里？repository decorator 是否合适？
11. 目前很多错误消息在 biz/data 中直接写字符串，后续多语言或前端展示会有什么问题？
12. 领域常量如角色、状态、优先级放在 data 包中是否合理？是否应该放到 domain 包？
13. GORM model 和业务实体是否应该分离？当前合在一起有什么利弊？
14. proto message、HTTP DTO、GORM model 三套结构如何保持一致？
15. 你如何防止前端枚举、后端枚举、数据库枚举漂移？
16. 代码中哪些地方通过测试锁定了层间契约？
17. 如果 handler 测试需要 mock gRPC client，你如何设计接口？
18. 如果 biz 测试需要 mock repository，你如何避免 mock 太脆？
19. 哪些测试应该使用真实数据库，而不是 mock？
20. 当前目录结构是否支持多人协作？如果团队扩展到 5 人，代码归属如何划分？

## 4. API Gateway 与 HTTP 设计

1. Gateway 中间件顺序为什么是 `MaxBodySize -> SecurityHeaders -> RequestID -> Trace -> Metrics -> AccessLog -> CORS -> RateLimit(IP) -> Auth -> RateLimit(user)`？
2. 如果把 Auth 放在 RateLimit(IP) 前面，会有什么影响？
3. 如果把 RequestID 放在 AccessLog 后面，会出现什么问题？
4. 写请求的幂等检查为什么放在 handler 中，而不是统一中间件？
5. 幂等检查为什么要在参数绑定和校验之后执行？
6. `MaxBodySize(1MB)` 对哪些接口有意义？未来支持附件上传时如何处理？
7. CORS 配置应该如何限制？开发环境和生产环境有哪些差异？
8. Security Headers 主要防御什么风险？对 API 服务是否必要？
9. 请求响应统一 envelope `{ code, message, request_id, data }` 的优缺点是什么？
10. 为什么业务错误不直接返回 gRPC 原始 status，而要转换成 HTTP envelope？
11. 对于 REST API，项目中 `POST /tasks/:id/status` 和 `POST /tasks/:id/assign` 为什么比 `PUT /tasks/:id` 更合适？
12. `UpdateTask` 不允许修改 status/assignee，这个约束应在前端、gateway、service、biz 哪几层体现？
13. 当前项目写接口全部支持 `Idempotency-Key`，是否所有写接口都真的需要？
14. 如果客户端不传 `Idempotency-Key`，系统如何表现？是否应该强制要求？
15. gateway 中如何处理 HTTP 401、403、404、409、429、500 的语义？
16. 非成员访问项目或任务时返回 404，这和统一错误响应如何结合？
17. 如果 handler 已经向客户端写了一部分响应，再发生错误，幂等缓存会如何处理？
18. `bodyCaptureWriter` 捕获响应体用于幂等缓存，有哪些内存和边界问题？
19. 如果响应是流式或大文件，当前幂等实现是否适用？
20. gateway 如何处理客户端断开连接？context cancellation 是否会传播到 gRPC 和 DB？
21. HTTP timeout 是在哪里配置的？如果没有配置可能有什么风险？
22. gateway 是否应该做请求体 JSON schema 校验？当前 Gin binding 能覆盖哪些问题？
23. `X-Request-Id` 是客户端生成还是服务端生成？如果客户端伪造重复 request_id 怎么办？
24. 响应中暴露 `request_id` 对排障有什么帮助？
25. API 路径中为什么使用 `/api/v1`？后续版本升级如何做？
26. REST 路径里 `project_id` 有时是 path，有时是 query/body，这样设计的依据是什么？
27. `GET /tasks` 使用 query filter，如何避免筛选条件过多导致接口变复杂？
28. 如果要支持排序字段和多条件搜索，gateway 参数层如何扩展？
29. gateway 是否应该做用户信息 enrichment？为什么不让 task-service 直接返回昵称头像？
30. 批量用户查询 enrichment 如何避免 N+1 调用？

## 5. 认证、JWT 与安全

1. 登录和注册为什么直接返回 JWT？注册后是否应该自动登录？
2. JWT 使用 HS256，有什么优缺点？如果改成 RS256 会改变哪些模块？
3. `JWT_SECRET` 为什么要求至少 32 字符？默认占位符为什么要拒绝启动？
4. JWT claims 中放了 `sub`、`jti`、`iat`、`exp`、`username`，分别有什么作用？
5. 为什么 logout 需要 Redis 黑名单？JWT 不是无状态的吗？
6. 黑名单 key 使用 `blacklist:<jti>`，TTL 应该设置多久？如何计算剩余有效期？
7. 如果 Redis 黑名单查询失败，当前 auth 中间件返回 500。这个设计相比 fail-open/fail-close 的取舍是什么？
8. 认证接口如何防暴力破解？IP 级 auth 限流是否足够？
9. 登录失败时为什么返回统一的“invalid account or password”，而不区分用户不存在和密码错误？
10. 用户被禁用后，已经签发的 token 是否还能继续使用？当前系统如何处理？
11. 如果用户修改密码，旧 token 是否应该失效？怎么实现？
12. 当前 token TTL 是 2 小时，为什么不是 15 分钟或 7 天？
13. 是否需要 refresh token？引入后数据模型和安全策略如何变化？
14. JWT secret 在 gateway 和 user-service 都需要配置，如何避免两个服务配置不一致？
15. `x-user-id` 来自 gateway 解析的 JWT，内部服务是否完全信任它？为什么？
16. 内部服务如何校验 `x-internal-token`？如果泄露了怎么办？
17. 当前 gRPC 使用 insecure transport credentials，在内网是否可接受？生产环境如何改？
18. 密码使用 bcrypt cost=10，如何在安全和性能之间取舍？
19. 项目里 `BCRYPT_COST` 可配置，为什么限制在 4 到 14？
20. 登录时发现旧 hash cost 低会 rehash，这个过程为什么不阻塞登录失败？
21. 弱密码黑名单是如何加载的？是否区分大小写？是否需要包含常见变体？
22. 用户名、邮箱、密码的校验规则分别是什么？有没有国际化问题？
23. 邮箱使用 `net/mail.ParseAddress` 是否足够？它可能接受哪些非预期格式？
24. 邮箱存储前转小写，这对国际化邮箱有什么影响？
25. token 存在前端哪里？localStorage 的 XSS 风险如何处理？
26. 前端 401 后清理 token 并派发 `auth:expired`，如何避免多个请求同时 401 导致重复跳转？
27. 如果浏览器关闭后 token 仍在，是否符合业务预期？
28. API 是否需要 CSRF 防护？使用 Bearer token 时风险模型是什么？
29. 如果引入 cookie 认证，CORS、CSRF、SameSite 配置会怎么变？
30. 你如何设计审计日志来记录登录失败、权限拒绝、token 撤销等安全事件？

## 6. 权限模型与业务规则

1. 项目中 owner/admin/member 三个角色的权限边界是什么？
2. 为什么 owner 不能直接退出项目，必须先转让所有权？
3. 为什么 admin 只能邀请 member，不能邀请 admin？
4. 为什么 admin 不能移除 owner 或其他 admin？
5. 成员访问不存在资源和无权限资源都返回 404，会不会影响前端用户体验？
6. 非成员返回 404 的安全收益是什么？有没有排障成本？
7. 项目归档后“全员只读”，哪些接口应该禁止？哪些接口仍允许？
8. 当前 `ListProjectMembers` 在归档项目下是否应该允许？为什么？
9. 任务创建时为什么只检查项目成员身份，不区分 owner/admin/member？
10. member 只能编辑自己的任务，这里的“自己的”指 creator 还是 assignee？为什么？
11. member 删除任务时还要求 todo 状态，这个业务规则的目的是什么？
12. member 是否应该能变更指派给自己的任务状态？当前逻辑按 creator 判断是否合理？
13. 如果 creator 离开项目，他创建的任务权限应该如何处理？
14. 如果 assignee 被移出项目，任务的 assignee_id 应该清空吗？
15. 添加成员时 task-service 通过 user-service 校验用户存在且 active，这个校验是否有竞态？
16. 转让 owner 时为什么要求目标用户已经是项目成员？
17. 转让 owner 的事务中，旧 owner 被改成 admin 是否一定合理？能不能改成 member？
18. 数据库部分唯一索引保证一个项目最多一个 owner，但业务上如何保证至少一个 owner？
19. `projects.owner_id` 和 `project_members.role=owner` 两处数据如何避免不一致？
20. 如果转让过程中更新 project 成功，但更新 member role 失败，系统如何回滚？
21. 如果两个 owner 转让请求并发执行，乐观锁和唯一索引分别发挥什么作用？
22. 修改成员角色时为什么不能直接把别人改成 owner，而必须走 transfer ownership？
23. 删除评论的权限规则是什么？owner/admin/member 分别能删哪些评论？
24. 评论属于 task，task 属于 project，删除评论时如何确认调用者是项目成员？
25. 操作日志查询是否也需要权限校验？非成员能否看到日志？
26. 操作日志中是否会记录敏感信息？`detail_json` 的边界是什么？
27. 权限判断如果散落在多个 biz 方法中，如何保证新增接口不会漏校验？
28. 是否可以把权限矩阵抽象成策略表或 policy engine？现在为什么没有这么做？
29. 前端权限控制和后端权限控制如何配合？前端隐藏按钮是否能替代后端校验？
30. 如果产品要求“项目访客只读角色”，需要改哪些表、枚举、接口和前端逻辑？

## 7. 任务状态机与领域约束

1. 当前任务状态机支持哪些状态流转？
2. 为什么 `done -> todo` 不允许，但 `done -> doing` 允许？
3. 为什么 `cancelled -> todo` 允许？是否应该允许 `cancelled -> doing`？
4. `ChangeTaskStatus` 为什么要求传 version，而 `AssignTask` 使用当前 task version？
5. 如果两个用户同时拖拽同一个任务到不同状态，后端如何处理？
6. 状态机校验应该放在前端还是后端？前端 hook `useStatusTransitions` 的作用是什么？
7. 前后端都维护状态机，如何防止规则不一致？
8. `UpdateTask` 不接受 status 和 assignee，如何防止绕过状态机？
9. 任务 priority 的合法值在哪里校验？如果前端传 999 会怎样？
10. due_time 用 string 从 proto 传输，为什么不用 `google.protobuf.Timestamp`？
11. due_time 字符串由哪个层解析？时区如何处理？
12. 任务 `extra` 使用 JSONB，为什么当前 proto 没有暴露？后续如何扩展？
13. `extra` 约定只放 labels/checklist/attachments，数据库是否强约束？业务层是否强约束？
14. 如果任务标题为空，在哪一层会被拦截？
15. `UpdateTask` 中 title 为空表示“不更新标题”，那如何把标题更新为空？业务是否允许？
16. `content` 每次 update 都写入 updates，即使为空。这里和 title 的语义为什么不同？
17. 创建任务默认 priority 是 normal，status 是 todo，这些默认值放在代码还是数据库更好？
18. 删除任务使用软删除，对评论和操作日志有什么影响？
19. 软删除任务后，历史操作日志是否还能查到？是否应该保留？
20. 任务列表的 keyword 只查 title，前端匹配却同时查 title/content，这是否一致？

## 8. 数据库设计与事务一致性

1. 为什么使用 PostgreSQL 16？项目中用到了哪些 PG 特性？
2. 为什么一个数据库实例下拆 `user_svc` 和 `task_svc` 两个 schema？
3. 为什么没有声明跨 schema 外键？优缺点是什么？
4. 没有数据库外键时，如何保证 `owner_id`、`creator_id`、`assignee_id` 引用有效用户？
5. `users`、`projects`、`tasks` 使用软删除，`comments`、`members` 物理删除，依据是什么？
6. 软删除配合唯一索引为什么要使用 `WHERE deleted_at IS NULL` 的部分唯一索引？
7. 如果软删除用户后又注册同名用户名，会带来什么历史数据问题？
8. `projects(owner_id, name)` 唯一索引解决什么问题？是否应该项目成员范围内唯一？
9. `project_members(project_id, user_id)` 唯一索引在并发添加成员时如何工作？
10. `project_members(project_id) WHERE role=0` 的部分唯一索引如何保证 owner 唯一？
11. owner 唯一索引为什么不能保证 owner 一定存在？
12. `tasks` 表为什么按 `(project_id, created_at DESC, id DESC)` 建索引？
13. `status`、`assignee_id` 的组合查询如何利用索引？
14. `keyword` 使用 `ILIKE '%xxx%'`，在数据量大时会有什么性能问题？
15. 如果要优化 keyword 搜索，你会用 trigram index、全文索引，还是引入 Elasticsearch？
16. `operation_logs.detail` 用 JSONB，有什么查询和索引上的考虑？
17. JSONB 适合存哪些扩展数据？哪些数据不该放 JSONB？
18. 乐观锁字段 `version` 为什么用 bigint？
19. GORM `Updates(map)` 对零值字段如何处理？当前代码是否依赖这个行为？
20. repository 捕获 duplicate key 时通过字符串包含 `"duplicate key"` 判断，有什么问题？
21. 数据库错误是否应该根据 SQLSTATE 判断？Go/GORM 中如何实现？
22. `TransferOwnership` 在一个事务里更新 project 和 member，隔离级别是否足够？
23. 并发转让 owner 时，除了乐观锁，还会不会触发唯一索引冲突？
24. 事务中调用外部 user-service 是否合适？当前转让是在事务前校验用户，为什么？
25. 如果用户校验后、事务提交前，目标用户被禁用，会产生什么一致性窗口？
26. 如何设计补偿机制或最终一致性来处理跨服务校验？
27. `FindByID` 使用 GORM 默认软删除过滤，这个隐式行为有什么风险？
28. 成员表物理删除后，历史“谁曾参与项目”如何追溯？
29. 评论物理删除后，操作日志是否应该记录删除前内容？
30. 新增迁移为什么必须同时提供 up/down？down 迁移在生产中是否一定可执行？

## 9. 并发控制、幂等与重试

1. 项目里哪些操作需要乐观锁？为什么 project/task 需要，comment/member 不需要？
2. 乐观锁更新 SQL 的核心条件是什么？
3. 更新影响行数为 0 时为什么返回 `ABORTED` 而不是 `NOT_FOUND`？
4. 客户端收到 `ABORTED` 应该如何恢复？
5. 服务端是否应该自动重试乐观锁冲突？什么时候不应该重试？
6. 前端乐观更新把 version +1，如果后端返回冲突，会如何回滚？
7. 幂等和乐观锁分别解决什么问题？二者是否可以互相替代？
8. `Idempotency-Key` 为什么要按用户 ID 做 scope？
9. 如果两个不同用户使用同一个 idempotency key，会互相影响吗？
10. 当前幂等实现先 `SETNX key pending`，成功后缓存响应体。并发重复请求看到 pending 时直接返回 OK，这是否符合严格幂等语义？
11. 如果第一个请求还在执行，第二个请求收到“request already processed”，但实际第一个请求最后失败，会发生什么？
12. 是否应该让重复请求等待第一个请求完成？如何用 Redis 实现？
13. 幂等缓存的是响应体，不缓存 HTTP status 和 headers，会有什么边界问题？
14. 如果接口返回 201/204，当前幂等重放是否保持语义？
15. 幂等 key 在错误时删除，为什么？哪些错误应该允许重试，哪些不应该？
16. 如果业务成功但缓存响应失败，客户端重试会不会重复执行？
17. 如果 Redis 在 `SETNX` 后宕机或网络失败，幂等保证如何退化？
18. 24 小时 TTL 的依据是什么？太短或太长分别有什么问题？
19. 幂等 key 是否需要绑定 method/path/body hash？当前只绑定用户和 key 有什么风险？
20. 如果客户端复用同一个 key 请求不同接口，会得到什么结果？
21. 前端每次 mutation 用 randomUUID 生成 idempotency key，这能防重复提交吗？和“用户双击”有什么关系？
22. 如果 mutation 因网络超时重试，前端是否能复用同一个 idempotency key？
23. React Query mutation 默认重试策略是否会影响幂等？
24. 数据库唯一约束是否也是一种幂等？例如添加成员重复请求会返回什么？
25. 创建任务没有天然唯一键，为什么更依赖 Idempotency-Key？
26. 分布式系统中“exactly once”是否存在？本项目能提供到什么语义？
27. 幂等实现放 gateway，如果未来绕过 gateway 直接调 gRPC，会发生什么？
28. 是否应该在 service/biz 层也实现幂等？成本是什么？
29. 高并发下 `bodyCaptureWriter` 捕获大响应是否可能造成内存压力？
30. 你会如何给幂等模块补充更严格的测试场景？

## 10. Redis：缓存、限流、黑名单与降级

1. Redis 在项目里承担了哪些职责？
2. 把缓存、限流、幂等、JWT 黑名单都放在同一个 Redis 实例，有什么风险？
3. 如果 Redis 不可用，各模块应该 fail-open 还是 fail-close？
4. 当前限流 Redis 出错时 fail-open，为什么？
5. 当前 auth 黑名单 Redis 出错时返回 500，为什么不是 fail-open？
6. 当前幂等 Redis 出错时继续执行但失去保证，是否合理？
7. 用户缓存和项目缓存都使用 cache-aside，读写流程是什么？
8. 缓存 TTL 5 分钟的依据是什么？
9. 写操作后如何失效项目缓存？哪些操作必须失效？
10. 添加/移除成员是否需要失效项目缓存？为什么项目缓存只缓存 Project 本身可能没问题？
11. 用户信息更新后如何失效用户缓存？当前是否有用户更新接口？
12. `BatchGetUsers` 从缓存命中部分用户、DB 查 missed，返回顺序是否与请求顺序一致？这对前端有影响吗？
13. 如果 BatchGetUsers 传入重复 ID，当前行为如何？是否需要去重？
14. 缓存穿透、击穿、雪崩分别是什么？本项目是否有防护？
15. 项目详情缓存是否可能导致非成员读到数据？权限校验和缓存读取顺序为什么重要？
16. token bucket 使用 Redis Lua 脚本有什么好处？
17. 令牌桶中的 `tokens` 和 `last_refill` 如何更新？
18. 为什么用 Lua 脚本而不是 GET/SET 多条命令？
19. 令牌桶 key 的过期时间为什么是 `ceil(capacity/rate)+60`？
20. IP 限流、Auth 限流、User 限流的默认速率分别是多少？为什么这样设？
21. `c.ClientIP()` 依赖代理头时有什么安全问题？
22. 如果部署在 Nginx/Ingress 后面，如何正确获取真实 IP？
23. 限流 label 或 key 如果包含原始 URL 会有什么高基数问题？
24. 黑名单 key 是否应该加命名空间区分环境？
25. Redis key 命名如何避免冲突？当前有哪些 prefix？
26. Redis 操作是否设置了 timeout？在哪里设置？
27. Redis 连接池如何配置？默认值是否足够 1k QPS？
28. Redis 持久化 AOF 对 JWT 黑名单和幂等 key 有意义吗？
29. 如果 Redis 主从切换导致短暂丢 key，会对系统造成什么影响？
30. 你会如何给 Redis 相关模块设计监控指标和告警？

## 11. 分页、查询与数据返回

1. 项目列表为什么使用 offset 分页，而任务列表使用 cursor 分页？
2. offset 分页在大 offset 时有什么性能和一致性问题？
3. cursor 分页为什么按 `(created_at DESC, id DESC)` 排序？
4. cursor 中为什么需要同时包含 `created_at` 和 `id`？
5. 如果两个任务 created_at 相同，id 如何保证稳定排序？
6. cursor 为什么使用 base64url 编码 JSON？是否需要加密或签名？
7. 客户端能否篡改 cursor？当前服务端如何处理？
8. cursor 中的 `filter_hash` 解决什么问题？
9. `filter_hash` 用 SHA256 前 16 位，碰撞风险是否可接受？
10. filter_hash 的输入参数如何构造？status 用 rune 转字符串是否直观？
11. 如果新增排序条件，filter_hash 需要包含吗？
12. 如果用户翻页过程中新增任务，会不会看到重复或漏数据？
13. 如果用户翻页过程中删除任务，会发生什么？
14. 操作日志列表也使用 cursor，但没有 filter_hash，为什么？
15. 评论列表使用 `after_id` 正向分页，为什么不复用 cursor？
16. 评论 `after_id` 先查 anchor，再按 `(created_at,id)>anchor` 查询，性能如何？
17. `limit` 最大值为什么 tasks 是 50，comments/logs 是 100？
18. 查询接口默认 limit 20 是否适合前端看板？
19. 任务列表按项目过滤是否必须？如果支持“我的任务”全局列表，索引如何调整？
20. `assignee_id` 为空代表不过滤，那如何查询“未指派任务”？
21. keyword 只查 title 是否符合前端搜索预期？
22. 批量返回任务时是否需要返回 creator/assignee 的用户信息？当前在哪里 enrich？
23. API 返回数据是否包含 created_at/updated_at？proto 中当前是否缺失？
24. 没有返回时间字段会影响前端排序、展示和冲突提示吗？
25. 如果需要支持服务端排序和字段选择，接口如何扩展？

## 12. 操作日志与异步处理

1. 为什么操作日志要异步写入，而不是跟主业务事务一起提交？
2. 操作日志是审计数据还是普通展示数据？这会影响可靠性要求吗？
3. `LogWriter` channel 容量为什么是 1024？
4. batch size 64、flush interval 100ms 是如何取舍的？
5. channel 满时为什么降级同步写，而不是丢弃或扩容 channel？
6. 同步降级写会不会拖慢主请求？这是否违背异步化目的？
7. 如果数据库持续慢，channel 一直满，系统会出现什么现象？
8. `flushWithRetry` 最多重试 3 次并指数退避，为什么没有把失败日志放入死信队列？
9. 操作日志写入失败是否应该影响主业务结果？
10. 如果业务操作成功但日志写入失败，审计完整性如何保证？
11. `Shutdown` 时先 close done，再 drain channel，这个流程是否可能漏写？
12. `run` 中同时监听 `done`、`ch`、`ticker`，关闭时 batch 和 channel 剩余数据如何处理？
13. worker panic 后自动重启，为什么要记录 `worker_panic_total`？
14. `prometheus.MustRegister` 多次注册会 panic，项目如何用 `sync.Once` 避免？
15. operation log detail 目前很多是 `{}`，如果要支持可读审计，应记录哪些字段？
16. 记录变更前后值会不会泄露敏感信息？
17. 操作日志按 project_id 和 task_id 查询，索引如何设计？
18. task 操作日志同时写 project_id 和 task_id，有什么好处？
19. 如果任务被删除，操作日志是否仍可通过 task_id 查询？
20. 如果操作日志量很大，是否需要分区表、归档或冷热存储？
21. 如果需要保证审计日志不可篡改，可以怎么设计？
22. 为什么没有使用 Kafka/RabbitMQ？当前 channel worker 的适用边界是什么？
23. 多实例 task-service 下，每个实例都有自己的 LogWriter，会有什么问题？
24. 如果实例被强杀，内存 channel 中的日志会丢失，是否可接受？
25. 要做到更可靠的异步日志，你会用 outbox pattern 还是消息队列？

## 13. 错误码、异常处理与可观测性

1. 统一错误码为什么选 12 个？是否覆盖了所有业务场景？
2. `CodeInternal` 对外隐藏真实错误消息，为什么？
3. 哪些错误应该返回 400，哪些应该返回 409？
4. 乐观锁冲突为什么映射到 HTTP 409？
5. 归档项目只读为什么返回 `FAILED_PRECONDITION`，而不是 `PERMISSION_DENIED`？
6. 非成员返回 `NOT_FOUND`，但真实资源不存在也返回 `NOT_FOUND`，日志里如何区分？
7. data 层把所有 DB 查询失败包成 `INTERNAL`，是否会丢失排障信息？
8. gRPC status 到 HTTP status 的映射在哪里完成？
9. 如果一个未知错误从 biz 层冒出，最终 HTTP 会是什么响应？
10. panic recovery 如何保证响应中仍有 request_id？
11. AccessLog 记录哪些字段？如何避免记录密码或 token？
12. request_id 和 trace_id 的区别是什么？
13. 如果没有 trace_id，仅靠 request_id 能否串联 gateway 和两个服务？
14. OpenTelemetry exporter 初始化失败时服务继续启动，这是什么取舍？
15. Sampler 使用 AlwaysSample 在生产是否合适？
16. Prometheus 指标里为什么不能把原始 URL 当 label？
17. HTTP/gRPC latency histogram 如何选 bucket？
18. RED 指标是什么？项目里如何体现？
19. DB/Redis 操作指标能帮助定位什么问题？
20. `rate_limit_hits_total` 和 `gateway_rate_limiter_errors_total` 分别代表什么？
21. 操作日志 channel full 指标升高说明什么？
22. Grafana 19 个面板中，你认为最关键的 5 个是什么？
23. 如果用户报告“接口偶尔 504”，你会按什么顺序看 metrics、logs、traces？
24. 如果 P95 延迟升高但错误率不高，你会怎么定位？
25. 如果 DB 指标正常但 gRPC 客户端耗时升高，可能是什么问题？
26. 如果 trace 不出现，应排查哪些配置？
27. `/healthz` 和 `/readyz` 的语义区别是什么？
28. admin HTTP 服务暴露 `/metrics` 是否需要鉴权？生产环境如何保护？
29. 日志中如何关联用户 ID？是否会带来隐私问题？
30. 如果需要支持告警，你会为哪些指标设置阈值？

## 14. 测试体系与质量保障

1. 项目有哪些测试层级？单元测试、集成测试、E2E、压测分别验证什么？
2. 为什么集成测试使用 testcontainers 拉真实 PostgreSQL 和 Redis？
3. 哪些场景必须用真实数据库才能测出来？
4. repository 单元测试和集成测试的边界是什么？
5. biz 层测试最应该覆盖哪些权限和状态机边界？
6. handler 测试如何验证统一错误响应和幂等行为？
7. middleware 测试如何验证 RateLimit、Auth、CORS、SecurityHeaders？
8. 如何测试 Redis 不可用时各模块的降级行为？
9. 如何测试 log writer channel 满时的同步降级？
10. 如何测试 worker panic 后重启？
11. 如何测试乐观锁并发冲突？需要真正并发吗？
12. 如何构造两个请求使用相同 Idempotency-Key 的测试？
13. 如何测试 filter_hash 不匹配时拒绝翻页？
14. 如何测试非成员访问返回 404 而不是 403？
15. 如何测试 owner 转让事务的原子性？
16. 如何测试 DB 唯一索引在并发添加成员时的效果？
17. `go test ./...` 和 `go test -tags=integration` 的区别是什么？
18. 覆盖率 80% 的意义是什么？为什么覆盖率高不代表质量高？
19. 哪些代码可以不追求覆盖率？生成代码、cmd、配置装配是否需要？
20. 如果 CI 中 Docker 不可用，集成测试如何处理？
21. 前端 Vitest 主要测什么？Playwright E2E 主要测什么？
22. MSW mock 模式对前端 E2E 有什么优点？有什么缺点？
23. 使用 mock E2E 会不会掩盖前后端契约不一致？
24. 如何增加契约测试保证 proto/HTTP DTO/前端 types 一致？
25. 压测脚本如何准备 token、项目和用户数据？
26. 登录压测为什么容易被 bcrypt cost 影响？
27. 你如何判断 1k QPS 的测试结果可信？
28. 压测中 P50/P95/P99 分别有什么意义？
29. 如果压测出现 429，是系统性能差还是限流配置生效？
30. 提交前你会跑哪些命令？如果只改前端或只改后端，检查范围如何缩小？

## 15. 前端架构与用户体验

1. 前端为什么选择 React + TypeScript + Vite + Ant Design？
2. Zustand 和 React Query 的职责如何划分？
3. auth.store 中 `loading/authenticated/unauthenticated` 三态解决什么问题？
4. token 为什么封装在 `utils/token`，而不是直接存在 store？
5. Axios request interceptor 做了哪些事情？
6. Axios response interceptor 如何处理统一 envelope？
7. 401 时清 token 并派发 `auth:expired`，页面如何响应？
8. 前端每个写请求如何生成和传递 `Idempotency-Key`？
9. 当前 idempotency key 每次 mutation 都新生成，网络重试时是否会复用？
10. React Query 的 queryKey 如何设计？为什么 task list key 包含 projectId/status/assigneeId/keyword？
11. `useInfiniteQuery` 如何处理 cursor 分页？
12. 前端乐观更新如何同时更新 detail query 和 list query？
13. `snapshotTaskListQueries` 为什么要保存多个列表缓存？
14. 如果任务状态变化后不匹配当前筛选条件，前端如何从列表移除它？
15. 前端 `taskMatchesFilters` 会查 content，但后端 keyword 只查 title，这会导致什么问题？
16. mutation onError 如何回滚乐观更新？
17. onSuccess 后为什么还要 onSettled invalidate？
18. 如果后端返回新版 version，前端如何同步？
19. 如果两个浏览器同时编辑同一任务，前端如何展示冲突？
20. 状态机 hook `useStatusTransitions` 和后端状态机如何保持一致？
21. 看板拖拽时如何避免非法状态跳转？
22. 权限 hook `useTaskPermission` 和 `useProjectPermission` 是否只用于 UI 控制？
23. 前端隐藏按钮是否能防止越权？为什么后端仍必须校验？
24. 项目归档后前端如何进入只读模式？
25. ErrorBoundary 处理哪些错误？网络错误和渲染错误分别在哪里处理？
26. MSW fixtures 如何模拟 owner/admin/member 权限矩阵？
27. 前端类型来自手写 types，是否应该由 OpenAPI/proto 自动生成？
28. Ant Design 组件对 bundle size 有什么影响？`stats.html` 如何分析？
29. 前端请求超时 10 秒，后端 gRPC deadline 如果更短或更长，会有什么体验差异？
30. 如果要支持离线编辑或弱网重试，React Query 和幂等设计需要怎么改？

## 16. 配置、部署与运维

1. 项目为什么支持 local/dev/docker 三套配置？
2. Viper YAML + 环境变量覆盖的优点是什么？
3. 哪些配置必须通过环境变量注入，不能写入 YAML？
4. `JWT_SECRET`、`INTERNAL_TOKEN` 在生产如何管理？
5. Docker Compose 中为什么只编排基础设施，没有编排三个 Go 服务？
6. PostgreSQL 暴露 5433、Redis 暴露 6380 的原因是什么？
7. Prometheus 通过 `host.docker.internal` 抓取宿主机服务指标，有什么平台兼容问题？
8. Jaeger 开启 OTLP 4317/4318，服务如何连接？
9. Grafana dashboard provisioning 如何工作？
10. 数据库迁移脚本为什么为两个 schema 使用独立 migrations table？
11. 服务启动顺序如何保证？如果 DB/Redis 未 ready，会发生什么？
12. `/readyz` 应该检查哪些依赖？当前 ready 语义是否足够严格？
13. 优雅关闭时需要关闭哪些资源？gRPC conn、Redis conn、LogWriter、HTTP server 分别如何处理？
14. 如果 task-service 收到 SIGTERM，如何保证剩余操作日志尽量 flush？
15. 生产部署时是否应该把 gateway 和服务做成多个副本？哪些组件需要共享状态？
16. 多副本部署下 Redis 限流是否天然全局生效？
17. 多副本部署下操作日志 channel 是每个实例独立的，这有什么影响？
18. 多副本 gateway 下 Idempotency-Key 是否仍然有效？为什么？
19. 如果要上 Kubernetes，需要新增哪些 manifest 或 Helm values？
20. 健康检查、资源限制、HPA、日志采集、Secret、ConfigMap 分别怎么设计？

## 17. 性能、容量与压测

1. 项目声称单机 1k QPS，测试场景是什么？读写比例如何？
2. 1k QPS 是否包含登录接口？为什么登录接口单独压测？
3. bcrypt cost 对登录 P99 的影响为什么很大？
4. DB 连接池 `max open`、`max idle` 如何影响吞吐？
5. Redis 连接池在高并发下可能成为瓶颈吗？
6. gateway 到 gRPC 服务的连接是否复用？连接数如何控制？
7. Gin、gRPC、GORM、Redis 客户端各自的超时在哪里设置？
8. 如果压测中 P99 高但 P95 正常，可能是什么原因？
9. 如果 QPS 上去后错误率增加，如何判断是限流、DB、Redis、gRPC 还是 CPU？
10. 操作日志异步化对写接口延迟有多大帮助？
11. 幂等响应缓存会不会增加写接口延迟？
12. 用户信息 enrichment 使用 BatchGetUsers 能减少多少 RPC 调用？
13. 批量查询用户缓存命中率低时，性能瓶颈在哪里？
14. cursor 分页相比 offset 分页对大数据量有什么优势？
15. `ILIKE '%keyword%'` 在任务量大时会拖慢列表接口，如何量化和优化？
16. Prometheus metrics 本身会不会带来性能开销？
17. AlwaysSample trace 在高 QPS 生产环境会带来什么成本？
18. 前端 bundle 中 antd 是否过大？如何拆包或按需加载？
19. 如果要支持 10 万项目、1000 万任务，数据库索引和分表策略如何改？
20. 如果要支持 1 万并发在线用户，gateway、Redis、DB 如何扩容？

## 18. 前后端契约与 API 演进

1. proto 是内部 RPC 契约，HTTP API 契约在哪里定义？
2. docs/api-reference 和 handler 实现如何保持一致？
3. 前端 `api/types.ts` 是否由后端自动生成？手写类型有什么风险？
4. 如果 task proto 新增字段，前端如何感知？
5. 如果 HTTP 响应 envelope 新增 `details` 字段，前端兼容吗？
6. proto3 字段默认值会带来什么问题？例如 status=0 如何区分“未传”和 todo？
7. `ListTasksRequest.status` 是 int32 默认 0，这是否会导致无法区分不过滤和过滤 todo？
8. 当前是否有额外机制处理 status 可选？如果没有，bug 会表现成什么？
9. 如果需要可选字段，proto3 应该用 `optional`、wrapper，还是额外 bool？
10. `assignee_id` 空字符串表示未传，这和“查询未指派任务”冲突吗？
11. HTTP PUT 是否应该支持 partial update？当前 title 空字符串语义是什么？
12. 如果 API v2 要改字段名，如何兼容旧客户端？
13. Postman collection 自动同步 version 变量，这解决什么测试痛点？
14. OpenAPI 文档是否应该从代码生成？引入后有什么收益？
15. gRPC reflection 在开发和生产环境分别是否应该开启？

## 19. 代码细节与潜在风险追问

1. `SetupIdempotency` 返回 pending 时直接 HTTP 200，这是否会让客户端误以为业务成功？
2. 幂等缓存没有保存 status code，重复 DELETE 的语义是否正确？
3. Auth middleware 中 Redis 黑名单查询失败返回 500，和“Redis 缓存失败降级”的策略是否一致？
4. RateLimit 的 public path 使用 `strings.TrimSuffix`，Auth middleware public path 不 trim，尾斜杠请求会怎样？
5. `BatchGetUsers` 先追加缓存命中用户，再 append DB missed 用户，返回顺序是否稳定？
6. `TaskBiz.ListTasks` 中 statusStr 用 `string(rune(*filter.Status + '0'))`，如果 status 大于 9 会怎样？
7. `ListTasksRequest.status` 默认值 0 是否可能导致所有请求默认过滤 todo？
8. `AssignTask` 不要求客户端传 version，而是用读取到的 task.Version 更新，这和 `ChangeTaskStatus` 是否一致？
9. `UpdateTask` 中 dueTime 为空就不更新，那如何清空 due_time？
10. `ProjectBiz.GetProject` 先校验成员再读缓存，为什么顺序不能反过来？
11. 项目缓存只在 update/archive/unarchive/transfer 时失效，create 后是否需要写缓存？
12. `ProjectBiz.AddProjectMember` 没有失效 project cache，这是否有问题？
13. `RemoveProjectMember` 返回 targetMember，但删除后前端如何确认最新成员列表？
14. `LeaveProject` 先查 member 再查 project，非成员和项目不存在都返回 not found，是否符合隐藏资源存在性的目标？
15. `projectRepo.Update` 使用 `Updates(map)`，如果 updates 里没有 `updated_at`，GORM 是否会自动更新？
16. `taskRepo.Delete` 先 Find 再 Delete，中间如果任务被别人删了会怎样？
17. `commentRepo.Delete` 不检查 RowsAffected，是否有风险？
18. `operationLogRepo.listLogs` 的 where 字符串由调用方传入，是否存在 SQL 注入风险？为什么当前可控？
19. `xcursor.EncodeCursor` 使用 `base64.URLEncoding` 会带 `=` padding，URL 中是否需要处理？
20. `ComputeFilterHash` 用 `strings.Join(params, "|")`，如果参数里包含 `|` 是否可能歧义？
21. `xjwt.Validate` 只检查 signing method 是 HMAC，是否需要固定 HS256？
22. JWT claims 是否校验 issuer/audience？当前没有会有什么影响？
23. `NewClients` 中第一个连接成功、第二个连接失败会关闭 userConn，这个资源处理是否完整？
24. `grpc.NewClient` 配合 `insecure.NewCredentials` 是否会在启动阶段发现地址不可达？
25. Prometheus counter 使用 `MustRegister`，测试中多次构造对象是否会 panic？项目如何处理？
26. `bcryptCost` 是包初始化时读取环境变量，测试中动态改 env 会不会生效？
27. `UserBiz.Register` 弱密码 map 是否大小写敏感？`Password123` 和 `password123` 如何处理？
28. `mail.ParseAddress` 可能接受 `"Name <a@b.com>"`，项目是否希望接受？
29. `ListProjectMembers` 是否会返回 owner/admin/member 的排序？前端是否依赖排序？
30. 前端 `window.crypto.randomUUID` 在不支持的浏览器下 fallback 是否足够？
31. 前端 request interceptor 每次都设置 `X-Request-Id`，如果用户传入自定义 request id 会被覆盖吗？
32. 前端 response interceptor 对 envelope.code 非 OK 抛 `AppError`，HTTP 200 但业务失败是否能处理？
33. 前端遇到非 envelope 响应直接返回 response.data，这是否可能掩盖后端异常？
34. MSW mock 的状态机和后端状态机是否完全一致？如何验证？
35. Playwright 在 mock 模式下跑，不经过真实 gateway，哪些问题测不到？

## 20. 扩展设计与系统演进

1. 如果要支持项目邀请链接，权限和数据模型怎么改？
2. 如果要支持组织/团队空间，用户、项目、成员关系如何重构？
3. 如果要支持任务附件，文件存储、权限校验、扫描和过期清理怎么设计？
4. 如果要支持任务评论 @ 提醒，通知服务如何接入？
5. 如果要支持全文搜索，你会在 PG 内做还是引入 Elasticsearch/OpenSearch？
6. 如果要支持任务历史版本回滚，数据模型如何设计？
7. 如果要支持审计日志不可删除，数据库和权限如何调整？
8. 如果要支持多租户，tenant_id 应该放在哪些表和索引中？
9. 如果要支持企业 SSO/OAuth2，认证服务如何改？
10. 如果要支持 refresh token 和多端登录管理，需要新增哪些表？
11. 如果要支持移动端离线提交，幂等、冲突解决和版本号如何设计？
12. 如果要支持任务依赖关系，状态机如何处理“前置任务未完成”？
13. 如果要支持项目模板，哪些数据需要复制？如何避免复制成员？
14. 如果要支持软删除恢复，tasks/projects 的 deleted_at 和唯一索引会遇到什么问题？
15. 如果要支持数据导出，如何避免大查询拖垮主库？
16. 如果要支持读写分离，事务、缓存一致性、read-after-write 如何处理？
17. 如果要支持分库分表，project_id 是否适合作为分片键？
18. 如果要拆 user-service 独立数据库，跨服务用户展示信息如何处理？
19. 如果要把操作日志改成 Kafka，你如何保证消息和业务 DB 的一致性？
20. 如果要做灰度发布，proto/HTTP API/数据库迁移如何保证向前兼容？

## 21. 项目复盘与个人贡献

1. 这个项目从 0 到完成分了哪些阶段？每个阶段的可验证交付是什么？
2. 哪个阶段风险最大？你是如何降低风险的？
3. 你在实现过程中遇到最难定位的 bug 是什么？最后怎么解决？
4. 哪个设计是你后来推翻重做的？为什么？
5. 你如何判断某个模块需要先写测试？
6. 如果时间只够做 3 个亮点，你会保留哪些，砍掉哪些？
7. 这个项目中你学到的最重要的工程经验是什么？
8. 你如何向面试官证明这些代码是你自己理解并实现的？
9. 如果让我现场让你改一个需求，你最有信心改哪一块？最担心哪一块？
10. 如果团队接手这个项目，你会先补哪些文档和自动化检查？
11. 你觉得项目最大的技术债是什么？
12. 如果重新开始，你会继续选择微服务吗？为什么？
13. 如果面试官质疑“这个项目过度设计”，你会如何回应？
14. 如果面试官质疑“这个项目不是真正生产级”，你会承认哪些不足？
15. 你如何用这个项目说明你具备排障、压测、测试和架构取舍能力？

## 22. 高频连环追问示例

### 22.1 幂等连环问

1. 为什么写接口需要幂等？
2. 为什么不用数据库唯一键解决所有幂等问题？
3. Redis `SETNX` 如何保证并发下只有一个请求执行？
4. 如果第一个请求执行中，第二个请求来了，当前实现返回什么？
5. 这个返回是否严格正确？如果不严格，你如何改？
6. 如果业务执行成功但缓存响应失败，重试会怎样？
7. 如果业务执行失败，为什么删除 key？
8. 如果客户端没有复用同一个 key，幂等是否还生效？
9. 幂等 key 是否应该绑定请求体 hash？
10. 幂等模块应该放 gateway 还是 service？

### 22.2 权限连环问

1. owner/admin/member 各自能做什么？
2. 为什么非成员返回 404？
3. 如果前端要区分“无权限”和“不存在”，你还会返回 404 吗？
4. 权限判断为什么放 biz 层？
5. 新增一个接口时如何保证不会漏权限？
6. owner 转让如何保证原子性？
7. owner 唯一索引能解决哪些问题，不能解决哪些问题？
8. 如果 creator 离开项目，他创建的任务谁能维护？
9. 如果 assignee 不是成员，为什么拒绝指派？
10. 如果用户被禁用，历史任务如何处理？

### 22.3 缓存连环问

1. 用户详情为什么要缓存？
2. 项目详情缓存前为什么先校验成员？
3. cache-aside 流程是什么？
4. 缓存失效发生在哪些写操作后？
5. Redis 挂了读接口会怎样？
6. 缓存穿透怎么处理？
7. 如何监控缓存命中率？
8. BatchGetUsers 返回顺序是否稳定？
9. 如何避免缓存脏读？
10. 如果用户昵称更新，任务列表展示如何及时更新？

### 22.4 乐观锁连环问

1. 为什么不用悲观锁？
2. 乐观锁 SQL 怎么写？
3. version 从哪里来？
4. 冲突时为什么返回 409？
5. 客户端怎么处理冲突？
6. 服务端自动重试是否合适？
7. AssignTask 为什么没有让客户端传 version？
8. ChangeTaskStatus 为什么要传 version？
9. 前端乐观更新和后端乐观锁如何配合？
10. 如果两个请求都拿到 version=1，会发生什么？

### 22.5 可观测性连环问

1. 请求慢了你先看什么？
2. request_id 和 trace_id 怎么关联？
3. HTTP/gRPC/DB/Redis 指标分别能定位什么？
4. 为什么不能把 user_id 作为 Prometheus label？
5. AlwaysSample 生产环境有什么问题？
6. Jaeger 没有 trace 你怎么排查？
7. 429 突增说明什么？
8. log writer channel full 突增说明什么？
9. 503 和 504 分别代表什么？
10. 你会加哪些告警？

### 22.6 前端连环问

1. React Query 管服务端状态，Zustand 管客户端状态，为什么这样分？
2. task queryKey 为什么要包含筛选条件？
3. 乐观更新如何回滚？
4. 多个任务列表缓存如何同时更新？
5. 状态变化后任务不满足当前筛选条件怎么办？
6. 前端状态机和后端状态机不一致怎么办？
7. 401 如何全局处理？
8. MSW E2E 测不到哪些问题？
9. idempotency key 如何跟 mutation 重试配合？
10. 如何减少 antd bundle 体积？

