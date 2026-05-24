import { http, HttpResponse } from 'msw'

import type {
  ApiEnvelope,
  AssignTaskRequest,
  ChangeTaskStatusRequest,
  CreateTaskRequest,
  CreateTaskData,
  ListTasksData,
  Task,
  TaskData,
  UpdateTaskRequest,
} from '@/api/types'
import { Priority, TaskStatus, type TaskStatus as TaskStatusValue } from '@/utils/constants'
import { createMockTasks } from '../fixtures/tasks'

let tasks = createMockTasks()
let nextTaskNumber = 100

const validTransitions: Record<TaskStatusValue, TaskStatusValue[]> = {
  [TaskStatus.Todo]: [TaskStatus.Doing, TaskStatus.Done, TaskStatus.Cancelled],
  [TaskStatus.Doing]: [TaskStatus.Done, TaskStatus.Cancelled, TaskStatus.Todo],
  [TaskStatus.Done]: [TaskStatus.Doing],
  [TaskStatus.Cancelled]: [TaskStatus.Todo],
}

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

function findTask(taskId: string) {
  return tasks.find((task) => task.id === taskId)
}

function hasVersionConflict(task: Task, version: number | undefined) {
  return version === undefined || version !== task.version
}

export function resetMockTasks() {
  tasks = createMockTasks()
  nextTaskNumber = 100
}

export const taskHandlers = [
  http.get('*/api/v1/tasks', ({ request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const url = new URL(request.url)
    const projectId = url.searchParams.get('project_id')
    if (!projectId) {
      return error('INVALID_ARGUMENT', 'project_id is required', 400)
    }

    const limit = Number(url.searchParams.get('limit') ?? 20)
    const cursor = Number(url.searchParams.get('cursor') || 0)
    const statusParam = url.searchParams.get('status')
    const assigneeId = url.searchParams.get('assignee_id') || ''
    const keyword = (url.searchParams.get('keyword') || '').trim().toLowerCase()

    let visibleTasks = tasks.filter((task) => task.project_id === projectId)

    if (statusParam !== null && statusParam !== '') {
      visibleTasks = visibleTasks.filter((task) => task.status === Number(statusParam))
    }

    if (assigneeId) {
      visibleTasks = visibleTasks.filter((task) => task.assignee_id === assigneeId)
    }

    if (keyword) {
      visibleTasks = visibleTasks.filter(
        (task) =>
          task.title.toLowerCase().includes(keyword) ||
          task.content.toLowerCase().includes(keyword),
      )
    }

    const pageTasks = visibleTasks.slice(cursor, cursor + limit)
    const nextCursor = cursor + limit < visibleTasks.length ? String(cursor + limit) : ''

    return ok<ListTasksData>({
      next_cursor: nextCursor,
      tasks: pageTasks,
    })
  }),

  http.post('*/api/v1/tasks', async ({ request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const payload = (await request.json()) as Partial<CreateTaskRequest>
    const title = payload.title?.trim()

    if (!payload.project_id || !title) {
      return error('INVALID_ARGUMENT', 'project_id and title are required', 400)
    }

    const task: Task = {
      id: `task-${nextTaskNumber++}`,
      project_id: payload.project_id,
      title,
      content: payload.content?.trim() ?? '',
      status: TaskStatus.Todo,
      priority: Priority.Normal,
      assignee_id: '',
      creator_id: 'user-1',
      due_time: '',
      version: 1,
    }

    tasks = [task, ...tasks]

    return ok<CreateTaskData>({ task }, 201)
  }),

  http.get('*/api/v1/tasks/:taskId', ({ params, request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const task = findTask(String(params.taskId))
    if (!task) {
      return error('NOT_FOUND', 'task not found', 404)
    }

    return ok<TaskData>({ task })
  }),

  http.put('*/api/v1/tasks/:taskId', async ({ params, request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const task = findTask(String(params.taskId))
    if (!task) {
      return error('NOT_FOUND', 'task not found', 404)
    }

    const payload = (await request.json()) as Partial<UpdateTaskRequest>
    if (hasVersionConflict(task, payload.version)) {
      return error('ABORTED', 'version conflict', 409)
    }

    const title = payload.title?.trim()
    if (!title) {
      return error('INVALID_ARGUMENT', 'title is required', 400)
    }

    task.title = title
    task.content = payload.content?.trim() ?? ''
    task.priority = payload.priority ?? task.priority
    task.due_time = payload.due_time ?? ''
    task.version += 1

    return ok<TaskData>({ task })
  }),

  http.post('*/api/v1/tasks/:taskId/assign', async ({ params, request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const task = findTask(String(params.taskId))
    if (!task) {
      return error('NOT_FOUND', 'task not found', 404)
    }

    const payload = (await request.json()) as Partial<AssignTaskRequest>
    if (!payload.assignee_id) {
      return error('INVALID_ARGUMENT', 'assignee_id is required', 400)
    }

    task.assignee_id = payload.assignee_id
    task.version += 1

    return ok<TaskData>({ task })
  }),

  http.post('*/api/v1/tasks/:taskId/status', async ({ params, request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const task = findTask(String(params.taskId))
    if (!task) {
      return error('NOT_FOUND', 'task not found', 404)
    }

    const payload = (await request.json()) as Partial<ChangeTaskStatusRequest>
    if (payload.status === undefined) {
      return error('INVALID_ARGUMENT', 'status is required', 400)
    }

    if (hasVersionConflict(task, payload.version)) {
      return error('ABORTED', 'version conflict', 409)
    }

    if (task.status === payload.status) {
      return error('FAILED_PRECONDITION', 'task is already in this status', 400)
    }

    if (!validTransitions[task.status].includes(payload.status)) {
      return error('FAILED_PRECONDITION', 'invalid status transition', 400)
    }

    task.status = payload.status
    task.version += 1

    return ok<TaskData>({ task })
  }),
]
