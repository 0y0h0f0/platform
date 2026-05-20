package service

import (
	"context"

	taskv1 "task-platform/gen/go/task/v1"
	"task-platform/internal/task/biz"
	"task-platform/internal/task/data"
	"task-platform/pkg/xerr"
)

type projectBiz interface {
	CreateProject(ctx context.Context, callerID, name, description string) (*data.Project, error)
	UpdateProject(ctx context.Context, projectID, callerID, name, description string, version int64) (*data.Project, error)
	ArchiveProject(ctx context.Context, projectID, callerID string) (*data.Project, error)
	UnarchiveProject(ctx context.Context, projectID, callerID string) (*data.Project, error)
	TransferOwnership(ctx context.Context, projectID, callerID, targetUserID string) (*data.Project, error)
	AddProjectMember(ctx context.Context, projectID, callerID, targetUserID string, role int32) (*data.ProjectMember, error)
	RemoveProjectMember(ctx context.Context, projectID, callerID, targetUserID string) (*data.ProjectMember, error)
	UpdateProjectMemberRole(ctx context.Context, projectID, callerID, targetUserID string, role int32) (*data.ProjectMember, error)
	LeaveProject(ctx context.Context, projectID, callerID string) (*data.ProjectMember, error)
	ListProjectMembers(ctx context.Context, projectID, callerID string) ([]*data.ProjectMember, error)
	CheckProjectMember(ctx context.Context, projectID, userID string) (bool, int32, error)
	ListProjects(ctx context.Context, callerID string, includeArchived bool, limit, offset int) ([]*data.Project, error)
	GetProject(ctx context.Context, projectID, callerID string) (*data.Project, error)
}

type taskBiz interface {
	CreateTask(ctx context.Context, projectID, callerID, title, content string) (*data.Task, error)
	UpdateTask(ctx context.Context, taskID, callerID, title, content string, priority int32, dueTime string, version int64) (*data.Task, error)
	DeleteTask(ctx context.Context, taskID, callerID string) (*data.Task, error)
	GetTask(ctx context.Context, taskID, callerID string) (*data.Task, error)
	ListTasks(ctx context.Context, projectID, callerID string, filter biz.TaskListFilter) ([]*data.Task, string, error)
	AssignTask(ctx context.Context, taskID, callerID, assigneeID string) (*data.Task, error)
	ChangeTaskStatus(ctx context.Context, taskID, callerID string, status int32, version int64) (*data.Task, error)
}

type commentBiz interface {
	CreateComment(ctx context.Context, taskID, callerID, content string) (*data.TaskComment, error)
	DeleteComment(ctx context.Context, taskID, commentID, callerID string) (*data.TaskComment, error)
	ListComments(ctx context.Context, taskID, callerID string, limit int32, afterID string) ([]*data.TaskComment, error)
}

type opLogBiz interface {
	ListProjectLogs(ctx context.Context, projectID, callerID string, limit int, cursor string) ([]*data.OperationLog, string, error)
	ListTaskLogs(ctx context.Context, taskID, callerID string, limit int, cursor string) ([]*data.OperationLog, string, error)
}

type TaskService struct {
	taskv1.UnimplementedTaskServiceServer
	biz        projectBiz
	taskBiz    taskBiz
	commentBiz commentBiz
	opLogBiz   opLogBiz
}

func NewTaskService(b *biz.ProjectBiz, tb *biz.TaskBiz, cb commentBiz, opLogBiz opLogBiz) *TaskService {
	return &TaskService{biz: b, taskBiz: tb, commentBiz: cb, opLogBiz: opLogBiz}
}

func (s *TaskService) CreateProject(ctx context.Context, req *taskv1.CreateProjectRequest) (*taskv1.CreateProjectResponse, error) {
	callerID := GetUserID(ctx)
	if callerID == "" {
		return nil, xerr.NewError(xerr.CodeUnauthenticated, "missing user identity")
	}

	project, err := s.biz.CreateProject(ctx, callerID, req.Name, req.Description)
	if err != nil {
		return nil, err
	}
	return &taskv1.CreateProjectResponse{Project: toProtoProject(project)}, nil
}

func (s *TaskService) UpdateProject(ctx context.Context, req *taskv1.UpdateProjectRequest) (*taskv1.UpdateProjectResponse, error) {
	project, err := s.biz.UpdateProject(ctx, req.ProjectId, getCallerID(ctx), req.Name, req.Description, req.Version)
	if err != nil {
		return nil, err
	}
	return &taskv1.UpdateProjectResponse{Project: toProtoProject(project)}, nil
}

