import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'

import LoginPage from '../../src/pages/auth/LoginPage'
import { useAuthStore } from '../../src/stores/auth.store'
import { getToken } from '../../src/utils/token'
import { renderWithProviders } from '../test-utils'

describe('LoginPage', () => {
  it('logs in and navigates to projects', async () => {
    const user = userEvent.setup()

    renderWithProviders(
      <MemoryRouter initialEntries={['/login']}>
        <Routes>
          <Route element={<LoginPage />} path="/login" />
          <Route element={<div>我的项目</div>} path="/projects" />
        </Routes>
      </MemoryRouter>,
    )

    await user.type(screen.getByLabelText('账号或邮箱'), 'demo_user')
    await user.type(screen.getByLabelText('密码'), 'password123')
    await user.click(screen.getByRole('button', { name: /登\s*录/ }))

    expect(await screen.findByText('我的项目')).toBeInTheDocument()
    await waitFor(() => expect(useAuthStore.getState().status).toBe('authenticated'))
    expect(getToken()).toBe('mock-access-token')
  })
})
