import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import type { Project } from '../../src/api/types'
import { ProjectEditModal } from '../../src/components/project/ProjectEditModal'
import { ProjectStatus } from '../../src/utils/constants'
import { setToken } from '../../src/utils/token'
import { renderWithProviders } from '../test-utils'

const project: Project = {
  id: 'project-web-console',
  name: '前端管理台',
  description: '搭建注册登录、项目列表和任务看板的前端工作台。',
  owner_id: 'user-1',
  status: ProjectStatus.Active,
  version: 3,
}

describe('ProjectEditModal', () => {
  it('submits updated project data with the current version', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    const onUpdated = vi.fn()
    setToken('mock-access-token')

    renderWithProviders(
      <ProjectEditModal onClose={onClose} onUpdated={onUpdated} open project={project} />,
    )

    const dialog = await screen.findByRole('dialog', { name: '编辑项目' })
    await user.clear(within(dialog).getByLabelText('项目名称'))
    await user.type(within(dialog).getByLabelText('项目名称'), '前端管理台升级')
    await user.clear(within(dialog).getByLabelText('项目描述'))
    await user.type(within(dialog).getByLabelText('项目描述'), '补齐项目设置和成员协作能力')
    await user.click(within(dialog).getByRole('button', { name: /保\s*存/ }))

    await waitFor(() =>
      expect(onUpdated).toHaveBeenCalledWith(
        expect.objectContaining({
          description: '补齐项目设置和成员协作能力',
          name: '前端管理台升级',
          version: 4,
        }),
      ),
    )
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
