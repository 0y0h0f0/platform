import '@testing-library/jest-dom/vitest'
import { afterAll, afterEach, beforeAll } from 'vitest'

import { useAuthStore } from '../src/stores/auth.store'
import { clearToken } from '../src/utils/token'
import { server } from '../src/mocks/server'
import { resetMockProjects } from '../src/mocks/handlers/project.handlers'
import { resetMockTasks } from '../src/mocks/handlers/task.handlers'
import { resetMockComments } from '../src/mocks/handlers/comment.handlers'

const getComputedStyle = window.getComputedStyle.bind(window)
Object.defineProperty(window, 'getComputedStyle', {
  writable: true,
  value: (element: Element) => getComputedStyle(element),
})

Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => undefined,
    removeListener: () => undefined,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
    dispatchEvent: () => false,
  }),
})

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => {
  server.resetHandlers()
  resetMockProjects()
  resetMockTasks()
  resetMockComments()
  clearToken()
  useAuthStore.setState({ accessToken: null, status: 'unauthenticated', user: null })
})
afterAll(() => server.close())
