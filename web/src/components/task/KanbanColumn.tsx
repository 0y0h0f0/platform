import { Badge, Typography } from 'antd'
import type { DragEvent } from 'react'

import type { Task } from '@/api/types'
import { EmptyState } from '@/components/common/EmptyState'
import { TaskStatusColors, TaskStatusLabels, type TaskStatus } from '@/utils/constants'
import { TaskCard } from './TaskCard'

const { Title } = Typography

interface KanbanColumnProps {
  disabled?: boolean
  onChangeStatus: (task: Task, status: TaskStatus) => void
  onDropTask?: (taskId: string, status: TaskStatus) => void
  onTaskOpen?: (task: Task) => void
  status: TaskStatus
  tasks: Task[]
}

function readTaskId(event: DragEvent<HTMLElement>) {
  return (
    event.dataTransfer.getData('application/x-task-id') || event.dataTransfer.getData('text/plain')
  )
}

export function KanbanColumn({
  disabled = false,
  onChangeStatus,
  onDropTask,
  onTaskOpen,
  status,
  tasks,
}: KanbanColumnProps) {
  const handleDragOver = (event: DragEvent<HTMLElement>) => {
    if (disabled) {
      return
    }

    event.preventDefault()
    event.dataTransfer.dropEffect = 'move'
  }

  const handleDrop = (event: DragEvent<HTMLElement>) => {
    if (disabled) {
      return
    }

    const taskId = readTaskId(event)
    if (!taskId) {
      return
    }

    event.preventDefault()
    onDropTask?.(taskId, status)
  }

  return (
    <section
      aria-label={TaskStatusLabels[status]}
      className="kanban-column"
      onDragOver={handleDragOver}
      onDrop={handleDrop}
    >
      <header className="kanban-column-header">
        <Title level={2}>{TaskStatusLabels[status]}</Title>
        <Badge color={TaskStatusColors[status]} count={tasks.length} showZero />
      </header>

      <div className="kanban-column-body">
        {tasks.length > 0 ? (
          tasks.map((task) => (
            <TaskCard
              disabled={disabled}
              key={task.id}
              onChangeStatus={onChangeStatus}
              onOpen={onTaskOpen}
              task={task}
            />
          ))
        ) : (
          <EmptyState title="暂无任务" />
        )}
      </div>
    </section>
  )
}
