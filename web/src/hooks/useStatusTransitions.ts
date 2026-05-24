import { useMemo } from 'react'

import { TaskStatus, type TaskStatus as TaskStatusValue } from '@/utils/constants'

const transitions: Record<TaskStatusValue, TaskStatusValue[]> = {
  [TaskStatus.Todo]: [TaskStatus.Doing, TaskStatus.Done, TaskStatus.Cancelled],
  [TaskStatus.Doing]: [TaskStatus.Done, TaskStatus.Cancelled, TaskStatus.Todo],
  [TaskStatus.Done]: [TaskStatus.Doing],
  [TaskStatus.Cancelled]: [TaskStatus.Todo],
}

export function getAllowedStatusTransitions(status: TaskStatusValue): TaskStatusValue[] {
  return transitions[status]
}

export function isAllowedStatusTransition(from: TaskStatusValue, to: TaskStatusValue): boolean {
  return transitions[from].includes(to)
}

export function useStatusTransitions(status: TaskStatusValue): TaskStatusValue[] {
  return useMemo(() => getAllowedStatusTransitions(status), [status])
}
