import { FolderOpenOutlined } from '@ant-design/icons'
import { Card, Space, Tag, Typography } from 'antd'
import { Link } from 'react-router-dom'

import type { Project } from '@/api/types'
import { ProjectStatus, ProjectStatusColors, ProjectStatusLabels } from '@/utils/constants'

const { Paragraph, Text } = Typography

export function ProjectCard({ project }: { project: Project }) {
  const archived = project.status === ProjectStatus.Archived

  return (
    <Card
      actions={[
        <Link key="detail" to={`/projects/${project.id}`}>
          进入项目
        </Link>,
      ]}
      className={`project-card${archived ? ' is-archived' : ''}`}
      extra={
        <Tag color={ProjectStatusColors[project.status]}>{ProjectStatusLabels[project.status]}</Tag>
      }
      title={
        <Space size={8}>
          <FolderOpenOutlined />
          <span>{project.name}</span>
        </Space>
      }
    >
      <Paragraph
        className="project-card-description"
        type={project.description ? undefined : 'secondary'}
      >
        {project.description || '暂无项目描述'}
      </Paragraph>
      <div className="project-card-meta">
        <span>Owner</span>
        <Text code>{project.owner_username || project.owner_nickname || project.owner_id}</Text>
        <span>Version</span>
        <Text>{project.version}</Text>
      </div>
    </Card>
  )
}
