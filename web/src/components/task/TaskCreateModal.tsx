import { App, Form, Input, Modal } from 'antd'

import type { Task } from '@/api/types'
import { useCreateTaskMutation } from '@/queries/task.queries'

const { TextArea } = Input

interface TaskCreateFormValues {
  content?: string
  title: string
}

interface TaskCreateModalProps {
  onClose: () => void
  onCreated?: (task: Task) => void
  open: boolean
  projectId: string
}

export function TaskCreateModal({ onClose, onCreated, open, projectId }: TaskCreateModalProps) {
  const { message } = App.useApp()
  const [form] = Form.useForm<TaskCreateFormValues>()
  const createTaskMutation = useCreateTaskMutation(projectId)

  const handleSubmit = (values: TaskCreateFormValues) => {
    createTaskMutation.mutate(
      {
        content: values.content?.trim() ?? '',
        title: values.title.trim(),
      },
      {
        onSuccess: (data) => {
          message.success('任务已创建')
          form.resetFields()
          onCreated?.(data.task)
          onClose()
        },
        onError: (error) => {
          message.error(error instanceof Error ? error.message : '创建任务失败')
        },
      },
    )
  }

  const handleCancel = () => {
    form.resetFields()
    onClose()
  }

  return (
    <Modal
      confirmLoading={createTaskMutation.isPending}
      okText="创建"
      onCancel={handleCancel}
      onOk={() => form.submit()}
      open={open}
      title="创建任务"
    >
      <Form form={form} layout="vertical" onFinish={handleSubmit}>
        <Form.Item
          label="任务标题"
          name="title"
          rules={[
            { required: true, message: '请输入任务标题' },
            { max: 120, message: '任务标题不能超过 120 个字符' },
          ]}
        >
          <Input autoFocus maxLength={120} placeholder="例如：完成看板筛选联调" />
        </Form.Item>

        <Form.Item
          label="任务内容"
          name="content"
          rules={[{ max: 2000, message: '任务内容不能超过 2000 个字符' }]}
        >
          <TextArea
            autoSize={{ minRows: 4, maxRows: 8 }}
            maxLength={2000}
            placeholder="补充任务背景、验收标准或注意事项"
            showCount
          />
        </Form.Item>
      </Form>
    </Modal>
  )
}
