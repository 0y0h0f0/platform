import { Alert, Button, Space, Typography } from 'antd'

import { EmptyState } from '@/components/common/EmptyState'
import { LoadingSpinner } from '@/components/common/LoadingSpinner'
import { useProjectOperationLogsQuery } from '@/queries/operationLog.queries'
import { getErrorMessage } from '@/utils/error'
import { OperationLogItem } from './OperationLogItem'

const { Text } = Typography

interface OperationLogListProps {
  limit?: number
  projectId: string
}

export function OperationLogList({ limit = 20, projectId }: OperationLogListProps) {
  const logsQuery = useProjectOperationLogsQuery({ limit, projectId })
  const logs = logsQuery.data?.pages.flatMap((page) => page.logs ?? []) ?? []

  if (logsQuery.isLoading) {
    return <LoadingSpinner tip="正在加载操作日志" />
  }

  if (logsQuery.isError) {
    return (
      <Alert
        action={
          <Button onClick={() => logsQuery.refetch()} size="small">
            重试
          </Button>
        }
        description={getErrorMessage(logsQuery.error)}
        message="操作日志加载失败"
        showIcon
        type="error"
      />
    )
  }

  return (
    <Space className="operation-log-list" direction="vertical" size={12}>
      <Text type="secondary">共 {logs.length} 条记录</Text>
      {logs.length > 0 ? (
        logs.map((log) => <OperationLogItem key={log.id} log={log} />)
      ) : (
        <EmptyState title="暂无操作日志" />
      )}

      {logsQuery.hasNextPage ? (
        <div className="operation-log-load-more">
          <Button loading={logsQuery.isFetchingNextPage} onClick={() => logsQuery.fetchNextPage()}>
            加载更多
          </Button>
        </div>
      ) : null}
    </Space>
  )
}
