import { describe, expect, it } from 'vitest'

import { useAuthStore } from '../../src/stores/auth.store'
import { getToken, setToken } from '../../src/utils/token'

const user = {
  id: 'user_1',
  username: 'alice',
  email: 'alice@example.com',
  nickname: 'Alice',
  avatar_url: '',
  status: 0,
}

describe('auth store', () => {
  it('hydrates to loading when token exists', () => {
    setToken('token-1')

    useAuthStore.getState().hydrate()

    expect(useAuthStore.getState().status).toBe('loading')
    expect(useAuthStore.getState().accessToken).toBe('token-1')
  })

  it('sets authenticated user and persists token', () => {
    useAuthStore.getState().setAuthenticated(user, 'token-2')

    expect(useAuthStore.getState().status).toBe('authenticated')
    expect(useAuthStore.getState().user?.username).toBe('alice')
    expect(getToken()).toBe('token-2')
  })

  it('clears auth state', () => {
    useAuthStore.getState().setAuthenticated(user, 'token-3')
    useAuthStore.getState().setUnauthenticated()

    expect(useAuthStore.getState().status).toBe('unauthenticated')
    expect(useAuthStore.getState().user).toBeNull()
    expect(getToken()).toBeNull()
  })
})
