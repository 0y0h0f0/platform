import { useMutation, useQuery } from '@tanstack/react-query'

import * as authApi from '@/api/auth'
import type { LoginRequest, RegisterRequest } from '@/api/types'
import { useAuthStore } from '@/stores/auth.store'

export const authQueryKeys = {
  currentUser: ['currentUser'] as const,
}

function createIdempotencyKey() {
  return window.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

export function useCurrentUserQuery(enabled: boolean) {
  return useQuery({
    queryKey: authQueryKeys.currentUser,
    queryFn: authApi.getMe,
    enabled,
    retry: false,
  })
}

export function useLoginMutation() {
  const setAuthenticated = useAuthStore((state) => state.setAuthenticated)

  return useMutation({
    mutationFn: (payload: LoginRequest) => authApi.login(payload, createIdempotencyKey()),
    onSuccess: (data) => {
      setAuthenticated(data.user, data.access_token)
    },
  })
}

export function useRegisterMutation() {
  const setAuthenticated = useAuthStore((state) => state.setAuthenticated)

  return useMutation({
    mutationFn: (payload: RegisterRequest) => authApi.register(payload, createIdempotencyKey()),
    onSuccess: (data) => {
      setAuthenticated(data.user, data.access_token)
    },
  })
}

export function useLogoutMutation() {
  const setUnauthenticated = useAuthStore((state) => state.setUnauthenticated)

  return useMutation({
    mutationFn: () => authApi.logout(createIdempotencyKey()),
    onSettled: () => {
      setUnauthenticated()
    },
  })
}