func (s *TaskService) ArchiveProject(ctx context.Context, req *taskv1.ArchiveProjectRequest) (*taskv1.ArchiveProjectResponse, error) {
	project, err := s.biz.ArchiveProject(ctx, req.ProjectId, getCallerID(ctx))
	if err != nil {
		return nil, err
	}
	return &taskv1.ArchiveProjectResponse{Project: toProtoProject(project)}, nil
}

func (s *TaskService) UnarchiveProject(ctx context.Context, req *taskv1.UnarchiveProjectRequest) (*taskv1.UnarchiveProjectResponse, error) {
	project, err := s.biz.UnarchiveProject(ctx, req.ProjectId, getCallerID(ctx))
	if err != nil {
		return nil, err
	}
	return &taskv1.UnarchiveProjectResponse{Project: toProtoProject(project)}, nil
}

func (s *TaskService) TransferProjectOwnership(ctx context.Context, req *taskv1.TransferProjectOwnershipRequest) (*taskv1.TransferProjectOwnershipResponse, error) {
	project, err := s.biz.TransferOwnership(ctx, req.ProjectId, getCallerID(ctx), req.TargetUserId)
	if err != nil {
		return nil, err
	}
	return &taskv1.TransferProjectOwnershipResponse{Project: toProtoProject(project)}, nil
}

func (s *TaskService) ListProjects(ctx context.Context, req *taskv1.ListProjectsRequest) (*taskv1.ListProjectsResponse, error) {
	projects, err := s.biz.ListProjects(ctx, getCallerID(ctx), req.IncludeArchived, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, err
	}
	protoProjects := make([]*taskv1.Project, len(projects))
	for i, p := range projects {
		protoProjects[i] = toProtoProject(p)
	}
	return &taskv1.ListProjectsResponse{Projects: protoProjects}, nil
}

func (s *TaskService) GetProject(ctx context.Context, req *taskv1.GetProjectRequest) (*taskv1.GetProjectResponse, error) {
	project, err := s.biz.GetProject(ctx, req.ProjectId, getCallerID(ctx))
	if err != nil {
		return nil, err
	}
	return &taskv1.GetProjectResponse{Project: toProtoProject(project)}, nil
}

func (s *TaskService) AddProjectMember(ctx context.Context, req *taskv1.AddProjectMemberRequest) (*taskv1.AddProjectMemberResponse, error) {
	member, err := s.biz.AddProjectMember(ctx, req.ProjectId, getCallerID(ctx), req.UserId, req.Role)
	if err != nil {
		return nil, err
	}
	return &taskv1.AddProjectMemberResponse{Member: toProtoMember(member)}, nil
}

func (s *TaskService) RemoveProjectMember(ctx context.Context, req *taskv1.RemoveProjectMemberRequest) (*taskv1.RemoveProjectMemberResponse, error) {
	member, err := s.biz.RemoveProjectMember(ctx, req.ProjectId, getCallerID(ctx), req.UserId)
	if err != nil {
		return nil, err
	}
	return &taskv1.RemoveProjectMemberResponse{Member: toProtoMember(member)}, nil
}

func (s *TaskService) UpdateProjectMemberRole(ctx context.Context, req *taskv1.UpdateProjectMemberRoleRequest) (*taskv1.UpdateProjectMemberRoleResponse, error) {
	member, err := s.biz.UpdateProjectMemberRole(ctx, req.ProjectId, getCallerID(ctx), req.UserId, req.Role)
	if err != nil {
		return nil, err
	}
	return &taskv1.UpdateProjectMemberRoleResponse{Member: toProtoMember(member)}, nil
}

func (s *TaskService) LeaveProject(ctx context.Context, req *taskv1.LeaveProjectRequest) (*taskv1.LeaveProjectResponse, error) {
	member, err := s.biz.LeaveProject(ctx, req.ProjectId, getCallerID(ctx))
	if err != nil {
		return nil, err
	}
	return &taskv1.LeaveProjectResponse{Member: toProtoMember(member)}, nil
}

