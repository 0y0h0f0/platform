import type { User } from '@/api/types'

export const mockUsers: User[] = [
  {
    id: 'user-1',
    username: 'demo_user',
    email: 'demo@example.com',
    nickname: '演示用户',
    avatar_url: '',
    status: 0,
  },
  {
    id: 'user-2',
    username: 'admin_user',
    email: 'admin@example.com',
    nickname: '管理员',
    avatar_url: '',
    status: 0,
  },
  {
    id: 'user-3',
    username: 'member_user',
    email: 'member@example.com',
    nickname: '协作者',
    avatar_url: '',
    status: 0,
  },
  {
    id: 'user-4',
    username: 'qa_user',
    email: 'qa@example.com',
    nickname: '测试同学',
    avatar_url: '',
    status: 0,
  },
]

export const mockUser = mockUsers[0]

export function findMockUser(userId: string) {
  return mockUsers.find((user) => user.id === userId)
}
