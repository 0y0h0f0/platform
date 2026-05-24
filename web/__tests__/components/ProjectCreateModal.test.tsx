import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import { ProjectCreateModal } from '../../src/components/project/ProjectCreateModal'
import { setToken } from '../../src/utils/token'
import { renderWithProviders } from '../test-utils'

describe('ProjectCreateModal', () => {
  it('submits project data', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    const onCreated = vi.fn()
    setToken('mock-access-token')

    renderWithProviders(<ProjectCreateModal onClose={onClose} onCreated={onCreated} open />)

    const dialog = await screen.findByRole('dialog', { name: '创建项目' })
    await user.type(within(dialog).getByLabelText('项目名称'), '客户成功空间')
    await user.type(within(dialog).getByLabelText('项目描述'), '沉淀客户交付任务')
    await user.click(within(dialog).getByRole('button', { name: /创\s*建/ }))

    await waitFor(() =>
      expect(onCreated).toHaveBeenCalledWith(
        expect.objectContaining({
          description: '沉淀客户交付任务',
          name: '客户成功空间',
        }),
      ),
    )
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
