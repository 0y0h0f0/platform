import { Layout } from 'antd'
import { Outlet } from 'react-router-dom'

import { AppHeader } from './AppHeader'
import { AppSider } from './AppSider'

const { Content } = Layout

export function AppLayout() {
  return (
    <Layout className="app-shell">
      <AppSider />
      <Layout>
        <AppHeader />
        <Content className="app-content">
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}
