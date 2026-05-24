import { useInfiniteQuery } from '@tanstack/react-query'

import * as operationLogApi from '@/api/operationLog'

export interface OperationLogListParams {
  limit: number
  projectId: string
}

export const operationLogQueryKeys = {
  all: ['operation-logs'] as const,
  project: (projectId: string) => ['operation-logs', 'projects', projectId] as const,
}

export function useProjectOperationLogsQuery(params: OperationLogListParams) {
  return useInfiniteQuery({
    queryKey: operationLogQueryKeys.project(params.projectId),
    queryFn: ({ pageParam }) =>
      operationLogApi.listProjectOperationLogs(params.projectId, {
        cursor: pageParam || undefined,
        limit: params.limit,
      }),
    enabled: Boolean(params.projectId),
    getNextPageParam: (lastPage) => lastPage.next_cursor || undefined,
    initialPageParam: '',
  })
}
