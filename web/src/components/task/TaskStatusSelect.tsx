import { App, Select, Typography } from 'antd'

import type { Task } from '@/api/types'
import { useStatusTransitions } from '@/hooks/useStatusTransitions'
import { useChangeTaskStatusMutation } from '@/queries/task.queries'
import { AppError, getErrorMessage } from '@/utils/error'
import { TaskStatusLabels, type TaskStatus } from '@/utils/constants'

const { Text } = Typography

interface TaskStatusSelectProps {
  disabled?: boolean
  onChanged?: (task: Task) => void
  onConflict?: () => void
  task: Task
}

function isVersionConflict(error: unknown) {
  return error instanceof AppError && error.code === 'ABORTED'
}

export function TaskStatusSelect({
  disabled = false,
  onChanged,
  onConflict,
  task,
}: TaskStatusSelectProps) {
  const { message } = App.useApp()
  const transitions = useStatusTransitions(task.status)
  const changeStatusMutation = useChangeTaskStatusMutation()
  const options = [task.status, ...transitions].map((status) => ({
    label: TaskStatusLabels[status],
    value: status,
  }))

  const handleChange = (status: TaskStatus) => {
    if (status === task.status) {
      return
    }

    changeStatusMutation.mutate(
      { status, task },
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
          message.success('状态已更新')
          onChanged?.(data.task)
        },
      },
    )
  }

  return (
    <label className="task-detail-control">
      <Text type="secondary">状态</Text>
      <Select
        aria-label="任务状态"
        disabled={disabled || changeStatusMutation.isPending}
        loading={changeStatusMutation.isPending}
        onChange={handleChange}
        options={options}
        value={task.status}
      />
    </label>
  )
}
