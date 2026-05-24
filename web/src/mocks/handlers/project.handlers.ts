import { http, HttpResponse } from 'msw'

import type {
  ApiEnvelope,
  CreateProjectRequest,
  CreateProjectData,
  ListProjectsData,
  Project,
  ProjectMember,
  ProjectData,
  TransferProjectOwnershipRequest,
  UpdateProjectRequest,
} from '@/api/types'
import { ProjectStatus, Role } from '@/utils/constants'
import {
  addProject,
  addProjectMember,
  appendOperationLog,
  findProject,
  findProjectMember,
  projectMockState,
  resetProjectMockState,
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

function hasVersionConflict(project: Project, version: number | undefined) {
  return version === undefined || version !== project.version
}

export function resetMockProjects() {
  resetProjectMockState()
}

export const projectHandlers = [
  http.get('*/api/v1/projects', ({ request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const url = new URL(request.url)
    const limit = Number(url.searchParams.get('limit') ?? 20)
    const offset = Number(url.searchParams.get('offset') ?? 0)
    const includeArchived = url.searchParams.get('include_archived') === 'true'

    const visibleProjects = includeArchived
      ? projectMockState.projects
      : projectMockState.projects.filter((project) => project.status === ProjectStatus.Active)

    return ok<ListProjectsData>({
      projects: visibleProjects.slice(offset, offset + limit),
    })
  }),

  http.post('*/api/v1/projects', async ({ request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const payload = (await request.json()) as Partial<CreateProjectRequest>
    const name = payload.name?.trim()

    if (!name) {
      return error('INVALID_ARGUMENT', 'project name is required', 400)
    }

    const project: Project = {
      id: `project-${projectMockState.nextProjectNumber++}`,
      name,
      description: payload.description?.trim() ?? '',
      owner_id: 'user-1',
      status: ProjectStatus.Active,
      version: 1,
    }
    const member: ProjectMember = {
      id: `member-${projectMockState.nextMemberNumber++}`,
      project_id: project.id,
      role: Role.Owner,
      user_id: project.owner_id,
    }

    addProject(project)
    addProjectMember(member)
    appendOperationLog({
      action: 'project.create',
      detail_json: '{}',
      operator_id: project.owner_id,
      project_id: project.id,
    })

    return ok<CreateProjectData>({ project }, 201)
  }),

  http.get('*/api/v1/projects/:projectId', ({ params, request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const project = findProject(String(params.projectId))
    if (!project) {
      return error('NOT_FOUND', 'project not found', 404)
    }

    return ok<ProjectData>({ project })
  }),

  http.put('*/api/v1/projects/:projectId', async ({ params, request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const project = findProject(String(params.projectId))
    if (!project) {
      return error('NOT_FOUND', 'project not found', 404)
    }

    if (project.status === ProjectStatus.Archived) {
      return error('FAILED_PRECONDITION', 'archived project is read-only', 400)
    }

    const payload = (await request.json()) as Partial<UpdateProjectRequest>
    if (hasVersionConflict(project, payload.version)) {
      return error('ABORTED', 'version conflict', 409)
    }

    const name = payload.name?.trim()
    if (!name) {
      return error('INVALID_ARGUMENT', 'project name is required', 400)
    }

    project.name = name
    project.description = payload.description?.trim() ?? ''
    project.version += 1

    appendOperationLog({
      action: 'project.update',
      detail_json: '{}',
      operator_id: project.owner_id,
      project_id: project.id,
    })

    return ok<ProjectData>({ project })
  }),

  http.post('*/api/v1/projects/:projectId/archive', ({ params, request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const project = findProject(String(params.projectId))
    if (!project) {
      return error('NOT_FOUND', 'project not found', 404)
    }

    project.status = ProjectStatus.Archived
    project.version += 1
    appendOperationLog({
      action: 'project.archive',
      detail_json: '{}',
      operator_id: project.owner_id,
      project_id: project.id,
    })

    return ok<ProjectData>({ project })
  }),

  http.post('*/api/v1/projects/:projectId/unarchive', ({ params, request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const project = findProject(String(params.projectId))
    if (!project) {
      return error('NOT_FOUND', 'project not found', 404)
    }

    project.status = ProjectStatus.Active
    project.version += 1
    appendOperationLog({
      action: 'project.unarchive',
      detail_json: '{}',
      operator_id: project.owner_id,
      project_id: project.id,
    })

    return ok<ProjectData>({ project })
  }),

  http.post('*/api/v1/projects/:projectId/transfer', async ({ params, request }) => {
    if (!isAuthorized(request)) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    const project = findProject(String(params.projectId))
    if (!project) {
      return error('NOT_FOUND', 'project not found', 404)
    }

    if (project.status === ProjectStatus.Archived) {
      return error('FAILED_PRECONDITION', 'archived project is read-only', 400)
    }

    const payload = (await request.json()) as Partial<TransferProjectOwnershipRequest>
    const targetUserId = payload.target_user_id?.trim()
    if (!targetUserId) {
      return error('INVALID_ARGUMENT', 'target_user_id is required', 400)
    }

    if (targetUserId === project.owner_id) {
      return error('INVALID_ARGUMENT', 'cannot transfer ownership to yourself', 400)
    }

    const oldOwnerMember = findProjectMember(project.id, project.owner_id)
    const targetMember = findProjectMember(project.id, targetUserId)
    if (!targetMember) {
      return error('FAILED_PRECONDITION', 'target user is not a project member', 400)
    }

    if (oldOwnerMember) {
      oldOwnerMember.role = Role.Admin
    }
    targetMember.role = Role.Owner
    const fromUserId = project.owner_id
    project.owner_id = targetUserId
    project.version += 1

    appendOperationLog({
      action: 'project.transfer_ownership',
      detail_json: JSON.stringify({ from_user_id: fromUserId, to_user_id: targetUserId }),
      operator_id: fromUserId,
      project_id: project.id,
    })

    return ok<ProjectData>({ project })
  }),
]
