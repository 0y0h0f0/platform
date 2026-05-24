import { http, HttpResponse } from 'msw'

import type {
  ApiEnvelope,
  CreateCommentRequest,
  CreateCommentData,
  DeleteCommentData,
  ListCommentsData,
  TaskComment,
} from '@/api/types'
import { createMockComments } from '../fixtures/comments'
import { createMockTasks } from '../fixtures/tasks'

let comments = createMockComments()
let nextCommentNumber = 100

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

function taskExists(taskId: string) {
  return createMockTasks().some((task) => task.id === taskId)
}

export function resetMockComments() {
  comments = createMockComments()
  nextCommentNumber = 100
}

export const commentHandlers = [
  http.get('*/api/v1/tasks/:taskId/comments', ({ params, request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const taskId = String(params.taskId)
    if (!taskExists(taskId)) {
      return error('NOT_FOUND', 'task not found', 404)
    }

    const url = new URL(request.url)
    const limit = Number(url.searchParams.get('limit') ?? 20)
    const afterId = url.searchParams.get('after_id') || ''
    const taskComments = comments.filter((comment) => comment.task_id === taskId)
    const afterIndex = afterId ? taskComments.findIndex((comment) => comment.id === afterId) : -1

    if (afterId && afterIndex === -1) {
      return error('NOT_FOUND', 'after_id comment not found', 404)
    }

    return ok<ListCommentsData>({
      comments: taskComments.slice(afterIndex + 1, afterIndex + 1 + limit),
    })
  }),

  http.post('*/api/v1/tasks/:taskId/comments', async ({ params, request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const taskId = String(params.taskId)
    if (!taskExists(taskId)) {
      return error('NOT_FOUND', 'task not found', 404)
    }

    const payload = (await request.json()) as Partial<CreateCommentRequest>
    const content = payload.content?.trim()
    if (!content) {
      return error('INVALID_ARGUMENT', 'comment content is required', 400)
    }

    const comment: TaskComment = {
      id: `comment-${nextCommentNumber++}`,
      task_id: taskId,
      user_id: 'user-1',
      username: 'owner',
      nickname: '项目负责人',
      avatar_url: '',
      content,
    }

    comments = [...comments, comment]

    return ok<CreateCommentData>({ comment }, 201)
  }),

  http.delete('*/api/v1/tasks/:taskId/comments/:commentId', ({ params, request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const taskId = String(params.taskId)
    const commentId = String(params.commentId)
    const comment = comments.find((item) => item.id === commentId && item.task_id === taskId)

    if (!comment) {
      return error('NOT_FOUND', 'comment not found', 404)
    }

    comments = comments.filter((item) => item.id !== commentId)

    return ok<DeleteCommentData>({ comment })
  }),
]
