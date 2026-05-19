package service

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	taskv1 "task-platform/gen/go/task/v1"
	"task-platform/internal/task/data"
)

type mockBiz struct {
	createProjectFn  func(ctx context.Context, callerID, name, description string) (*data.Project, error)
	updateProjectFn  func(ctx context.Context, projectID, callerID, name, description string, version int64) (*data.Project, error)
	archiveProjectFn func(ctx context.Context, projectID, callerID string) (*data.Project, error)
	unarchiveFn      func(ctx context.Context, projectID, callerID string) (*data.Project, error)
	transferFn       func(ctx context.Context, projectID, callerID, targetUserID string) (*data.Project, error)
	addMemberFn      func(ctx context.Context, projectID, callerID, targetUserID string, role int32) (*data.ProjectMember, error)
	removeMemberFn   func(ctx context.Context, projectID, callerID, targetUserID string) (*data.ProjectMember, error)
	updateRoleFn     func(ctx context.Context, projectID, callerID, targetUserID string, role int32) (*data.ProjectMember, error)
	leaveFn          func(ctx context.Context, projectID, callerID string) (*data.ProjectMember, error)
	listMembersFn    func(ctx context.Context, projectID, callerID string) ([]*data.ProjectMember, error)
	checkMemberFn    func(ctx context.Context, projectID, userID string) (bool, int32, error)
	listProjectsFn   func(ctx context.Context, callerID string, includeArchived bool, limit, offset int) ([]*data.Project, error)
	getProjectFn     func(ctx context.Context, projectID, callerID string) (*data.Project, error)
}

func (m *mockBiz) CreateProject(ctx context.Context, callerID, name, description string) (*data.Project, error) {
	return m.createProjectFn(ctx, callerID, name, description)
}
func (m *mockBiz) UpdateProject(ctx context.Context, projectID, callerID, name, description string, version int64) (*data.Project, error) {
	return m.updateProjectFn(ctx, projectID, callerID, name, description, version)
}
func (m *mockBiz) ArchiveProject(ctx context.Context, projectID, callerID string) (*data.Project, error) {
	return m.archiveProjectFn(ctx, projectID, callerID)
}
func (m *mockBiz) UnarchiveProject(ctx context.Context, projectID, callerID string) (*data.Project, error) {
	return m.unarchiveFn(ctx, projectID, callerID)
}
func (m *mockBiz) TransferOwnership(ctx context.Context, projectID, callerID, targetUserID string) (*data.Project, error) {
	return m.transferFn(ctx, projectID, callerID, targetUserID)
}
func (m *mockBiz) AddProjectMember(ctx context.Context, projectID, callerID, targetUserID string, role int32) (*data.ProjectMember, error) {
	return m.addMemberFn(ctx, projectID, callerID, targetUserID, role)
}
func (m *mockBiz) RemoveProjectMember(ctx context.Context, projectID, callerID, targetUserID string) (*data.ProjectMember, error) {
	return m.removeMemberFn(ctx, projectID, callerID, targetUserID)
}
func (m *mockBiz) UpdateProjectMemberRole(ctx context.Context, projectID, callerID, targetUserID string, role int32) (*data.ProjectMember, error) {
	return m.updateRoleFn(ctx, projectID, callerID, targetUserID, role)
}
func (m *mockBiz) LeaveProject(ctx context.Context, projectID, callerID string) (*data.ProjectMember, error) {
	return m.leaveFn(ctx, projectID, callerID)
}
func (m *mockBiz) ListProjectMembers(ctx context.Context, projectID, callerID string) ([]*data.ProjectMember, error) {
	return m.listMembersFn(ctx, projectID, callerID)
}
func (m *mockBiz) CheckProjectMember(ctx context.Context, projectID, userID string) (bool, int32, error) {
	return m.checkMemberFn(ctx, projectID, userID)
}
func (m *mockBiz) ListProjects(ctx context.Context, callerID string, includeArchived bool, limit, offset int) ([]*data.Project, error) {
	return m.listProjectsFn(ctx, callerID, includeArchived, limit, offset)
}
func (m *mockBiz) GetProject(ctx context.Context, projectID, callerID string) (*data.Project, error) {
	return m.getProjectFn(ctx, projectID, callerID)
}

func ctxWithUser(userID, username string) context.Context {
	ctx := context.Background()
	ctx = WithUserID(ctx, userID)
	ctx = WithUsername(ctx, username)
	return ctx
}

