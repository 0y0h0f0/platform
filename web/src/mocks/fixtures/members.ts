import type { ProjectMember } from '@/api/types'
import { Role } from '@/utils/constants'

export function createMockMembers(): ProjectMember[] {
  return [
    {
      id: 'member-owner-web',
      project_id: 'project-web-console',
      user_id: 'user-1',
      role: Role.Owner,
    },
    {
      id: 'member-admin-web',
      project_id: 'project-web-console',
      user_id: 'user-2',
      role: Role.Admin,
    },
    {
      id: 'member-member-web',
      project_id: 'project-web-console',
      user_id: 'user-3',
      role: Role.Member,
    },
    {
      id: 'member-owner-mobile',
      project_id: 'project-mobile-board',
      user_id: 'user-1',
      role: Role.Owner,
    },
    {
      id: 'member-owner-archived',
      project_id: 'project-archived-migration',
      user_id: 'user-1',
      role: Role.Owner,
    },
  ]
}
