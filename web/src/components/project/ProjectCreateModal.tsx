import { App, Form, Input, Modal } from 'antd'

import type { Project } from '@/api/types'
import { useCreateProjectMutation } from '@/queries/project.queries'

const { TextArea } = Input

interface ProjectCreateFormValues {
  description?: string
  name: string
}

interface ProjectCreateModalProps {
  onClose: () => void
  onCreated?: (project: Project) => void
  open: boolean
}

export function ProjectCreateModal({ onClose, onCreated, open }: ProjectCreateModalProps) {
  const { message } = App.useApp()
  const [form] = Form.useForm<ProjectCreateFormValues>()
  const createProjectMutation = useCreateProjectMutation()

  const handleSubmit = (values: ProjectCreateFormValues) => {
    createProjectMutation.mutate(
      {
        description: values.description?.trim() ?? '',
        name: values.name.trim(),
      },
      {
        onSuccess: (data) => {
          message.success('项目已创建')
          form.resetFields()
          onCreated?.(data.project)
          onClose()
        },
        onError: (error) => {
          message.error(error instanceof Error ? error.message : '创建项目失败')
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
      confirmLoading={createProjectMutation.isPending}
      okText="创建"
      onCancel={handleCancel}
      onOk={() => form.submit()}
      open={open}
      title="创建项目"
    >
      <Form form={form} layout="vertical" onFinish={handleSubmit}>
        <Form.Item
          label="项目名称"
          name="name"
          rules={[
            { required: true, message: '请输入项目名称' },
            { max: 64, message: '项目名称不能超过 64 个字符' },
          ]}
        >
          <Input autoFocus maxLength={64} placeholder="例如：移动端看板改版" />
        </Form.Item>

        <Form.Item
          label="项目描述"
          name="description"
          rules={[{ max: 500, message: '项目描述不能超过 500 个字符' }]}
        >
          <TextArea
            autoSize={{ minRows: 4, maxRows: 8 }}
            maxLength={500}
            placeholder="补充项目目标、范围或协作说明"
            showCount
          />
        </Form.Item>
      </Form>
    </Modal>
  )
}