func (s *TaskService) ListProjectMembers(ctx context.Context, req *taskv1.ListProjectMembersRequest) (*taskv1.ListProjectMembersResponse, error) {
	members, err := s.biz.ListProjectMembers(ctx, req.ProjectId, getCallerID(ctx))
	if err != nil {
		return nil, err
	}
	protoMembers := make([]*taskv1.ProjectMember, len(members))
	for i, m := range members {
		protoMembers[i] = toProtoMember(m)
	}
	return &taskv1.ListProjectMembersResponse{Members: protoMembers}, nil
}

func (s *TaskService) CheckProjectMember(ctx context.Context, req *taskv1.CheckProjectMemberRequest) (*taskv1.CheckProjectMemberResponse, error) {
	isMember, role, err := s.biz.CheckProjectMember(ctx, req.ProjectId, req.UserId)
	if err != nil {
		return nil, err
	}
	return &taskv1.CheckProjectMemberResponse{IsMember: isMember, Role: role}, nil
}

func getCallerID(ctx context.Context) string {
	return GetUserID(ctx)
}

func toProtoProject(p *data.Project) *taskv1.Project {
	return &taskv1.Project{
		Id:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		OwnerId:     p.OwnerID,
		Status:      p.Status,
		Version:     p.Version,
	}
}

func toProtoMember(m *data.ProjectMember) *taskv1.ProjectMember {
	return &taskv1.ProjectMember{
		Id:        m.ID,
		ProjectId: m.ProjectID,
		UserId:    m.UserID,
		Role:      m.Role,
	}
}

func toProtoTask(t *data.Task) *taskv1.Task {
	task := &taskv1.Task{
		Id:        t.ID,
		ProjectId: t.ProjectID,
		Title:     t.Title,
		Content:   t.Content,
		Status:    t.Status,
		Priority:  t.Priority,
		CreatorId: t.CreatorID,
		Version:   t.Version,
	}
	if t.AssigneeID != nil {
		task.AssigneeId = *t.AssigneeID
	}
	if t.DueTime != nil {
		task.DueTime = *t.DueTime
	}
	return task
}

func (s *TaskService) CreateTask(ctx context.Context, req *taskv1.CreateTaskRequest) (*taskv1.CreateTaskResponse, error) {
	task, err := s.taskBiz.CreateTask(ctx, req.ProjectId, getCallerID(ctx), req.Title, req.Content)
	if err != nil {
		return nil, err
	}
	return &taskv1.CreateTaskResponse{Task: toProtoTask(task)}, nil
}

func (s *TaskService) UpdateTask(ctx context.Context, req *taskv1.UpdateTaskRequest) (*taskv1.UpdateTaskResponse, error) {
	task, err := s.taskBiz.UpdateTask(ctx, req.TaskId, getCallerID(ctx), req.Title, req.Content, req.Priority, req.DueTime, req.Version)
	if err != nil {
		return nil, err
	}
	return &taskv1.UpdateTaskResponse{Task: toProtoTask(task)}, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, req *taskv1.DeleteTaskRequest) (*taskv1.DeleteTaskResponse, error) {
	task, err := s.taskBiz.DeleteTask(ctx, req.TaskId, getCallerID(ctx))
	if err != nil {
		return nil, err
	}
	return &taskv1.DeleteTaskResponse{Task: toProtoTask(task)}, nil
}

func (s *TaskService) GetTask(ctx context.Context, req *taskv1.GetTaskRequest) (*taskv1.GetTaskResponse, error) {
	task, err := s.taskBiz.GetTask(ctx, req.TaskId, getCallerID(ctx))
	if err != nil {
		return nil, err
	}
	return &taskv1.GetTaskResponse{Task: toProtoTask(task)}, nil
}

func (s *TaskService) ListTasks(ctx context.Context, req *taskv1.ListTasksRequest) (*taskv1.ListTasksResponse, error) {
	filter := biz.TaskListFilter{
		AssigneeID: req.AssigneeId,
		Keyword:    req.Keyword,
		Limit:      int(req.Limit),
		Cursor:     req.Cursor,
	}
	if req.Status != -1 {
		filter.Status = &req.Status
	}
	tasks, nextCursor, err := s.taskBiz.ListTasks(ctx, req.ProjectId, getCallerID(ctx), filter)
	if err != nil {
		return nil, err
	}
	protoTasks := make([]*taskv1.Task, len(tasks))
	for i, t := range tasks {
		protoTasks[i] = toProtoTask(t)
	}
	return &taskv1.ListTasksResponse{Tasks: protoTasks, NextCursor: nextCursor}, nil
}

