import { Avatar, Space, Tag, Typography } from 'antd'

import type { OperationLog } from '@/api/types'
import { ActionLabel, type ActionType } from '@/utils/constants'

const { Text } = Typography

const detailLabels: Record<string, string> = {
  from_user_id: '原拥有者',
  to_user_id: '新拥有者',
  user_id: '成员',
}

function getDisplayName(log: OperationLog) {
  return log.nickname || log.username || log.operator_id
}

function getAvatarText(log: OperationLog) {
  return getDisplayName(log).slice(0, 1).toUpperCase()
}

function getActionLabel(action: OperationLog['action']) {
  return ActionLabel[action as ActionType] ?? action
}

function parseDetail(detailJson: string) {
  try {
    const parsed = JSON.parse(detailJson) as Record<string, unknown>
    return Object.entries(parsed).filter(
      ([, value]) => value !== null && value !== undefined && value !== '',
    )
  } catch {
    return []
  }
}

function formatDetailValue(value: unknown) {
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    return String(value)
  }

  return JSON.stringify(value)
}

export function OperationLogItem({ log }: { log: OperationLog }) {
  const details = parseDetail(log.detail_json)

  return (
    <article className="operation-log-item">
      <Avatar className="operation-log-avatar" src={log.avatar_url || undefined}>
        {getAvatarText(log)}
      </Avatar>
      <div className="operation-log-body">
        <Space className="operation-log-header" size={8} wrap>
          <Text strong>{getDisplayName(log)}</Text>
          <Text type="secondary">{log.operator_id}</Text>
          <Tag>{getActionLabel(log.action)}</Tag>
        </Space>
        <Space size={[6, 6]} wrap>
          {log.task_id ? <Tag color="blue">任务 {log.task_id}</Tag> : null}
          {details.map(([key, value]) => (
            <Tag key={key}>
              {detailLabels[key] ?? key}: {formatDetailValue(value)}
            </Tag>
          ))}
          <Text type="secondary">{log.id}</Text>
        </Space>
      </div>
    </article>
  )
}
