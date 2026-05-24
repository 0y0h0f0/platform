import { SaveOutlined } from '@ant-design/icons'
import { App, Button, DatePicker, Form, Input, Select } from 'antd'
import dayjs, { type Dayjs } from 'dayjs'
import { useEffect } from 'react'

import type { Task } from '@/api/types'
import { useUpdateTaskMutation } from '@/queries/task.queries'
import { Priority, PriorityLabels, type Priority as PriorityValue } from '@/utils/constants'
import { AppError, getErrorMessage } from '@/utils/error'

const { TextArea } = Input

interface TaskEditFormValues {
  content?: string
  dueTime?: Dayjs | null
  priority: PriorityValue
  title: string
}

interface TaskEditFormProps {
  disabled?: boolean
  onConflict?: () => void
  onUpdated?: (task: Task) => void
  task: Task
}

const priorityOptions = [Priority.Low, Priority.Normal, Priority.High, Priority.Urgent].map(
  (priority) => ({
    label: PriorityLabels[priority],
    value: priority,
  }),
)

function taskToFormValues(task: Task): TaskEditFormValues {
  return {
    content: task.content,
    dueTime: task.due_time ? dayjs(task.due_time) : null,
    priority: task.priority,
    title: task.title,
  }
}

function isVersionConflict(error: unknown) {
  return error instanceof AppError && error.code === 'ABORTED'
}

export function TaskEditForm({ disabled = false, onConflict, onUpdated, task }: TaskEditFormProps) {
  const { message } = App.useApp()
  const [form] = Form.useForm<TaskEditFormValues>()
  const updateTaskMutation = useUpdateTaskMutation()

  useEffect(() => {
    form.setFieldsValue(taskToFormValues(task))
  }, [form, task])

  const handleSubmit = (values: TaskEditFormValues) => {
    if (disabled) {
      return
    }

    updateTaskMutation.mutate(
      {
        task,
        values: {
          content: values.content?.trim() ?? '',
          due_time: values.dueTime?.toISOString() ?? '',
          priority: values.priority,
          title: values.title.trim(),
        },
      },
      {
        onError: (error) => {
          if (isVersionConflict(error)) {
            message.warning('任务已被他人修改，已刷新最新数据')
            onConflict?.()
            return
          }

          message.error(getErrorMessage(error))
        },
        onSuccess: (data) => {
          message.success('任务已保存')
          form.setFieldsValue(taskToFormValues(data.task))
          onUpdated?.(data.task)
        },
      },
    )
  }

  return (
    <Form
      className="task-edit-form"
      disabled={disabled}
      form={form}
      initialValues={taskToFormValues(task)}
      layout="vertical"
      onFinish={handleSubmit}
    >
      <Form.Item
        label="任务标题"
        name="title"
        rules={[
          { required: true, message: '请输入任务标题' },
          { max: 120, message: '任务标题不能超过 120 个字符' },
        ]}
      >
        <Input maxLength={120} />
      </Form.Item>

      <Form.Item
        label="任务内容"
        name="content"
        rules={[{ max: 2000, message: '任务内容不能超过 2000 个字符' }]}
      >
        <TextArea autoSize={{ minRows: 5, maxRows: 10 }} maxLength={2000} showCount />
      </Form.Item>

      <div className="task-edit-grid">
        <Form.Item label="优先级" name="priority" rules={[{ required: true }]}>
          <Select options={priorityOptions} />
        </Form.Item>

        <Form.Item label="截止时间" name="dueTime">
          <DatePicker className="task-edit-date-picker" showTime />
        </Form.Item>
      </div>

      <div className="task-edit-actions">
        <Button
          disabled={disabled}
          htmlType="submit"
          icon={<SaveOutlined aria-hidden />}
          loading={updateTaskMutation.isPending}
          type="primary"
        >
          保存
        </Button>
      </div>
    </Form>
  )
}