func (s *TaskService) AssignTask(ctx context.Context, req *taskv1.AssignTaskRequest) (*taskv1.AssignTaskResponse, error) {
	task, err := s.taskBiz.AssignTask(ctx, req.TaskId, getCallerID(ctx), req.AssigneeId)
	if err != nil {
		return nil, err
	}
	return &taskv1.AssignTaskResponse{Task: toProtoTask(task)}, nil
}

func (s *TaskService) ChangeTaskStatus(ctx context.Context, req *taskv1.ChangeTaskStatusRequest) (*taskv1.ChangeTaskStatusResponse, error) {
	task, err := s.taskBiz.ChangeTaskStatus(ctx, req.TaskId, getCallerID(ctx), req.Status, req.Version)
	if err != nil {
		return nil, err
	}
	return &taskv1.ChangeTaskStatusResponse{Task: toProtoTask(task)}, nil
}

func (s *TaskService) CreateTaskComment(ctx context.Context, req *taskv1.CreateTaskCommentRequest) (*taskv1.CreateTaskCommentResponse, error) {
	comment, err := s.commentBiz.CreateComment(ctx, req.TaskId, getCallerID(ctx), req.Content)
	if err != nil {
		return nil, err
	}
	return &taskv1.CreateTaskCommentResponse{Comment: toProtoComment(comment)}, nil
}

func (s *TaskService) DeleteTaskComment(ctx context.Context, req *taskv1.DeleteTaskCommentRequest) (*taskv1.DeleteTaskCommentResponse, error) {
	comment, err := s.commentBiz.DeleteComment(ctx, req.TaskId, req.CommentId, getCallerID(ctx))
	if err != nil {
		return nil, err
	}
	return &taskv1.DeleteTaskCommentResponse{Comment: toProtoComment(comment)}, nil
}

func (s *TaskService) ListTaskComments(ctx context.Context, req *taskv1.ListTaskCommentsRequest) (*taskv1.ListTaskCommentsResponse, error) {
	comments, err := s.commentBiz.ListComments(ctx, req.TaskId, getCallerID(ctx), req.Limit, req.AfterId)
	if err != nil {
		return nil, err
	}
	protoComments := make([]*taskv1.TaskComment, len(comments))
	for i, c := range comments {
		protoComments[i] = toProtoComment(c)
	}
	return &taskv1.ListTaskCommentsResponse{Comments: protoComments}, nil
}

func (s *TaskService) ListOperationLogs(ctx context.Context, req *taskv1.ListOperationLogsRequest) (*taskv1.ListOperationLogsResponse, error) {
	callerID := GetUserID(ctx)
	if callerID == "" {
		return nil, xerr.NewError(xerr.CodeUnauthenticated, "missing user identity")
	}

	var logs []*data.OperationLog
	var nextCursor string
	var err error

	limit := int(req.Limit)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	if req.ProjectId != "" {
		logs, nextCursor, err = s.opLogBiz.ListProjectLogs(ctx, req.ProjectId, callerID, limit, req.Cursor)
	} else if req.TaskId != "" {
		logs, nextCursor, err = s.opLogBiz.ListTaskLogs(ctx, req.TaskId, callerID, limit, req.Cursor)
	} else {
		return nil, xerr.NewError(xerr.CodeInvalidArgument, "project_id or task_id is required")
	}
	if err != nil {
		return nil, err
	}

	protoLogs := make([]*taskv1.OperationLog, len(logs))
	for i, l := range logs {
		protoLogs[i] = toProtoOperationLog(l)
	}
	return &taskv1.ListOperationLogsResponse{Logs: protoLogs, NextCursor: nextCursor}, nil
}

func toProtoComment(c *data.TaskComment) *taskv1.TaskComment {
	return &taskv1.TaskComment{
		Id:      c.ID,
		TaskId:  c.TaskID,
		UserId:  c.UserID,
		Content: c.Content,
	}
}

func toProtoOperationLog(l *data.OperationLog) *taskv1.OperationLog {
	ol := &taskv1.OperationLog{
		Id:         l.ID,
		OperatorId: l.OperatorID,
		Action:     l.Action,
		DetailJson: l.Detail,
	}
	if l.ProjectID != nil {
		ol.ProjectId = *l.ProjectID
	}
	if l.TaskID != nil {
		ol.TaskId = *l.TaskID
	}
	return ol
}
