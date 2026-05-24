# 团队任务协作平台 - 前端实施计划

## 1. 项目定位

为 `task-platform` 后端项目构建一个完整的前端管理界面，覆盖注册登录、项目管理、任务看板、评论系统、成员管理和操作日志等全部功能。

**技术选型：** 基于现有 `web/` 脚手架增量建设（React 19 + TypeScript 6）。阶段 1 实测 Vite 8/Rolldown 在当前环境会 SIGBUS，故固定为 Vite 7.3.3 + @vitejs/plugin-react 5.2.0；新增 Ant Design 5 + TanStack Query + Zustand + React Router + Axios。不要为了前端计划把脚手架降级到 React 18 / Vite 6。

## 2. 目录结构

```
web/
├── index.html
├── package.json
├── tsconfig.json
├── tsconfig.app.json
├── tsconfig.node.json
├── vite.config.ts
├── eslint.config.js                   # ESLint flat config
├── .prettierrc                        # Prettier 配置
├── .env.development
├── .env.production
├── src/
│   ├── main.tsx
│   ├── App.tsx
│   ├── vite-env.d.ts                  # 含 ImportMetaEnv 类型扩展
│   │
│   ├── api/
│   │   ├── client.ts                  # Axios 实例：拦截器（token/幂等/401/统一解包）
│   │   ├── auth.ts                    # register, login, logout, getMe
│   │   ├── project.ts                 # 项目 CRUD + 归档/转让
│   │   ├── task.ts                    # 任务 CRUD + 指派/状态变更
│   │   ├── comment.ts                 # 评论 创建/列表/删除
│   │   ├── member.ts                  # 成员 添加/修改/移除/离开
│   │   ├── operationLog.ts            # 项目+任务操作日志
│   │   ├── types.ts                   # REST DTO 类型（以 HTTP JSON envelope/data 为准）
│   │   └── generated/                 # 可选：buf 生成的 proto 基础类型；评论/日志需 REST 扩展类型
│   │
│   ├── stores/
│   │   ├── auth.store.ts              # Zustand：三态鉴权（loading/authenticated/unauthenticated）
│   │   └── ui.store.ts                # Zustand：仅全局 UI（侧边栏折叠），project 作用域状态走 URL params
│   │
│   ├── hooks/
│   │   ├── useAuth.ts
│   │   ├── useProjectPermission.ts    # 根据 role 返回权限布尔值
│   │   ├── useTaskPermission.ts       # 根据 role + creator 返回权限布尔值
│   │   └── useStatusTransitions.ts    # 返回当前状态允许的目标状态列表
│   │
│   ├── pages/
│   │   ├── auth/
│   │   │   ├── LoginPage.tsx
│   │   │   └── RegisterPage.tsx
│   │   ├── projects/
│   │   │   ├── ProjectListPage.tsx
│   │   │   └── ProjectDetailPage.tsx  # 看板 Tab + 设置 Tab
│   │   └── NotFoundPage.tsx
│   │
│   ├── components/
│   │   ├── layout/
│   │   │   ├── AppLayout.tsx          # Ant Design Layout 骨架
│   │   │   ├── AppHeader.tsx          # Logo + 用户下拉菜单
│   │   │   ├── AppSider.tsx           # 导航
│   │   │   └── PageSkeleton.tsx       # 路由级 Suspense 的 fallback
│   │   ├── auth/
│   │   │   ├── AuthGuard.tsx          # 三态：loading→骨架屏 / unauthenticated→/login / authenticated→Outlet
│   │   │   └── GuestGuard.tsx         # 已登录 → /projects
│   │   ├── project/
│   │   │   ├── ProjectCard.tsx
│   │   │   ├── ProjectCreateModal.tsx
│   │   │   ├── ProjectEditModal.tsx
│   │   │   ├── KanbanBoard.tsx        # 四列看板
│   │   │   ├── KanbanColumn.tsx
│   │   │   ├── TaskCard.tsx           # 看板中的任务卡片
│   │   │   ├── TaskCreateModal.tsx
│   │   │   ├── TaskDetailDrawer.tsx   # 任务详情抽屉（最复杂的组件）
│   │   │   ├── TaskEditForm.tsx
│   │   │   ├── TaskStatusSelect.tsx   # 合法状态转换下拉
│   │   │   ├── TaskAssignSelect.tsx   # 指派成员选择器
│   │   │   ├── CommentList.tsx        # after_id 分页
│   │   │   ├── CommentItem.tsx
│   │   │   ├── CommentInput.tsx
│   │   │   ├── OperationLogList.tsx   # cursor 分页
│   │   │   ├── OperationLogItem.tsx
│   │   │   ├── MemberList.tsx
│   │   │   ├── MemberAddModal.tsx
│   │   │   ├── ProjectToolbar.tsx     # 搜索/筛选栏
│   │   │   └── ArchiveToggle.tsx
│   │   └── common/
│   │       ├── ErrorBoundary.tsx
│   │       ├── LoadingSpinner.tsx
│   │       ├── EmptyState.tsx
│   │       └── PriorityTag.tsx
│   │
│   ├── queries/                       # TanStack Query hooks（每个资源一个文件）
│   │   ├── auth.queries.ts
│   │   ├── project.queries.ts
│   │   ├── task.queries.ts
│   │   ├── comment.queries.ts
│   │   ├── member.queries.ts
│   │   └── operationLog.queries.ts
│   │
│   ├── mocks/                         # MSW mock handlers（按接口模块拆分，同时驱动开发+测试）
│   │   ├── browser.ts                 # setupWorker(...handlers)
│   │   ├── server.ts                  # setupServer(...handlers) (Vitest)
│   │   ├── handlers/
│   │   │   ├── auth.handlers.ts
│   │   │   ├── project.handlers.ts
│   │   │   ├── task.handlers.ts
│   │   │   ├── comment.handlers.ts
│   │   │   ├── member.handlers.ts
│   │   │   └── operationLog.handlers.ts
│   │   └── fixtures/                  # 模拟响应数据
│   │
│   ├── utils/
│   │   ├── constants.ts               # 类型安全的枚举映射、中文标签、颜色
│   │   ├── cursor.ts                  # base64url cursor 编解码
│   │   ├── error.ts                   # 错误码 → 用户中文提示
│   │   ├── token.ts                   # localStorage token 读写
│   │   └── time.ts                    # dayjs 时间格式化 + 中文 locale
│   │
│   └── styles/
│       ├── global.css                 # 全局样式 + Ant Design 主题覆盖
│       └── kanban.css                 # 看板布局样式
│
├── __tests__/                         # Vitest + React Testing Library
│   ├── setup.ts                       # 全局测试配置（MSW server、清理）
│   ├── utils/
│   ├── hooks/
│   ├── stores/
│   └── components/
│
└── public/
    └── favicon.ico
```