func TestCreateProject_MissingCallerID(t *testing.T) {
	svc := &TaskService{}
	_, err := svc.CreateProject(context.Background(), &taskv1.CreateProjectRequest{Name: "Test"})
	st, _ := status.FromError(err)
	if st.Code() != codes.Unauthenticated {
		t.Errorf("code = %s, want Unauthenticated", st.Code())
	}
}

func TestGetUserID_Empty(t *testing.T) {
	if id := GetUserID(context.Background()); id != "" {
		t.Errorf("expected empty, got %s", id)
	}
}

func TestGetUsername_Empty(t *testing.T) {
	if name := GetUsername(context.Background()); name != "" {
		t.Errorf("expected empty, got %s", name)
	}
}

func TestWithUserID(t *testing.T) {
	ctx := WithUserID(context.Background(), "user-1")
	if id := GetUserID(ctx); id != "user-1" {
		t.Errorf("id = %s, want user-1", id)
	}
}

func TestWithUsername(t *testing.T) {
	ctx := WithUsername(context.Background(), "alice")
	if name := GetUsername(ctx); name != "alice" {
		t.Errorf("name = %s, want alice", name)
	}
}

func TestCreateProject_Success(t *testing.T) {
	mock := &mockBiz{
		createProjectFn: func(_ context.Context, callerID, name, _ string) (*data.Project, error) {
			return &data.Project{ID: "p1", Name: name, OwnerID: callerID}, nil
		},
	}
	svc := &TaskService{biz: mock}
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.CreateProject(ctx, &taskv1.CreateProjectRequest{Name: "Test"})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if resp.Project.Id != "p1" {
		t.Errorf("id = %s", resp.Project.Id)
	}
}

func TestCreateProject_BizError(t *testing.T) {
	mock := &mockBiz{
		createProjectFn: func(_ context.Context, _, _, _ string) (*data.Project, error) {
			return nil, status.Error(codes.InvalidArgument, "bad name")
		},
	}
	svc := &TaskService{biz: mock}
	ctx := ctxWithUser("user-1", "alice")

	_, err := svc.CreateProject(ctx, &taskv1.CreateProjectRequest{Name: ""})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("code = %s, want InvalidArgument", status.Code(err))
	}
}

func TestUpdateProject_Success(t *testing.T) {
	mock := &mockBiz{
		updateProjectFn: func(_ context.Context, projectID, _, name, _ string, version int64) (*data.Project, error) {
			return &data.Project{ID: projectID, Name: name, Version: version + 1}, nil
		},
	}
	svc := &TaskService{biz: mock}
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.UpdateProject(ctx, &taskv1.UpdateProjectRequest{ProjectId: "p1", Name: "New", Version: 1})
	if err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}
	if resp.Project.Name != "New" {
		t.Errorf("name = %s", resp.Project.Name)
	}
}

func TestArchiveProject_Success(t *testing.T) {
	mock := &mockBiz{
		archiveProjectFn: func(_ context.Context, projectID, _ string) (*data.Project, error) {
			return &data.Project{ID: projectID, Status: data.ProjectStatusArchived}, nil
		},
	}
	svc := &TaskService{biz: mock}
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.ArchiveProject(ctx, &taskv1.ArchiveProjectRequest{ProjectId: "p1"})
	if err != nil {
		t.Fatalf("ArchiveProject: %v", err)
	}
	if resp.Project.Status != data.ProjectStatusArchived {
		t.Errorf("status = %d", resp.Project.Status)
	}
}

func TestUnarchiveProject_Success(t *testing.T) {
	mock := &mockBiz{
		unarchiveFn: func(_ context.Context, projectID, _ string) (*data.Project, error) {
			return &data.Project{ID: projectID, Status: data.ProjectStatusActive}, nil
		},
	}
	svc := &TaskService{biz: mock}
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.UnarchiveProject(ctx, &taskv1.UnarchiveProjectRequest{ProjectId: "p1"})
	if err != nil {
		t.Fatalf("UnarchiveProject: %v", err)
	}
	if resp.Project.Status != data.ProjectStatusActive {
		t.Errorf("status = %d", resp.Project.Status)
	}
}

