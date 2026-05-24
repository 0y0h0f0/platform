import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'

import type { Project } from '../../src/api/types'
import { MemberList } from '../../src/components/project/MemberList'
import { getProjectPermission } from '../../src/hooks/useProjectPermission'
import { useProjectMembersQuery } from '../../src/queries/member.queries'
import { ProjectStatus, Role } from '../../src/utils/constants'
import { setToken } from '../../src/utils/token'
import { renderWithProviders } from '../test-utils'

const project: Project = {
  id: 'project-web-console',
  name: '前端管理台',
  description: '搭建注册登录、项目列表和任务看板的前端工作台。',
  owner_id: 'user-1',
  status: ProjectStatus.Active,
  version: 3,
}

function OwnerMemberListHarness() {
  const membersQuery = useProjectMembersQuery(project.id)
  const members = membersQuery.data?.members ?? []
  const permission = getProjectPermission(project, Role.Owner, 'user-1')

  return (
    <MemberList
      currentRole={Role.Owner}
      currentUserId="user-1"
      members={members}
      permission={permission}
      projectId={project.id}
    />
  )
}

describe('MemberList', () => {
  it('renders owner controls and removes a member', async () => {
    const user = userEvent.setup()
    setToken('mock-access-token')

    renderWithProviders(<OwnerMemberListHarness />)

    expect(await screen.findByText('user-2')).toBeInTheDocument()
    expect(screen.getByText('user-3')).toBeInTheDocument()
    expect(screen.getAllByLabelText('修改成员 user-2 角色').length).toBeGreaterThan(0)

    await user.click(screen.getByLabelText('移除成员 user-3'))
    const removeButtons = await screen.findAllByRole('button', { name: /移\s*除/ })
    await user.click(removeButtons[removeButtons.length - 1])

    await waitFor(() => expect(screen.queryByText('user-3')).not.toBeInTheDocument())
  })

  it('limits admin member actions', async () => {
    setToken('mock-access-token')

    const permission = getProjectPermission(project, Role.Admin, 'user-2')
    renderWithProviders(
      <MemberList
        currentRole={Role.Admin}
        currentUserId="user-2"
        members={[
          { id: 'member-owner', project_id: project.id, role: Role.Owner, user_id: 'user-1' },
          { id: 'member-admin', project_id: project.id, role: Role.Admin, user_id: 'user-2' },
          { id: 'member-user', project_id: project.id, role: Role.Member, user_id: 'user-3' },
        ]}
        permission={permission}
        projectId={project.id}
      />,
    )

    expect(screen.queryByLabelText('修改成员 user-3 角色')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('移除成员 user-2')).not.toBeInTheDocument()
    expect(screen.getByLabelText('移除成员 user-3')).toBeInTheDocument()
  })
})
