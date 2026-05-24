import { App, Alert, Button, Space, Typography } from 'antd'

import { EmptyState } from '@/components/common/EmptyState'
import { LoadingSpinner } from '@/components/common/LoadingSpinner'
import { useCommentsQuery, useDeleteCommentMutation } from '@/queries/comment.queries'
import { Role, type Role as RoleValue } from '@/utils/constants'
import { getErrorMessage } from '@/utils/error'
import { CommentItem } from './CommentItem'

const { Text } = Typography

interface CommentListProps {
  currentRole?: RoleValue | null
  currentUserId?: string | null
  limit?: number
  readOnly?: boolean
  taskId: string
}

function canDeleteComment(
  commentUserId: string,
  currentRole: RoleValue | null | undefined,
  currentUserId: string | null | undefined,
  readOnly: boolean,
) {
  if (readOnly) {
    return false
  }

  return commentUserId === currentUserId || currentRole === Role.Owner || currentRole === Role.Admin
}

export function CommentList({
  currentRole,
  currentUserId,
  limit = 20,
  readOnly = false,
  taskId,
}: CommentListProps) {
  const { message } = App.useApp()
  const commentsQuery = useCommentsQuery({ limit, taskId })
  const deleteCommentMutation = useDeleteCommentMutation(taskId)
  const comments = commentsQuery.data?.pages.flatMap((page) => page.comments) ?? []
  const deletingCommentId = deleteCommentMutation.variables

  const handleDelete = (commentId: string) => {
    deleteCommentMutation.mutate(commentId, {
      onError: (error) => {
        message.error(getErrorMessage(error))
      },
      onSuccess: () => {
        message.success('评论已删除')
      },
    })
  }

  if (commentsQuery.isLoading) {
    return <LoadingSpinner tip="正在加载评论" />
  }

  if (commentsQuery.isError) {
    return (
      <Alert
        action={
          <Button onClick={() => commentsQuery.refetch()} size="small">
            重试
          </Button>
        }
        description={getErrorMessage(commentsQuery.error)}
        message="评论加载失败"
        showIcon
        type="error"
      />
    )
  }

  return (
    <Space className="comment-list" direction="vertical" size={12}>
      <Text type="secondary">共 {comments.length} 条评论</Text>
      {comments.length > 0 ? (
        comments.map((comment) => (
          <CommentItem
            canDelete={canDeleteComment(comment.user_id, currentRole, currentUserId, readOnly)}
            comment={comment}
            deleting={deletingCommentId === comment.id && deleteCommentMutation.isPending}
            key={comment.id}
            onDelete={handleDelete}
          />
        ))
      ) : (
        <EmptyState title="暂无评论" />
      )}

      {commentsQuery.hasNextPage ? (
        <div className="comment-load-more">
          <Button
            loading={commentsQuery.isFetchingNextPage}
            onClick={() => commentsQuery.fetchNextPage()}
          >
            加载更多
          </Button>
        </div>
      ) : null}
    </Space>
  )
}
