import { SendOutlined } from '@ant-design/icons'
import { App, Button, Form, Input } from 'antd'

import type { TaskComment } from '@/api/types'
import { useCreateCommentMutation } from '@/queries/comment.queries'
import { getErrorMessage } from '@/utils/error'

const { TextArea } = Input

interface CommentInputValues {
  content: string
}

interface CommentInputProps {
  disabled?: boolean
  onCreated?: (comment: TaskComment) => void
  taskId: string
}

export function CommentInput({ disabled = false, onCreated, taskId }: CommentInputProps) {
  const { message } = App.useApp()
  const [form] = Form.useForm<CommentInputValues>()
  const createCommentMutation = useCreateCommentMutation(taskId)

  const handleSubmit = (values: CommentInputValues) => {
    const content = values.content.trim()
    if (!content || disabled) {
      return
    }

    createCommentMutation.mutate(
      { content },
      {
        onError: (error) => {
          message.error(getErrorMessage(error))
        },
        onSuccess: (data) => {
          message.success('评论已发送')
          form.resetFields()
          onCreated?.(data.comment)
        },
      },
    )
  }

  return (
    <Form className="comment-input" form={form} layout="vertical" onFinish={handleSubmit}>
      <Form.Item
        name="content"
        rules={[
          { required: true, message: '请输入评论内容' },
          { max: 2000, message: '评论内容不能超过 2000 个字符' },
        ]}
      >
        <TextArea
          aria-label="评论内容"
          autoSize={{ minRows: 3, maxRows: 6 }}
          disabled={disabled || createCommentMutation.isPending}
          maxLength={2000}
          placeholder={disabled ? '当前不能评论' : '写下评论'}
          showCount
        />
      </Form.Item>
      <div className="comment-input-actions">
        <Button
          disabled={disabled}
          htmlType="submit"
          icon={<SendOutlined aria-hidden />}
          loading={createCommentMutation.isPending}
          type="primary"
        >
          发送
        </Button>
      </div>
    </Form>
  )
}
