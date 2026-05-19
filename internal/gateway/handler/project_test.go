package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	taskv1 "task-platform/gen/go/task/v1"
	"task-platform/internal/gateway/handler"
	"task-platform/internal/gateway/middleware"
)

type stubTaskClient struct {
	taskv1.TaskServiceClient
	createProjectRes *taskv1.CreateProjectResponse
	createProjectErr error
	listProjectsRes  *taskv1.ListProjectsResponse
	listProjectsErr  error
	getProjectRes    *taskv1.GetProjectResponse
	getProjectErr    error
	updateProjectRes *taskv1.UpdateProjectResponse
	updateProjectErr error
	archiveRes       *taskv1.ArchiveProjectResponse
	archiveErr       error
	unarchiveRes     *taskv1.UnarchiveProjectResponse
	unarchiveErr     error
	transferRes      *taskv1.TransferProjectOwnershipResponse
	transferErr      error
	addMemberRes     *taskv1.AddProjectMemberResponse
	addMemberErr     error
	listMembersRes   *taskv1.ListProjectMembersResponse
	listMembersErr   error
	updateRoleRes    *taskv1.UpdateProjectMemberRoleResponse
	updateRoleErr    error
	removeMemberRes  *taskv1.RemoveProjectMemberResponse
	removeMemberErr  error
	leaveRes         *taskv1.LeaveProjectResponse
	leaveErr         error
}

func (s *stubTaskClient) CreateProject(_ context.Context, _ *taskv1.CreateProjectRequest, _ ...grpc.CallOption) (*taskv1.CreateProjectResponse, error) {
	return s.createProjectRes, s.createProjectErr
}
func (s *stubTaskClient) ListProjects(_ context.Context, _ *taskv1.ListProjectsRequest, _ ...grpc.CallOption) (*taskv1.ListProjectsResponse, error) {
	return s.listProjectsRes, s.listProjectsErr
}
func (s *stubTaskClient) GetProject(_ context.Context, _ *taskv1.GetProjectRequest, _ ...grpc.CallOption) (*taskv1.GetProjectResponse, error) {
	return s.getProjectRes, s.getProjectErr
}
func (s *stubTaskClient) UpdateProject(_ context.Context, _ *taskv1.UpdateProjectRequest, _ ...grpc.CallOption) (*taskv1.UpdateProjectResponse, error) {
	return s.updateProjectRes, s.updateProjectErr
}
func (s *stubTaskClient) ArchiveProject(_ context.Context, _ *taskv1.ArchiveProjectRequest, _ ...grpc.CallOption) (*taskv1.ArchiveProjectResponse, error) {
	return s.archiveRes, s.archiveErr
}
func (s *stubTaskClient) UnarchiveProject(_ context.Context, _ *taskv1.UnarchiveProjectRequest, _ ...grpc.CallOption) (*taskv1.UnarchiveProjectResponse, error) {
	return s.unarchiveRes, s.unarchiveErr
}
func (s *stubTaskClient) TransferProjectOwnership(_ context.Context, _ *taskv1.TransferProjectOwnershipRequest, _ ...grpc.CallOption) (*taskv1.TransferProjectOwnershipResponse, error) {
	return s.transferRes, s.transferErr
}
func (s *stubTaskClient) AddProjectMember(_ context.Context, _ *taskv1.AddProjectMemberRequest, _ ...grpc.CallOption) (*taskv1.AddProjectMemberResponse, error) {
	return s.addMemberRes, s.addMemberErr
}
func (s *stubTaskClient) ListProjectMembers(_ context.Context, _ *taskv1.ListProjectMembersRequest, _ ...grpc.CallOption) (*taskv1.ListProjectMembersResponse, error) {
	return s.listMembersRes, s.listMembersErr
}
func (s *stubTaskClient) UpdateProjectMemberRole(_ context.Context, _ *taskv1.UpdateProjectMemberRoleRequest, _ ...grpc.CallOption) (*taskv1.UpdateProjectMemberRoleResponse, error) {
	return s.updateRoleRes, s.updateRoleErr
}
func (s *stubTaskClient) RemoveProjectMember(_ context.Context, _ *taskv1.RemoveProjectMemberRequest, _ ...grpc.CallOption) (*taskv1.RemoveProjectMemberResponse, error) {
	return s.removeMemberRes, s.removeMemberErr
}
func (s *stubTaskClient) LeaveProject(_ context.Context, _ *taskv1.LeaveProjectRequest, _ ...grpc.CallOption) (*taskv1.LeaveProjectResponse, error) {
	return s.leaveRes, s.leaveErr
}

