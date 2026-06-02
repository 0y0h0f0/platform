import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import * as memberApi from '@/api/member'
import type { AddProjectMemberRequest, UpdateProjectMemberRoleRequest } from '@/api/types'
import { operationLogQueryKeys } from '@/queries/operationLog.queries'
import { projectQueryKeys } from '@/queries/project.queries'

interface UpdateMemberRoleVariables {
  role: UpdateProjectMemberRoleRequest['role']
  userId: string
}

// memberQueryKeys aliases project member keys so settings panels and mutations
// invalidate the same cache entries.
export const memberQueryKeys = {
  project: (projectId: string) => projectQueryKeys.members(projectId),
}

function createIdempotencyKey() {
  // Member mutations are writes and participate in gateway idempotency.
  return window.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function invalidateMemberQueries(
  queryClient: ReturnType<typeof useQueryClient>,
  projectId: string,
) {
  // Membership changes also append operation logs, so both views refresh
  // together.
  queryClient.invalidateQueries({ queryKey: memberQueryKeys.project(projectId) })
  queryClient.invalidateQueries({ queryKey: operationLogQueryKeys.project(projectId) })
}

// useProjectMembersQuery loads project-scoped membership for permissions and UI.
export function useProjectMembersQuery(projectId: string) {
  return useQuery({
    queryKey: memberQueryKeys.project(projectId),
    queryFn: () => memberApi.listProjectMembers(projectId),
    enabled: Boolean(projectId),
  })
}

export function useAddProjectMemberMutation(projectId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: AddProjectMemberRequest) =>
      memberApi.addProjectMember(projectId, payload, createIdempotencyKey()),
    onSuccess: () => {
      invalidateMemberQueries(queryClient, projectId)
    },
  })
}

export function useUpdateProjectMemberRoleMutation(projectId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ role, userId }: UpdateMemberRoleVariables) =>
      memberApi.updateProjectMemberRole(projectId, userId, { role }, createIdempotencyKey()),
    onSuccess: () => {
      invalidateMemberQueries(queryClient, projectId)
    },
  })
}

export function useRemoveProjectMemberMutation(projectId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (userId: string) =>
      memberApi.removeProjectMember(projectId, userId, createIdempotencyKey()),
    onSuccess: () => {
      invalidateMemberQueries(queryClient, projectId)
    },
  })
}

export function useLeaveProjectMutation(projectId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => memberApi.leaveProject(projectId, createIdempotencyKey()),
    onSuccess: () => {
      invalidateMemberQueries(queryClient, projectId)
      queryClient.invalidateQueries({ queryKey: projectQueryKeys.all })
    },
  })
}
