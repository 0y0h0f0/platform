import { request } from './client'
import type {
  AssignTaskRequest,
  ChangeTaskStatusRequest,
  CreateTaskRequest,
  CreateTaskData,
  GetTaskData,
  ListTasksRequest,
  ListTasksData,
  TaskData,
  UpdateTaskRequest,
} from './types'

export function createTask(payload: CreateTaskRequest, idempotencyKey?: string) {
  return request<CreateTaskData>({
    url: '/tasks',
    method: 'POST',
    data: payload,
    idempotencyKey,
  })
}

export function listTasks(params: ListTasksRequest) {
  return request<ListTasksData>({
    url: '/tasks',
    method: 'GET',
    params,
  })
}

export function getTask(taskId: string) {
  return request<GetTaskData>({
    url: `/tasks/${taskId}`,
    method: 'GET',
  })
}

export function updateTask(taskId: string, payload: UpdateTaskRequest, idempotencyKey?: string) {
  return request<TaskData>({
    url: `/tasks/${taskId}`,
    method: 'PUT',
    data: payload,
    idempotencyKey,
  })
}

export function deleteTask(taskId: string, idempotencyKey?: string) {
  return request<TaskData>({
    url: `/tasks/${taskId}`,
    method: 'DELETE',
    idempotencyKey,
  })
}

export function assignTask(taskId: string, payload: AssignTaskRequest, idempotencyKey?: string) {
  return request<TaskData>({
    url: `/tasks/${taskId}/assign`,
    method: 'POST',
    data: payload,
    idempotencyKey,
  })
}

export function changeTaskStatus(
  taskId: string,
  payload: ChangeTaskStatusRequest,
  idempotencyKey?: string,
) {
  return request<TaskData>({
    url: `/tasks/${taskId}/status`,
    method: 'POST',
    data: payload,
    idempotencyKey,
  })
}