func setupProjectHandler(t *testing.T) (*handler.ProjectHandler, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	stub := &stubTaskClient{
		createProjectRes: &taskv1.CreateProjectResponse{
			Project: &taskv1.Project{Id: "proj-1", Name: "Test", OwnerId: "user-1"},
		},
		listProjectsRes: &taskv1.ListProjectsResponse{
			Projects: []*taskv1.Project{{Id: "proj-1", Name: "Test"}},
		},
		getProjectRes: &taskv1.GetProjectResponse{
			Project: &taskv1.Project{Id: "proj-1", Name: "Test"},
		},
		updateProjectRes: &taskv1.UpdateProjectResponse{
			Project: &taskv1.Project{Id: "proj-1", Name: "Updated"},
		},
		archiveRes: &taskv1.ArchiveProjectResponse{
			Project: &taskv1.Project{Id: "proj-1", Status: 1},
		},
		unarchiveRes: &taskv1.UnarchiveProjectResponse{
			Project: &taskv1.Project{Id: "proj-1", Status: 0},
		},
		transferRes: &taskv1.TransferProjectOwnershipResponse{
			Project: &taskv1.Project{Id: "proj-1", OwnerId: "user-2"},
		},
		addMemberRes: &taskv1.AddProjectMemberResponse{
			Member: &taskv1.ProjectMember{Id: "mem-1", UserId: "user-2", Role: 2},
		},
		listMembersRes: &taskv1.ListProjectMembersResponse{
			Members: []*taskv1.ProjectMember{{Id: "mem-1", UserId: "user-1", Role: 0}},
		},
		updateRoleRes: &taskv1.UpdateProjectMemberRoleResponse{
			Member: &taskv1.ProjectMember{Id: "mem-1", UserId: "user-2", Role: 1},
		},
		removeMemberRes: &taskv1.RemoveProjectMemberResponse{
			Member: &taskv1.ProjectMember{Id: "mem-1", UserId: "user-2", Role: 2},
		},
		leaveRes: &taskv1.LeaveProjectResponse{
			Member: &taskv1.ProjectMember{Id: "mem-1", UserId: "user-1", Role: 2},
		},
	}

	h := handler.NewProjectHandler(stub)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), middleware.CtxKeyRequestID, "req-123"))
		c.Next()
	})
	return h, r
}

func doRequest(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	r.ServeHTTP(w, req)
	return w
}

