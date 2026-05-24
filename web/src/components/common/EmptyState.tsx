import { Empty, Typography, type EmptyProps } from 'antd'

import emptyStateImage from '@/assets/empty-state.svg'
import type { ReactNode } from 'react'

const { Text } = Typography

interface EmptyStateProps {
  action?: ReactNode
  description?: string
  image?: EmptyProps['image']
  title: string
}

export function EmptyState({
  action,
  description,
  image = emptyStateImage,
  title,
}: EmptyStateProps) {
  return (
    <div className="empty-state">
      <Empty description={<Text strong>{title}</Text>} image={image}>
        {description ? <Text type="secondary">{description}</Text> : null}
        {action ? <div className="empty-state-action">{action}</div> : null}
      </Empty>
    </div>
  )
}