func TestTransferProjectOwnership_Success(t *testing.T) {
	mock := &mockBiz{
		transferFn: func(_ context.Context, projectID, _, targetUserID string) (*data.Project, error) {
			return &data.Project{ID: projectID, OwnerID: targetUserID}, nil
		},
	}
	svc := &TaskService{biz: mock}
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.TransferProjectOwnership(ctx, &taskv1.TransferProjectOwnershipRequest{
		ProjectId: "p1", TargetUserId: "user-2",
	})
	if err != nil {
		t.Fatalf("TransferProjectOwnership: %v", err)
	}
	if resp.Project.OwnerId != "user-2" {
		t.Errorf("owner = %s", resp.Project.OwnerId)
	}
}

func TestListProjects_Success(t *testing.T) {
	mock := &mockBiz{
		listProjectsFn: func(_ context.Context, _ string, _ bool, _, _ int) ([]*data.Project, error) {
			return []*data.Project{{ID: "p1"}, {ID: "p2"}}, nil
		},
	}
	svc := &TaskService{biz: mock}
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.ListProjects(ctx, &taskv1.ListProjectsRequest{Limit: 20})
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(resp.Projects) != 2 {
		t.Errorf("len = %d, want 2", len(resp.Projects))
	}
}

func TestGetProject_Success(t *testing.T) {
	mock := &mockBiz{
		getProjectFn: func(_ context.Context, projectID, _ string) (*data.Project, error) {
			return &data.Project{ID: projectID, Name: "Test"}, nil
		},
	}
	svc := &TaskService{biz: mock}
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.GetProject(ctx, &taskv1.GetProjectRequest{ProjectId: "p1"})
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if resp.Project.Name != "Test" {
		t.Errorf("name = %s", resp.Project.Name)
	}
}

