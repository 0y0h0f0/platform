import { create } from 'zustand'

import type { User } from '@/api/types'
import { clearToken, getToken, setToken } from '@/utils/token'

export type AuthStatus = 'loading' | 'authenticated' | 'unauthenticated'

interface AuthState {
  status: AuthStatus
  user: User | null
  accessToken: string | null
  hydrate: () => void
  setAuthenticated: (user: User, accessToken?: string) => void
  setUser: (user: User) => void
  setUnauthenticated: () => void
}

function readInitialToken() {
  // Guards SSR/test environments where localStorage is not available.
  if (typeof window === 'undefined') {
    return null
  }
  return getToken()
}

const initialToken = readInitialToken()

// useAuthStore is the single source of truth for auth session state. The token
// is persisted separately so reloads can rehydrate before /me succeeds.
export const useAuthStore = create<AuthState>((set, get) => ({
  status: initialToken ? 'loading' : 'unauthenticated',
  user: null,
  accessToken: initialToken,
  hydrate: () => {
    // Re-read storage because the API client may clear the token after a 401.
    const token = getToken()
    set({
      accessToken: token,
      status: token ? 'loading' : 'unauthenticated',
      user: token ? get().user : null,
    })
  },
  setAuthenticated: (user, accessToken) => {
    if (accessToken) {
      setToken(accessToken)
    }
    set({
      accessToken: accessToken ?? get().accessToken,
      status: 'authenticated',
      user,
    })
  },
  setUser: (user) => {
    set({ status: 'authenticated', user })
  },
  setUnauthenticated: () => {
    // Clearing storage first prevents a reload from reviving an expired session.
    clearToken()
    set({ accessToken: null, status: 'unauthenticated', user: null })
  },
}))
