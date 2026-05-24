import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import * as projectApi from '@/api/project'
import type {
  CreateProjectRequest,
  ListProjectsRequest,
  Project,
  UpdateProjectRequest,
} from '@/api/types'

export interface ProjectListParams {
  includeArchived: boolean
  limit: number
  offset: number
}

export const projectQueryKeys = {
  all: ['projects'] as const,
  list: (params: ProjectListParams) => ['projects', params] as const,
  detail: (projectId: string) => ['projects', projectId] as const,
  members: (projectId: string) => ['projects', projectId, 'members'] as const,
}

function createIdempotencyKey() {
  return window.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function toListRequest(params: ProjectListParams): ListProjectsRequest {
  return {
    include_archived: params.includeArchived,
    limit: params.limit,
    offset: params.offset,
  }
}

function writeProjectDetail(queryClient: ReturnType<typeof useQueryClient>, project: Project) {
  queryClient.setQueryData(projectQueryKeys.detail(project.id), { project })
}

function invalidateProjectLists(queryClient: ReturnType<typeof useQueryClient>) {
  queryClient.invalidateQueries({ queryKey: projectQueryKeys.all })
}

function invalidateProjectDetail(
  queryClient: ReturnType<typeof useQueryClient>,
  projectId: string,
) {
  queryClient.invalidateQueries({ queryKey: projectQueryKeys.detail(projectId) })
}

function invalidateProjectMembers(
  queryClient: ReturnType<typeof useQueryClient>,
  projectId: string,
) {
  queryClient.invalidateQueries({ queryKey: projectQueryKeys.members(projectId) })
}

function invalidateProjectLogs(queryClient: ReturnType<typeof useQueryClient>, projectId: string) {
  queryClient.invalidateQueries({ queryKey: ['operation-logs', 'projects', projectId] })
}

export function useProjectsQuery(params: ProjectListParams) {
  return useQuery({
    queryKey: projectQueryKeys.list(params),
    queryFn: () => projectApi.listProjects(toListRequest(params)),
    placeholderData: keepPreviousData,
  })
}

export function useProjectQuery(projectId: string) {
  return useQuery({
    queryKey: projectQueryKeys.detail(projectId),
    queryFn: () => projectApi.getProject(projectId),
    enabled: Boolean(projectId),
  })
}

export function useCreateProjectMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: CreateProjectRequest) =>
      projectApi.createProject(payload, createIdempotencyKey()),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: projectQueryKeys.all })
    },
  })
}
export function useUpdateProjectMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      project,
      values,
    }: {
      project: Project
      values: Omit<UpdateProjectRequest, 'version'>
    }) =>
      projectApi.updateProject(
        project.id,
        {
          ...values,
          version: project.version,
        },
        createIdempotencyKey(),
      ),
    onSuccess: (data) => {
      writeProjectDetail(queryClient, data.project)
      invalidateProjectLists(queryClient)
      invalidateProjectLogs(queryClient, data.project.id)
    },
  })
}

export function useArchiveProjectMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (projectId: string) => projectApi.archiveProject(projectId, createIdempotencyKey()),
    onSuccess: (data) => {
      writeProjectDetail(queryClient, data.project)
      invalidateProjectLists(queryClient)
      invalidateProjectLogs(queryClient, data.project.id)
    },
  })
}

export function useUnarchiveProjectMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (projectId: string) =>
      projectApi.unarchiveProject(projectId, createIdempotencyKey()),
    onSuccess: (data) => {
      writeProjectDetail(queryClient, data.project)
      invalidateProjectLists(queryClient)
      invalidateProjectLogs(queryClient, data.project.id)
    },
  })
}

export function useTransferProjectOwnershipMutation(projectId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (targetUserId: string) =>
      projectApi.transferProjectOwnership(
        projectId,
        { target_user_id: targetUserId },
        createIdempotencyKey(),
      ),
    onSuccess: (data) => {
      writeProjectDetail(queryClient, data.project)
      invalidateProjectLists(queryClient)
      invalidateProjectDetail(queryClient, data.project.id)
      invalidateProjectMembers(queryClient, data.project.id)
      invalidateProjectLogs(queryClient, data.project.id)
    },
  })
}
