import { request } from './client'
import type { AuthData, GetMeData, LoginRequest, RegisterRequest } from './types'

export function register(payload: RegisterRequest, idempotencyKey?: string) {
  return request<AuthData>({
    url: '/auth/register',
    method: 'POST',
    data: payload,
    idempotencyKey,
    skipAuth: true,
  })
}

export function login(payload: LoginRequest, idempotencyKey?: string) {
  return request<AuthData>({
    url: '/auth/login',
    method: 'POST',
    data: payload,
    idempotencyKey,
    skipAuth: true,
  })
}

export function logout(idempotencyKey?: string) {
  return request<null>({
    url: '/auth/logout',
    method: 'POST',
    idempotencyKey,
  })
}

export function getMe() {
  return request<GetMeData>({
    url: '/users/me',
    method: 'GET',
  })
}
