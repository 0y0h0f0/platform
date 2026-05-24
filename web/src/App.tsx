import { lazy, Suspense, type ReactNode } from 'react'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'

import { AuthGuard } from '@/components/auth/AuthGuard'
import { GuestGuard } from '@/components/auth/GuestGuard'
import { ErrorBoundary } from '@/components/common/ErrorBoundary'
import { AppLayout } from '@/components/layout/AppLayout'
import { PageSkeleton } from '@/components/layout/PageSkeleton'

const LoginPage = lazy(() => import('@/pages/auth/LoginPage'))
const RegisterPage = lazy(() => import('@/pages/auth/RegisterPage'))
const ProjectListPage = lazy(() => import('@/pages/projects/ProjectListPage'))
const ProjectDetailPage = lazy(() => import('@/pages/projects/ProjectDetailPage'))
const NotFoundPage = lazy(() => import('@/pages/NotFoundPage'))

function RouteElement({ children }: { children: ReactNode }) {
  return (
    <Suspense fallback={<PageSkeleton />}>
      <ErrorBoundary>{children}</ErrorBoundary>
    </Suspense>
  )
}

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<GuestGuard />}>
          <Route
            element={
              <RouteElement>
                <LoginPage />
              </RouteElement>
            }
            path="/login"
          />
          <Route
            element={
              <RouteElement>
                <RegisterPage />
              </RouteElement>
            }
            path="/register"
          />
        </Route>

        <Route element={<AuthGuard />}>
          <Route element={<AppLayout />}>
            <Route index element={<Navigate replace to="/projects" />} />
            <Route
              element={
                <RouteElement>
                  <ProjectListPage />
                </RouteElement>
              }
              path="/projects"
            />
            <Route
              element={
                <RouteElement>
                  <ProjectDetailPage />
                </RouteElement>
              }
              path="/projects/:id"
            />
          </Route>
        </Route>

        <Route
          element={
            <RouteElement>
              <NotFoundPage />
            </RouteElement>
          }
          path="*"
        />
      </Routes>
    </BrowserRouter>
  )
}

export default App
