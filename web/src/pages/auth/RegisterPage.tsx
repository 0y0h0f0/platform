import { LockOutlined, MailOutlined, UserOutlined } from '@ant-design/icons'
import { App, Button, Card, Form, Input, Typography } from 'antd'
import { Link, useNavigate } from 'react-router-dom'

import type { RegisterRequest } from '@/api/types'
import { useRegisterMutation } from '@/queries/auth.queries'
import { getErrorMessage } from '@/utils/error'

const { Text, Title } = Typography

export default function RegisterPage() {
  const { message } = App.useApp()
  const navigate = useNavigate()
  const registerMutation = useRegisterMutation()

  const onFinish = (values: RegisterRequest) => {
    registerMutation.mutate(values, {
      onSuccess: () => {
        message.success('注册成功')
        navigate('/projects', { replace: true })
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
          <Title level={1}>注册</Title>
          <Text type="secondary">创建账号后会自动登录并进入项目列表</Text>
        </div>

        <Form<RegisterRequest> layout="vertical" onFinish={onFinish} requiredMark={false}>
          <Form.Item
            label="用户名"
            name="username"
            rules={[
              { required: true, message: '请输入用户名' },
              { pattern: /^[a-zA-Z0-9_]{3,32}$/, message: '用户名需为 3-32 位字母、数字或下划线' },
            ]}
          >
            <Input autoComplete="username" prefix={<UserOutlined />} />
          </Form.Item>
          <Form.Item
            label="邮箱"
            name="email"
            rules={[
              { required: true, message: '请输入邮箱' },
              { type: 'email', message: '邮箱格式不正确' },
            ]}
          >
            <Input autoComplete="email" prefix={<MailOutlined />} />
          </Form.Item>
          <Form.Item
            label="密码"
            name="password"
            rules={[
              { required: true, message: '请输入密码' },
              { min: 8, message: '密码至少 8 位' },
            ]}
          >
            <Input.Password autoComplete="new-password" prefix={<LockOutlined />} />
          </Form.Item>
          <Button block htmlType="submit" loading={registerMutation.isPending} type="primary">
            注册
          </Button>
        </Form>

        <div className="auth-footer">
          <Text type="secondary">已有账号？</Text>
          <Link to="/login">登录</Link>
        </div>
      </Card>
    </main>
  )
}
