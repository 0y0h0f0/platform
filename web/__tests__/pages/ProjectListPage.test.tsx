import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'

import ProjectListPage from '../../src/pages/projects/ProjectListPage'
import { setToken } from '../../src/utils/token'
import { renderWithProviders } from '../test-utils'

function renderProjectListPage(initialPath = '/projects') {
  renderWithProviders(
    <MemoryRouter initialEntries={[initialPath]}>
      <Routes>
        <Route element={<ProjectListPage />} path="/projects" />
      </Routes>
    </MemoryRouter>,
  )
}

describe('ProjectListPage', () => {
  it('shows project cards and toggles archived projects', async () => {
    const user = userEvent.setup()
    setToken('mock-access-token')

    renderProjectListPage()

    expect(await screen.findByText('前端管理台')).toBeInTheDocument()
    expect(screen.getByText('移动端看板优化')).toBeInTheDocument()
    expect(screen.queryByText('历史归档迁移')).not.toBeInTheDocument()

    await user.click(screen.getByRole('switch', { name: '显示归档项目' }))

    expect(await screen.findByText('历史归档迁移')).toBeInTheDocument()
  })

  it('creates a project and refreshes the list', async () => {
    const user = userEvent.setup()
    setToken('mock-access-token')

    renderProjectListPage()

    await screen.findByText('前端管理台')
    await user.click(screen.getByRole('button', { name: '创建项目' }))

    const dialog = await screen.findByRole('dialog', { name: '创建项目' })
    await user.type(within(dialog).getByLabelText('项目名称'), '阶段三验收项目')
    await user.type(within(dialog).getByLabelText('项目描述'), '验证项目列表创建流程')
    await user.click(within(dialog).getByRole('button', { name: /创\s*建/ }))

    expect(await screen.findByText('阶段三验收项目')).toBeInTheDocument()
    await waitFor(() =>
      expect(screen.queryByRole('dialog', { name: '创建项目' })).not.toBeInTheDocument(),
    )
  })
})