## 3. 路由设计

| 路径 | 页面 | 鉴权 | 说明 |
|------|------|------|------|
| `/login` | `LoginPage` | 仅游客 | 账号密码登录 |
| `/register` | `RegisterPage` | 仅游客 | 注册表单 |
| `/projects` | `ProjectListPage` | 需登录 | 项目卡片列表（首页） |
| `/projects/:id` | `ProjectDetailPage` | 需登录 | 看板 + 项目设置双 Tab |
| `/` | Redirect → `/projects` | - | 默认跳转 |
| `*` | `NotFoundPage` | 无 | 404 |

所有页面组件使用 `React.lazy()` + `<Suspense fallback={<PageSkeleton />}>` 做路由级代码分割。每个页面出口包 `<ErrorBoundary>` 隔离崩溃。

AuthGuard 采用**三态鉴权**（`loading` / `authenticated` / `unauthenticated`），避免页面刷新时因 Zustand 初始值短暂为 `false` 而误将已登录用户重定向到 `/login`：
- `loading`：渲染骨架屏，等待 authStore 从 localStorage 同步完成
- `unauthenticated`：重定向到 `/login`
- `authenticated`：渲染 `<Outlet />`

## 4. 组件树

```
<App>
  <QueryClientProvider>
    <BrowserRouter>
      <Routes>
        <Route element={<GuestGuard />}>           // 已登录则跳转 /projects
          <Route path="/login" element={
            <Suspense fallback={<PageSkeleton />}>
              <ErrorBoundary><LoginPage /></ErrorBoundary>
            </Suspense>
          } />
          <Route path="/register" element={
            <Suspense fallback={<PageSkeleton />}>
              <ErrorBoundary><RegisterPage /></ErrorBoundary>
            </Suspense>
          } />
        </Route>
        <Route element={<AuthGuard />}>             // 三态鉴权：loading→骨架屏 / unauthenticated→/login / authenticated→Outlet
          <Route element={<AppLayout />}>           // Header + Sider + Content
            <Route path="/projects" element={
              <Suspense fallback={<PageSkeleton />}>
                <ErrorBoundary><ProjectListPage /></ErrorBoundary>
              </Suspense>
            } />
            <Route path="/projects/:id" element={
              <Suspense fallback={<PageSkeleton />}>
                <ErrorBoundary><ProjectDetailPage /></ErrorBoundary>
              </Suspense>
            } />
            <Route index element={<Navigate to="/projects" replace />} />
          </Route>
        </Route>
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </BrowserRouter>
  </QueryClientProvider>
</App>
```

### 4.1 AppLayout

```
+--------------------------------------------------------------+
| AppHeader: [Logo] 团队任务协作平台    [当前用户 / 退出登录]      |
+--------------------------------------------------------------+
| AppSider (可折叠)              | <Outlet /> (Content)         |
|   - 项目列表 (nav link)        |                              |
|   - 创建项目 (button)          |                              |
+-------------------------------+------------------------------+
```

### 4.2 ProjectListPage

```
+--------------------------------------------------------------+
| 头部: "我的项目" [+ 创建项目按钮] [归档项目开关]                 |
+--------------------------------------------------------------+
| 卡片网格: [ProjectCard] [ProjectCard] [ProjectCard] ...       |
| 上一页/下一页（offset-based；后端暂不返回 total，不能使用依赖 total 的完整分页器） |
+--------------------------------------------------------------+
```

### 4.3 ProjectDetailPage

```
+--------------------------------------------------------------+
| 项目头部: 名称, 描述, [项目设置按钮]                             |
+--------------------------------------------------------------+
| ProjectToolbar: [关键词搜索] [负责人筛选] [状态筛选]              |
+--------------------------------------------------------------+
| <Tabs>                                                        |
|   <Tab key="kanban" tab="看板">                               |
|     <KanbanBoard>                                             |
|       待办 | 进行中 | 已完成 | 已取消                            |
|     </KanbanBoard>                                            |
|   </Tab>                                                      |
|   <Tab key="settings" tab="项目设置">                          |
|     编辑项目信息 | 成员管理 | 危险操作区 | 操作日志                |
|   </Tab>                                                      |
| </Tabs>                                                       |
|                                                               |
| <TaskDetailDrawer open={selectedTaskId} />                    |
+--------------------------------------------------------------+
```

### 4.4 TaskDetailDrawer（右侧抽屉，宽度 ~720px）

从上到下依次：
1. **任务头部**：标题（可编辑）、优先级标签、状态徽章、创建者、负责人、截止日期
2. **操作栏**：编辑、删除（Popconfirm）、指派成员下拉、状态变更下拉
3. **编辑表单**（点击编辑时显示）：标题、内容、优先级、截止日期
4. **评论区**：`<CommentList>` + `<CommentInput>`
5. **操作日志**：`<OperationLogList>`（可折叠）

