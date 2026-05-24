import { request } from './client'
import type {
  CreateProjectRequest,
  CreateProjectData,
  GetProjectData,
  ListProjectsRequest,
  ListProjectsData,
  ProjectData,
  TransferProjectOwnershipRequest,
  UpdateProjectRequest,
} from './types'

export function listProjects(params: ListProjectsRequest) {
  return request<ListProjectsData>({
    url: '/projects',
    method: 'GET',
    params,
  })
}

export function createProject(payload: CreateProjectRequest, idempotencyKey?: string) {
  return request<CreateProjectData>({
    url: '/projects',
    method: 'POST',
    data: payload,
    idempotencyKey,
  })
}

export function getProject(projectId: string) {
  return request<GetProjectData>({
    url: `/projects/${projectId}`,
    method: 'GET',
  })
}

export function updateProject(
  projectId: string,
  payload: UpdateProjectRequest,
  idempotencyKey?: string,
) {
  return request<ProjectData>({
    url: `/projects/${projectId}`,
    method: 'PUT',
    data: payload,
    idempotencyKey,
  })
}

export function archiveProject(projectId: string, idempotencyKey?: string) {
  return request<ProjectData>({
    url: `/projects/${projectId}/archive`,
    method: 'POST',
    idempotencyKey,
  })
}

export function unarchiveProject(projectId: string, idempotencyKey?: string) {
  return request<ProjectData>({
    url: `/projects/${projectId}/unarchive`,
    method: 'POST',
    idempotencyKey,
  })
}

export function transferProjectOwnership(
  projectId: string,
  payload: TransferProjectOwnershipRequest,
  idempotencyKey?: string,
) {
  return request<ProjectData>({
    url: `/projects/${projectId}/transfer`,
    method: 'POST',
    data: payload,
    idempotencyKey,
  })
}
