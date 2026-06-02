import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query'

import * as commentApi from '@/api/comment'
import type { CreateCommentRequest } from '@/api/types'

// CommentListParams controls forward pagination for a task's comments.
export interface CommentListParams {
  limit: number
  taskId: string
}

// commentQueryKeys scopes comment caches under their task ID.
export const commentQueryKeys = {
  all: ['comments'] as const,
  task: (taskId: string) => ['tasks', taskId, 'comments'] as const,
}

function createIdempotencyKey() {
  // Comment create/delete requests can be safely retried by the gateway.
  return window.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

// useCommentsQuery supports both backend next_cursor and legacy after_id paging
// used by the mock/test layer.
export function useCommentsQuery(params: CommentListParams) {
  return useInfiniteQuery({
    queryKey: commentQueryKeys.task(params.taskId),
    queryFn: ({ pageParam }) =>
      commentApi.listComments(params.taskId, {
        after_id: pageParam || undefined,
        limit: params.limit,
      }),
    enabled: Boolean(params.taskId),
    getNextPageParam: (lastPage) => {
      if (lastPage.next_cursor) {
        return lastPage.next_cursor
      }

      if (lastPage.comments.length < params.limit) {
        return undefined
      }

      return lastPage.comments.at(-1)?.id
    },
    initialPageParam: '',
  })
}

export function useCreateCommentMutation(taskId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: CreateCommentRequest) =>
      commentApi.createComment(taskId, payload, createIdempotencyKey()),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: commentQueryKeys.task(taskId) })
    },
  })
}

export function useDeleteCommentMutation(taskId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (commentId: string) =>
      commentApi.deleteComment(taskId, commentId, createIdempotencyKey()),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: commentQueryKeys.task(taskId) })
    },
  })
}
