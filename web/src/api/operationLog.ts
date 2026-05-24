import { request } from './client'
import type { ListOperationLogsRequest, ListOperationLogsData } from './types'

export function listProjectOperationLogs(projectId: string, params: ListOperationLogsRequest) {
  return request<ListOperationLogsData>({
    url: `/projects/${projectId}/operation-logs`,
    method: 'GET',
    params,
  })
}

export function listTaskOperationLogs(taskId: string, params: ListOperationLogsRequest) {
  return request<ListOperationLogsData>({
    url: `/tasks/${taskId}/operation-logs`,
    method: 'GET',
    params,
  })
}
