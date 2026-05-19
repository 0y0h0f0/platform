package service

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	taskv1 "task-platform/gen/go/task/v1"
	"task-platform/internal/task/biz"
	"task-platform/internal/task/data"
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

type TaskService struct {
	taskv1.UnimplementedTaskServiceServer
	biz projectBiz
}

func NewTaskService(b *biz.ProjectBiz) *TaskService {
	return &TaskService{biz: b}
}

func (s *TaskService) CreateProject(ctx context.Context, req *taskv1.CreateProjectRequest) (*taskv1.CreateProjectResponse, error) {
	callerID := GetUserID(ctx)
	if callerID == "" {
		return nil, status.Error(codes.Unauthenticated, "missing user identity")
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
