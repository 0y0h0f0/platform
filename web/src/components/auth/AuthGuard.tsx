import { Navigate, Outlet, useLocation } from 'react-router-dom'

import { PageSkeleton } from '@/components/layout/PageSkeleton'
import { useAuthBootstrap } from '@/hooks/useAuth'

export function AuthGuard() {
  const location = useLocation()
  const { status } = useAuthBootstrap()

  if (status === 'loading') {
    return <PageSkeleton />
  }

  if (status === 'unauthenticated') {
    return <Navigate replace state={{ from: location }} to="/login" />
  }

  return <Outlet />
}
