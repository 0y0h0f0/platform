import { useMemo } from 'react'

import type { Project } from '@/api/types'
import { ProjectStatus, Role, type Role as RoleValue } from '@/utils/constants'

export interface ProjectPermission {
  canAddMember: boolean
  canArchive: boolean
  canChangeMemberRole: boolean
  canEditProject: boolean
  canLeave: boolean
  canRemoveMember: boolean
  canTransfer: boolean
  canUnarchive: boolean
}

export function getProjectPermission(
  project: Project | null | undefined,
  role: RoleValue | null | undefined,
  userId: string | null | undefined,
): ProjectPermission {
  const isArchived = project?.status === ProjectStatus.Archived
  const isOwner = Boolean(project && userId && project.owner_id === userId) || role === Role.Owner
  const isAdmin = role === Role.Admin

  return {
    canAddMember: Boolean(project && !isArchived && (isOwner || isAdmin)),
    canArchive: Boolean(project && !isArchived && isOwner),
    canChangeMemberRole: Boolean(project && !isArchived && isOwner),
    canEditProject: Boolean(project && !isArchived && isOwner),
    canLeave: Boolean(project && !isArchived && !isOwner),
    canRemoveMember: Boolean(project && !isArchived && (isOwner || isAdmin)),
    canTransfer: Boolean(project && !isArchived && isOwner),
    canUnarchive: Boolean(project && isArchived && isOwner),
  }
}

export function useProjectPermission(
  project: Project | null | undefined,
  role: RoleValue | null | undefined,
  userId: string | null | undefined,
): ProjectPermission {
  return useMemo(() => getProjectPermission(project, role, userId), [project, role, userId])
}
