import { useMemo } from 'react'

import type { Project, Task } from '@/api/types'
import { ProjectStatus, Role, type Role as RoleValue } from '@/utils/constants'

// TaskPermission mirrors backend task rules in a UI-friendly shape.
export interface TaskPermission {
  canAssignTask: boolean
  canChangeStatus: boolean
  canComment: boolean
  canEditTask: boolean
  canViewTask: boolean
  isReadOnly: boolean
}

// getTaskPermission is pure so tests and components share the same permission
// calculation.
export function getTaskPermission(
  project: Project | null | undefined,
  role: RoleValue | null | undefined,
  task: Task | null | undefined,
  userId: string | null | undefined,
): TaskPermission {
  const sameProject = Boolean(project && task && project.id === task.project_id)
  const isArchived = project?.status === ProjectStatus.Archived
  const hasProjectRole = role !== null && role !== undefined
  const isOwner = Boolean(project && userId && project.owner_id === userId) || role === Role.Owner
  const isAdmin = role === Role.Admin
  const isCreator = Boolean(task && userId && task.creator_id === userId)
  // Owners may be missing from a stale member list, so owner_id is accepted as a
  // fallback role signal.
  const canViewTask = Boolean(sameProject && userId && (hasProjectRole || isOwner))
  const canWriteTask = Boolean(canViewTask && !isArchived && (isOwner || isAdmin || isCreator))

  return {
    canAssignTask: canWriteTask,
    canChangeStatus: canWriteTask,
    canComment: Boolean(canViewTask && !isArchived),
    canEditTask: canWriteTask,
    canViewTask,
    isReadOnly: Boolean(sameProject && isArchived),
  }
}

// useTaskPermission memoizes the pure permission calculation for React views.
export function useTaskPermission(
  project: Project | null | undefined,
  role: RoleValue | null | undefined,
  task: Task | null | undefined,
  userId: string | null | undefined,
): TaskPermission {
  return useMemo(
    () => getTaskPermission(project, role, task, userId),
    [project, role, task, userId],
  )
}
