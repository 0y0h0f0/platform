import { Button, Card, Space, Tag, Typography } from 'antd'
import type { DragEvent } from 'react'

import type { Task } from '@/api/types'
import { PriorityTag } from '@/components/common/PriorityTag'
import { useStatusTransitions } from '@/hooks/useStatusTransitions'
import { TaskStatusColors, TaskStatusLabels, type TaskStatus } from '@/utils/constants'
import { formatDateTime } from '@/utils/time'

const { Paragraph, Text } = Typography

interface TaskCardProps {
  disabled?: boolean
  onChangeStatus: (task: Task, status: TaskStatus) => void
  onDragStart?: (task: Task, event: DragEvent<HTMLElement>) => void
  onOpen?: (task: Task) => void
  task: Task
}

export function TaskCard({
  disabled = false,
  onChangeStatus,
  onDragStart,
  onOpen,
  task,
}: TaskCardProps) {
  const transitions = useStatusTransitions(task.status)
  const title = onOpen ? (
    <button className="task-card-title-button" onClick={() => onOpen(task)} type="button">
      {task.title}
    </button>
  ) : (
    <span className="task-card-title">{task.title}</span>
  )

  const handleDragStart = (event: DragEvent<HTMLElement>) => {
    if (disabled) {
      event.preventDefault()
      return
    }

    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/plain', task.id)
    event.dataTransfer.setData('application/x-task-id', task.id)
    onDragStart?.(task, event)
  }

  return (
    <Card
      className="task-card"
      draggable={!disabled}
      extra={<PriorityTag priority={task.priority} />}
      onDragStart={handleDragStart}
      size="small"
      title={title}
    >
      <Paragraph className="task-card-content" type={task.content ? undefined : 'secondary'}>
        {task.content || '暂无内容'}
      </Paragraph>

      <div className="task-card-meta">
        <Tag color={TaskStatusColors[task.status]}>{TaskStatusLabels[task.status]}</Tag>
        <Text type="secondary">
          负责人 {task.assignee_username || task.assignee_id || '未指派'}
        </Text>
        <Text type="secondary">截止 {formatDateTime(task.due_time)}</Text>
      </div>

      <Space className="task-card-actions" size={8} wrap>
        {transitions.map((status) => (
          <Button
            disabled={disabled}
            key={status}
            onClick={() => onChangeStatus(task, status)}
            size="small"
          >
            {TaskStatusLabels[status]}
          </Button>
        ))}
      </Space>
    </Card>
  )
}
