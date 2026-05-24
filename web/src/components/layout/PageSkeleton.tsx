import { Skeleton, Space } from 'antd'

export function PageSkeleton() {
  return (
    <div className="page-skeleton" aria-label="页面加载中">
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <Skeleton.Input active block style={{ height: 32 }} />
        <Skeleton active paragraph={{ rows: 6 }} />
      </Space>
    </div>
  )
}
