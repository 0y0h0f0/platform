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
  if (typeof window === 'undefined') {
    return null
  }
  return getToken()
}

const initialToken = readInitialToken()

export const useAuthStore = create<AuthState>((set, get) => ({
  status: initialToken ? 'loading' : 'unauthenticated',
  user: null,
  accessToken: initialToken,
  hydrate: () => {
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
    clearToken()
    set({ accessToken: null, status: 'unauthenticated', user: null })
  },
}))