## 5. 数据流

### 5.1 鉴权流程

Token 存储在 `localStorage`（access token 2h TTL，无 refresh token）。需注意 localStorage 对 XSS 攻击敏感——依赖后端 SecurityHeaders 中间件的 CSP 头（`plan.md` §9.1）作为缓解措施。若后续引入 refresh token，必须改用 httpOnly cookie。

```
注册/登录
    ↓
POST /api/v1/auth/register 或 /api/v1/auth/login
    ↓
响应: { access_token, user }
    ↓
├─ token → localStorage.setItem("token", access_token)
├─ user  → authStore.setUser(user), status = "authenticated"
└─ 跳转到 /projects

页面刷新
    ↓
authStore 初始状态为 "loading"（三态：loading | authenticated | unauthenticated）
    ↓
authStore.hydrate() 同步读取 localStorage
    ├─ token 存在 → 发起 GET /api/v1/users/me 验证
    │     ├─ 成功 → status = "authenticated"
    │     └─ 401  → status = "unauthenticated" → 清除 token → 跳转 /login
    └─ token 不存在 → status = "unauthenticated" → 跳转 /login

AuthGuard 渲染逻辑：
    ├─ status === "loading"   → 渲染 <PageSkeleton />
    ├─ status === "unauthenticated" → <Navigate to="/login" />
    └─ status === "authenticated"   → <Outlet />
```

### 5.2 API 调用流程

```
组件调用 TanStack Query hook
    ↓
queries/*.ts (useQuery / useMutation)
    ↓
api/*.ts (具体请求函数)
    ↓
api/client.ts (Axios 实例)
    ├─ 请求拦截器：
    │    - 附加 Authorization: Bearer <token>
    │    - POST/PUT/DELETE 附加 Idempotency-Key（同一次用户意图必须复用同一个 key，见 §6.7）
    │    - 附加 X-Request-Id
    ├─ Vite proxy → Go 后端 :8080
    └─ 响应拦截器：
         - code !== "OK" → 抛出 AppError
         - 返回 response.data.data（解包 envelope）
         - HTTP 401 → 清除 auth → 跳转 /login
         - 网络错误 → AppError("NETWORK_ERROR")
```

### 5.3 状态管理分工

**TanStack Query（服务端状态）：**
- 所有 API 数据：projects, tasks, comments, members, operationLogs
- 缓存、后台刷新、乐观更新、mutation 后自动失效

**Zustand（客户端状态）：**
- `authStore`：user, accessToken, status（`"loading" | "authenticated" | "unauthenticated"`）
- `uiStore`：仅存储跨页面的全局 UI 状态（侧边栏折叠）

**URL search params（project 作用域状态）：**
- 选中的任务 ID（`?task=<taskId>`，控制 TaskDetailDrawer 开关）
- 看板筛选条件（`?status=&assignee=&keyword=`），切换项目时自动重置

将 project-scoped 状态放在 URL 而非全局 store，确保切换到不同项目时状态自动重置，避免跨项目污染（如项目 A 的 selectedTaskId 在切换到项目 B 后仍然持有陈旧值）。

### 5.4 Query Key 设计

```
['currentUser']
['projects', { includeArchived, limit, offset }]
['projects', projectId]
['projects', projectId, 'members']
['projects', projectId, 'logs']
['tasks', { projectId, status, assigneeId, keyword }]
['tasks', taskId]
['tasks', taskId, 'comments']
['tasks', taskId, 'logs']
```

### 5.5 Mutation 后缓存失效规则

| 操作 | 失效的 Query Key |
|------|-----------------|
| 创建项目 | `['projects']` |
| 编辑/归档/转让项目 | `['projects']`, `['projects', id]` |
| 创建任务 | `['tasks', { projectId }]` |
| 编辑/删除任务 | `['tasks', { projectId }]`, `['tasks', taskId]` |
| 变更任务状态 | `['tasks', { projectId }]`, `['tasks', taskId]` |
| 指派任务 | `['tasks', { projectId }]`, `['tasks', taskId]` |
| 添加/移除成员 | `['projects', projectId, 'members']` |
| 创建/删除评论 | `['tasks', taskId, 'comments']` |
| 离开项目/转让 | `['projects']`, `['projects', projectId]`, `['projects', projectId, 'members']` |

## 6. 关键实现细节

### 6.1 统一响应解包（api/client.ts）

后端统一返回 `{ code, message, request_id, details?, data? }`。Axios 响应拦截器负责：
- `code === "OK"` → 返回 `response.data.data`（组件只拿到业务数据）
- `code !== "OK"` → 抛出 `AppError(code, message, request_id, details)`
- HTTP 401 → 清除 auth + 跳转 /login
- 网络错误 → `AppError("NETWORK_ERROR", "网络连接失败")`

```typescript
// types
interface ApiEnvelope<T = any> {
  code: string;
  message: string;
  request_id: string;
  details?: { field: string; reason: string }[];
  data?: T;
}

class AppError extends Error {
  constructor(
    public code: string,
    message: string,
    public requestId?: string,
    public details?: { field: string; reason: string }[],
  ) {
    super(message);
  }
}
```

### 6.2 错误码 → 中文提示（utils/error.ts）

| 错误码 | 用户提示 |
|--------|---------|
| `UNAUTHENTICATED` | 登录已过期，请重新登录 |
| `PERMISSION_DENIED` | 没有权限执行此操作 |
| `NOT_FOUND` | 资源不存在 |
| `FAILED_PRECONDITION` | 当前状态不允许此操作 |
| `INVALID_ARGUMENT` | 请求参数有误 |
| `ALREADY_EXISTS` | 资源已存在 |
| `ABORTED` | 数据已被他人修改，请刷新后重试 |
| `RESOURCE_EXHAUSTED` | 请求过于频繁，请稍后重试 |
| `INTERNAL` | 服务器内部错误，请稍后重试 |
| `UNAVAILABLE` | 服务暂不可用，请稍后重试 |
| `DEADLINE_EXCEEDED` | 请求超时，请稍后重试 |
| `NETWORK_ERROR` | 网络连接失败，请检查网络 |

