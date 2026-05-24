import { App, Form, Input, Modal } from 'antd'
import { useEffect } from 'react'

import type { Project } from '@/api/types'
import { useUpdateProjectMutation } from '@/queries/project.queries'
import { getErrorMessage } from '@/utils/error'

const { TextArea } = Input

interface ProjectEditFormValues {
  description?: string
  name: string
}

interface ProjectEditModalProps {
  onClose: () => void
  onUpdated?: (project: Project) => void
  open: boolean
  project: Project
}

export function ProjectEditModal({ onClose, onUpdated, open, project }: ProjectEditModalProps) {
  const { message } = App.useApp()
  const [form] = Form.useForm<ProjectEditFormValues>()
  const updateProjectMutation = useUpdateProjectMutation()

  useEffect(() => {
    if (open) {
      form.setFieldsValue({
        description: project.description,
        name: project.name,
      })
    }
  }, [form, open, project.description, project.name])

  const handleSubmit = (values: ProjectEditFormValues) => {
    updateProjectMutation.mutate(
      {
        project,
        values: {
          description: values.description?.trim() ?? '',
          name: values.name.trim(),
        },
      },
      {
        onError: (error) => {
          message.error(getErrorMessage(error))
        },
        onSuccess: (data) => {
          message.success('项目已更新')
          onUpdated?.(data.project)
          onClose()
        },
      },
    )
  }

  const handleCancel = () => {
    form.setFieldsValue({
      description: project.description,
      name: project.name,
    })
    onClose()
  }

  return (
    <Modal
      confirmLoading={updateProjectMutation.isPending}
      okText="保存"
      onCancel={handleCancel}
      onOk={() => form.submit()}
      open={open}
      title="编辑项目"
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
