# Task Platform Web

React + TypeScript + Vite frontend for the team task collaboration platform.

## Prerequisites

- Node.js 22+
- npm 10+
- Optional backend gateway at `http://localhost:8080` when not using MSW

## Install

```bash
npm install
cp .env.example .env.local
```

## Development

Run against the real Go API gateway through the Vite `/api` proxy:

```bash
npm run dev
```

Run in standalone mock mode with MSW, no backend services required:

```bash
npm run dev:mock
```

The app uses `VITE_API_BASE_URL=/api/v1` by default. Set `VITE_ENABLE_MSW=true` to make the browser service worker intercept `/api/v1/*` requests with the local handlers in `src/mocks/`.

## Quality Checks

```bash
npm run typecheck
npm run lint
npm run test
npm run build
```

## E2E Tests

Playwright runs against the Vite dev server in MSW mode, so backend services are not required for browser E2E coverage.

```bash
npx playwright install chromium
npm run e2e
```

The suite covers register, login, project creation, member management, task creation, kanban drag-and-drop, comments, archive behavior, and owner/admin/member permission matrix checks.

## Bundle Stats

Generate `stats.html` with Rollup Visualizer:

```bash
npm run build:stats
```

Open `stats.html` in a browser and inspect the `antd`, `vendor`, `query`, and route chunks before release.
