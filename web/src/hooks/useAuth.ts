import { useEffect } from 'react'

import { useCurrentUserQuery } from '@/queries/auth.queries'
import { useAuthStore } from '@/stores/auth.store'

export function useAuth() {
  return useAuthStore()
}

export function useAuthBootstrap() {
  const status = useAuthStore((state) => state.status)
  const accessToken = useAuthStore((state) => state.accessToken)
  const hydrate = useAuthStore((state) => state.hydrate)
  const setUser = useAuthStore((state) => state.setUser)
  const setUnauthenticated = useAuthStore((state) => state.setUnauthenticated)

  useEffect(() => {
    hydrate()
  }, [hydrate])

  const currentUserQuery = useCurrentUserQuery(status === 'loading' && Boolean(accessToken))

  useEffect(() => {
    if (status === 'loading' && !accessToken) {
      setUnauthenticated()
    }
  }, [accessToken, setUnauthenticated, status])

  useEffect(() => {
    if (currentUserQuery.data?.user) {
      setUser(currentUserQuery.data.user)
    }
  }, [currentUserQuery.data, setUser])

  useEffect(() => {
    if (currentUserQuery.isError) {
      setUnauthenticated()
    }
  }, [currentUserQuery.isError, setUnauthenticated])

  return {
    accessToken,
    isChecking: status === 'loading',
    status,
    user: useAuthStore((state) => state.user),
  }
}
