import { Alert, Button, Drawer, Space, Typography } from 'antd'
import { useState } from 'react'

import type { Project, ProjectMember } from '@/api/types'
import { CommentInput } from '@/components/comment/CommentInput'
import { CommentList } from '@/components/comment/CommentList'
import { LoadingSpinner } from '@/components/common/LoadingSpinner'
import { useTaskPermission } from '@/hooks/useTaskPermission'
import { useTaskQuery } from '@/queries/task.queries'
import type { Role } from '@/utils/constants'
import { getErrorMessage } from '@/utils/error'
import { TaskAssignSelect } from './TaskAssignSelect'
import { TaskEditForm } from './TaskEditForm'
import { TaskStatusSelect } from './TaskStatusSelect'

const { Text, Title } = Typography

interface TaskDetailDrawerProps {
  currentRole?: Role | null
  currentUserId?: string | null
  members: ProjectMember[]
  onClose: () => void
  open: boolean
  project: Project
  taskId?: string
}

function TaskReadonlyAlert({ isReadOnly }: { isReadOnly: boolean }) {
  return (
    <Alert
      message={isReadOnly ? '项目已归档，任务只读' : '没有编辑此任务的权限'}
      showIcon
      type="info"
    />
  )
}

export function TaskDetailDrawer({
  currentRole,
  currentUserId,
  members,
  onClose,
  open,
  project,
  taskId,
}: TaskDetailDrawerProps) {
  const [conflictTaskId, setConflictTaskId] = useState<string | null>(null)
  const queryTaskId = open ? (taskId ?? '') : ''
  const taskQuery = useTaskQuery(queryTaskId)
  const raw = taskQuery.data?.task
  const task = raw ? { ...raw, status: raw.status ?? 0, version: raw.version ?? 0 } : undefined
  const permission = useTaskPermission(project, currentRole, task, currentUserId)
  const conflictVisible = conflictTaskId === queryTaskId

  const handleConflict = () => {
    setConflictTaskId(queryTaskId)
    taskQuery.refetch()
  }

  const handleChanged = () => {
    setConflictTaskId(null)
  }

  return (
    <Drawer
      destroyOnClose
      extra={task ? <Text type="secondary">版本 {task.version}</Text> : null}
      onClose={onClose}
      open={open}
      title={task?.title ?? '任务详情'}
      width={560}
    >
      {taskQuery.isLoading ? <LoadingSpinner tip="正在加载任务" /> : null}

      {taskQuery.isError ? (
        <Alert
          action={
            <Button onClick={() => taskQuery.refetch()} size="small">
              重试
            </Button>
          }
          description={getErrorMessage(taskQuery.error)}
          message="任务加载失败"
          showIcon
          type="error"
        />
      ) : null}

      {task ? (
        <Space className="task-detail-content" direction="vertical" size={16}>
          <section className="task-detail-heading">
            <Title level={2}>{task.title}</Title>
            <div className="task-detail-meta">
              <Text type="secondary">创建者 {task.creator_username || task.creator_id}</Text>
              <Text type="secondary">任务 ID {task.id}</Text>
            </div>
          </section>

          {conflictVisible ? (
            <Alert
              description="已刷新最新任务，请基于最新内容重新提交。"
              message="任务版本已更新"
              showIcon
              type="warning"
            />
          ) : null}

          {!permission.canEditTask ? (
            <TaskReadonlyAlert isReadOnly={permission.isReadOnly} />
          ) : null}

          <div className="task-detail-controls">
            <TaskStatusSelect
              disabled={!permission.canChangeStatus}
              onChanged={handleChanged}
              onConflict={handleConflict}
              task={task}
            />
            <TaskAssignSelect
              disabled={!permission.canAssignTask}
              members={members}
              onChanged={handleChanged}
              onConflict={handleConflict}
              task={task}
            />
          </div>

          <TaskEditForm
            disabled={!permission.canEditTask}
            onConflict={handleConflict}
            onUpdated={handleChanged}
            task={task}
          />

          <section className="task-comments-section">
            <Title level={3}>评论</Title>
            <CommentInput disabled={!permission.canComment} taskId={task.id} />
            <CommentList
              currentRole={currentRole}
              currentUserId={currentUserId}
              readOnly={!permission.canComment}
              taskId={task.id}
            />
          </section>
        </Space>
      ) : null}
    </Drawer>
  )
}
