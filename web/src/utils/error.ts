import type { ApiCode, FieldDetail } from '@/api/types'

// errorMessages maps stable backend codes to user-facing fallback messages.
const errorMessages: Partial<Record<ApiCode, string>> = {
  UNAUTHENTICATED: '登录已过期，请重新登录',
  PERMISSION_DENIED: '没有权限执行此操作',
  NOT_FOUND: '资源不存在',
  FAILED_PRECONDITION: '当前状态不允许此操作',
  INVALID_ARGUMENT: '请求参数有误',
  ALREADY_EXISTS: '资源已存在',
  ABORTED: '数据已被他人修改，请刷新后重试',
  RESOURCE_EXHAUSTED: '请求过于频繁，请稍后重试',
  INTERNAL: '服务器内部错误，请稍后重试',
  UNAVAILABLE: '服务暂不可用，请稍后重试',
  DEADLINE_EXCEEDED: '请求超时，请稍后重试',
  NETWORK_ERROR: '网络连接失败，请检查网络',
}

// AppError preserves backend code and request ID after Axios unwraps responses.
export class AppError extends Error {
  readonly code: ApiCode | string
  readonly requestId?: string
  readonly details?: FieldDetail[]

  constructor(
    code: ApiCode | string,
    message: string,
    requestId?: string,
    details?: FieldDetail[],
  ) {
    super(message)
    this.name = 'AppError'
    this.code = code
    this.requestId = requestId
    this.details = details
  }
}

// getErrorMessage normalizes unknown errors for components and mutations.
export function getErrorMessage(error: unknown): string {
  if (error instanceof AppError) {
    return errorMessages[error.code as ApiCode] ?? error.message
  }

  if (error instanceof Error) {
    return error.message
  }

  return '操作失败，请稍后重试'
}
