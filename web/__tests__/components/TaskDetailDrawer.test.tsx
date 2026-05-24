import { http, HttpResponse } from 'msw'
import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'

import type { Project } from '../../src/api/types'
import { TaskDetailDrawer } from '../../src/components/task/TaskDetailDrawer'
import { createMockMembers } from '../../src/mocks/fixtures/members'
import { server } from '../../src/mocks/server'
import { ProjectStatus, Role } from '../../src/utils/constants'
import { setToken } from '../../src/utils/token'
import { renderWithProviders } from '../test-utils'

const project: Project = {
  id: 'project-web-console',
  name: '前端管理台',
  description: '管理任务和项目',
  owner_id: 'user-1',
  status: ProjectStatus.Active,
  version: 1,
}

const archivedProject = {
  ...project,
  status: ProjectStatus.Archived,
}

const members = createMockMembers().filter((member) => member.project_id === project.id)

function getSelectControl(dialog: HTMLElement, label: string) {
  return within(dialog).getAllByLabelText(label)[0]
}

function openSelect(dialog: HTMLElement, label: string) {
  const select = getSelectControl(dialog, label)
  fireEvent.mouseDown(select.querySelector('.ant-select-selector') ?? select)
}

function renderDrawer(taskId = 'task-kanban-api') {
  renderDrawerWithAccess({ taskId })
}

function renderDrawerWithAccess({
  currentRole = Role.Owner,
  currentUserId = 'user-1',
  currentProject = project,
  taskId = 'task-kanban-api',
}: {
  currentRole?: Role
  currentUserId?: string
  currentProject?: Project
  taskId?: string
}) {
  setToken('mock-access-token')

  renderWithProviders(
    <TaskDetailDrawer
      currentRole={currentRole}
      currentUserId={currentUserId}
      members={members}
      onClose={() => undefined}
      open
      project={currentProject}
      taskId={taskId}
    />,
  )
}

describe('TaskDetailDrawer', () => {
  it('edits task fields, status, and assignee', async () => {
    const user = userEvent.setup()
    renderDrawer()

    const dialog = await screen.findByRole('dialog', { name: '接入看板任务接口' })
    await user.clear(within(dialog).getByLabelText('任务标题'))
    await user.type(within(dialog).getByLabelText('任务标题'), '接入任务详情抽屉')
    await user.click(within(dialog).getByRole('button', { name: '保存' }))

    await waitFor(() =>
      expect(within(dialog).getByDisplayValue('接入任务详情抽屉')).toBeInTheDocument(),
    )

    openSelect(dialog, '任务状态')
    await user.click(await screen.findByText('已完成'))
    await waitFor(() => expect(getSelectControl(dialog, '任务状态')).toHaveTextContent('已完成'))

    openSelect(dialog, '负责人')
    await user.click(await screen.findByText('user-2 · 管理员'))
    await waitFor(() => expect(getSelectControl(dialog, '负责人')).toHaveTextContent('user-2'))
  })

  it('renders archived project tasks as read-only', async () => {
    renderDrawerWithAccess({ currentProject: archivedProject })

    const dialog = await screen.findByRole('dialog', { name: '接入看板任务接口' })

    expect(screen.getByText('项目已归档，任务只读')).toBeInTheDocument()
    expect(within(dialog).getByLabelText('任务标题')).toBeDisabled()
    expect(within(dialog).getByRole('button', { name: '保存' })).toBeDisabled()
  })

  it('blocks task editing when the member has no write permission', async () => {
    renderDrawerWithAccess({ currentRole: Role.Member, currentUserId: 'user-3' })

    const dialog = await screen.findByRole('dialog', { name: '接入看板任务接口' })

    expect(screen.getByText('没有编辑此任务的权限')).toBeInTheDocument()
    expect(within(dialog).getByLabelText('任务标题')).toBeDisabled()
    expect(within(dialog).getByRole('button', { name: '保存' })).toBeDisabled()
  })

  it('shows conflict warning and refreshes latest task after optimistic lock failure', async () => {
    const user = userEvent.setup()
    server.use(
      http.put('*/api/v1/tasks/:taskId', () =>
        HttpResponse.json(
          {
            code: 'ABORTED',
            message: 'version conflict',
            request_id: 'conflict-request-id',
          },
          { status: 409 },
        ),
      ),
    )

    renderDrawer('task-login-wireframe')

    const dialog = await screen.findByRole('dialog', { name: '设计登录页' })
    await user.clear(within(dialog).getByLabelText('任务标题'))
    await user.type(within(dialog).getByLabelText('任务标题'), '冲突标题')
    await user.click(within(dialog).getByRole('button', { name: '保存' }))

    expect(await screen.findByText('任务版本已更新')).toBeInTheDocument()
    await waitFor(() => expect(within(dialog).getByDisplayValue('设计登录页')).toBeInTheDocument())
  })
})
