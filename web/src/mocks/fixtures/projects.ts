import type { Project } from '@/api/types'
import { ProjectStatus } from '@/utils/constants'

export function createMockProjects(): Project[] {
  return [
    {
      id: 'project-web-console',
      name: '前端管理台',
      description: '搭建注册登录、项目列表和任务看板的前端工作台。',
      owner_id: 'user-1',
      status: ProjectStatus.Active,
      version: 3,
    },
    {
      id: 'project-mobile-board',
      name: '移动端看板优化',
      description: '梳理移动端任务流转体验，降低跨角色协作成本。',
      owner_id: 'user-1',
      status: ProjectStatus.Active,
      version: 1,
    },
    {
      id: 'project-archived-migration',
      name: '历史归档迁移',
      description: '已完成的旧系统项目迁移验证。',
      owner_id: 'user-1',
      status: ProjectStatus.Archived,
      version: 7,
    },
  ]
}
