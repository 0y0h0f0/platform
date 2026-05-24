import type { OperationLog } from '@/api/types'

export function createMockOperationLogs(): OperationLog[] {
  return [
    {
      id: 'log-005',
      project_id: 'project-web-console',
      operator_id: 'user-1',
      username: 'demo_user',
      nickname: '演示用户',
      avatar_url: '',
      action: 'comment.create',
      detail_json: '{}',
      task_id: 'task-kanban-api',
    },
    {
      id: 'log-004',
      project_id: 'project-web-console',
      operator_id: 'user-1',
      username: 'demo_user',
      nickname: '演示用户',
      avatar_url: '',
      action: 'task.status_change',
      detail_json: '{}',
      task_id: 'task-kanban-api',
    },
    {
      id: 'log-003',
      project_id: 'project-web-console',
      operator_id: 'user-1',
      username: 'demo_user',
      nickname: '演示用户',
      avatar_url: '',
      action: 'member.add',
      detail_json: '{"user_id":"user-3"}',
    },
    {
      id: 'log-002',
      project_id: 'project-web-console',
      operator_id: 'user-1',
      username: 'demo_user',
      nickname: '演示用户',
      avatar_url: '',
      action: 'project.update',
      detail_json: '{}',
    },
    {
      id: 'log-001',
      project_id: 'project-web-console',
      operator_id: 'user-1',
      username: 'demo_user',
      nickname: '演示用户',
      avatar_url: '',
      action: 'project.create',
      detail_json: '{}',
    },
  ]
}
