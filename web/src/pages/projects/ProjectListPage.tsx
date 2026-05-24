import { PlusOutlined } from '@ant-design/icons'
import { Alert, Button, Space, Typography } from 'antd'
import { useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'

import { EmptyState } from '@/components/common/EmptyState'
import { LoadingSpinner } from '@/components/common/LoadingSpinner'
import { ArchiveToggle } from '@/components/project/ArchiveToggle'
import { ProjectCard } from '@/components/project/ProjectCard'
import { ProjectCreateModal } from '@/components/project/ProjectCreateModal'
import { useProjectsQuery } from '@/queries/project.queries'

const { Text, Title } = Typography
const PAGE_SIZE = 6

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : '项目列表加载失败'
}

export default function ProjectListPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [includeArchived, setIncludeArchived] = useState(false)
  const [offset, setOffset] = useState(0)
  const createModalOpen = searchParams.get('create') === '1'

  const queryParams = useMemo(
    () => ({
      includeArchived,
      limit: PAGE_SIZE,
      offset,
    }),
    [includeArchived, offset],
  )

  const projectsQuery = useProjectsQuery(queryParams)
  const projects = projectsQuery.data?.projects ?? []
  const page = Math.floor(offset / PAGE_SIZE) + 1
  const canGoPrevious = offset > 0
  const canGoNext = projects.length === PAGE_SIZE

  const openCreateModal = () => {
    const next = new URLSearchParams(searchParams)
    next.set('create', '1')
    setSearchParams(next)
  }

  const closeCreateModal = () => {
    const next = new URLSearchParams(searchParams)
    next.delete('create')
    setSearchParams(next, { replace: true })
  }

  const handleIncludeArchivedChange = (checked: boolean) => {
    setIncludeArchived(checked)
    setOffset(0)
  }

  return (
    <div className="project-list-page">
      <section className="page-heading page-heading-row">
        <span>
          <Title level={1}>我的项目</Title>
          <Text type="secondary">查看项目卡片，创建新项目，按需显示已归档项目。</Text>
        </span>
        <Space className="project-list-actions" size={12} wrap>
          <ArchiveToggle
            checked={includeArchived}
            disabled={projectsQuery.isFetching}
            onChange={handleIncludeArchivedChange}
          />
          <Button icon={<PlusOutlined aria-hidden />} onClick={openCreateModal} type="primary">
            创建项目
          </Button>
        </Space>
      </section>

      {projectsQuery.isLoading ? <LoadingSpinner tip="正在加载项目" /> : null}

      {projectsQuery.isError ? (
        <Alert
          action={
            <Button onClick={() => projectsQuery.refetch()} size="small">
              重试
            </Button>
          }
          message="项目列表加载失败"
          description={getErrorMessage(projectsQuery.error)}
          showIcon
          type="error"
        />
      ) : null}

      {!projectsQuery.isLoading && !projectsQuery.isError && projects.length === 0 ? (
        <EmptyState
          action={
            <Button icon={<PlusOutlined aria-hidden />} onClick={openCreateModal} type="primary">
              创建项目
            </Button>
          }
          description={
            includeArchived
              ? '当前账号还没有项目。'
              : '当前没有活跃项目，可显示归档项目或创建新项目。'
          }
          title="暂无项目"
        />
      ) : null}

      {!projectsQuery.isLoading && !projectsQuery.isError && projects.length > 0 ? (
        <>
          <div className="project-grid">
            {projects.map((project) => (
              <ProjectCard key={project.id} project={project} />
            ))}
          </div>

          <footer className="project-pagination">
            <Button
              disabled={!canGoPrevious || projectsQuery.isFetching}
              onClick={() => setOffset(offset - PAGE_SIZE)}
            >
              上一页
            </Button>
            <Text type="secondary">第 {page} 页</Text>
            <Button
              disabled={!canGoNext || projectsQuery.isFetching}
              onClick={() => setOffset(offset + PAGE_SIZE)}
            >
              下一页
            </Button>
          </footer>
        </>
      ) : null}

      <ProjectCreateModal
        onClose={closeCreateModal}
        onCreated={() => setOffset(0)}
        open={createModalOpen}
      />
    </div>
  )
}
