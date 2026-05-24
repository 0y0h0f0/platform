import type { OperationLog, Project, ProjectMember } from '@/api/types'
import { findMockUser, mockUser } from '../fixtures/users'
import { createMockMembers } from '../fixtures/members'
import { createMockOperationLogs } from '../fixtures/operationLogs'
import { createMockProjects } from '../fixtures/projects'

export const projectMockState = {
  members: createMockMembers(),
  nextLogNumber: 100,
  nextMemberNumber: 100,
  nextProjectNumber: 100,
  operationLogs: createMockOperationLogs(),
  projects: createMockProjects(),
}

export function resetProjectMockState() {
  projectMockState.members = createMockMembers()
  projectMockState.nextLogNumber = 100
  projectMockState.nextMemberNumber = 100
  projectMockState.nextProjectNumber = 100
  projectMockState.operationLogs = createMockOperationLogs()
  projectMockState.projects = createMockProjects()
}

export function findProject(projectId: string) {
  return projectMockState.projects.find((project) => project.id === projectId)
}

export function findProjectMember(projectId: string, userId: string) {
  return projectMockState.members.find(
    (member) => member.project_id === projectId && member.user_id === userId,
  )
}

export function listProjectMembers(projectId: string) {
  return projectMockState.members.filter((member) => member.project_id === projectId)
}

export function addProject(project: Project) {
  projectMockState.projects = [project, ...projectMockState.projects]
}

export function addProjectMember(member: ProjectMember) {
  projectMockState.members = [member, ...projectMockState.members]
}

export function removeProjectMember(projectId: string, userId: string) {
  const member = findProjectMember(projectId, userId)
  projectMockState.members = projectMockState.members.filter(
    (current) => !(current.project_id === projectId && current.user_id === userId),
  )
  return member
}

export function appendOperationLog(
  log: Omit<OperationLog, 'avatar_url' | 'id' | 'nickname' | 'username'> & { id?: string },
) {
  const user = findMockUser(log.operator_id) ?? mockUser
  const operationLog: OperationLog = {
    ...log,
    id: log.id ?? `log-${projectMockState.nextLogNumber++}`,
    username: user.username,
    nickname: user.nickname,
    avatar_url: user.avatar_url,
  }
  projectMockState.operationLogs = [operationLog, ...projectMockState.operationLogs]
  return operationLog
}
