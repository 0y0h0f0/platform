import {
  type InfiniteData,
  type QueryClient,
  type QueryKey,
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'

import * as taskApi from '@/api/task'
import type {
  CreateTaskRequest,
  ListTasksRequest,
  ListTasksData,
  Task,
  TaskData,
  UpdateTaskRequest,
} from '@/api/types'
import type { TaskStatus } from '@/utils/constants'

export interface TaskListParams {
  assigneeId?: string
  keyword?: string
  limit: number
  projectId: string
  status?: TaskStatus
}

export interface TaskListCacheFilters {
  assigneeId: string
  keyword: string
  projectId: string
  status: TaskStatus | null
}

interface ChangeTaskStatusVariables {
  status: TaskStatus
  task: Task
}

interface AssignTaskVariables {
  assigneeId: string
  task: Task
}

interface UpdateTaskVariables {
  task: Task
  values: Omit<UpdateTaskRequest, 'version'>
}

type TaskListQueryKey = readonly ['tasks', TaskListCacheFilters]
type TaskListSnapshot = Array<[TaskListQueryKey, InfiniteData<ListTasksData> | undefined]>

export const taskQueryKeys = {
  all: ['tasks'] as const,
  list: (params: TaskListParams) =>
    [
      'tasks',
      {
        assigneeId: params.assigneeId ?? '',
        keyword: params.keyword ?? '',
        projectId: params.projectId,
        status: params.status ?? null,
      },
    ] as const,
  detail: (taskId: string) => ['tasks', taskId] as const,
}

function createIdempotencyKey() {
  return window.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function toListRequest(params: TaskListParams, cursor: string): ListTasksRequest {
  return {
    assignee_id: params.assigneeId || undefined,
    cursor: cursor || undefined,
    keyword: params.keyword || undefined,
    limit: params.limit,
    project_id: params.projectId,
    status: params.status,
  }
}

function isTaskListQueryKey(queryKey: QueryKey): queryKey is TaskListQueryKey {
  if (queryKey.length !== 2 || queryKey[0] !== 'tasks') {
    return false
  }

  const filters = queryKey[1]
  return (
    typeof filters === 'object' &&
    filters !== null &&
    'projectId' in filters &&
    'assigneeId' in filters &&
    'keyword' in filters &&
    'status' in filters
  )
}

function taskMatchesFilters(task: Task, filters: TaskListCacheFilters) {
  if (task.project_id !== filters.projectId) {
    return false
  }

  if (filters.status !== null && task.status !== filters.status) {
    return false
  }

  if (filters.assigneeId && task.assignee_id !== filters.assigneeId) {
    return false
  }

  const keyword = filters.keyword.trim().toLowerCase()
  if (!keyword) {
    return true
  }

  return task.title.toLowerCase().includes(keyword) || task.content.toLowerCase().includes(keyword)
}

export function upsertTaskInInfiniteData(
  data: InfiniteData<ListTasksData> | undefined,
  task: Task,
  filters: TaskListCacheFilters,
): InfiniteData<ListTasksData> | undefined {
  if (!data) {
    return data
  }

  const shouldKeepTask = taskMatchesFilters(task, filters)
  let foundTask = false
  const pages = data.pages.map((page) => {
    const tasks = page.tasks.reduce<Task[]>((acc, currentTask) => {
      if (currentTask.id !== task.id) {
        acc.push(currentTask)
        return acc
      }

      foundTask = true
      if (shouldKeepTask) {
        acc.push(task)
      }
      return acc
    }, [])

    return { ...page, tasks }
  })

  if (!foundTask && shouldKeepTask && pages[0]) {
    pages[0] = {
      ...pages[0],
      tasks: [task, ...pages[0].tasks],
    }
  }

  return { ...data, pages }
}

function snapshotTaskListQueries(queryClient: QueryClient): TaskListSnapshot {
  return queryClient
    .getQueriesData<InfiniteData<ListTasksData>>({
      predicate: (query) => isTaskListQueryKey(query.queryKey),
      queryKey: taskQueryKeys.all,
    })
    .filter((entry): entry is [TaskListQueryKey, InfiniteData<ListTasksData> | undefined] =>
      isTaskListQueryKey(entry[0]),
    )
}

function restoreTaskListQueries(queryClient: QueryClient, snapshot: TaskListSnapshot) {
  snapshot.forEach(([queryKey, data]) => {
    queryClient.setQueryData(queryKey, data)
  })
}

function writeTaskToListQueries(queryClient: QueryClient, task: Task) {
  snapshotTaskListQueries(queryClient).forEach(([queryKey]) => {
    queryClient.setQueryData<InfiniteData<ListTasksData>>(queryKey, (current) =>
      upsertTaskInInfiniteData(current, task, queryKey[1]),
    )
  })
}

function writeTaskToDetailQuery(queryClient: QueryClient, task: Task) {
  queryClient.setQueryData<TaskData>(taskQueryKeys.detail(task.id), { task })
}

function restoreOptimisticSnapshot(
  queryClient: QueryClient,
  snapshot:
    | {
        detail?: TaskData
        lists: TaskListSnapshot
      }
    | undefined,
  taskId: string,
) {
  if (!snapshot) {
    return
  }

  restoreTaskListQueries(queryClient, snapshot.lists)
  queryClient.setQueryData(taskQueryKeys.detail(taskId), snapshot.detail)
}

async function applyOptimisticTask(
  queryClient: QueryClient,
  task: Task,
): Promise<{ detail?: TaskData; lists: TaskListSnapshot }> {
  await queryClient.cancelQueries({ queryKey: taskQueryKeys.all })

  const snapshot = {
    detail: queryClient.getQueryData<TaskData>(taskQueryKeys.detail(task.id)),
    lists: snapshotTaskListQueries(queryClient),
  }

  writeTaskToDetailQuery(queryClient, task)
  writeTaskToListQueries(queryClient, task)

  return snapshot
}

function invalidateTaskQueries(queryClient: QueryClient, taskId: string) {
  queryClient.invalidateQueries({ queryKey: taskQueryKeys.detail(taskId) })
  queryClient.invalidateQueries({ queryKey: taskQueryKeys.all })
}

export function useTasksQuery(params: TaskListParams) {
  return useInfiniteQuery({
    queryKey: taskQueryKeys.list(params),
    queryFn: ({ pageParam }) => taskApi.listTasks(toListRequest(params, pageParam as string)),
    enabled: Boolean(params.projectId),
    getNextPageParam: (lastPage) => lastPage.next_cursor || undefined,
    initialPageParam: '',
  })
}

export function useTaskQuery(taskId: string) {
  return useQuery({
    queryKey: taskQueryKeys.detail(taskId),
    queryFn: () => taskApi.getTask(taskId),
    enabled: Boolean(taskId),
  })
}

export function useCreateTaskMutation(projectId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: Omit<CreateTaskRequest, 'project_id'>) =>
      taskApi.createTask({ ...payload, project_id: projectId }, createIdempotencyKey()),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: taskQueryKeys.all })
    },
  })
}

