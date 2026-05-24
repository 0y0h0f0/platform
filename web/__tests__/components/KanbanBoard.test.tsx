import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'

import { KanbanBoard } from '../../src/components/task/KanbanBoard'
import { setToken } from '../../src/utils/token'
import { renderWithProviders } from '../test-utils'

describe('KanbanBoard', () => {
  it('renders four columns and only legal status transitions', async () => {
    setToken('mock-access-token')

    renderWithProviders(<KanbanBoard projectId="project-web-console" />)

    expect(await screen.findByRole('region', { name: '待办' })).toBeInTheDocument()
    expect(screen.getByRole('region', { name: '进行中' })).toBeInTheDocument()
    expect(screen.getByRole('region', { name: '已完成' })).toBeInTheDocument()
    expect(screen.getByRole('region', { name: '已取消' })).toBeInTheDocument()

    const todoCard = screen.getByText('设计登录页').closest('.task-card')
    expect(todoCard).not.toBeNull()

    const card = within(todoCard as HTMLElement)
    expect(card.getByRole('button', { name: '进行中' })).toBeInTheDocument()
    expect(card.getByRole('button', { name: '已完成' })).toBeInTheDocument()
    expect(card.getByRole('button', { name: '已取消' })).toBeInTheDocument()
    expect(card.queryByRole('button', { name: '待办' })).not.toBeInTheDocument()
  })

  it('filters tasks by keyword', async () => {
    setToken('mock-access-token')

    renderWithProviders(<KanbanBoard keyword="MSW" projectId="project-web-console" />)

    expect(await screen.findByText('完善 MSW 鉴权 mock')).toBeInTheDocument()
    expect(screen.queryByText('设计登录页')).not.toBeInTheDocument()
  })

  it('moves a task when dropped on a valid status column', async () => {
    setToken('mock-access-token')

    renderWithProviders(<KanbanBoard projectId="project-web-console" />)

    await screen.findByText('设计登录页')
    const todoCard = screen.getByText('设计登录页').closest('.task-card')
    expect(todoCard).not.toBeNull()

    const dataTransfer = {
      dropEffect: 'none',
      effectAllowed: 'all',
      getData: (type: string) => (type === 'application/x-task-id' ? 'task-login-wireframe' : ''),
      setData: () => undefined,
    }

    const doingColumn = screen.getByRole('region', { name: '进行中' })
    fireEvent.dragStart(todoCard as HTMLElement, { dataTransfer })
    fireEvent.dragOver(doingColumn, { dataTransfer })
    fireEvent.drop(doingColumn, { dataTransfer })

    await waitFor(() => expect(within(doingColumn).getByText('设计登录页')).toBeInTheDocument())
  })

  it('moves a task after status change', async () => {
    const user = userEvent.setup()
    setToken('mock-access-token')

    renderWithProviders(<KanbanBoard projectId="project-web-console" />)

    await screen.findByText('设计登录页')
    const todoCard = screen.getByText('设计登录页').closest('.task-card')
    expect(todoCard).not.toBeNull()

    await user.click(within(todoCard as HTMLElement).getByRole('button', { name: '进行中' }))

    const doingColumn = screen.getByRole('region', { name: '进行中' })
    await waitFor(() => expect(within(doingColumn).getByText('设计登录页')).toBeInTheDocument())
  })
})