所有 mutation 的 `onError` 回调使用 `message.error()` 弹出中文提示。

### 6.3 权限控制（hooks/useProjectPermission.ts / useTaskPermission.ts）

权限纯前端计算，仅用于 UI 展示（隐藏/禁用按钮），后端才是最终权威。

```typescript
// useProjectPermission(project, role, userId)
canEditProject       // 未归档 && owner
canAddMember         // 未归档 && (owner || admin)；owner 可加 admin/member，admin 只能加 member
canRemoveMember      // 未归档 && (owner || (admin && 目标是 member))
canChangeMemberRole  // 未归档 && owner
canTransfer          // 未归档 && owner
canArchive           // 未归档 && owner
canUnarchive         // 已归档 && owner
canLeave             // 未归档 && 非 owner

// useTaskPermission(task, role, userId)
canEditTask          // 未归档 && (owner || admin || 自己是创建者)
canDeleteTask        // 未归档 && (owner || admin || (自己是创建者 && 状态=todo))
canAssignTask        // 未归档 && (owner || admin || 自己是创建者)
canChangeStatus      // 未归档 && (owner || admin || 自己是创建者)；后端按 creator 判断，不按 assignee 判断
canComment           // 未归档 && 项目成员
```

### 6.4 任务状态流转（hooks/useStatusTransitions.ts）

```typescript
todo(0)      → [doing(1), done(2), cancelled(3)]
doing(1)     → [done(2), cancelled(3), todo(0)]
done(2)      → [doing(1)]
cancelled(3) → [todo(0)]
```

`TaskStatusSelect` 组件只展示合法目标状态。在 Kanban 中，只能拖拽到合法列。

### 6.5 乐观更新

以下操作实现乐观更新（TanStack Query `onMutate` + `onError` 回滚）：

- **状态变更**（看板拖拽或下拉）：立即更新缓存中的 status，失败则回滚
- **指派变更**：立即更新缓存中的 assignee_id，失败则回滚
- **项目归档/取消归档**：立即更新 status，失败则回滚

**注意：评论创建不做乐观更新**，原因：
1. 评论使用 `useInfiniteQuery` + `after_id` 分页，直接修改 `pages` 数组结构复杂易出错
2. 评论创建成功后执行 `invalidateQueries({ queryKey: ['tasks', taskId, 'comments'] })` 即可

**乐观更新对 useQuery 和 useInfiniteQuery 使用不同回滚策略**。任务列表使用 `useInfiniteQuery`，其缓存值类型为 `InfiniteData`，不能用示例代码直接 `setQueryData` 回滚。通用回滚模式：

```typescript
// 针对 useInfiniteQuery 的乐观更新
onMutate: async (vars) => {
  await queryClient.cancelQueries({ queryKey: ['tasks', { projectId }] });
  const previous = queryClient.getQueryData<InfiniteData<TaskPage>>(['tasks', { projectId }]);
  // 乐观修改 pages 数组中的对应数据...
  return { previous };
},
onError: (_err, _vars, context) => {
  // 回滚：直接写回完整的 InfiniteData 快照
  if (context?.previous) {
    queryClient.setQueryData(['tasks', { projectId }], context.previous);
  }
  message.error('操作失败');
},
onSettled: () => {
  queryClient.invalidateQueries({ queryKey: ['tasks', { projectId }] });
},
```

`getQueryData`（单数，取一个 query 的快照）比 `getQueriesData`（复数）更精确，避免跨 query key 的误操作。

### 6.6 乐观锁冲突处理

当 `PUT` 操作收到 `ABORTED`（version 冲突）时，`onError` 中：
1. 弹出警告："数据已被他人修改，正在刷新..."
2. 自动 invalidate 对应 query 以获取最新版本
3. 用户在刷新后重新编辑/提交

### 6.7 幂等性（Idempotency-Key）

不能简单在 Axios 请求拦截器里“每次请求自动生成新 UUID”。快速双击会生成两个不同 key，后端会当成两次独立写入，无法满足“只创建一个资源”。

推荐策略：
1. mutation 开始时在业务层生成 `Idempotency-Key`，并在同一次用户意图的重试/重复点击中复用。
2. 按按钮或表单提交过程禁用提交控件，直到 mutation settle。
3. 请求失败且确认后端未缓存成功响应时，下一次新的用户意图再生成新 key。
4. Axios 只负责透传调用方传入的 key；没有显式 key 时可生成兜底 key，但不能依赖它解决双击幂等。

### 6.8 分页策略

| 资源 | 后端分页方式 | 前端实现 |
|------|------------|---------|
| 项目列表 | offset-based | 后端暂不返回 `total`；使用上一页/下一页，或先改后端返回 total 后再用完整 Pagination |
| 任务列表 | cursor-based | `useInfiniteQuery`，query key 不包含 cursor，`pageParam` 传 cursor，`getNextPageParam` 提取 `next_cursor` |
| 评论列表 | after_id | `useInfiniteQuery`，`getNextPageParam` 用最后评论的 id |
| 操作日志 | cursor-based | `useInfiniteQuery`，提取 `next_cursor` |

看板按筛选条件拉取任务，`limit` 不要超过后端最大值 50。若 `next_cursor` 非空，列底部展示“加载更多”；不能用 `limit=200` 假设一次性拿全量，因为后端超过 50 会回退默认分页。

