package service

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	taskv1 "task-platform/gen/go/task/v1"
	"task-platform/internal/task/biz"
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
	createTaskFn     func(ctx context.Context, projectID, callerID, title, content string) (*data.Task, error)
	updateTaskFn     func(ctx context.Context, taskID, callerID, title, content string, priority int32, dueTime string, version int64) (*data.Task, error)
	deleteTaskFn     func(ctx context.Context, taskID, callerID string) (*data.Task, error)
	getTaskFn        func(ctx context.Context, taskID, callerID string) (*data.Task, error)
	listTasksFn      func(ctx context.Context, projectID, callerID string, filter biz.TaskListFilter) ([]*data.Task, string, error)
	assignTaskFn     func(ctx context.Context, taskID, callerID, assigneeID string) (*data.Task, error)
	changeStatusFn   func(ctx context.Context, taskID, callerID string, status int32, version int64) (*data.Task, error)
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
func (m *mockBiz) CreateTask(ctx context.Context, projectID, callerID, title, content string) (*data.Task, error) {
	return m.createTaskFn(ctx, projectID, callerID, title, content)
}
func (m *mockBiz) UpdateTask(ctx context.Context, taskID, callerID, title, content string, priority int32, dueTime string, version int64) (*data.Task, error) {
	return m.updateTaskFn(ctx, taskID, callerID, title, content, priority, dueTime, version)
}
func (m *mockBiz) DeleteTask(ctx context.Context, taskID, callerID string) (*data.Task, error) {
	return m.deleteTaskFn(ctx, taskID, callerID)
}
func (m *mockBiz) GetTask(ctx context.Context, taskID, callerID string) (*data.Task, error) {
	return m.getTaskFn(ctx, taskID, callerID)
}
func (m *mockBiz) ListTasks(ctx context.Context, projectID, callerID string, filter biz.TaskListFilter) ([]*data.Task, string, error) {
	return m.listTasksFn(ctx, projectID, callerID, filter)
}
func (m *mockBiz) AssignTask(ctx context.Context, taskID, callerID, assigneeID string) (*data.Task, error) {
	return m.assignTaskFn(ctx, taskID, callerID, assigneeID)
}
func (m *mockBiz) ChangeTaskStatus(ctx context.Context, taskID, callerID string, status int32, version int64) (*data.Task, error) {
	return m.changeStatusFn(ctx, taskID, callerID, status, version)
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
	svc := NewTaskService(nil, nil)
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

// --- Task service tests ---

func makeTaskSvc(mock *mockBiz) *TaskService {
	return &TaskService{biz: mock, taskBiz: mock}
}

func TestCreateTask_Success(t *testing.T) {
	mock := &mockBiz{
		createTaskFn: func(_ context.Context, projectID, callerID, title, content string) (*data.Task, error) {
			return &data.Task{ID: "t1", ProjectID: projectID, Title: title, CreatorID: callerID, Status: data.TaskStatusTodo}, nil
		},
	}
	svc := makeTaskSvc(mock)
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.CreateTask(ctx, &taskv1.CreateTaskRequest{
		ProjectId: "proj-1", Title: "My Task", Content: "desc",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if resp.Task.Id != "t1" || resp.Task.Title != "My Task" {
		t.Errorf("task = %+v", resp.Task)
	}
}

func TestCreateTask_BizError(t *testing.T) {
	mock := &mockBiz{
		createTaskFn: func(_ context.Context, _, _, _, _ string) (*data.Task, error) {
			return nil, status.Error(codes.InvalidArgument, "invalid title")
		},
	}
	svc := makeTaskSvc(mock)
	ctx := ctxWithUser("user-1", "alice")

	_, err := svc.CreateTask(ctx, &taskv1.CreateTaskRequest{ProjectId: "proj-1", Title: ""})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("code = %s, want InvalidArgument", status.Code(err))
	}
}

func TestUpdateTask_Success(t *testing.T) {
	mock := &mockBiz{
		updateTaskFn: func(_ context.Context, taskID, _, title, _ string, priority int32, _ string, version int64) (*data.Task, error) {
			return &data.Task{ID: taskID, Title: title, Priority: priority, Version: version + 1}, nil
		},
	}
	svc := makeTaskSvc(mock)
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.UpdateTask(ctx, &taskv1.UpdateTaskRequest{
		TaskId: "t1", Title: "New Title", Priority: 2, Version: 0,
	})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if resp.Task.Title != "New Title" || resp.Task.Priority != 2 {
		t.Errorf("task = %+v", resp.Task)
	}
}

func TestUpdateTask_BizError(t *testing.T) {
	mock := &mockBiz{
		updateTaskFn: func(_ context.Context, _, _, _, _ string, _ int32, _ string, _ int64) (*data.Task, error) {
			return nil, status.Error(codes.Aborted, "version conflict")
		},
	}
	svc := makeTaskSvc(mock)
	ctx := ctxWithUser("user-1", "alice")

	_, err := svc.UpdateTask(ctx, &taskv1.UpdateTaskRequest{TaskId: "t1", Title: "X", Version: 999})
	if status.Code(err) != codes.Aborted {
		t.Errorf("code = %s, want Aborted", status.Code(err))
	}
}

func TestDeleteTask_Success(t *testing.T) {
	mock := &mockBiz{
		deleteTaskFn: func(_ context.Context, taskID, _ string) (*data.Task, error) {
			return &data.Task{ID: taskID, Title: "Deleted"}, nil
		},
	}
	svc := makeTaskSvc(mock)
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.DeleteTask(ctx, &taskv1.DeleteTaskRequest{TaskId: "t1"})
	if err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}
	if resp.Task.Id != "t1" {
		t.Errorf("id = %s", resp.Task.Id)
	}
}

func TestDeleteTask_BizError(t *testing.T) {
	mock := &mockBiz{
		deleteTaskFn: func(_ context.Context, _, _ string) (*data.Task, error) {
			return nil, status.Error(codes.PermissionDenied, "not allowed")
		},
	}
	svc := makeTaskSvc(mock)
	ctx := ctxWithUser("user-1", "alice")

	_, err := svc.DeleteTask(ctx, &taskv1.DeleteTaskRequest{TaskId: "t1"})
	if status.Code(err) != codes.PermissionDenied {
		t.Errorf("code = %s, want PermissionDenied", status.Code(err))
	}
}

func TestGetTask_Success(t *testing.T) {
	mock := &mockBiz{
		getTaskFn: func(_ context.Context, taskID, _ string) (*data.Task, error) {
			return &data.Task{ID: taskID, Title: "My Task", ProjectID: "proj-1"}, nil
		},
	}
	svc := makeTaskSvc(mock)
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.GetTask(ctx, &taskv1.GetTaskRequest{TaskId: "t1"})
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if resp.Task.Title != "My Task" {
		t.Errorf("title = %s", resp.Task.Title)
	}
}

func TestGetTask_NotFound(t *testing.T) {
	mock := &mockBiz{
		getTaskFn: func(_ context.Context, _, _ string) (*data.Task, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	svc := makeTaskSvc(mock)
	ctx := ctxWithUser("user-1", "alice")

	_, err := svc.GetTask(ctx, &taskv1.GetTaskRequest{TaskId: "t1"})
	if status.Code(err) != codes.NotFound {
		t.Errorf("code = %s, want NotFound", status.Code(err))
	}
}

func TestListTasks_Success(t *testing.T) {
	mock := &mockBiz{
		listTasksFn: func(_ context.Context, projectID, _ string, _ biz.TaskListFilter) ([]*data.Task, string, error) {
			return []*data.Task{{ID: "t1"}, {ID: "t2"}}, "cursor-abc", nil
		},
	}
	svc := makeTaskSvc(mock)
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.ListTasks(ctx, &taskv1.ListTasksRequest{ProjectId: "proj-1", Limit: 20, Status: -1})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(resp.Tasks) != 2 {
		t.Errorf("len = %d, want 2", len(resp.Tasks))
	}
	if resp.NextCursor != "cursor-abc" {
		t.Errorf("cursor = %s", resp.NextCursor)
	}
}

func TestListTasks_WithStatusFilter(t *testing.T) {
	mock := &mockBiz{
		listTasksFn: func(_ context.Context, _ string, _ string, filter biz.TaskListFilter) ([]*data.Task, string, error) {
			if filter.Status == nil || *filter.Status != data.TaskStatusDoing {
				t.Error("expected status=1 (doing) filter")
			}
			return []*data.Task{{ID: "t1", Status: data.TaskStatusDoing}}, "", nil
		},
	}
	svc := makeTaskSvc(mock)
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.ListTasks(ctx, &taskv1.ListTasksRequest{ProjectId: "proj-1", Limit: 20, Status: 1})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(resp.Tasks) != 1 {
		t.Errorf("len = %d, want 1", len(resp.Tasks))
	}
}

func TestListTasks_WithStatusTodo(t *testing.T) {
	mock := &mockBiz{
		listTasksFn: func(_ context.Context, _ string, _ string, filter biz.TaskListFilter) ([]*data.Task, string, error) {
			if filter.Status == nil || *filter.Status != data.TaskStatusTodo {
				t.Error("expected status=0 (todo) filter")
			}
			return []*data.Task{{ID: "t1", Status: data.TaskStatusTodo}}, "", nil
		},
	}
	svc := makeTaskSvc(mock)
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.ListTasks(ctx, &taskv1.ListTasksRequest{ProjectId: "proj-1", Limit: 20, Status: 0})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(resp.Tasks) != 1 {
		t.Errorf("len = %d, want 1", len(resp.Tasks))
	}
}

func TestListTasks_BizError(t *testing.T) {
	mock := &mockBiz{
		listTasksFn: func(_ context.Context, _, _ string, _ biz.TaskListFilter) ([]*data.Task, string, error) {
			return nil, "", status.Error(codes.NotFound, "not found")
		},
	}
	svc := makeTaskSvc(mock)
	ctx := ctxWithUser("user-1", "alice")

	_, err := svc.ListTasks(ctx, &taskv1.ListTasksRequest{ProjectId: "proj-1", Status: -1})
	if status.Code(err) != codes.NotFound {
		t.Errorf("code = %s, want NotFound", status.Code(err))
	}
}

func TestAssignTask_Success(t *testing.T) {
	mock := &mockBiz{
		assignTaskFn: func(_ context.Context, taskID, _, assigneeID string) (*data.Task, error) {
			return &data.Task{ID: taskID, AssigneeID: &assigneeID}, nil
		},
	}
	svc := makeTaskSvc(mock)
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.AssignTask(ctx, &taskv1.AssignTaskRequest{TaskId: "t1", AssigneeId: "user-2"})
	if err != nil {
		t.Fatalf("AssignTask: %v", err)
	}
	if resp.Task.AssigneeId != "user-2" {
		t.Errorf("assignee = %s", resp.Task.AssigneeId)
	}
}

func TestAssignTask_BizError(t *testing.T) {
	mock := &mockBiz{
		assignTaskFn: func(_ context.Context, _, _, _ string) (*data.Task, error) {
			return nil, status.Error(codes.FailedPrecondition, "assignee not a member")
		},
	}
	svc := makeTaskSvc(mock)
	ctx := ctxWithUser("user-1", "alice")

	_, err := svc.AssignTask(ctx, &taskv1.AssignTaskRequest{TaskId: "t1", AssigneeId: "user-99"})
	if status.Code(err) != codes.FailedPrecondition {
		t.Errorf("code = %s, want FailedPrecondition", status.Code(err))
	}
}

func TestChangeTaskStatus_Success(t *testing.T) {
	mock := &mockBiz{
		changeStatusFn: func(_ context.Context, taskID, _ string, status int32, _ int64) (*data.Task, error) {
			return &data.Task{ID: taskID, Status: status, Version: 1}, nil
		},
	}
	svc := makeTaskSvc(mock)
	ctx := ctxWithUser("user-1", "alice")

	resp, err := svc.ChangeTaskStatus(ctx, &taskv1.ChangeTaskStatusRequest{
		TaskId: "t1", Status: 1, Version: 0,
	})
	if err != nil {
		t.Fatalf("ChangeTaskStatus: %v", err)
	}
	if resp.Task.Status != 1 {
		t.Errorf("status = %d", resp.Task.Status)
	}
}

func TestChangeTaskStatus_BizError(t *testing.T) {
	mock := &mockBiz{
		changeStatusFn: func(_ context.Context, _, _ string, _ int32, _ int64) (*data.Task, error) {
			return nil, status.Error(codes.FailedPrecondition, "invalid transition")
		},
	}
	svc := makeTaskSvc(mock)
	ctx := ctxWithUser("user-1", "alice")

	_, err := svc.ChangeTaskStatus(ctx, &taskv1.ChangeTaskStatusRequest{TaskId: "t1", Status: 3, Version: 0})
	if status.Code(err) != codes.FailedPrecondition {
		t.Errorf("code = %s, want FailedPrecondition", status.Code(err))
	}
}

func TestToProtoTask(t *testing.T) {
	assigneeID := "user-2"
	dueTime := "2025-12-31"
	task := &data.Task{
		ID: "t1", ProjectID: "proj-1", Title: "Test", Content: "body",
		Status: data.TaskStatusDoing, Priority: data.PriorityHigh,
		CreatorID: "user-1", AssigneeID: &assigneeID, DueTime: &dueTime,
		Version: 3,
	}
	proto := toProtoTask(task)
	if proto.Id != "t1" || proto.Title != "Test" || proto.Version != 3 {
		t.Error("proto conversion mismatch")
	}
	if proto.AssigneeId != "user-2" {
		t.Errorf("assignee_id = %s", proto.AssigneeId)
	}
	if proto.DueTime != "2025-12-31" {
		t.Errorf("due_time = %s", proto.DueTime)
	}
}

func TestToProtoTask_NoAssignee(t *testing.T) {
	task := &data.Task{ID: "t1"}
	proto := toProtoTask(task)
	if proto.AssigneeId != "" {
		t.Errorf("assignee_id should be empty, got %s", proto.AssigneeId)
	}
}
