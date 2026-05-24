import { FolderOpenOutlined, PlusOutlined } from '@ant-design/icons'
import { Button, Layout } from 'antd'
import { NavLink, useNavigate } from 'react-router-dom'

const { Sider } = Layout

export function AppSider() {
  const navigate = useNavigate()

  return (
    <Sider className="app-sider" width={232} breakpoint="lg" collapsedWidth={0}>
      <div className="app-brand">团队任务协作平台</div>
      <nav className="app-nav" aria-label="主导航">
        <NavLink
          className={({ isActive }) => `app-nav-item${isActive ? ' is-active' : ''}`}
          to="/projects"
        >
          <FolderOpenOutlined />
          <span>项目列表</span>
        </NavLink>
        <Button
          className="app-create-button"
          icon={<PlusOutlined aria-hidden />}
          onClick={() => navigate('/projects?create=1')}
          type="primary"
          block
        >
          创建项目
        </Button>
      </nav>
    </Sider>
  )
}
