import { DeleteOutlined, UserAddOutlined } from '@ant-design/icons'
import { App, Button, Popconfirm, Select, Space, Table, Tag, Typography } from 'antd'
import type { ColumnsType } from 'antd/es/table'

import type { ProjectMember } from '@/api/types'
import { EmptyState } from '@/components/common/EmptyState'
import {
  useRemoveProjectMemberMutation,
  useUpdateProjectMemberRoleMutation,
} from '@/queries/member.queries'
import { Role, RoleColors, RoleLabels, type Role as RoleValue } from '@/utils/constants'
import { getErrorMessage } from '@/utils/error'
import type { ProjectPermission } from '@/hooks/useProjectPermission'

const { Text } = Typography

interface MemberListProps {
  currentRole?: RoleValue | null
  currentUserId?: string | null
  members: ProjectMember[]
  onAddMember?: () => void
  permission: Pick<ProjectPermission, 'canAddMember' | 'canChangeMemberRole' | 'canRemoveMember'>
  projectId: string
}

const editableRoleOptions = [
  { label: RoleLabels[Role.Admin], value: Role.Admin },
  { label: RoleLabels[Role.Member], value: Role.Member },
]

function canChangeRole(member: ProjectMember, permission: MemberListProps['permission']) {
  return permission.canChangeMemberRole && member.role !== Role.Owner
}

function canRemoveMember(
  member: ProjectMember,
  currentRole: RoleValue | null | undefined,
  currentUserId: string | null | undefined,
  permission: MemberListProps['permission'],
) {
  if (
    !permission.canRemoveMember ||
    member.user_id === currentUserId ||
    member.role === Role.Owner
  ) {
    return false
  }

  if (currentRole === Role.Admin && member.role !== Role.Member) {
    return false
  }

  return true
}

export function MemberList({
  currentRole,
  currentUserId,
  members,
  onAddMember,
  permission,
  projectId,
}: MemberListProps) {
  const { message } = App.useApp()
  const updateRoleMutation = useUpdateProjectMemberRoleMutation(projectId)
  const removeMemberMutation = useRemoveProjectMemberMutation(projectId)

  const handleRoleChange = (member: ProjectMember, role: RoleValue) => {
    updateRoleMutation.mutate(
      { role, userId: member.user_id },
      {
        onError: (error) => {
          message.error(getErrorMessage(error))
        },
        onSuccess: () => {
          message.success('成员角色已更新')
        },
      },
    )
  }

  const handleRemove = (member: ProjectMember) => {
    removeMemberMutation.mutate(member.user_id, {
      onError: (error) => {
        message.error(getErrorMessage(error))
      },
      onSuccess: () => {
        message.success('成员已移除')
      },
    })
  }

  const columns: ColumnsType<ProjectMember> = [
    {
      dataIndex: 'user_id',
      key: 'user_id',
      title: '成员',
      render: (userId: ProjectMember['user_id']) => (
        <Space direction="vertical" size={0}>
          <Text strong>{userId}</Text>
          <Text type="secondary">成员 ID</Text>
        </Space>
      ),
    },
    {
      dataIndex: 'role',
      key: 'role',
      title: '角色',
      width: 180,
      render: (_role: ProjectMember['role'], member) =>
        canChangeRole(member, permission) ? (
          <Select<RoleValue>
            aria-label={`修改成员 ${member.user_id} 角色`}
            className="member-role-select"
            disabled={
              updateRoleMutation.isPending &&
              updateRoleMutation.variables?.userId === member.user_id
            }
            onChange={(role) => handleRoleChange(member, role)}
            options={editableRoleOptions}
            value={member.role}
          />
        ) : (
          <Tag color={RoleColors[member.role]}>{RoleLabels[member.role]}</Tag>
        ),
    },
    {
      key: 'actions',
      title: '操作',
      width: 120,
      render: (_value, member) => {
        const removable = canRemoveMember(member, currentRole, currentUserId, permission)
        const removing =
          removeMemberMutation.isPending && removeMemberMutation.variables === member.user_id

        return removable ? (
          <Popconfirm
            cancelText="取消"
            okButtonProps={{ danger: true, loading: removing }}
            okText="移除"
            onConfirm={() => handleRemove(member)}
            title="移除成员"
          >
            <Button
              aria-label={`移除成员 ${member.user_id}`}
              danger
              disabled={removing}
              icon={<DeleteOutlined aria-hidden />}
              size="small"
              type="text"
            />
          </Popconfirm>
        ) : (
          <Text type="secondary">-</Text>
        )
      },
    },
  ]

  return (
    <section className="settings-section">
      <header className="settings-section-header">
        <span>
          <Typography.Title level={2}>成员管理</Typography.Title>
          <Text type="secondary">共 {members.length} 位成员</Text>
        </span>
        <Button
          disabled={!permission.canAddMember}
          icon={<UserAddOutlined aria-hidden />}
          onClick={onAddMember}
          type="primary"
        >
          添加成员
        </Button>
      </header>

      {members.length > 0 ? (
        <Table<ProjectMember>
          className="member-table"
          columns={columns}
          dataSource={members}
          pagination={false}
          rowKey={(member) => member.id || `${member.project_id}-${member.user_id}`}
          size="middle"
        />
      ) : (
        <EmptyState title="暂无成员" />
      )}
    </section>
  )
}
