import { LogoutOutlined, UserOutlined } from '@ant-design/icons'
import { App, Avatar, Button, Dropdown, Space, Typography, type MenuProps } from 'antd'
import { useNavigate } from 'react-router-dom'

import { useLogoutMutation } from '@/queries/auth.queries'
import { useAuthStore } from '@/stores/auth.store'

const { Text } = Typography

export function AppHeader() {
  const { message } = App.useApp()
  const navigate = useNavigate()
  const user = useAuthStore((state) => state.user)
  const logoutMutation = useLogoutMutation()

  const items: MenuProps['items'] = [
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      onClick: () => {
        logoutMutation.mutate(undefined, {
          onSuccess: () => {
            message.success('已退出登录')
            navigate('/login', { replace: true })
          },
        })
      },
    },
  ]

  return (
    <header className="app-header">
      <Dropdown menu={{ items }} trigger={['click']}>
        <Button type="text" className="user-menu-button">
          <Space size={10}>
            <Avatar size={28} icon={<UserOutlined />} src={user?.avatar_url || undefined} />
            <Text className="user-menu-name">{user?.nickname || user?.username || '用户'}</Text>
          </Space>
        </Button>
      </Dropdown>
    </header>
  )
}
