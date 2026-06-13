import { Button, Result, Space, Tabs, Tag, Typography } from 'antd'
import { useMemo } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'

import type { Task } from '@/api/types'
import { LoadingSpinner } from '@/components/common/LoadingSpinner'
import { ProjectSettingsPanel } from '@/components/project/ProjectSettingsPanel'
import { ProjectToolbar, type ProjectTaskFilters } from '@/components/project/ProjectToolbar'
import { KanbanBoard } from '@/components/task/KanbanBoard'
import { TaskCreateModal } from '@/components/task/TaskCreateModal'
import { TaskDetailDrawer } from '@/components/task/TaskDetailDrawer'
import { useProjectMembersQuery } from '@/queries/member.queries'
import { useProjectQuery } from '@/queries/project.queries'
import { useAuthStore } from '@/stores/auth.store'
import {
  ProjectStatus,
  ProjectStatusColors,
  ProjectStatusLabels,
  Role,
  TaskStatus,
  type TaskStatus as TaskStatusValue,
} from '@/utils/constants'

const { Paragraph, Text, Title } = Typography

function parseStatus(value: string | null): ProjectTaskFilters['status'] {
  // Query strings are untrusted, so invalid status values are ignored instead
  // of being sent to the API.
  if (value === null || value === '') {
    return undefined
  }

  const numeric = Number(value) as TaskStatusValue
  const allowed: TaskStatusValue[] = [
    TaskStatus.Todo,
    TaskStatus.Doing,
    TaskStatus.Done,
    TaskStatus.Cancelled,
  ]
  return allowed.includes(numeric) ? numeric : undefined
}

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : '项目加载失败'
}

export default function ProjectDetailPage() {
  const { id = '' } = useParams()
  const [searchParams, setSearchParams] = useSearchParams()
  const user = useAuthStore((state) => state.user)
  const projectQuery = useProjectQuery(id)
  const membersQuery = useProjectMembersQuery(id)
  const rawProject = projectQuery.data?.project
  const project = rawProject
    ? { ...rawProject, status: rawProject.status ?? 0, version: rawProject.version ?? 0 }
    : undefined
  const members = membersQuery.data?.members ?? []
  const activeTab = searchParams.get('tab') || 'kanban'
  const activeTaskId = searchParams.get('task') || undefined
  const createTaskOpen = searchParams.get('createTask') === '1'

  const filters = useMemo<ProjectTaskFilters>(
    // Keep board filters in the URL so detail drawers and reloads preserve the
    // current project view.
    () => ({
      assigneeId: searchParams.get('assignee') || undefined,
      keyword: searchParams.get('keyword') || undefined,
      status: parseStatus(searchParams.get('status')),
    }),
    [searchParams],
  )

  const currentMember = members.find((member) => member.user_id === user?.id)
  const currentRole = currentMember?.role ?? (project?.owner_id === user?.id ? Role.Owner : null)
  const canCreateTask = Boolean(project && user && (currentMember || project.owner_id === user.id))
  const archived = project?.status === ProjectStatus.Archived

  const updateSearchParams = (nextValues: Record<string, string | number | undefined>) => {
    // Merge partial updates so opening a task drawer does not reset filters/tabs.
    const next = new URLSearchParams(searchParams)
    Object.entries(nextValues).forEach(([key, value]) => {
      if (value === undefined || value === '') {
        next.delete(key)
      } else {
        next.set(key, String(value))
      }
    })
    setSearchParams(next)
  }

  const handleFiltersChange = (nextFilters: ProjectTaskFilters) => {
    updateSearchParams({
      assignee: nextFilters.assigneeId,
      keyword: nextFilters.keyword,
      status: nextFilters.status,
    })
  }

  const openTaskModal = () => updateSearchParams({ createTask: '1' })
  const closeTaskModal = () => updateSearchParams({ createTask: undefined })
  const openTaskDetail = (task: Task) => updateSearchParams({ task: task.id })
  const closeTaskDetail = () => updateSearchParams({ task: undefined })

  if (projectQuery.isLoading) {
    return <LoadingSpinner tip="正在加载项目" />
  }

  if (projectQuery.isError || !project) {
    return (
      <Result
        extra={
          <Button type="primary">
            <Link to="/projects">返回项目列表</Link>
          </Button>
        }
        status="error"
        subTitle={getErrorMessage(projectQuery.error)}
        title="项目加载失败"
      />
    )
  }

  return (
    <div className="project-detail-page">
      <section className="project-detail-heading">
        <Space align="start" className="project-detail-title-row" size={12} wrap>
          <span>
            <Title level={1}>{project.name}</Title>
            <Paragraph type={project.description ? undefined : 'secondary'}>
              {project.description || '暂无项目描述'}
            </Paragraph>
          </span>
          <Tag color={ProjectStatusColors[project.status]}>
            {ProjectStatusLabels[project.status]}
          </Tag>
        </Space>
        <Text type="secondary">项目版本 {project.version}</Text>
      </section>

      <Tabs
        activeKey={activeTab}
        onChange={(key) => updateSearchParams({ tab: key })}
        items={[
          {
            key: 'kanban',
            label: '看板',
            children: (
              <Space className="project-kanban-panel" direction="vertical" size={16}>
                <ProjectToolbar
                  disabled={archived || !canCreateTask}
                  filters={filters}
                  members={members}
                  onCreateTask={openTaskModal}
                  onFiltersChange={handleFiltersChange}
                />
                <KanbanBoard
                  assigneeId={filters.assigneeId}
                  disabled={archived}
                  keyword={filters.keyword}
                  onTaskOpen={openTaskDetail}
                  projectId={project.id}
                  status={filters.status}
                />
              </Space>
            ),
          },
          {
            key: 'settings',
            label: '项目设置',
            children: (
              <ProjectSettingsPanel
                currentRole={currentRole}
                currentUserId={user?.id}
                members={members}
                project={project}
              />
            ),
          },
        ]}
      />

      <TaskCreateModal onClose={closeTaskModal} open={createTaskOpen} projectId={project.id} />
      <TaskDetailDrawer
        currentRole={currentRole}
        currentUserId={user?.id}
        members={members}
        onClose={closeTaskDetail}
        open={Boolean(activeTaskId)}
        project={project}
        taskId={activeTaskId}
      />
    </div>
  )
}
