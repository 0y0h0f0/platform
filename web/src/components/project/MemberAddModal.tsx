import { App, Form, Input, Modal, Select } from 'antd'
import { useEffect, useMemo } from 'react'

import { useAddProjectMemberMutation } from '@/queries/member.queries'
import { Role, RoleLabels, type Role as RoleValue } from '@/utils/constants'
import { getErrorMessage } from '@/utils/error'

interface MemberAddFormValues {
  role: RoleValue
  userId: string
}

interface MemberAddModalProps {
  canAddAdmin?: boolean
  onClose: () => void
  open: boolean
  projectId: string
}

export function MemberAddModal({
  canAddAdmin = false,
  onClose,
  open,
  projectId,
}: MemberAddModalProps) {
  const { message } = App.useApp()
  const [form] = Form.useForm<MemberAddFormValues>()
  const addMemberMutation = useAddProjectMemberMutation(projectId)
  const roleOptions = useMemo(
    () => [
      ...(canAddAdmin ? [{ label: RoleLabels[Role.Admin], value: Role.Admin }] : []),
      { label: RoleLabels[Role.Member], value: Role.Member },
    ],
    [canAddAdmin],
  )

  useEffect(() => {
    if (open) {
      form.setFieldsValue({ role: Role.Member })
    }
  }, [form, open])

  const handleSubmit = (values: MemberAddFormValues) => {
    addMemberMutation.mutate(
      {
        role: values.role,
        user_id: values.userId.trim(),
      },
      {
        onError: (error) => {
          message.error(getErrorMessage(error))
        },
        onSuccess: () => {
          message.success('成员已添加')
          form.resetFields()
          onClose()
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
      confirmLoading={addMemberMutation.isPending}
      okText="添加"
      onCancel={handleCancel}
      onOk={() => form.submit()}
      open={open}
      title="添加成员"
    >
      <Form form={form} layout="vertical" onFinish={handleSubmit}>
        <Form.Item
          label="用户 ID"
          name="userId"
          rules={[
            { required: true, message: '请输入用户 ID' },
            { max: 64, message: '用户 ID 不能超过 64 个字符' },
          ]}
        >
          <Input autoFocus maxLength={64} placeholder="例如：user-4" />
        </Form.Item>

        <Form.Item label="角色" name="role" rules={[{ required: true, message: '请选择角色' }]}>
          <Select<RoleValue> options={roleOptions} />
        </Form.Item>
      </Form>
    </Modal>
  )
}
