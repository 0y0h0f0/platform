import { describe, expect, it } from 'vitest'

import type { Project, Task } from '../../src/api/types'
import { getTaskPermission } from '../../src/hooks/useTaskPermission'
import { Priority, ProjectStatus, Role, TaskStatus } from '../../src/utils/constants'

const project: Project = {
  id: 'project-web-console',
  name: '前端管理台',
  description: '',
  owner_id: 'user-1',
  status: ProjectStatus.Active,
  version: 1,
}

const task: Task = {
  id: 'task-1',
  project_id: project.id,
  title: '任务',
  content: '',
  status: TaskStatus.Todo,
  priority: Priority.Normal,
  assignee_id: 'user-2',
  creator_id: 'user-2',
  due_time: '',
  version: 1,
}

describe('useTaskPermission', () => {
  it('allows owner and admin to operate any task', () => {
    expect(getTaskPermission(project, Role.Owner, task, 'user-1').canEditTask).toBe(true)
    expect(getTaskPermission(project, Role.Admin, task, 'user-3').canAssignTask).toBe(true)
  })

  it('allows regular members to operate only tasks they created', () => {
    expect(getTaskPermission(project, Role.Member, task, 'user-2').canChangeStatus).toBe(true)
    expect(getTaskPermission(project, Role.Member, task, 'user-3').canEditTask).toBe(false)
    expect(getTaskPermission(project, Role.Member, task, 'user-3').canComment).toBe(true)
  })

  it('marks archived project tasks as read only', () => {
    const archivedProject = { ...project, status: ProjectStatus.Archived }
    const permission = getTaskPermission(archivedProject, Role.Owner, task, 'user-1')

    expect(permission.isReadOnly).toBe(true)
    expect(permission.canEditTask).toBe(false)
    expect(permission.canAssignTask).toBe(false)
    expect(permission.canChangeStatus).toBe(false)
    expect(permission.canComment).toBe(false)
  })
})
