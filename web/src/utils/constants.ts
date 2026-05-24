export const Role = { Owner: 0, Admin: 1, Member: 2 } as const
export type Role = (typeof Role)[keyof typeof Role]

export const TaskStatus = { Todo: 0, Doing: 1, Done: 2, Cancelled: 3 } as const
export type TaskStatus = (typeof TaskStatus)[keyof typeof TaskStatus]

export const Priority = { Low: 0, Normal: 1, High: 2, Urgent: 3 } as const
export type Priority = (typeof Priority)[keyof typeof Priority]

export const ProjectStatus = { Active: 0, Archived: 1 } as const
export type ProjectStatus = (typeof ProjectStatus)[keyof typeof ProjectStatus]

export const RoleLabels: Record<Role, string> = {
  [Role.Owner]: '拥有者',
  [Role.Admin]: '管理员',
  [Role.Member]: '成员',
}

export const RoleColors: Record<Role, string> = {
  [Role.Owner]: 'red',
  [Role.Admin]: 'orange',
  [Role.Member]: 'blue',
}

export const TaskStatusLabels: Record<TaskStatus, string> = {
  [TaskStatus.Todo]: '待办',
  [TaskStatus.Doing]: '进行中',
  [TaskStatus.Done]: '已完成',
  [TaskStatus.Cancelled]: '已取消',
}

export const TaskStatusColors: Record<TaskStatus, string> = {
  [TaskStatus.Todo]: 'default',
  [TaskStatus.Doing]: 'processing',
  [TaskStatus.Done]: 'success',
  [TaskStatus.Cancelled]: 'warning',
}

export const PriorityLabels: Record<Priority, string> = {
  [Priority.Low]: '低',
  [Priority.Normal]: '普通',
  [Priority.High]: '高',
  [Priority.Urgent]: '紧急',
}

export const PriorityColors: Record<Priority, string> = {
  [Priority.Low]: 'default',
  [Priority.Normal]: 'blue',
  [Priority.High]: 'orange',
  [Priority.Urgent]: 'red',
}

export const ProjectStatusLabels: Record<ProjectStatus, string> = {
  [ProjectStatus.Active]: '活跃',
  [ProjectStatus.Archived]: '已归档',
}

export const ProjectStatusColors: Record<ProjectStatus, string> = {
  [ProjectStatus.Active]: 'green',
  [ProjectStatus.Archived]: 'default',
}

export const ActionLabel = {
  'task.create': '创建了任务',
  'task.update': '更新了任务',
  'task.assign': '指派了任务',
  'task.status_change': '变更了任务状态',
  'task.delete': '删除了任务',
  'comment.create': '发表了评论',
  'comment.delete': '删除了评论',
  'member.add': '添加了成员',
  'member.remove': '移除了成员',
  'member.role_change': '修改了成员角色',
  'member.leave': '退出了项目',
  'project.create': '创建了项目',
  'project.update': '更新了项目',
  'project.archive': '归档了项目',
  'project.unarchive': '取消了归档',
  'project.transfer_ownership': '转让了项目',
} as const

export type ActionType = keyof typeof ActionLabel