### 6.9 常量定义（utils/constants.ts）

使用数字字面量联合类型 + `Record` 确保类型安全，避免 `RoleLabels[999]` 等越界访问；不要使用 `const enum`，以兼容 Vite/esbuild 的 `isolatedModules`：

```typescript
export const Role = { Owner: 0, Admin: 1, Member: 2 } as const;
export type Role = (typeof Role)[keyof typeof Role];

export const TaskStatus = { Todo: 0, Doing: 1, Done: 2, Cancelled: 3 } as const;
export type TaskStatus = (typeof TaskStatus)[keyof typeof TaskStatus];

export const Priority = { Low: 0, Normal: 1, High: 2, Urgent: 3 } as const;
export type Priority = (typeof Priority)[keyof typeof Priority];

export const ProjectStatus = { Active: 0, Archived: 1 } as const;
export type ProjectStatus = (typeof ProjectStatus)[keyof typeof ProjectStatus];

// Record<联合类型, string> 确保覆盖所有取值，漏写会编译报错
export const RoleLabels: Record<Role, string> = {
  [Role.Owner]: '拥有者', [Role.Admin]: '管理员', [Role.Member]: '成员',
};
export const RoleColors: Record<Role, string> = {
  [Role.Owner]: 'red', [Role.Admin]: 'orange', [Role.Member]: 'blue',
};

export const TaskStatusLabels: Record<TaskStatus, string> = {
  [TaskStatus.Todo]: '待办', [TaskStatus.Doing]: '进行中',
  [TaskStatus.Done]: '已完成', [TaskStatus.Cancelled]: '已取消',
};
export const TaskStatusColors: Record<TaskStatus, string> = {
  [TaskStatus.Todo]: 'default', [TaskStatus.Doing]: 'processing',
  [TaskStatus.Done]: 'success', [TaskStatus.Cancelled]: 'warning',
};

export const PriorityLabels: Record<Priority, string> = {
  [Priority.Low]: '低', [Priority.Normal]: '普通', [Priority.High]: '高', [Priority.Urgent]: '紧急',
};
export const PriorityColors: Record<Priority, string> = {
  [Priority.Low]: 'default', [Priority.Normal]: 'blue', [Priority.High]: 'orange', [Priority.Urgent]: 'red',
};

export const ProjectStatusLabels: Record<ProjectStatus, string> = {
  [ProjectStatus.Active]: '活跃', [ProjectStatus.Archived]: '已归档',
};

// 操作日志 Action（字符串枚举，联合类型约束 key）
export const ActionLabel = {
  'task.create': '创建了任务', 'task.update': '更新了任务',
  'task.assign': '指派了任务', 'task.status_change': '变更了任务状态',
  'task.delete': '删除了任务', 'comment.create': '发表了评论',
  'comment.delete': '删除了评论', 'member.add': '添加了成员',
  'member.remove': '移除了成员', 'member.role_change': '修改了成员角色',
  'member.leave': '退出了项目', 'project.create': '创建了项目',
  'project.update': '更新了项目', 'project.archive': '归档了项目',
  'project.unarchive': '取消了归档', 'project.transfer_ownership': '转让了项目',
} as const;
export type ActionType = keyof typeof ActionLabel;
```

**注意：** 前端常量要与 `api/proto/task/v1/task.proto` 和后端 `internal/task/data/model.go` 中的数值保持一致。i18n 待评估——如后续需支持多语言，将 `xxxLabels` 替换为 `react-i18next` 的 `t()` 调用，同时在 `src/locales/zh-CN.ts` 中集中管理文案。当前阶段中文硬编码可接受。

## 7. Vite 配置

```typescript
// vite.config.ts
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { '@': path.resolve(__dirname, './src') },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    target: 'es2020',
    outDir: 'dist',
    sourcemap: false,
    rollupOptions: {
      output: {
        // 使用函数形式分包，按 node_modules 包名精确匹配
        manualChunks(id) {
          if (id.includes('node_modules/react-dom') || id.includes('node_modules/react/') || id.includes('node_modules/react-router')) return 'vendor';
          if (id.includes('node_modules/antd') || id.includes('node_modules/@ant-design/icons')) return 'antd';
          if (id.includes('node_modules/@tanstack/react-query')) return 'query';
        },
      },
    },
  },
});
```

`vite-env.d.ts` 需扩展环境变量类型：

```typescript
/// <reference types="vite/client" />
interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string;
}
interface ImportMeta {
  readonly env: ImportMetaEnv;
}
```

## 8. 依赖清单

```json
{
  "dependencies": {
    "react": "^19.2.6",
    "react-dom": "^19.2.6",
    "react-router-dom": "^6.30.0",
    "antd": "^5.22.0",
    "@ant-design/icons": "^5.5.0",
    "@tanstack/react-query": "^5.60.0",
    "zustand": "^5.0.0",
    "axios": "^1.7.0",
    "dayjs": "^1.11.0"
  },
  "devDependencies": {
    "@vitejs/plugin-react": "^5.2.0",
    "typescript": "~6.0.2",
    "vite": "^7.3.3",
    "vitest": "^3.0.0",
    "@testing-library/react": "^16.0.0",
    "@testing-library/jest-dom": "^6.6.0",
    "@testing-library/user-event": "^14.5.0",
    "jsdom": "^25.0.0",
    "msw": "^2.7.0",
    "@types/node": "^24.12.3",
    "@types/react": "^19.2.14",
    "@types/react-dom": "^19.2.3",
    "eslint": "^10.3.0",
    "@eslint/js": "^10.0.1",
    "typescript-eslint": "^8.59.2",
    "eslint-plugin-react-hooks": "^7.1.1",
    "eslint-plugin-react-refresh": "^0.5.2",
    "prettier": "^3.4.0",
    "eslint-config-prettier": "^9.0.0",
    "husky": "^9.0.0",
    "lint-staged": "^15.0.0",
    "@tanstack/react-query-devtools": "^5.60.0",
    "rollup-plugin-visualizer": "^5.12.0"
  }
}
```

