import axios, { AxiosHeaders, type AxiosRequestConfig } from 'axios'

import type { ApiEnvelope } from './types'
import { AppError } from '@/utils/error'
import { clearToken, getToken } from '@/utils/token'

declare module 'axios' {
  export interface AxiosRequestConfig {
    idempotencyKey?: string
    skipAuth?: boolean
  }
}

const writeMethods = new Set(['post', 'put', 'delete'])

function createRequestId(): string {
  return window.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

export const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 10_000,
})

apiClient.interceptors.request.use((config) => {
  const headers = AxiosHeaders.from(config.headers)
  const token = getToken()

  if (token && !config.skipAuth) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  headers.set('X-Request-Id', createRequestId())

  const method = config.method?.toLowerCase()
  if (method && writeMethods.has(method) && config.idempotencyKey) {
    headers.set('Idempotency-Key', config.idempotencyKey)
  }

  config.headers = headers
  return config
})

apiClient.interceptors.response.use(
  (response) => {
    const envelope = response.data as ApiEnvelope

    if (!envelope || typeof envelope !== 'object' || !('code' in envelope)) {
      return response.data
    }

    if (envelope.code !== 'OK') {
      throw new AppError(envelope.code, envelope.message, envelope.request_id, envelope.details)
    }

    return envelope.data ?? null
  },
  (error: unknown) => {
    if (!axios.isAxiosError(error)) {
      return Promise.reject(error)
    }

    if (error.response?.status === 401) {
      clearToken()
      window.dispatchEvent(new CustomEvent('auth:expired'))
    }

    const envelope = error.response?.data as Partial<ApiEnvelope> | undefined
    if (envelope?.code) {
      return Promise.reject(
        new AppError(
          envelope.code,
          envelope.message ?? '请求失败',
          envelope.request_id,
          envelope.details,
        ),
      )
    }

    if (error.code === 'ECONNABORTED') {
      return Promise.reject(new AppError('DEADLINE_EXCEEDED', '请求超时'))
    }

    return Promise.reject(new AppError('NETWORK_ERROR', '网络连接失败'))
  },
)

export function request<T>(config: AxiosRequestConfig): Promise<T> {
  return apiClient.request<unknown, T>(config)
}