export function useUpdateTaskMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ task, values }: UpdateTaskVariables) =>
      taskApi.updateTask(
        task.id,
        {
          ...values,
          version: task.version,
        },
        createIdempotencyKey(),
      ),
    onMutate: ({ task, values }) =>
      applyOptimisticTask(queryClient, {
        ...task,
        ...values,
        version: task.version + 1,
      }),
    onError: (_error, variables, snapshot) => {
      restoreOptimisticSnapshot(queryClient, snapshot, variables.task.id)
    },
    onSuccess: (data) => {
      writeTaskToDetailQuery(queryClient, data.task)
      writeTaskToListQueries(queryClient, data.task)
    },
    onSettled: (_data, _error, variables) => {
      invalidateTaskQueries(queryClient, variables.task.id)
    },
  })
}

export function useAssignTaskMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ assigneeId, task }: AssignTaskVariables) =>
      taskApi.assignTask(task.id, { assignee_id: assigneeId }, createIdempotencyKey()),
    onMutate: ({ assigneeId, task }) =>
      applyOptimisticTask(queryClient, {
        ...task,
        assignee_id: assigneeId,
        version: task.version + 1,
      }),
    onError: (_error, variables, snapshot) => {
      restoreOptimisticSnapshot(queryClient, snapshot, variables.task.id)
    },
    onSuccess: (data) => {
      writeTaskToDetailQuery(queryClient, data.task)
      writeTaskToListQueries(queryClient, data.task)
    },
    onSettled: (_data, _error, variables) => {
      invalidateTaskQueries(queryClient, variables.task.id)
    },
  })
}

export function useChangeTaskStatusMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ status, task }: ChangeTaskStatusVariables) =>
      taskApi.changeTaskStatus(
        task.id,
        {
          status,
          version: task.version,
        },
        createIdempotencyKey(),
      ),
    onMutate: ({ status, task }) =>
      applyOptimisticTask(queryClient, {
        ...task,
        status,
        version: task.version + 1,
      }),
    onError: (_error, variables, snapshot) => {
      restoreOptimisticSnapshot(queryClient, snapshot, variables.task.id)
    },
    onSuccess: (data) => {
      writeTaskToDetailQuery(queryClient, data.task)
      writeTaskToListQueries(queryClient, data.task)
    },
    onSettled: (_data, _error, variables) => {
      invalidateTaskQueries(queryClient, variables.task.id)
    },
  })
}
