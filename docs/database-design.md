# 数据库设计

项目使用 PostgreSQL 16，一个数据库实例内拆分两个 schema：`user_svc` 和 `task_svc`。这样既保持用户域与任务协作域的逻辑边界，又避免在当前规模引入跨库事务和多实例部署复杂度。

## Schema

| Schema | 归属服务 | 表 |
|--------|----------|----|
| `user_svc` | `user-service` | `users` |
| `task_svc` | `task-service` | `projects`、`project_members`、`tasks`、`task_comments`、`operation_logs` |

## 逻辑关系

```text
user_svc.users
  |
  | user id referenced by owner_id / user_id / creator_id / assignee_id
  v
task_svc.projects
  +-- task_svc.project_members
  +-- task_svc.tasks
        +-- task_svc.task_comments
        +-- task_svc.operation_logs
```

当前迁移未声明跨 schema 外键，跨服务一致性由业务层校验：添加成员、指派任务时通过 `user-service` 校验用户存在且启用。

## `user_svc.users`

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `UUID` | 主键，默认 `gen_random_uuid()` |
| `username` | `VARCHAR(32)` | 用户名，软删除条件下唯一 |
| `email` | `VARCHAR(320)` | 邮箱，软删除条件下唯一 |
| `password_hash` | `VARCHAR(255)` | bcrypt hash |
| `nickname` | `VARCHAR(64)` | 昵称 |
| `avatar_url` | `TEXT` | 头像地址 |
| `status` | `SMALLINT` | `0=active`，`1=disabled` |
| `created_at` | `TIMESTAMPTZ` | 创建时间 |
| `updated_at` | `TIMESTAMPTZ` | 更新时间 |
| `deleted_at` | `TIMESTAMPTZ` | GORM 软删除字段 |

索引：

| 索引 | 说明 |
|------|------|
| `idx_users_username` | `username` 部分唯一索引，条件 `deleted_at IS NULL` |
| `idx_users_email` | `email` 部分唯一索引，条件 `deleted_at IS NULL` |

## `task_svc.projects`

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `UUID` | 主键 |
| `name` | `VARCHAR(100)` | 项目名 |
| `description` | `TEXT` | 项目描述 |
| `owner_id` | `UUID` | 项目 owner 用户 ID |
| `status` | `SMALLINT` | `0=active`，`1=archived` |
| `version` | `BIGINT` | 乐观锁版本 |
| `created_at` | `TIMESTAMPTZ` | 创建时间 |
| `updated_at` | `TIMESTAMPTZ` | 更新时间 |
| `deleted_at` | `TIMESTAMPTZ` | 软删除字段 |

索引：

| 索引 | 说明 |
|------|------|
| `idx_projects_owner_name` | `(owner_id, name)` 部分唯一索引，防止同一 owner 下项目重名 |

## `task_svc.project_members`

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `UUID` | 主键 |
| `project_id` | `UUID` | 项目 ID |
| `user_id` | `UUID` | 用户 ID |
| `role` | `SMALLINT` | `0=owner`，`1=admin`，`2=member` |
| `joined_at` | `TIMESTAMPTZ` | 加入时间 |
| `updated_at` | `TIMESTAMPTZ` | 更新时间 |

索引：

| 索引 | 说明 |
|------|------|
| `idx_project_members_project_user` | `(project_id, user_id)` 唯一索引，防重复成员 |
| `idx_project_members_owner` | `(project_id) WHERE role = 0` 部分唯一索引，确保一个项目最多一个 owner |
| `idx_project_members_user` | `(user_id, project_id)`，支持查询用户参与项目 |

`project_members` 使用物理删除。owner 不能直接退出项目，必须先转让所有权。

## `task_svc.tasks`

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `UUID` | 主键 |
| `project_id` | `UUID` | 所属项目 |
| `title` | `VARCHAR(200)` | 标题 |
| `content` | `TEXT` | 内容 |
| `status` | `SMALLINT` | `0=todo`，`1=doing`，`2=done`，`3=cancelled` |
| `priority` | `SMALLINT` | `0=low`，`1=medium`，`2=high` |
| `assignee_id` | `UUID` | 指派人，可空 |
| `creator_id` | `UUID` | 创建人 |
| `due_time` | `TIMESTAMPTZ` | 截止时间，可空 |
| `version` | `BIGINT` | 乐观锁版本 |
| `extra` | `JSONB` | 扩展字段，约定只放 `labels`、`checklist`、`attachments` |
| `created_at` | `TIMESTAMPTZ` | 创建时间 |
| `updated_at` | `TIMESTAMPTZ` | 更新时间 |
| `deleted_at` | `TIMESTAMPTZ` | 软删除字段 |

索引：

| 索引 | 说明 |
|------|------|
| `idx_tasks_default` | `(project_id, created_at DESC, id DESC)`，默认游标分页 |
| `idx_tasks_status` | `(project_id, status, created_at DESC, id DESC)`，状态过滤 |
| `idx_tasks_assignee` | `(project_id, assignee_id, created_at DESC, id DESC)`，指派人过滤 |
| `idx_tasks_due_time` | `due_time` 非空任务查询 |

## `task_svc.task_comments`

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `UUID` | 主键 |
| `task_id` | `UUID` | 任务 ID |
| `user_id` | `UUID` | 评论人 |
| `content` | `TEXT` | 评论内容 |
| `created_at` | `TIMESTAMPTZ` | 创建时间 |

索引：

| 索引 | 说明 |
|------|------|
| `idx_task_comments_task` | `(task_id, id)`，支持 `after_id` 分页 |

评论使用物理删除。

## `task_svc.operation_logs`

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `UUID` | 主键 |
| `project_id` | `UUID` | 项目 ID，可空 |
| `task_id` | `UUID` | 任务 ID，可空 |
| `operator_id` | `UUID` | 操作人 |
| `action` | `VARCHAR(50)` | 操作类型 |
| `detail` | `JSONB` | 操作详情 |
| `created_at` | `TIMESTAMPTZ` | 创建时间 |

索引：

| 索引 | 说明 |
|------|------|
| `idx_operation_logs_project` | `(project_id, created_at DESC, id DESC)` |
| `idx_operation_logs_task` | `(task_id, created_at DESC, id DESC)` |

## 并发控制

`projects` 和 `tasks` 使用乐观锁：

```sql
UPDATE ...
SET ..., version = version + 1
WHERE id = $1 AND version = $2;
```

更新影响行数为 0 时，服务返回 `ABORTED`，客户端需要重新读取最新版本后重试。

## 迁移策略

迁移文件位于：

```text
migrations/user_svc/
migrations/task_svc/
```

本地执行：

```bash
make migrate
```

脚本会：

1. 从 `.env` 读取数据库连接配置。
2. 等待 PostgreSQL 端口可用。
3. 使用 `migrate/migrate:v4.18.3` Docker 镜像分别执行 `user_svc`、`task_svc` 迁移。
4. 为每个 schema 使用独立 migrations table：`schema_migrations_user_svc` 和 `schema_migrations_task_svc`。

新增迁移时必须同时提供 `.up.sql` 和 `.down.sql`，并保持 schema 级别的迁移目录隔离。
