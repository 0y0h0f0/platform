import { EditOutlined, InboxOutlined, LogoutOutlined, SwapOutlined } from '@ant-design/icons'
import { App, Button, Popconfirm, Select, Space, Tag, Typography } from 'antd'
import { useMemo, useState } from 'react'

import type { Project, ProjectMember } from '@/api/types'
import { MemberAddModal } from '@/components/project/MemberAddModal'
import { MemberList } from '@/components/project/MemberList'
import { OperationLogList } from '@/components/project/OperationLogList'
import { ProjectEditModal } from '@/components/project/ProjectEditModal'
import { useProjectPermission } from '@/hooks/useProjectPermission'
import { useLeaveProjectMutation } from '@/queries/member.queries'
import {
  useArchiveProjectMutation,
  useTransferProjectOwnershipMutation,
  useUnarchiveProjectMutation,
} from '@/queries/project.queries'
import {
  ProjectStatus,
  ProjectStatusColors,
  ProjectStatusLabels,
  Role,
  RoleLabels,
  type Role as RoleValue,
} from '@/utils/constants'
import { getErrorMessage } from '@/utils/error'

const { Paragraph, Text, Title } = Typography

interface ProjectSettingsPanelProps {
  currentRole?: RoleValue | null
  currentUserId?: string | null
  members: ProjectMember[]
  project: Project
}

export function ProjectSettingsPanel({
  currentRole,
  currentUserId,
  members,
  project,
}: ProjectSettingsPanelProps) {
  const { message } = App.useApp()
  const [editOpen, setEditOpen] = useState(false)
  const [addMemberOpen, setAddMemberOpen] = useState(false)
  const [transferTargetId, setTransferTargetId] = useState<string>()
  const permission = useProjectPermission(project, currentRole, currentUserId)
  const archiveProjectMutation = useArchiveProjectMutation()
  const unarchiveProjectMutation = useUnarchiveProjectMutation()
  const transferOwnershipMutation = useTransferProjectOwnershipMutation(project.id)
  const leaveProjectMutation = useLeaveProjectMutation(project.id)
  const isArchived = project.status === ProjectStatus.Archived
  const canAddAdmin = currentRole === Role.Owner

  const transferOptions = useMemo(
    () =>
      members
        .filter((member) => member.role !== Role.Owner)
        .map((member) => ({
          label: `${member.username || member.user_id} · ${RoleLabels[member.role]}`,
          value: member.user_id,
        })),
    [members],
  )

  const handleArchiveToggle = () => {
    const mutation = isArchived ? unarchiveProjectMutation : archiveProjectMutation
    mutation.mutate(project.id, {
      onError: (error) => {
        message.error(getErrorMessage(error))
      },
      onSuccess: () => {
        message.success(isArchived ? '项目已取消归档' : '项目已归档')
      },
    })
  }

  const handleTransfer = () => {
    if (!transferTargetId) {
      return
    }

    transferOwnershipMutation.mutate(transferTargetId, {
      onError: (error) => {
        message.error(getErrorMessage(error))
      },
      onSuccess: () => {
        message.success('项目已转让')
        setTransferTargetId(undefined)
      },
    })
  }

  const handleLeave = () => {
    leaveProjectMutation.mutate(undefined, {
      onError: (error) => {
        message.error(getErrorMessage(error))
      },
      onSuccess: () => {
        message.success('已退出项目')
      },
    })
  }

  return (
    <div className="project-settings-panel">
      <section className="settings-section">
        <header className="settings-section-header">
          <span>
            <Title level={2}>项目信息</Title>
            <Text type="secondary">版本 {project.version}</Text>
          </span>
          <Space size={8} wrap>
            <Button
              disabled={!permission.canEditProject}
              icon={<EditOutlined aria-hidden />}
              onClick={() => setEditOpen(true)}
            >
              编辑项目
            </Button>
            <Popconfirm
              cancelText="取消"
              okButtonProps={{
                danger: !isArchived,
                loading: archiveProjectMutation.isPending || unarchiveProjectMutation.isPending,
              }}
              okText={isArchived ? '取消归档' : '归档'}
              onConfirm={handleArchiveToggle}
              title={isArchived ? '取消归档项目' : '归档项目'}
            >
              <Button
                danger={!isArchived}
                disabled={isArchived ? !permission.canUnarchive : !permission.canArchive}
                icon={<InboxOutlined aria-hidden />}
              >
                {isArchived ? '取消归档' : '归档项目'}
              </Button>
            </Popconfirm>
          </Space>
        </header>
        <div className="settings-project-summary">
          <Tag color={ProjectStatusColors[project.status]}>
            {ProjectStatusLabels[project.status]}
          </Tag>
          <Paragraph>{project.description || '暂无项目描述'}</Paragraph>
        </div>
      </section>

      <MemberList
        currentRole={currentRole}
        currentUserId={currentUserId}
        members={members}
        onAddMember={() => setAddMemberOpen(true)}
        permission={permission}
        projectId={project.id}
      />

      <section className="settings-section">
        <header className="settings-section-header">
          <span>
            <Title level={2}>项目操作</Title>
            <Text type="secondary">拥有者、管理员和成员按权限执行操作</Text>
          </span>
        </header>
        <div className="settings-actions-grid">
          <div className="settings-action-row">
            <Select
              allowClear
              className="settings-transfer-select"
              disabled={!permission.canTransfer || transferOptions.length === 0}
              onChange={setTransferTargetId}
              options={transferOptions}
              placeholder="选择新拥有者"
              value={transferTargetId}
            />
            <Button
              disabled={!permission.canTransfer || !transferTargetId}
              icon={<SwapOutlined aria-hidden />}
              loading={transferOwnershipMutation.isPending}
              onClick={handleTransfer}
              type="primary"
            >
              转让项目
            </Button>
          </div>
          <Popconfirm
            cancelText="取消"
            okButtonProps={{ danger: true, loading: leaveProjectMutation.isPending }}
            okText="退出"
            onConfirm={handleLeave}
            title="退出项目"
          >
            <Button danger disabled={!permission.canLeave} icon={<LogoutOutlined aria-hidden />}>
              退出项目
            </Button>
          </Popconfirm>
        </div>
      </section>

      <section className="settings-section">
        <header className="settings-section-header">
          <span>
            <Title level={2}>操作日志</Title>
            <Text type="secondary">项目级操作记录</Text>
          </span>
        </header>
        <OperationLogList projectId={project.id} />
      </section>

      <ProjectEditModal onClose={() => setEditOpen(false)} open={editOpen} project={project} />
      <MemberAddModal
        canAddAdmin={canAddAdmin}
        onClose={() => setAddMemberOpen(false)}
        open={addMemberOpen}
        projectId={project.id}
      />
    </div>
  )
}