func TestGetProject_Error(t *testing.T) {
	mock := &mockBiz{
		getProjectFn: func(_ context.Context, _, _ string) (*data.Project, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	svc := &TaskService{biz: mock}
	ctx := ctxWithUser("user-1", "alice")

	_, err := svc.GetProject(ctx, &taskv1.GetProjectRequest{ProjectId: "p1"})
	if status.Code(err) != codes.NotFound {
		t.Errorf("code = %s, want NotFound", status.Code(err))
	}
}

func TestAddProjectMember_Success(t *testing.T) {
	mock := &mockBiz{
		addMemberFn: func(_ context.Context, projectID, _, userID string, role int32) (*data.ProjectMember, error) {
			return &data.ProjectMember{ID: "m1", ProjectID: projectID, UserID: userID, Role: role}, nil
		},
	}
	svc := &TaskService{biz: mock}
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.AddProjectMember(ctx, &taskv1.AddProjectMemberRequest{
		ProjectId: "p1", UserId: "user-2", Role: data.RoleMember,
	})
	if err != nil {
		t.Fatalf("AddProjectMember: %v", err)
	}
	if resp.Member.Role != data.RoleMember {
		t.Errorf("role = %d", resp.Member.Role)
	}
}

func TestRemoveProjectMember_Success(t *testing.T) {
	mock := &mockBiz{
		removeMemberFn: func(_ context.Context, projectID, _, userID string) (*data.ProjectMember, error) {
			return &data.ProjectMember{ID: "m1", ProjectID: projectID, UserID: userID}, nil
		},
	}
	svc := &TaskService{biz: mock}
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.RemoveProjectMember(ctx, &taskv1.RemoveProjectMemberRequest{
		ProjectId: "p1", UserId: "user-2",
	})
	if err != nil {
		t.Fatalf("RemoveProjectMember: %v", err)
	}
	if resp.Member.UserId != "user-2" {
		t.Errorf("user_id = %s", resp.Member.UserId)
	}
}

func TestUpdateProjectMemberRole_Success(t *testing.T) {
	mock := &mockBiz{
		updateRoleFn: func(_ context.Context, projectID, _, userID string, role int32) (*data.ProjectMember, error) {
			return &data.ProjectMember{ID: "m1", ProjectID: projectID, UserID: userID, Role: role}, nil
		},
	}
	svc := &TaskService{biz: mock}
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.UpdateProjectMemberRole(ctx, &taskv1.UpdateProjectMemberRoleRequest{
		ProjectId: "p1", UserId: "user-2", Role: data.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("UpdateProjectMemberRole: %v", err)
	}
	if resp.Member.Role != data.RoleAdmin {
		t.Errorf("role = %d", resp.Member.Role)
	}
}

func TestLeaveProject_Success(t *testing.T) {
	mock := &mockBiz{
		leaveFn: func(_ context.Context, projectID, userID string) (*data.ProjectMember, error) {
			return &data.ProjectMember{ID: "m1", ProjectID: projectID, UserID: userID}, nil
		},
	}
	svc := &TaskService{biz: mock}
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.LeaveProject(ctx, &taskv1.LeaveProjectRequest{ProjectId: "p1"})
	if err != nil {
		t.Fatalf("LeaveProject: %v", err)
	}
	if resp.Member.UserId != "user-1" {
		t.Errorf("user_id = %s", resp.Member.UserId)
	}
}

func TestListProjectMembers_Success(t *testing.T) {
	mock := &mockBiz{
		listMembersFn: func(_ context.Context, projectID, _ string) ([]*data.ProjectMember, error) {
			return []*data.ProjectMember{
				{ID: "m1", ProjectID: projectID, UserID: "user-1", Role: data.RoleOwner},
				{ID: "m2", ProjectID: projectID, UserID: "user-2", Role: data.RoleMember},
			}, nil
		},
	}
	svc := &TaskService{biz: mock}
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.ListProjectMembers(ctx, &taskv1.ListProjectMembersRequest{ProjectId: "p1"})
	if err != nil {
		t.Fatalf("ListProjectMembers: %v", err)
	}
	if len(resp.Members) != 2 {
		t.Errorf("len = %d, want 2", len(resp.Members))
	}
}

func TestCheckProjectMember_Success(t *testing.T) {
	mock := &mockBiz{
		checkMemberFn: func(_ context.Context, _, _ string) (bool, int32, error) {
			return true, int32(0), nil
		},
	}
	svc := &TaskService{biz: mock}

	resp, err := svc.CheckProjectMember(context.Background(), &taskv1.CheckProjectMemberRequest{
		ProjectId: "p1", UserId: "user-1",
	})
	if err != nil {
		t.Fatalf("CheckProjectMember: %v", err)
	}
	if !resp.IsMember {
		t.Error("expected member")
	}
}

func TestCheckProjectMember_NotMember(t *testing.T) {
	mock := &mockBiz{
		checkMemberFn: func(_ context.Context, _, _ string) (bool, int32, error) {
			return false, int32(0), nil
		},
	}
	svc := &TaskService{biz: mock}

	resp, err := svc.CheckProjectMember(context.Background(), &taskv1.CheckProjectMemberRequest{
		ProjectId: "p1", UserId: "user-99",
	})
	if err != nil {
		t.Fatalf("CheckProjectMember: %v", err)
	}
	if resp.IsMember {
		t.Error("expected not member")
	}
}

func TestCheckProjectMember_Error(t *testing.T) {
	mock := &mockBiz{
		checkMemberFn: func(_ context.Context, _, _ string) (bool, int32, error) {
			return false, int32(0), status.Error(codes.Internal, "db error")
		},
	}
	svc := &TaskService{biz: mock}

	_, err := svc.CheckProjectMember(context.Background(), &taskv1.CheckProjectMemberRequest{
		ProjectId: "p1", UserId: "user-1",
	})
	if status.Code(err) != codes.Internal {
		t.Errorf("code = %s, want Internal", status.Code(err))
	}
}

func TestListProjects_EmptyList(t *testing.T) {
	mock := &mockBiz{
		listProjectsFn: func(_ context.Context, _ string, _ bool, _, _ int) ([]*data.Project, error) {
			return nil, nil
		},
	}
	svc := &TaskService{biz: mock}
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.ListProjects(ctx, &taskv1.ListProjectsRequest{})
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(resp.Projects) != 0 {
		t.Error("expected empty projects")
	}
}

func TestNewTaskService(t *testing.T) {
	svc := NewTaskService(nil)
	if svc == nil {
		t.Error("expected non-nil TaskService")
	}
}

func TestToProtoProject(t *testing.T) {
	p := &data.Project{
		ID: "p1", Name: "Test", Description: "desc",
		OwnerID: "user-1", Status: data.ProjectStatusActive, Version: 3,
	}
	proto := toProtoProject(p)
	if proto.Id != "p1" || proto.Name != "Test" || proto.Version != 3 {
		t.Error("proto conversion mismatch")
	}
}

func TestToProtoMember(t *testing.T) {
	m := &data.ProjectMember{
		ID: "m1", ProjectID: "p1", UserID: "user-1", Role: data.RoleOwner,
	}
	proto := toProtoMember(m)
	if proto.Id != "m1" || proto.Role != data.RoleOwner {
		t.Error("proto member conversion mismatch")
	}
}
