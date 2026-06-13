import { DeleteOutlined } from '@ant-design/icons'
import { Avatar, Button, Popconfirm, Space, Typography } from 'antd'

import type { TaskComment } from '@/api/types'

const { Paragraph, Text } = Typography

interface CommentItemProps {
  canDelete?: boolean
  comment: TaskComment
  deleting?: boolean
  onDelete?: (commentId: string) => void
}

function getDisplayName(comment: TaskComment) {
  return comment.nickname || comment.username || comment.user_id
}

function getAvatarText(comment: TaskComment) {
  return getDisplayName(comment).slice(0, 1).toUpperCase()
}

export function CommentItem({
  canDelete = false,
  comment,
  deleting = false,
  onDelete,
}: CommentItemProps) {
  const displayName = getDisplayName(comment)

  return (
    <article className="comment-item">
      <Avatar className="comment-avatar" src={comment.avatar_url || undefined}>
        {getAvatarText(comment)}
      </Avatar>
      <div className="comment-body">
        <Space className="comment-header" size={8} wrap>
          <Text strong>{displayName}</Text>
        </Space>
        <Paragraph className="comment-content">{comment.content}</Paragraph>
      </div>
      {canDelete ? (
        <Popconfirm
          cancelText="取消"
          okButtonProps={{ danger: true, loading: deleting }}
          okText="删除"
          onConfirm={() => onDelete?.(comment.id)}
          title="删除评论"
        >
          <Button
            aria-label={`删除评论 ${comment.id}`}
            danger
            disabled={deleting}
            icon={<DeleteOutlined aria-hidden />}
            size="small"
            type="text"
          />
        </Popconfirm>
      ) : null}
    </article>
  )
}
