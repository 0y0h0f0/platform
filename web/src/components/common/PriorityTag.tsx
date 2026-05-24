import { Tag } from 'antd'

import { PriorityColors, PriorityLabels, type Priority } from '@/utils/constants'

export function PriorityTag({ priority }: { priority: Priority }) {
  return <Tag color={PriorityColors[priority]}>{PriorityLabels[priority]}</Tag>
}
