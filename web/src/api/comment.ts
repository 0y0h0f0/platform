import { request } from './client'
import type {
  CreateCommentRequest,
  CreateCommentData,
  DeleteCommentData,
  ListCommentsRequest,
  ListCommentsData,
} from './types'

export function createComment(
  taskId: string,
  payload: CreateCommentRequest,
  idempotencyKey?: string,
) {
  return request<CreateCommentData>({
    url: `/tasks/${taskId}/comments`,
    method: 'POST',
    data: payload,
    idempotencyKey,
  })
}

export function listComments(taskId: string, params: ListCommentsRequest) {
  return request<ListCommentsData>({
    url: `/tasks/${taskId}/comments`,
    method: 'GET',
    params,
  })
}

export function deleteComment(taskId: string, commentId: string, idempotencyKey?: string) {
  return request<DeleteCommentData>({
    url: `/tasks/${taskId}/comments/${commentId}`,
    method: 'DELETE',
    idempotencyKey,
  })
}
