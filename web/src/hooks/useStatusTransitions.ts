import { useMemo } from 'react'

import { TaskStatus, type TaskStatus as TaskStatusValue } from '@/utils/constants'

// transitions must stay in sync with internal/task/biz/task.go.
const transitions: Record<TaskStatusValue, TaskStatusValue[]> = {
  [TaskStatus.Todo]: [TaskStatus.Doing, TaskStatus.Done, TaskStatus.Cancelled],
  [TaskStatus.Doing]: [TaskStatus.Done, TaskStatus.Cancelled, TaskStatus.Todo],
  [TaskStatus.Done]: [TaskStatus.Doing],
  [TaskStatus.Cancelled]: [TaskStatus.Todo],
}

// getAllowedStatusTransitions returns legal next states for a task status.
export function getAllowedStatusTransitions(status: TaskStatusValue): TaskStatusValue[] {
  return transitions[status]
}

// isAllowedStatusTransition is used before optimistic drag/drop mutations.
export function isAllowedStatusTransition(from: TaskStatusValue, to: TaskStatusValue): boolean {
  return transitions[from].includes(to)
}

export function useStatusTransitions(status: TaskStatusValue): TaskStatusValue[] {
  return useMemo(() => getAllowedStatusTransitions(status), [status])
}
