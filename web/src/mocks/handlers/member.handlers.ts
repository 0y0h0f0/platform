import { http, HttpResponse } from 'msw'

import type {
  AddProjectMemberRequest,
  ApiEnvelope,
  ListProjectMembersData,
  ProjectMember,
  ProjectMemberData,
  UpdateProjectMemberRoleRequest,
} from '@/api/types'
import { Role } from '@/utils/constants'
import { mockUser } from '../fixtures/users'
import {
  addProjectMember,
  appendOperationLog,
  findProject,
  findProjectMember,
  listProjectMembers,
  projectMockState,
  removeProjectMember,
} from './project.state'

function ok<T>(data: T, status = 200) {
  return HttpResponse.json<ApiEnvelope<T>>(
    {
      code: 'OK',
      message: 'ok',
      request_id: 'mock-request-id',
      data,
    },
    { status },
  )
}

function error(code: string, message: string, status: number) {
  return HttpResponse.json<ApiEnvelope>(
    {
      code,
      message,
      request_id: 'mock-request-id',
    },
    { status },
  )
}

function isAuthorized(request: Request) {
  return Boolean(request.headers.get('authorization'))
}

function isEditableRole(role: number | undefined): role is typeof Role.Admin | typeof Role.Member {
  return role === Role.Admin || role === Role.Member
}

export const memberHandlers = [
  http.get('*/api/v1/projects/:projectId/members', ({ params, request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const projectId = String(params.projectId)
    const project = findProject(projectId)
    if (!project) {
      return error('NOT_FOUND', 'project not found', 404)
    }

    return ok<ListProjectMembersData>({
      members: listProjectMembers(projectId),
    })
  }),

  http.post('*/api/v1/projects/:projectId/members', async ({ params, request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const projectId = String(params.projectId)
    const project = findProject(projectId)
    if (!project) {
      return error('NOT_FOUND', 'project not found', 404)
    }

    const payload = (await request.json()) as Partial<AddProjectMemberRequest>
    const userId = payload.user_id?.trim()
    if (!userId || !isEditableRole(payload.role)) {
      return error('INVALID_ARGUMENT', 'user_id and role are required', 400)
    }

    if (findProjectMember(projectId, userId)) {
      return error('ALREADY_EXISTS', 'member already exists', 409)
    }

    const member: ProjectMember = {
      id: `member-${projectMockState.nextMemberNumber++}`,
      project_id: projectId,
      role: payload.role,
      user_id: userId,
    }

    addProjectMember(member)
    appendOperationLog({
      action: 'member.add',
      detail_json: JSON.stringify({ user_id: userId }),
      operator_id: mockUser.id,
      project_id: projectId,
    })

    return ok<ProjectMemberData>({ member }, 201)
  }),

  http.put('*/api/v1/projects/:projectId/members/:userId', async ({ params, request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const projectId = String(params.projectId)
    const userId = String(params.userId)
    const member = findProjectMember(projectId, userId)
    if (!member) {
      return error('NOT_FOUND', 'member not found', 404)
    }

    const payload = (await request.json()) as Partial<UpdateProjectMemberRoleRequest>
    if (!isEditableRole(payload.role)) {
      return error('INVALID_ARGUMENT', 'invalid role', 400)
    }

    if (member.role === Role.Owner) {
      return error('FAILED_PRECONDITION', 'cannot change owner role', 400)
    }

    member.role = payload.role
    appendOperationLog({
      action: 'member.role_change',
      detail_json: JSON.stringify({ user_id: userId }),
      operator_id: mockUser.id,
      project_id: projectId,
    })

    return ok<ProjectMemberData>({ member })
  }),

  http.delete('*/api/v1/projects/:projectId/members/:userId', ({ params, request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const projectId = String(params.projectId)
    const userId = String(params.userId)
    const member = findProjectMember(projectId, userId)
    if (!member) {
      return error('NOT_FOUND', 'member not found', 404)
    }

    if (member.role === Role.Owner) {
      return error('FAILED_PRECONDITION', 'cannot remove owner', 400)
    }

    removeProjectMember(projectId, userId)
    appendOperationLog({
      action: 'member.remove',
      detail_json: JSON.stringify({ user_id: userId }),
      operator_id: mockUser.id,
      project_id: projectId,
    })

    return ok<ProjectMemberData>({ member })
  }),

  http.post('*/api/v1/projects/:projectId/members/me/leave', ({ params, request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const projectId = String(params.projectId)
    const member = findProjectMember(projectId, mockUser.id)
    if (!member) {
      return error('NOT_FOUND', 'member not found', 404)
    }

    if (member.role === Role.Owner) {
      return error('FAILED_PRECONDITION', 'owner cannot leave', 400)
    }

    removeProjectMember(projectId, mockUser.id)
    appendOperationLog({
      action: 'member.leave',
      detail_json: '{}',
      operator_id: mockUser.id,
      project_id: projectId,
    })

    return ok<ProjectMemberData>({ member })
  }),
]
