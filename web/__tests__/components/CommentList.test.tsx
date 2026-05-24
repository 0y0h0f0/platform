import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'

import { CommentList } from '../../src/components/comment/CommentList'
import { Role } from '../../src/utils/constants'
import { setToken } from '../../src/utils/token'
import { renderWithProviders } from '../test-utils'

describe('CommentList', () => {
  it('loads more comments with after_id pagination', async () => {
    const user = userEvent.setup()
    setToken('mock-access-token')

    renderWithProviders(
      <CommentList
        currentRole={Role.Owner}
        currentUserId="user-1"
        limit={2}
        taskId="task-kanban-api"
      />,
    )

    expect(await screen.findByText('接口分页已经联通，下一步补详情抽屉。')).toBeInTheDocument()
    expect(screen.getByText('项目负责人')).toBeInTheDocument()
    expect(screen.getByText('状态流转要复用同一套合法转换规则。')).toBeInTheDocument()
    expect(screen.queryByText('我会补充 after_id 分页测试。')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: '加载更多' }))

    expect(await screen.findByText('我会补充 after_id 分页测试。')).toBeInTheDocument()
    await waitFor(() =>
      expect(screen.queryByRole('button', { name: '加载更多' })).not.toBeInTheDocument(),
    )
  })

  it('only allows members to delete their own comments', async () => {
    const user = userEvent.setup()
    setToken('mock-access-token')

    renderWithProviders(
      <CommentList currentRole={Role.Member} currentUserId="user-3" taskId="task-kanban-api" />,
    )

    expect(await screen.findByText('接口分页已经联通，下一步补详情抽屉。')).toBeInTheDocument()
    expect(screen.queryByLabelText('删除评论 comment-001')).not.toBeInTheDocument()

    await user.click(screen.getByLabelText('删除评论 comment-003'))
    const deleteButtons = await screen.findAllByRole('button', { name: /删\s*除/ })
    await user.click(deleteButtons[deleteButtons.length - 1])

    await waitFor(() =>
      expect(screen.queryByText('我会补充 after_id 分页测试。')).not.toBeInTheDocument(),
    )
  })
})