func TestProjectCreate_Success(t *testing.T) {
	h, r := setupProjectHandler(t)
	r.POST("/projects", h.Create)

	body := `{"name": "Test Project", "description": "desc"}`
	w := doRequest(r, http.MethodPost, "/projects", body)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestProjectCreate_InvalidBody(t *testing.T) {
	h, r := setupProjectHandler(t)
	r.POST("/projects", h.Create)

	w := doRequest(r, http.MethodPost, "/projects", "not json")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestProjectCreate_GRPCError(t *testing.T) {
	h, r := setupProjectHandler(t)
	r.POST("/projects", h.Create)

	stub := &stubTaskClient{
		createProjectErr: status.Error(codes.InvalidArgument, "invalid name"),
	}
	h2 := handler.NewProjectHandler(stub)

	r2 := gin.New()
	r2.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), middleware.CtxKeyRequestID, "req-123"))
		c.Next()
	})
	r2.POST("/projects", h2.Create)

	w := doRequest(r2, http.MethodPost, "/projects", `{"name": ""}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestProjectList_Success(t *testing.T) {
	h, r := setupProjectHandler(t)
	r.GET("/projects", h.List)

	w := doRequest(r, http.MethodGet, "/projects?limit=10&offset=0", "")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestProjectList_Defaults(t *testing.T) {
	h, r := setupProjectHandler(t)
	r.GET("/projects", h.List)

	w := doRequest(r, http.MethodGet, "/projects", "")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}
}

func TestProjectGet_Success(t *testing.T) {
	h, r := setupProjectHandler(t)
	r.GET("/projects/:id", h.Get)

	w := doRequest(r, http.MethodGet, "/projects/proj-1", "")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestProjectGet_NotFound(t *testing.T) {
	stub := &stubTaskClient{
		getProjectErr: status.Error(codes.NotFound, "not found"),
	}
	h := handler.NewProjectHandler(stub)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), middleware.CtxKeyRequestID, "req-123"))
		c.Next()
	})
	r.GET("/projects/:id", h.Get)

	w := doRequest(r, http.MethodGet, "/projects/proj-1", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestProjectUpdate_Success(t *testing.T) {
	h, r := setupProjectHandler(t)
	r.PUT("/projects/:id", h.Update)

	body := `{"name": "Updated", "description": "new desc", "version": 1}`
	w := doRequest(r, http.MethodPut, "/projects/proj-1", body)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestProjectUpdate_InvalidBody(t *testing.T) {
	h, r := setupProjectHandler(t)
	r.PUT("/projects/:id", h.Update)

	w := doRequest(r, http.MethodPut, "/projects/proj-1", "bad")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestProjectArchive_Success(t *testing.T) {
	h, r := setupProjectHandler(t)
	r.POST("/projects/:id/archive", h.Archive)

	w := doRequest(r, http.MethodPost, "/projects/proj-1/archive", "")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestProjectUnarchive_Success(t *testing.T) {
	h, r := setupProjectHandler(t)
	r.POST("/projects/:id/unarchive", h.Unarchive)

	w := doRequest(r, http.MethodPost, "/projects/proj-1/unarchive", "")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestProjectTransfer_Success(t *testing.T) {
	h, r := setupProjectHandler(t)
	r.POST("/projects/:id/transfer", h.Transfer)

	body := `{"target_user_id": "user-2"}`
	w := doRequest(r, http.MethodPost, "/projects/proj-1/transfer", body)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestProjectTransfer_InvalidBody(t *testing.T) {
	h, r := setupProjectHandler(t)
	r.POST("/projects/:id/transfer", h.Transfer)

	w := doRequest(r, http.MethodPost, "/projects/proj-1/transfer", "bad")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAddMember_Success(t *testing.T) {
	h, r := setupProjectHandler(t)
	r.POST("/projects/:id/members", h.AddMember)

	body := `{"user_id": "user-2", "role": 2}`
	w := doRequest(r, http.MethodPost, "/projects/proj-1/members", body)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestAddMember_InvalidBody(t *testing.T) {
	h, r := setupProjectHandler(t)
	r.POST("/projects/:id/members", h.AddMember)

	w := doRequest(r, http.MethodPost, "/projects/proj-1/members", "bad")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestListMembers_Success(t *testing.T) {
	h, r := setupProjectHandler(t)
	r.GET("/projects/:id/members", h.ListMembers)

	w := doRequest(r, http.MethodGet, "/projects/proj-1/members", "")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestUpdateMemberRole_Success(t *testing.T) {
	h, r := setupProjectHandler(t)
	r.PUT("/projects/:id/members/:userId", h.UpdateMemberRole)

	body := `{"role": 1}`
	w := doRequest(r, http.MethodPut, "/projects/proj-1/members/user-2", body)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestUpdateMemberRole_InvalidBody(t *testing.T) {
	h, r := setupProjectHandler(t)
	r.PUT("/projects/:id/members/:userId", h.UpdateMemberRole)

	w := doRequest(r, http.MethodPut, "/projects/proj-1/members/user-2", "bad")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRemoveMember_Success(t *testing.T) {
	h, r := setupProjectHandler(t)
	r.DELETE("/projects/:id/members/:userId", h.RemoveMember)

	w := doRequest(r, http.MethodDelete, "/projects/proj-1/members/user-2", "")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestLeaveProject_Success(t *testing.T) {
	h, r := setupProjectHandler(t)
	r.POST("/projects/:id/members/me/leave", h.Leave)

	w := doRequest(r, http.MethodPost, "/projects/proj-1/members/me/leave", "")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestProjectHandler_ErrorResponses(t *testing.T) {
	// Test that errors return proper JSON
	stub := &stubTaskClient{
		archiveErr: status.Error(codes.PermissionDenied, "only owner can archive"),
	}
	h := handler.NewProjectHandler(stub)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), middleware.CtxKeyRequestID, "req-456"))
		c.Next()
	})
	r.POST("/projects/:id/archive", h.Archive)

	w := doRequest(r, http.MethodPost, "/projects/proj-1/archive", "")

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["code"] != "PERMISSION_DENIED" {
		t.Errorf("code = %s, want PERMISSION_DENIED", resp["code"])
	}
	if resp["request_id"] != "req-456" {
		t.Errorf("request_id = %s, want req-456", resp["request_id"])
	}
}

func TestProjectHandler_NonGRPCError(t *testing.T) {
	stub := &stubTaskClient{
		leaveErr: status.Error(codes.Unknown, "something weird"),
	}
	h := handler.NewProjectHandler(stub)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), middleware.CtxKeyRequestID, "req-1"))
		c.Next()
	})
	r.POST("/projects/:id/members/me/leave", h.Leave)

	w := doRequest(r, http.MethodPost, "/projects/proj-1/members/me/leave", "")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
