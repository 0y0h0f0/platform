import { Alert, Button } from 'antd'
import { useMemo } from 'react'

import type { Task } from '@/api/types'
import { LoadingSpinner } from '@/components/common/LoadingSpinner'
import { isAllowedStatusTransition } from '@/hooks/useStatusTransitions'
import {
  useChangeTaskStatusMutation,
  useTasksQuery,
  type TaskListParams,
} from '@/queries/task.queries'
import { TaskStatus, type TaskStatus as TaskStatusValue } from '@/utils/constants'
import { KanbanColumn } from './KanbanColumn'

// BOARD_STATUSES defines the visible column order and matches backend statuses.
const BOARD_STATUSES = [
  TaskStatus.Todo,
  TaskStatus.Doing,
  TaskStatus.Done,
  TaskStatus.Cancelled,
] as const

interface KanbanBoardProps {
  assigneeId?: string
  disabled?: boolean
  keyword?: string
  onTaskOpen?: (task: Task) => void
  projectId: string
  status?: TaskStatusValue
}

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : '任务列表加载失败'
}

export function KanbanBoard({
  assigneeId,
  disabled = false,
  keyword,
  onTaskOpen,
  projectId,
  status,
}: KanbanBoardProps) {
  const queryParams = useMemo<TaskListParams>(
    () => ({
      assigneeId,
      keyword,
      limit: 50,
      projectId,
      status,
    }),
    [assigneeId, keyword, projectId, status],
  )

  const tasksQuery = useTasksQuery(queryParams)
  const changeStatusMutation = useChangeTaskStatusMutation()
  const tasks = useMemo(
    // The board renders all loaded pages at once; fetchNextPage appends more.
    () =>
      tasksQuery.data?.pages.flatMap((page) =>
        (page.tasks ?? []).map((t) => {
          if (
            t.status === undefined ||
            t.status === null ||
            t.version === undefined ||
            t.version === null
          ) {
            if (import.meta.env.DEV) {
              console.warn('Task missing required fields', {
                id: t.id,
                status: t.status,
                version: t.version,
              })
            }
          }
          return { ...t, status: t.status ?? 0, version: t.version ?? 0 }
        }),
      ) ?? [],
    [tasksQuery.data],
  )

  const tasksByStatus = useMemo(() => {
    // Build stable arrays per column so each column stays simple and stateless.
    return BOARD_STATUSES.reduce(
      (acc, boardStatus) => {
        acc[boardStatus] = tasks.filter((task) => task.status === boardStatus)
        return acc
      },
      {} as Record<TaskStatusValue, Task[]>,
    )
  }, [tasks])

  const handleChangeStatus = (task: Task, nextStatus: TaskStatusValue) => {
    // Guard the client-side interaction before the mutation reaches backend
    // transition validation.
    if (disabled || !isAllowedStatusTransition(task.status, nextStatus)) {
      return
    }

    changeStatusMutation.mutate({ status: nextStatus, task })
  }

  const handleDropTask = (taskId: string, nextStatus: TaskStatusValue) => {
    const task = tasks.find((currentTask) => currentTask.id === taskId)
    if (!task) {
      return
    }

    handleChangeStatus(task, nextStatus)
  }

  if (tasksQuery.isLoading) {
    return <LoadingSpinner tip="正在加载任务" />
  }

  if (tasksQuery.isError) {
    return (
      <Alert
        action={
          <Button onClick={() => tasksQuery.refetch()} size="small">
            重试
          </Button>
        }
        description={getErrorMessage(tasksQuery.error)}
        message="任务列表加载失败"
        showIcon
        type="error"
      />
    )
  }

  return (
    <div className="kanban-board">
      <div className="kanban-grid">
        {BOARD_STATUSES.map((boardStatus) => (
          <KanbanColumn
            disabled={disabled || changeStatusMutation.isPending}
            key={boardStatus}
            onChangeStatus={handleChangeStatus}
            onDropTask={handleDropTask}
            onTaskOpen={onTaskOpen}
            status={boardStatus}
            tasks={tasksByStatus[boardStatus]}
          />
        ))}
      </div>

      {tasksQuery.hasNextPage ? (
        <footer className="kanban-load-more">
          <Button
            loading={tasksQuery.isFetchingNextPage}
            onClick={() => tasksQuery.fetchNextPage()}
          >
            加载更多
          </Button>
        </footer>
      ) : null}
    </div>
  )
}
