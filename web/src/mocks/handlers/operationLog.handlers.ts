import { http, HttpResponse } from 'msw'

import type { ApiEnvelope, ListOperationLogsData } from '@/api/types'
import { projectMockState } from './project.state'

function ok<T>(data: T, status = 200) {
  return HttpResponse.json<ApiEnvelope<T>>(
    {
      code: 'OK',
      message: 'ok',
      request_id: 'mock-request-id',
      data,
    },
    { status },
  )
}

function error(code: string, message: string, status: number) {
  return HttpResponse.json<ApiEnvelope>(
    {
      code,
      message,
      request_id: 'mock-request-id',
    },
    { status },
  )
}

function isAuthorized(request: Request) {
  return Boolean(request.headers.get('authorization'))
}

function paginateLogs(request: Request, logs: ListOperationLogsData['logs']) {
  const url = new URL(request.url)
  const limit = Number(url.searchParams.get('limit') ?? 20)
  const cursor = Number(url.searchParams.get('cursor') || 0)
  const pageLogs = logs.slice(cursor, cursor + limit)
  const nextCursor = cursor + limit < logs.length ? String(cursor + limit) : ''

  return ok<ListOperationLogsData>({
    logs: pageLogs,
    next_cursor: nextCursor,
  })
}

export const operationLogHandlers = [
  http.get('*/api/v1/projects/:projectId/operation-logs', ({ params, request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const projectId = String(params.projectId)
    const logs = projectMockState.operationLogs.filter((log) => log.project_id === projectId)
    return paginateLogs(request, logs)
  }),

  http.get('*/api/v1/tasks/:taskId/operation-logs', ({ params, request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const taskId = String(params.taskId)
    const logs = projectMockState.operationLogs.filter((log) => log.task_id === taskId)
    return paginateLogs(request, logs)
  }),
]
