import { PlusOutlined, SearchOutlined } from '@ant-design/icons'
import { Button, Input, Select, Space } from 'antd'
import type { ProjectMember } from '@/api/types'
import { TaskStatus, TaskStatusLabels, type TaskStatus as TaskStatusValue } from '@/utils/constants'

export interface ProjectTaskFilters {
  assigneeId?: string
  keyword?: string
  status?: TaskStatusValue
}

interface ProjectToolbarProps {
  disabled?: boolean
  filters: ProjectTaskFilters
  members: ProjectMember[]
  onCreateTask: () => void
  onFiltersChange: (filters: ProjectTaskFilters) => void
}

const statusOptions = [
  TaskStatus.Todo,
  TaskStatus.Doing,
  TaskStatus.Done,
  TaskStatus.Cancelled,
].map((status) => ({ label: TaskStatusLabels[status], value: status }))

export function ProjectToolbar({
  disabled = false,
  filters,
  members,
  onCreateTask,
  onFiltersChange,
}: ProjectToolbarProps) {
  return (
    <div className="project-toolbar">
      <Input.Search
        allowClear
        enterButton={<SearchOutlined />}
        onChange={(event) =>
          onFiltersChange({ ...filters, keyword: event.target.value.trim() || undefined })
        }
        onSearch={(value) => onFiltersChange({ ...filters, keyword: value.trim() || undefined })}
        placeholder="搜索任务"
        value={filters.keyword ?? ''}
      />

      <Select
        allowClear
        className="project-toolbar-select"
        onChange={(value) => onFiltersChange({ ...filters, assigneeId: value })}
        options={members.map((member) => ({
          label: member.username || member.nickname || member.user_id,
          value: member.user_id,
        }))}
        placeholder="负责人"
        value={filters.assigneeId}
      />

      <Select
        allowClear
        className="project-toolbar-select"
        onChange={(value) => onFiltersChange({ ...filters, status: value })}
        options={statusOptions}
        placeholder="状态"
        value={filters.status}
      />

      <Space className="project-toolbar-actions">
        <Button
          disabled={disabled}
          icon={<PlusOutlined aria-hidden />}
          onClick={onCreateTask}
          type="primary"
        >
          创建任务
        </Button>
      </Space>
    </div>
  )
}