依赖说明：
- **zustand** `^5.0.0` — v5 移除了默认导出，必须使用命名导入 `import { create } from 'zustand'`
- **msw** — Mock Service Worker，浏览器端开发 mock + Vitest 服务端 handler 复用
- **husky + lint-staged** — pre-commit 自动 `eslint --fix` + `prettier --write` + `tsc --noEmit`
- **jsdom** — Vitest DOM 环境
- **rollup-plugin-visualizer** — 构建后生成 `stats.html`，分析包体积
- **@tanstack/react-query-devtools** — 仅 dev 模式引入，调试 query cache

## 9. 实施阶段（10 个阶段，约 15 天）

### 阶段 1：项目脚手架 + 工程化基础
- 基于现有 `web/` 目录增量改造，不重新执行 `npm create vite`，避免覆盖当前 React 19 / TypeScript 6 脚手架
- 安装所有依赖（含 devDependencies）
- 配置 `vite.config.ts`（proxy、alias、manualChunks、build.target）
- 配置 `tsconfig.json`（strict、路径别名）
- 配置 `eslint.config.js`（flat config）+ `.prettierrc`
- 配置 `husky` + `lint-staged`（pre-commit：`eslint --fix` + `prettier --write` + `tsc --noEmit`）
- 配置 `vitest.config.ts`（jsdom 环境、setup 文件、路径别名）
- 创建 `src/main.tsx`、`src/App.tsx`、`src/vite-env.d.ts`
- 配置全局 CSS + Ant Design `ConfigProvider` 主题 + `dayjs` 中文 locale
- 实现 `api/client.ts`（Axios + 拦截器）
- 实现 `utils/error.ts`、`utils/constants.ts`、`utils/token.ts`、`utils/cursor.ts`
- 搭建 `src/mocks/` 基础结构（`browser.ts`、`server.ts`）
- 编写 `auth.handlers.ts`（register/login/logout/getMe mock）
- **验证**：`npm run dev` 启动，MSW 拦截 `/api/v1/users/me` 返回 mock 数据；`npm run lint` 零报错

### 阶段 2：鉴权体系 + 测试
- `stores/auth.store.ts`（Zustand，三态鉴权，`create` 命名导入）
- `api/auth.ts`
- `queries/auth.queries.ts`
- `AuthGuard.tsx`（三态渲染）、`GuestGuard.tsx`
- `LoginPage.tsx`、`RegisterPage.tsx`
- `AppLayout.tsx`、`AppHeader.tsx`、`AppSider.tsx`、`PageSkeleton.tsx`
- `mocks/handlers/auth.handlers.ts`（完善 mock）
- **测试**：`auth.store.test.ts`、`AuthGuard.test.tsx`、`LoginPage.test.tsx`、`RegisterPage.test.tsx`
- **验证**：注册 → 登录 → 登出 → 刷新保持登录态；过期 token 自动跳转

### 阶段 3：项目列表 + 测试
- `api/project.ts`
- `queries/project.queries.ts`
- `ProjectCard.tsx`、`ProjectCreateModal.tsx`、`ArchiveToggle.tsx`
- `ProjectListPage.tsx`
- `PriorityTag.tsx`、`LoadingSpinner.tsx`、`EmptyState.tsx`
- `mocks/handlers/project.handlers.ts`
- **测试**：`project.queries.test.ts`、`ProjectListPage.test.tsx`、`ProjectCreateModal.test.tsx`
- **验证**：查看项目卡片、创建项目、切换归档显示

### 阶段 4：看板 + 任务列表 + 测试
- `queries/task.queries.ts`
- `KanbanBoard.tsx`、`KanbanColumn.tsx`、`TaskCard.tsx`
- `TaskCreateModal.tsx`、`ProjectToolbar.tsx`
- `ProjectDetailPage.tsx`（看板 Tab）
- `useStatusTransitions.ts`、`useProjectPermission.ts`
- `mocks/handlers/task.handlers.ts`
- **测试**：`KanbanBoard.test.tsx`、`TaskCreateModal.test.tsx`、`useStatusTransitions.test.ts`
- **验证**：看板四列展示、创建任务、筛选任务、看板只展示合法状态转换

### 阶段 5：任务详情抽屉 + 测试
- `TaskDetailDrawer.tsx`
- `TaskEditForm.tsx`、`TaskStatusSelect.tsx`、`TaskAssignSelect.tsx`
- `useTaskPermission.ts`
- 乐观更新（状态变更、指派变更，useInfiniteQuery 专用回滚）
- 乐观锁冲突处理
- **测试**：`TaskDetailDrawer.test.tsx`、`useTaskPermission.test.ts`、乐观更新单元测试
- **验证**：点击任务打开抽屉、编辑、变更状态、指派；版本冲突提示并刷新

### 阶段 6：评论系统 + 测试
- `api/comment.ts`
- `queries/comment.queries.ts`
- `CommentList.tsx`、`CommentItem.tsx`、`CommentInput.tsx`
- 集成到 `TaskDetailDrawer.tsx`
- `mocks/handlers/comment.handlers.ts`
- **测试**：`CommentList.test.tsx`、`CommentInput.test.tsx`、after_id 分页测试
- **验证**：添加/删除评论、after_id 分页、权限隔离

