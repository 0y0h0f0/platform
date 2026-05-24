import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import { TaskCreateModal } from '../../src/components/task/TaskCreateModal'
import { setToken } from '../../src/utils/token'
import { renderWithProviders } from '../test-utils'

describe('TaskCreateModal', () => {
  it('submits task data', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    const onCreated = vi.fn()
    setToken('mock-access-token')

    renderWithProviders(
      <TaskCreateModal
        onClose={onClose}
        onCreated={onCreated}
        open
        projectId="project-web-console"
      />,
    )

    const dialog = await screen.findByRole('dialog', { name: '创建任务' })
    await user.type(within(dialog).getByLabelText('任务标题'), '阶段四看板验收')
    await user.type(within(dialog).getByLabelText('任务内容'), '验证创建任务进入待办列')
    await user.click(within(dialog).getByRole('button', { name: /创\s*建/ }))

    await waitFor(() =>
      expect(onCreated).toHaveBeenCalledWith(
        expect.objectContaining({
          content: '验证创建任务进入待办列',
          project_id: 'project-web-console',
          title: '阶段四看板验收',
        }),
      ),
    )
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
