import { Spin, Typography } from 'antd'

const { Text } = Typography

export function LoadingSpinner({ tip = '加载中' }: { tip?: string }) {
  return (
    <div className="loading-state" role="status">
      <Spin />
      <Text type="secondary">{tip}</Text>
    </div>
  )
}
