import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'

import { OperationLogList } from '../../src/components/project/OperationLogList'
import { setToken } from '../../src/utils/token'
import { renderWithProviders } from '../test-utils'

describe('OperationLogList', () => {
  it('renders enriched operator fields and loads more operation logs', async () => {
    const user = userEvent.setup()
    setToken('mock-access-token')

    renderWithProviders(<OperationLogList limit={2} projectId="project-web-console" />)

    expect(await screen.findByText('发表了评论')).toBeInTheDocument()
    expect(screen.getByText('变更了任务状态')).toBeInTheDocument()
    expect(screen.getAllByText('演示用户').length).toBeGreaterThan(0)
    expect(screen.getAllByText('user-1').length).toBeGreaterThan(0)
    expect(screen.queryByText('添加了成员')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: '加载更多' }))

    expect(await screen.findByText('添加了成员')).toBeInTheDocument()
    expect(screen.getByText('成员: user-3')).toBeInTheDocument()
  })
})
