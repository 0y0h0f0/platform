# API 参考

HTTP API 由 `api-gateway` 暴露，默认监听 `http://127.0.0.1:8080`。所有业务接口统一使用 `/api/v1` 前缀。

## 通用约定

### 请求头

| Header | 必填 | 说明 |
|--------|------|------|
| `Authorization: Bearer <token>` | 除注册、登录外必填 | JWT access token，默认 2 小时有效 |
| `Content-Type: application/json` | 有请求体时必填 | JSON 请求体 |
| `Idempotency-Key` | 写接口建议填写 | 适用于 `POST`、`PUT`、`DELETE`，24 小时内重复提交返回缓存响应 |
| `X-Request-ID` | 可选 | 不传则服务端生成；响应和日志中会携带 |

### 统一响应

成功响应：

```json
{
  "code": "OK",
  "message": "ok",
  "request_id": "req-id",
  "data": {}
}
```

失败响应：

```json
{
  "code": "INVALID_ARGUMENT",
  "message": "invalid request body",
  "request_id": "req-id"
}
```

### 错误码

| Code | HTTP | 常见原因 |
|------|------|----------|
| `INVALID_ARGUMENT` | 400 | 参数格式错误、缺少必填字段 |
| `UNAUTHENTICATED` | 401 | 未登录、token 无效或已登出 |
| `PERMISSION_DENIED` | 403 | 当前用户无权执行操作 |
| `NOT_FOUND` | 404 | 资源不存在，或非成员访问资源 |
| `ALREADY_EXISTS` | 409 | 用户名、邮箱、项目成员等唯一约束冲突 |
| `FAILED_PRECONDITION` | 400 | 归档项目写入、非法任务状态流转、owner 退出项目等前置条件不满足 |
| `ABORTED` | 409 | 乐观锁版本冲突 |
| `RESOURCE_EXHAUSTED` | 429 | 限流命中 |
| `INTERNAL` | 500 | 服务端内部错误，响应会隐藏内部细节 |
| `UNAVAILABLE` | 503 | 下游服务或基础设施不可用 |
| `DEADLINE_EXCEEDED` | 504 | 内部 RPC 超时 |

### 枚举

| 类型 | 值 |
|------|----|
| 用户状态 `user.status` | `0=active`，`1=disabled` |
| 项目状态 `project.status` | `0=active`，`1=archived` |
| 项目角色 `member.role` | `0=owner`，`1=admin`，`2=member` |
| 任务状态 `task.status` | `0=todo`，`1=doing`，`2=done`，`3=cancelled` |
| 任务优先级 `task.priority` | `0=low`，`1=medium`，`2=high` |

## Auth

### `POST /api/v1/auth/register`

注册用户并返回 access token。

请求体：

```json
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "StrongPassword123"
}
```

响应：`201 Created`，`data` 为 `RegisterResponse`。

### `POST /api/v1/auth/login`

使用用户名或邮箱登录。

请求体：

```json
{
  "account": "alice",
  "password": "StrongPassword123"
}
```

响应：`200 OK`，`data` 为 `LoginResponse`。

### `POST /api/v1/auth/logout`

登出当前 token。Gateway 从 token 中读取 `jti` 和过期时间，将其写入 Redis 黑名单，TTL 等于 token 剩余有效期。

响应：`200 OK`。

## Users

### `GET /api/v1/users/me`

获取当前登录用户信息。

响应：`200 OK`，`data` 为 `GetUserResponse`。

