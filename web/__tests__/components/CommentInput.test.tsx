import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import { CommentInput } from '../../src/components/comment/CommentInput'
import { setToken } from '../../src/utils/token'
import { renderWithProviders } from '../test-utils'

describe('CommentInput', () => {
  it('submits a new task comment', async () => {
    const user = userEvent.setup()
    const onCreated = vi.fn()
    setToken('mock-access-token')

    renderWithProviders(<CommentInput onCreated={onCreated} taskId="task-kanban-api" />)

    await user.type(screen.getByLabelText('评论内容'), '补充阶段六评论验收')
    await user.click(screen.getByRole('button', { name: '发送' }))

    await waitFor(() =>
      expect(onCreated).toHaveBeenCalledWith(
        expect.objectContaining({
          content: '补充阶段六评论验收',
          task_id: 'task-kanban-api',
          user_id: 'user-1',
        }),
      ),
    )
    expect(screen.getByLabelText('评论内容')).toHaveValue('')
  })
})
