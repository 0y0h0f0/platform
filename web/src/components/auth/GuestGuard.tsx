import { Navigate, Outlet } from 'react-router-dom'

import { PageSkeleton } from '@/components/layout/PageSkeleton'
import { useAuthBootstrap } from '@/hooks/useAuth'

export function GuestGuard() {
  const { status } = useAuthBootstrap()

  if (status === 'loading') {
    return <PageSkeleton />
  }

  if (status === 'authenticated') {
    return <Navigate replace to="/projects" />
  }

  return <Outlet />
}
