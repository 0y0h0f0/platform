import { App, Select, Typography } from 'antd'

import type { ProjectMember, Task } from '@/api/types'
import { useAssignTaskMutation } from '@/queries/task.queries'
import { RoleLabels } from '@/utils/constants'
import { AppError, getErrorMessage } from '@/utils/error'

const { Text } = Typography

interface TaskAssignSelectProps {
  disabled?: boolean
  members: ProjectMember[]
  onChanged?: (task: Task) => void
  onConflict?: () => void
  task: Task
}

function isVersionConflict(error: unknown) {
  return error instanceof AppError && error.code === 'ABORTED'
}

export function TaskAssignSelect({
  disabled = false,
  members,
  onChanged,
  onConflict,
  task,
}: TaskAssignSelectProps) {
  const { message } = App.useApp()
  const assignTaskMutation = useAssignTaskMutation()
  const options = members.map((member) => ({
    label: `${member.user_id} · ${RoleLabels[member.role]}`,
    value: member.user_id,
  }))

  const handleChange = (assigneeId: string) => {
    if (assigneeId === task.assignee_id) {
      return
    }

    assignTaskMutation.mutate(
      { assigneeId, task },
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
          message.success('负责人已更新')
          onChanged?.(data.task)
        },
      },
    )
  }

  return (
    <label className="task-detail-control">
      <Text type="secondary">负责人</Text>
      <Select
        aria-label="负责人"
        disabled={disabled || assignTaskMutation.isPending}
        loading={assignTaskMutation.isPending}
        onChange={handleChange}
        optionFilterProp="label"
        options={options}
        placeholder="未指派"
        showSearch
        value={task.assignee_id || undefined}
      />
    </label>
  )
}
