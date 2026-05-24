import type { TaskComment } from '@/api/types'

export function createMockComments(): TaskComment[] {
  return [
    {
      id: 'comment-001',
      task_id: 'task-kanban-api',
      user_id: 'user-1',
      username: 'owner',
      nickname: '项目负责人',
      avatar_url: '',
      content: '接口分页已经联通，下一步补详情抽屉。',
    },
    {
      id: 'comment-002',
      task_id: 'task-kanban-api',
      user_id: 'user-2',
      username: 'admin',
      nickname: '管理员',
      avatar_url: '',
      content: '状态流转要复用同一套合法转换规则。',
    },
    {
      id: 'comment-003',
      task_id: 'task-kanban-api',
      user_id: 'user-3',
      username: 'member',
      nickname: '成员',
      avatar_url: '',
      content: '我会补充 after_id 分页测试。',
    },
    {
      id: 'comment-004',
      task_id: 'task-login-wireframe',
      user_id: 'user-2',
      username: 'admin',
      nickname: '管理员',
      avatar_url: '',
      content: '登录页错误状态需要和后端错误码对齐。',
    },
  ]
}
