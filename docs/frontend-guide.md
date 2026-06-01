# 前端指南

前端位于 `web/`，使用 React、TypeScript、Vite、Ant Design、React Query、Zustand、MSW、Vitest 和 Playwright。

## 技术栈

| 类别 | 技术 |
|------|------|
| Framework | React 19 |
| Language | TypeScript 6 |
| Build | Vite 7 |
| UI | Ant Design 5 |
| Data fetching | TanStack React Query 5 |
| State | Zustand 5 |
| HTTP | Axios |
| Mock | MSW 2 |
| Unit test | Vitest + Testing Library |
| E2E | Playwright |

## 环境准备

```bash
cd web
npm install
cp .env.example .env.local
```

配置：

```text
VITE_API_BASE_URL=/api/v1
VITE_ENABLE_MSW=false
```

## 开发模式

连接真实 Go gateway：

```bash
npm run dev
```

Vite 会将 `/api` 代理到后端 gateway。需要先启动 PostgreSQL、Redis、三个 Go 服务。

独立 mock 模式：

```bash
npm run dev:mock
```

该模式启用 MSW，浏览器 service worker 会拦截 `/api/v1/*` 请求，不需要后端服务。

## 目录结构

```text
web/src/api/             # Axios client 和接口函数
web/src/queries/         # React Query hooks
web/src/stores/          # Zustand store
web/src/pages/           # 页面
web/src/components/      # 业务组件和通用组件
web/src/hooks/           # 权限和状态机 hooks
web/src/mocks/           # MSW handlers 和 fixtures
web/src/utils/           # token、time、cursor、error 等工具
web/__tests__/           # 单元和组件测试
web/e2e/                 # Playwright E2E
```

## 主要页面

| 页面 | 说明 |
|------|------|
| `LoginPage` | 登录 |
| `RegisterPage` | 注册 |
| `ProjectListPage` | 项目列表、创建项目 |
| `ProjectDetailPage` | 项目详情、看板、成员、操作日志、设置 |
| `NotFoundPage` | 404 |

## API 对接

- API base URL 默认 `/api/v1`。
- token 存储由 `auth.store.ts` 和 token 工具负责。
- 业务请求封装在 `src/api/*.ts`。
- 服务端统一错误响应在 `src/utils/error.ts` 中转换为前端可展示错误。
- React Query hooks 位于 `src/queries/*.ts`，负责缓存、失效和乐观更新。

## Mock 数据

MSW handlers 位于：

```text
web/src/mocks/handlers/
```

fixtures 位于：

```text
web/src/mocks/fixtures/
```

新增后端接口时，前端应同步更新：

1. `src/api/types.ts`
2. 对应 `src/api/*.ts`
3. 对应 `src/queries/*.ts`
4. MSW handler 和 fixtures
5. 组件或页面测试

## 质量命令

```bash
npm run typecheck
npm run lint
npm run test
npm run build
```

覆盖率：

```bash
npm run coverage
```

格式化：

```bash
npm run format
npm run format:check
```

## E2E

```bash
npx playwright install chromium
npm run e2e
```

Playwright 配置会启动 Vite mock 模式，因此 E2E 不依赖后端服务。测试覆盖注册、登录、项目创建、成员管理、任务创建、看板拖拽、评论、归档和权限矩阵。

## 构建与包体分析

生产构建：

```bash
npm run build
```

生成 Rollup Visualizer：

```bash
npm run build:stats
```

输出 `web/stats.html`，用于检查 `antd`、`vendor`、`query` 和路由 chunk。

## 对接注意事项

- 写接口建议传 `Idempotency-Key`，尤其是创建项目、创建任务、评论和成员变更。
- 更新项目或任务必须携带最新 `version`，否则后端可能返回 `ABORTED`。
- 任务状态和指派人必须使用独立接口，不要放到普通任务更新请求里。
- 项目归档后所有写操作都会失败，前端需要在按钮和表单层禁用相关操作。
- 非成员访问可能返回 `NOT_FOUND`，不要把它简单解释为资源被删除。