## Projects

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/projects` | 创建项目 |
| `GET` | `/api/v1/projects` | 查询当前用户参与的项目 |
| `GET` | `/api/v1/projects/:id` | 获取项目详情 |
| `PUT` | `/api/v1/projects/:id` | 更新项目名称和描述 |
| `POST` | `/api/v1/projects/:id/archive` | 归档项目，仅 owner |
| `POST` | `/api/v1/projects/:id/unarchive` | 取消归档，仅 owner |
| `POST` | `/api/v1/projects/:id/transfer` | 转让项目所有权，仅 owner |
| `POST` | `/api/v1/projects/:id/members` | 添加成员 |
| `GET` | `/api/v1/projects/:id/members` | 查询成员 |
| `PUT` | `/api/v1/projects/:id/members/:userId` | 修改成员角色 |
| `DELETE` | `/api/v1/projects/:id/members/:userId` | 移除成员 |
| `POST` | `/api/v1/projects/:id/members/me/leave` | 当前用户退出项目 |
| `GET` | `/api/v1/projects/:id/operation-logs` | 查询项目操作日志 |

### 创建项目

```json
{
  "name": "Campus Task Platform",
  "description": "demo project"
}
```

响应：`201 Created`，`data.project` 为项目对象。

### 查询项目

Query：

| 参数 | 默认 | 说明 |
|------|------|------|
| `limit` | `20` | 每页数量 |
| `offset` | `0` | 偏移量 |
| `include_archived` | `false` | 是否包含归档项目 |

### 更新项目

```json
{
  "name": "New Name",
  "description": "new description",
  "version": 0
}
```

`version` 必须使用最近一次读取到的版本；冲突时返回 `ABORTED`。

### 转让所有权

```json
{
  "target_user_id": "uuid"
}
```

目标用户必须是项目成员。转让会在事务内更新 `projects.owner_id` 和 `project_members.role`。

### 添加或修改成员

添加成员：

```json
{
  "user_id": "uuid",
  "role": 2
}
```

修改角色：

```json
{
  "role": 1
}
```

Owner 可以邀请任意角色；admin 只能邀请或移除普通 member；member 无成员管理权限。

### 查询操作日志

Query：

| 参数 | 默认 | 说明 |
|------|------|------|
| `limit` | `20` | 每页数量 |
| `cursor` | 空 | 下一页游标 |

响应经过 gateway 聚合，会尽量补充操作人展示信息。

## Tasks

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/tasks` | 创建任务 |
| `GET` | `/api/v1/tasks` | 查询任务列表 |
| `GET` | `/api/v1/tasks/:id` | 获取任务详情 |
| `PUT` | `/api/v1/tasks/:id` | 更新任务基础字段 |
| `DELETE` | `/api/v1/tasks/:id` | 删除任务 |
| `POST` | `/api/v1/tasks/:id/assign` | 指派任务 |
| `POST` | `/api/v1/tasks/:id/status` | 修改任务状态 |
| `POST` | `/api/v1/tasks/:id/comments` | 创建评论 |
| `GET` | `/api/v1/tasks/:id/comments` | 查询评论 |
| `DELETE` | `/api/v1/tasks/:id/comments/:commentId` | 删除评论 |
| `GET` | `/api/v1/tasks/:id/operation-logs` | 查询任务操作日志 |

### 创建任务

```json
{
  "project_id": "uuid",
  "title": "Implement auth flow",
  "content": "Register, login and logout"
}
```

响应：`201 Created`，`data.task` 为任务对象。

### 查询任务

Query：

| 参数 | 必填 | 默认 | 说明 |
|------|------|------|------|
| `project_id` | 是 | 无 | 项目 ID |
| `limit` | 否 | `20` | 每页数量 |
| `cursor` | 否 | 空 | 下一页游标 |
| `status` | 否 | `-1` | 任务状态；不传表示全部 |
| `assignee_id` | 否 | 空 | 指派人过滤 |
| `keyword` | 否 | 空 | 标题/内容关键词 |

任务列表使用游标分页。游标包含 `created_at`、`id` 和筛选条件 hash；如果客户端换了筛选条件却复用旧游标，服务端会拒绝请求。

### 更新任务

```json
{
  "title": "Implement auth flow v2",
  "content": "update content",
  "priority": 2,
  "due_time": "2026-06-01T10:00:00Z",
  "version": 3
}
```

`PUT /tasks/:id` 不接受 `status` 和 `assignee_id`。状态变更和指派必须走独立端点。

### 指派任务

```json
{
  "assignee_id": "uuid"
}
```

被指派用户必须存在、启用且是项目成员。

### 修改任务状态

```json
{
  "status": 1,
  "version": 3
}
```

允许的状态流转：

```text
todo -> doing | done | cancelled
doing -> done | cancelled | todo
done -> doing
cancelled -> todo
```

非法流转返回 `FAILED_PRECONDITION`。

### 评论

创建评论：

```json
{
  "content": "Looks good"
}
```

查询评论 Query：

| 参数 | 默认 | 说明 |
|------|------|------|
| `limit` | `20` | 每页数量 |
| `after_id` | 空 | 从指定评论 ID 之后继续查询 |

评论列表响应会由 gateway 聚合用户展示信息。

## 幂等规则

所有写接口支持 `Idempotency-Key`：

- Redis key 前缀为 `idempotency:`，TTL 为 24 小时。
- 登录用户的 key 会加上 `user_id` 前缀，避免不同用户之间 key 冲突。
- 首次请求成功后缓存响应体；重复请求返回缓存响应，HTTP 状态为 `200 OK`。
- 首次请求返回 4xx 或 5xx 时会清理 key，允许客户端修正后重试。
- 如果重复请求命中仍处于 `pending` 的 key，会返回 `409 Conflict` + `ABORTED`，提示客户端稍后重试。

## 调试工具

推荐导入 [postman/task-platform.postman_collection.json](postman/task-platform.postman_collection.json)。集合中注册和登录请求会自动保存 token，后续接口可直接使用 Bearer token 调试。
