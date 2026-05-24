import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'

import RegisterPage from '../../src/pages/auth/RegisterPage'
import { useAuthStore } from '../../src/stores/auth.store'
import { renderWithProviders } from '../test-utils'

describe('RegisterPage', () => {
  it('registers and navigates to projects', async () => {
    const user = userEvent.setup()

    renderWithProviders(
      <MemoryRouter initialEntries={['/register']}>
        <Routes>
          <Route element={<RegisterPage />} path="/register" />
          <Route element={<div>我的项目</div>} path="/projects" />
        </Routes>
      </MemoryRouter>,
    )

    await user.type(screen.getByLabelText('用户名'), 'new_user')
    await user.type(screen.getByLabelText('邮箱'), 'new@example.com')
    await user.type(screen.getByLabelText('密码'), 'password123')
    await user.click(screen.getByRole('button', { name: /注\s*册/ }))

    expect(await screen.findByText('我的项目')).toBeInTheDocument()
    await waitFor(() => expect(useAuthStore.getState().user?.username).toBe('new_user'))
  })
})