### 阶段 7：项目设置 + 成员管理 + 操作日志 + 测试
- `queries/member.queries.ts`、`queries/operationLog.queries.ts`
- `MemberList.tsx`、`MemberAddModal.tsx`
- `ProjectEditModal.tsx`
- `OperationLogList.tsx`、`OperationLogItem.tsx`
- 设置 Tab：编辑项目、成员管理、归档/转让/退出、操作日志
- `mocks/handlers/member.handlers.ts`、`mocks/handlers/operationLog.handlers.ts`
- **测试**：`MemberList.test.tsx`、`ProjectEditModal.test.tsx`
- **验证**：完整的项目设置 CRUD、成员增删改、归档/转让/离开

### 阶段 8：REST DTO 收敛 + 可选 proto 基础类型生成
- 梳理所有 HTTP `data` 响应，集中到 `src/api/types.ts`，以 gateway 实际 JSON 为准
- 不强制所有 TypeScript 类型直接来自 `.proto`；proto message 描述内部 gRPC 合同，不完全等同 HTTP DTO
- 如需复用 proto 基础类型，可在 `buf.gen.yaml` 中增加 TypeScript 生成规则，输出到 `web/src/api/generated/`
- 为评论和操作日志定义 REST 扩展类型（含 `username` / `nickname` / `avatar_url` 等 gateway enrichment 字段）
- 将 `api/*.ts` 的入参/出参统一引用 REST DTO，减少散落类型
- **验证**：`tsc --noEmit` 零报错；评论、操作日志页面能正常访问补全后的用户展示字段

### 阶段 9：打磨与上线
- `ErrorBoundary.tsx`
- 加载骨架屏（Ant Design Skeleton）
- 空状态插图
- `React.lazy()` 路由级代码分割（已在阶段 1 的组件树中规划）
- `NotFoundPage.tsx`
- `rollup-plugin-visualizer` 生成 `stats.html`，检查包体积
- 边界情况测试：过期 token、归档项目、权限拒绝、幂等性
- README 启动说明（含 MSW 独立开发模式）

### 阶段 10：E2E 测试
- Playwright 或 Cypress 覆盖核心流程
- 场景：注册→登录→创建项目→添加成员→创建任务→看板拖拽→评论→归档
- 权限矩阵 E2E 验证（owner/admin/member 各角色操作权限）
- CI 集成（GitHub Actions，与 Go 后端 CI 统一流水线）

## 10. 验证清单

完成后逐项验证：

### 鉴权
- [ ] 注册新用户 → 自动登录跳转项目列表
- [ ] 登录 → 获取 token → 跳转项目列表
- [ ] 刷新页面 → 保持登录态
- [ ] 登出 → 清除 token → 跳转登录页
- [ ] Token 过期（等 2h 或手动删 Redis key）→ API 返回 401 → 自动跳转登录页

### 项目管理
- [ ] 创建项目 → 出现在列表中
- [ ] 编辑项目名称/描述 → 成功更新
- [ ] 归档项目 → 状态变为"已归档" → 看板内所有操作按钮禁用
- [ ] 取消归档 → 恢复可操作
- [ ] 转让项目 → owner 变更 → 原 owner 变为 admin

### 成员管理
- [ ] 添加成员（各角色）→ 成员列表更新
- [ ] 修改成员角色 → 权限随之变更
- [ ] 移除成员 → 成员从列表消失
- [ ] 非成员访问项目 → 404
- [ ] owner 不能退出项目（按钮不显示）
- [ ] admin/member 可以退出项目

### 任务管理
- [ ] 创建任务 → 出现在看板"待办"列
- [ ] 编辑任务 → 标题/内容/优先级/截止日期更新
- [ ] 删除任务 → 任务从看板消失
- [ ] member 只能编辑/删除自己的任务
- [ ] member 只能删除状态为 todo 的任务

### 状态流转
- [ ] todo → doing / done / cancelled（合法）
- [ ] doing → done / cancelled / todo（合法）
- [ ] done → doing（合法）
- [ ] cancelled → todo（合法）
- [ ] 非法转换（如 done → cancelled）→ 按钮不显示或后端拒绝
- [ ] 看板拖拽只允许拖到合法列

### 任务指派
- [ ] 指派任务给项目成员 → 成功
- [ ] 指派给非成员 → 后端拒绝
- [ ] member 只能指派自己创建的任务

### 评论
- [ ] 添加评论 → 立即显示在列表
- [ ] 删除自己的评论 → 成功
- [ ] owner/admin 删除他人评论 → 成功
- [ ] member 不能删除他人评论 → 删除按钮不显示
- [ ] after_id 分页加载更多 → 正常工作

### 操作日志
- [ ] 项目操作日志 → 显示所有成员操作记录
- [ ] 任务操作日志 → 显示该任务相关操作
- [ ] cursor 分页加载更多 → 正常工作
- [ ] 日志中显示操作者用户名/昵称/头像

### 权限矩阵
- [ ] 归档项目 → 所有写操作按钮禁用
- [ ] 非成员 → 项目/任务资源不可见
- [ ] owner → 所有操作可用
- [ ] admin → 不能编辑项目/归档/转让/修改成员角色
- [ ] member → 只能操作自己的任务

### 错误处理
- [ ] 网络断开 → 提示"网络连接失败"
- [ ] 参数错误 → 提示具体错误信息
- [ ] 版本冲突 → 提示"数据已被他人修改"并刷新

### 幂等性
- [ ] 快速双击创建按钮 → 只创建一个资源

### 工程化
- [ ] `npm run lint` → 零报错
- [ ] `npm run typecheck` (`tsc --noEmit`) → 零报错
- [ ] `npm run test` → 全部通过，覆盖率 ≥ 80%
- [ ] `npm run build` → 成功构建，`dist/` 产出正确
- [ ] 包体积 `stats.html` → 无异常大的 chunk
- [x] E2E 核心流程通过

## 11. 后端接口对照表

