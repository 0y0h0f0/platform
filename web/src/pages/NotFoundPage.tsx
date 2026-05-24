import { Button, Result } from 'antd'
import { Link } from 'react-router-dom'

export default function NotFoundPage() {
  return (
    <Result
      extra={
        <Button type="primary">
          <Link to="/projects">返回项目列表</Link>
        </Button>
      }
      status="404"
      subTitle="当前页面不存在或已经移动"
      title="404"
    />
  )
}
