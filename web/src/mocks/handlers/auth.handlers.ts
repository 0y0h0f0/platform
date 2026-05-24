import { http, HttpResponse } from 'msw'

import type { ApiEnvelope, AuthData, LoginRequest, RegisterRequest } from '@/api/types'
import { findMockUser, mockUser } from '../fixtures/users'

const mockAccessToken = 'mock-access-token'

function getMockUserByAccount(account: string | undefined) {
  if (account === 'admin_user' || account === 'admin@example.com') {
    return findMockUser('user-2') ?? mockUser
  }
  if (account === 'member_user' || account === 'member@example.com') {
    return findMockUser('user-3') ?? mockUser
  }
  return mockUser
}

function createMockAccessToken(userId: string) {
  return userId === mockUser.id ? mockAccessToken : `${mockAccessToken}:${userId}`
}

function getUserFromAuthorization(authorization: string | null) {
  const token = authorization?.replace(/^Bearer\s+/i, '') ?? ''
  const userId = token.includes(':') ? token.split(':').at(-1) : mockUser.id
  return findMockUser(userId || '') ?? mockUser
}

function ok<T>(data: T, status = 200) {
  return HttpResponse.json<ApiEnvelope<T>>(
    {
      code: 'OK',
      message: 'ok',
      request_id: 'mock-request-id',
      data,
    },
    { status },
  )
}

function error(code: string, message: string, status: number) {
  return HttpResponse.json<ApiEnvelope>(
    {
      code,
      message,
      request_id: 'mock-request-id',
    },
    { status },
  )
}

export const authHandlers = [
  http.post('*/api/v1/auth/register', async ({ request }) => {
    const payload = (await request.json()) as Partial<RegisterRequest>

    if (!payload.username || !payload.email || !payload.password) {
      return error('INVALID_ARGUMENT', 'invalid request body', 400)
    }

    const user = {
      ...mockUser,
      username: payload.username,
      email: payload.email,
      nickname: payload.username,
    }

    return ok<AuthData>({ access_token: mockAccessToken, user }, 201)
  }),

  http.post('*/api/v1/auth/login', async ({ request }) => {
    const payload = (await request.json()) as Partial<LoginRequest>

    if (!payload.account || !payload.password) {
      return error('INVALID_ARGUMENT', 'invalid request body', 400)
    }

    const user = getMockUserByAccount(payload.account)

    return ok<AuthData>({ access_token: createMockAccessToken(user.id), user })
  }),

  http.post('*/api/v1/auth/logout', () => ok(null)),

  http.get('*/api/v1/users/me', ({ request }) => {
    const authorization = request.headers.get('authorization')
    if (!authorization) {
      return error('UNAUTHENTICATED', 'missing token', 401)
    }

    return ok({ user: getUserFromAuthorization(authorization) })
  }),
]
