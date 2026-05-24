import type { ActionType, Priority, ProjectStatus, Role, TaskStatus } from '@/utils/constants'

export type ApiCode =
  | 'OK'
  | 'INVALID_ARGUMENT'
  | 'UNAUTHENTICATED'
  | 'PERMISSION_DENIED'
  | 'NOT_FOUND'
  | 'ALREADY_EXISTS'
  | 'FAILED_PRECONDITION'
  | 'ABORTED'
  | 'RESOURCE_EXHAUSTED'
  | 'INTERNAL'
  | 'UNAVAILABLE'
  | 'DEADLINE_EXCEEDED'
  | 'NETWORK_ERROR'

export interface FieldDetail {
  field: string
  reason: string
}

export interface ApiEnvelope<T = unknown> {
  code: ApiCode | string
  message: string
  request_id: string
  details?: FieldDetail[]
  data?: T
}

export interface RestUserEnrichment {
  username?: string
  nickname?: string
  avatar_url?: string
}

export interface RestUser {
  id: string
  username: string
  email: string
  nickname: string
  avatar_url: string
  status: number
}

export interface RestProject {
  id: string
  name: string
  description: string
  owner_id: string
  status: ProjectStatus
  version: number
}

export interface RestProjectMember {
  id: string
  project_id: string
  user_id: string
  role: Role
}

export interface RestTask {
  id: string
  project_id: string
  title: string
  content: string
  status: TaskStatus
  priority: Priority
  assignee_id: string
  creator_id: string
  due_time: string
  version: number
}

export interface RestTaskComment extends RestUserEnrichment {
  id: string
  task_id: string
  user_id: string
  content: string
}

export interface RestOperationLog extends RestUserEnrichment {
  id: string
  project_id?: string
  task_id?: string
  operator_id: string
  action: ActionType | string
  detail_json: string
}

export type User = RestUser
export type Project = RestProject
export type ProjectMember = RestProjectMember
export type Task = RestTask
export type TaskComment = RestTaskComment
export type OperationLog = RestOperationLog

export interface RegisterRequest {
  username: string
  email: string
  password: string
}

export interface LoginRequest {
  account: string
  password: string
}

export interface CreateProjectRequest {
  name: string
  description: string
}

export interface ListProjectsRequest {
  include_archived?: boolean
  limit?: number
  offset?: number
}

export interface UpdateProjectRequest {
  name: string
  description: string
  version: number
}

export interface TransferProjectOwnershipRequest {
  target_user_id: string
}

export interface AddProjectMemberRequest {
  user_id: string
  role: Role
}

export interface UpdateProjectMemberRoleRequest {
  role: Role
}

export interface ListOperationLogsRequest {
  cursor?: string
  limit?: number
}

export interface CreateTaskRequest {
  project_id: string
  title: string
  content: string
}

export interface UpdateTaskRequest {
  title: string
  content: string
  priority: Priority
  due_time: string
  version: number
}

export interface ListTasksRequest {
  assignee_id?: string
  cursor?: string
  keyword?: string
  limit?: number
  project_id: string
  status?: TaskStatus
}

export interface AssignTaskRequest {
  assignee_id: string
}

export interface ChangeTaskStatusRequest {
  status: TaskStatus
  version: number
}

export interface CreateCommentRequest {
  content: string
}

export interface ListCommentsRequest {
  after_id?: string
  limit?: number
}

export interface AuthData {
  access_token: string
  user: RestUser
}

export interface GetMeData {
  user: RestUser
}

export interface CreateProjectData {
  project: RestProject
}

export interface ListProjectsData {
  projects: RestProject[]
}

export interface GetProjectData {
  project: RestProject
}

export interface ProjectData {
  project: RestProject
}

export interface ListProjectMembersData {
  members: RestProjectMember[]
}

export interface ProjectMemberData {
  member: RestProjectMember
}

export interface ListOperationLogsData {
  logs: RestOperationLog[]
  next_cursor: string
}

export interface CreateTaskData {
  task: RestTask
}

export interface ListTasksData {
  tasks: RestTask[]
  next_cursor: string
}

export interface GetTaskData {
  task: RestTask
}

export interface TaskData {
  task: RestTask
}

export interface CreateCommentData {
  comment: RestTaskComment
}

export interface DeleteCommentData {
  comment: RestTaskComment
}

export interface ListCommentsData {
  comments: RestTaskComment[]
  next_cursor?: string
}

export type AuthResponse = AuthData
export type GetMeResponse = GetMeData
export type CreateProjectResponse = CreateProjectData
export type ListProjectsResponse = ListProjectsData
export type GetProjectResponse = GetProjectData
export type ProjectResponse = ProjectData
export type ListProjectMembersResponse = ListProjectMembersData
export type ProjectMemberResponse = ProjectMemberData
export type ListOperationLogsResponse = ListOperationLogsData
export type CreateTaskResponse = CreateTaskData
export type ListTasksResponse = ListTasksData
export type GetTaskResponse = GetTaskData
export type TaskResponse = TaskData
export type CreateCommentResponse = CreateCommentData
export type DeleteCommentResponse = DeleteCommentData
export type ListCommentsResponse = ListCommentsData
