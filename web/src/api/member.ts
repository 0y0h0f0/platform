import { request } from './client'
import type {
  AddProjectMemberRequest,
  ListProjectMembersData,
  ProjectMemberData,
  UpdateProjectMemberRoleRequest,
} from './types'

export function listProjectMembers(projectId: string) {
  return request<ListProjectMembersData>({
    url: `/projects/${projectId}/members`,
    method: 'GET',
  })
}

export function addProjectMember(
  projectId: string,
  payload: AddProjectMemberRequest,
  idempotencyKey?: string,
) {
  return request<ProjectMemberData>({
    url: `/projects/${projectId}/members`,
    method: 'POST',
    data: payload,
    idempotencyKey,
  })
}

export function updateProjectMemberRole(
  projectId: string,
  userId: string,
  payload: UpdateProjectMemberRoleRequest,
  idempotencyKey?: string,
) {
  return request<ProjectMemberData>({
    url: `/projects/${projectId}/members/${userId}`,
    method: 'PUT',
    data: payload,
    idempotencyKey,
  })
}

export function removeProjectMember(projectId: string, userId: string, idempotencyKey?: string) {
  return request<ProjectMemberData>({
    url: `/projects/${projectId}/members/${userId}`,
    method: 'DELETE',
    idempotencyKey,
  })
}

export function leaveProject(projectId: string, idempotencyKey?: string) {
  return request<ProjectMemberData>({
    url: `/projects/${projectId}/members/me/leave`,
    method: 'POST',
    idempotencyKey,
  })
}
