import { LockOutlined, UserOutlined } from '@ant-design/icons'
import { App, Button, Card, Form, Input, Typography } from 'antd'
import { Link, useLocation, useNavigate } from 'react-router-dom'

import type { LoginRequest } from '@/api/types'
import { useLoginMutation } from '@/queries/auth.queries'
import { getErrorMessage } from '@/utils/error'

const { Text, Title } = Typography

interface LocationState {
  from?: {
    pathname?: string
  }
}

export default function LoginPage() {
  const { message } = App.useApp()
  const navigate = useNavigate()
  const location = useLocation()
  const loginMutation = useLoginMutation()
  const from = (location.state as LocationState | null)?.from?.pathname || '/projects'

  const onFinish = (values: LoginRequest) => {
    loginMutation.mutate(values, {
      onSuccess: () => {
        message.success('登录成功')
        navigate(from, { replace: true })
      },
      onError: (error) => {
        message.error(getErrorMessage(error))
      },
    })
  }

  return (
    <main className="auth-page">
      <Card className="auth-card">
        <div className="auth-heading">
          <Title level={1}>登录</Title>
          <Text type="secondary">使用账号或邮箱进入团队任务协作平台</Text>
        </div>

        <Form<LoginRequest> layout="vertical" onFinish={onFinish} requiredMark={false}>
          <Form.Item
            label="账号或邮箱"
            name="account"
            rules={[{ required: true, message: '请输入账号或邮箱' }]}
          >
            <Input autoComplete="username" prefix={<UserOutlined />} />
          </Form.Item>
          <Form.Item
            label="密码"
            name="password"
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password autoComplete="current-password" prefix={<LockOutlined />} />
          </Form.Item>
          <Button block htmlType="submit" loading={loginMutation.isPending} type="primary">
            登录
          </Button>
        </Form>

        <div className="auth-footer">
          <Text type="secondary">还没有账号？</Text>
          <Link to="/register">注册</Link>
        </div>
      </Card>
    </main>
  )
}
