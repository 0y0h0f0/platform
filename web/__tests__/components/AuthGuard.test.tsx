import { http, HttpResponse } from 'msw'
import { screen, waitFor } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'

import { AuthGuard } from '../../src/components/auth/AuthGuard'
import { server } from '../../src/mocks/server'
import { useAuthStore } from '../../src/stores/auth.store'
import { getToken, setToken } from '../../src/utils/token'
import { renderWithProviders } from '../test-utils'

function renderGuard(initialEntry = '/projects') {
  renderWithProviders(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route element={<AuthGuard />}>
          <Route element={<div>项目页面</div>} path="/projects" />
        </Route>
        <Route element={<div>登录页面</div>} path="/login" />
      </Routes>
    </MemoryRouter>,
  )
}

describe('AuthGuard', () => {
  it('redirects unauthenticated users to login', async () => {
    renderGuard()

    expect(await screen.findByText('登录页面')).toBeInTheDocument()
  })

  it('clears an expired token and redirects to login', async () => {
    server.use(
      http.get('*/api/v1/users/me', () =>
        HttpResponse.json(
          {
            code: 'UNAUTHENTICATED',
            message: 'token expired',
            request_id: 'expired-request-id',
          },
          { status: 401 },
        ),
      ),
    )
    setToken('expired-access-token')
    useAuthStore.setState({ accessToken: 'expired-access-token', status: 'loading', user: null })

    renderGuard()

    expect(await screen.findByText('登录页面')).toBeInTheDocument()
    expect(getToken()).toBeNull()
    expect(useAuthStore.getState().status).toBe('unauthenticated')
  })

  it('validates an existing token before rendering protected content', async () => {
    setToken('mock-access-token')
    useAuthStore.setState({ accessToken: 'mock-access-token', status: 'loading', user: null })

    renderGuard()

    await waitFor(() => expect(screen.getByText('项目页面')).toBeInTheDocument())
    expect(useAuthStore.getState().status).toBe('authenticated')
  })
})