TypeScript 类型以 HTTP REST DTO 为准，集中维护在 `src/api/types.ts`。可选地通过 `buf generate` 从 `api/proto/` 生成 proto 基础类型到 `web/src/api/generated/`，但不能把生成的 proto 类型直接等同于 HTTP 响应类型：gateway 会对评论和操作日志补充用户展示字段。

### 完整接口清单（28 个）

| # | 方法 | 路径 | 说明 | 分页 |
|---|------|------|------|------|
| 1 | POST | `/api/v1/auth/register` | 注册 | - |
| 2 | POST | `/api/v1/auth/login` | 登录 | - |
| 3 | POST | `/api/v1/auth/logout` | 登出 | - |
| 4 | GET | `/api/v1/users/me` | 当前用户 | - |
| 5 | POST | `/api/v1/projects` | 创建项目 | - |
| 6 | GET | `/api/v1/projects` | 项目列表 | offset |
| 7 | GET | `/api/v1/projects/:id` | 项目详情 | - |
| 8 | PUT | `/api/v1/projects/:id` | 编辑项目 | - |
| 9 | POST | `/api/v1/projects/:id/archive` | 归档项目 | - |
| 10 | POST | `/api/v1/projects/:id/unarchive` | 取消归档 | - |
| 11 | POST | `/api/v1/projects/:id/transfer` | 转让项目 | - |
| 12 | POST | `/api/v1/projects/:id/members` | 添加成员 | - |
| 13 | GET | `/api/v1/projects/:id/members` | 成员列表 | - |
| 14 | PUT | `/api/v1/projects/:id/members/:userId` | 修改成员角色 | - |
| 15 | DELETE | `/api/v1/projects/:id/members/:userId` | 移除成员 | - |
| 16 | POST | `/api/v1/projects/:id/members/me/leave` | 退出项目 | - |
| 17 | GET | `/api/v1/projects/:id/operation-logs` | 项目操作日志 | cursor |
| 18 | POST | `/api/v1/tasks` | 创建任务 | - |
| 19 | GET | `/api/v1/tasks` | 任务列表 | cursor |
| 20 | GET | `/api/v1/tasks/:id` | 任务详情 | - |
| 21 | PUT | `/api/v1/tasks/:id` | 编辑任务 | - |
| 22 | DELETE | `/api/v1/tasks/:id` | 删除任务 | - |
| 23 | POST | `/api/v1/tasks/:id/assign` | 指派任务 | - |
| 24 | POST | `/api/v1/tasks/:id/status` | 变更状态 | - |
| 25 | POST | `/api/v1/tasks/:id/comments` | 添加评论 | - |
| 26 | GET | `/api/v1/tasks/:id/comments` | 评论列表 | after_id |
| 27 | DELETE | `/api/v1/tasks/:id/comments/:commentId` | 删除评论 | - |
| 28 | GET | `/api/v1/tasks/:id/operation-logs` | 任务操作日志 | cursor |

## 12. 风险与对策

| 风险 | 对策 |
|------|------|
| localStorage XSS 泄露 token | 依赖后端 CSP 头缓解；access token 仅 2h TTL 限制窗口期；如需 refresh token 则必须改用 httpOnly cookie |
| Ant Design 5 CSS-in-JS 与后端 CSP 冲突 | 后端当前 CSP 未声明 `style-src`；若由 gateway 托管生产前端，需选择 AntD `StyleProvider` nonce 方案或调整 CSP，否则样式可能被浏览器拦截 |
| REST DTO 与 proto 类型漂移 | 阶段 8 集中维护 REST DTO；可选生成 proto 基础类型，但评论/日志等 enrichment 响应必须显式建模 |
| 后端 envelope 格式与预期不符 | Axios 拦截器显式检查 `code === "OK"`，异常格式抛出 `AppError` |
| 乐观锁冲突频繁 | 明确提示"数据已被他人修改"，自动刷新并保留表单内容 |
| 看板任务量超过单页上限 | 首次实现按后端最大 `limit=50` 拉取；`next_cursor` 非空时展示“加载更多”；后续用 `react-window` 虚拟滚动 + 游标加载更多 |
| Token 过期中断 | Axios 拦截器捕获 401，自动跳转登录页并提示 |
| 拖拽库学习成本 | 阶段 4 看板先用手动状态变更按钮，稳定后再加拖拽（可选） |
| Ant Design 5 主题覆盖 | 使用 `ConfigProvider` + CSS 变量，不在组件级覆盖样式 |
| 页面刷新 AuthGuard 竞态导致误踢 | authStore 三态设计（loading/authenticated/unauthenticated），loading 阶段渲染骨架屏 |

## 13. 部署方案

**开发环境：**
```bash
cd web && npm run dev        # Vite :5173，proxy /api → :8080
make run/api-gateway          # Go 后端 :8080
```

**生产构建：**
```bash
cd web && npm run build       # → web/dist/
```

当前后端尚未实现 `web/dist/` 静态托管和 SPA fallback。生产部署有两个可选方案：

1. 推荐先用 Nginx/Caddy/对象存储托管 `web/dist/`，并将 `/api` 反向代理到 `api-gateway`。
2. 若坚持由 `api-gateway` 单进程托管，需要新增静态文件路由和 `NoRoute` fallback，并确保 `/assets`、`/favicon`、`index.html` 不经过 JWT Auth；同时处理 CSP 与 Ant Design 样式注入的兼容性。

不能只执行 `npm run build` 就假设 gateway 会自动托管前端。

**Makefile 补充：**
```makefile
web-install:
	cd web && npm ci

web-dev:
	cd web && npm run dev

web-build:
	cd web && npm run build

web-typecheck:
	cd web && npx tsc --noEmit

web-lint:
	cd web && npx eslint src/ && npx prettier --check src/

web-test:
	cd web && npx vitest run

web-test-watch:
	cd web && npx vitest

web-coverage:
	cd web && npx vitest run --coverage
```
