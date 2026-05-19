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

type stubTaskServiceClient struct {
	taskv1.TaskServiceClient
	createTaskRes      *taskv1.CreateTaskResponse
	createTaskErr      error
	listTasksRes       *taskv1.ListTasksResponse
	listTasksErr       error
	getTaskRes         *taskv1.GetTaskResponse
	getTaskErr         error
	updateTaskRes      *taskv1.UpdateTaskResponse
	updateTaskErr      error
	deleteTaskRes      *taskv1.DeleteTaskResponse
	deleteTaskErr      error
	assignTaskRes      *taskv1.AssignTaskResponse
	assignTaskErr      error
	changeStatusRes    *taskv1.ChangeTaskStatusResponse
	changeStatusErr    error
	createCommentRes   *taskv1.CreateTaskCommentResponse
	createCommentErr   error
	deleteCommentRes   *taskv1.DeleteTaskCommentResponse
	deleteCommentErr   error
	listCommentsRes    *taskv1.ListTaskCommentsResponse
	listCommentsErr    error
	listOpLogsRes      *taskv1.ListOperationLogsResponse
	listOpLogsErr      error
}

func (s *stubTaskServiceClient) CreateTask(_ context.Context, _ *taskv1.CreateTaskRequest, _ ...grpc.CallOption) (*taskv1.CreateTaskResponse, error) {
	return s.createTaskRes, s.createTaskErr
}
func (s *stubTaskServiceClient) ListTasks(_ context.Context, _ *taskv1.ListTasksRequest, _ ...grpc.CallOption) (*taskv1.ListTasksResponse, error) {
	return s.listTasksRes, s.listTasksErr
}
func (s *stubTaskServiceClient) GetTask(_ context.Context, _ *taskv1.GetTaskRequest, _ ...grpc.CallOption) (*taskv1.GetTaskResponse, error) {
	return s.getTaskRes, s.getTaskErr
}
func (s *stubTaskServiceClient) UpdateTask(_ context.Context, _ *taskv1.UpdateTaskRequest, _ ...grpc.CallOption) (*taskv1.UpdateTaskResponse, error) {
	return s.updateTaskRes, s.updateTaskErr
}
func (s *stubTaskServiceClient) DeleteTask(_ context.Context, _ *taskv1.DeleteTaskRequest, _ ...grpc.CallOption) (*taskv1.DeleteTaskResponse, error) {
	return s.deleteTaskRes, s.deleteTaskErr
}
func (s *stubTaskServiceClient) AssignTask(_ context.Context, _ *taskv1.AssignTaskRequest, _ ...grpc.CallOption) (*taskv1.AssignTaskResponse, error) {
	return s.assignTaskRes, s.assignTaskErr
}
func (s *stubTaskServiceClient) ChangeTaskStatus(_ context.Context, _ *taskv1.ChangeTaskStatusRequest, _ ...grpc.CallOption) (*taskv1.ChangeTaskStatusResponse, error) {
	return s.changeStatusRes, s.changeStatusErr
}
func (s *stubTaskServiceClient) CreateTaskComment(_ context.Context, _ *taskv1.CreateTaskCommentRequest, _ ...grpc.CallOption) (*taskv1.CreateTaskCommentResponse, error) {
	return s.createCommentRes, s.createCommentErr
}
func (s *stubTaskServiceClient) DeleteTaskComment(_ context.Context, _ *taskv1.DeleteTaskCommentRequest, _ ...grpc.CallOption) (*taskv1.DeleteTaskCommentResponse, error) {
	return s.deleteCommentRes, s.deleteCommentErr
}
func (s *stubTaskServiceClient) ListTaskComments(_ context.Context, _ *taskv1.ListTaskCommentsRequest, _ ...grpc.CallOption) (*taskv1.ListTaskCommentsResponse, error) {
	return s.listCommentsRes, s.listCommentsErr
}
func (s *stubTaskServiceClient) ListOperationLogs(_ context.Context, _ *taskv1.ListOperationLogsRequest, _ ...grpc.CallOption) (*taskv1.ListOperationLogsResponse, error) {
	return s.listOpLogsRes, s.listOpLogsErr
}

func setupTaskHandler(t *testing.T) (*handler.TaskHandler, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	stub := &stubTaskServiceClient{
		createTaskRes: &taskv1.CreateTaskResponse{
			Task: &taskv1.Task{Id: "task-1", ProjectId: "proj-1", Title: "My Task", Status: 0},
		},
		listTasksRes: &taskv1.ListTasksResponse{
			Tasks: []*taskv1.Task{{Id: "task-1", Title: "My Task"}, {Id: "task-2", Title: "Other"}},
		},
		getTaskRes: &taskv1.GetTaskResponse{
			Task: &taskv1.Task{Id: "task-1", Title: "My Task", ProjectId: "proj-1"},
		},
		updateTaskRes: &taskv1.UpdateTaskResponse{
			Task: &taskv1.Task{Id: "task-1", Title: "Updated", Version: 1},
		},
		deleteTaskRes: &taskv1.DeleteTaskResponse{
			Task: &taskv1.Task{Id: "task-1", Title: "Deleted"},
		},
		assignTaskRes: &taskv1.AssignTaskResponse{
			Task: &taskv1.Task{Id: "task-1", AssigneeId: "user-2"},
		},
		changeStatusRes: &taskv1.ChangeTaskStatusResponse{
			Task: &taskv1.Task{Id: "task-1", Status: 1},
		},
		createCommentRes: &taskv1.CreateTaskCommentResponse{
			Comment: &taskv1.TaskComment{Id: "comment-1", TaskId: "task-1", UserId: "user-1", Content: "nice"},
		},
		deleteCommentRes: &taskv1.DeleteTaskCommentResponse{
			Comment: &taskv1.TaskComment{Id: "comment-1", TaskId: "task-1", UserId: "user-1"},
		},
		listCommentsRes: &taskv1.ListTaskCommentsResponse{
			Comments: []*taskv1.TaskComment{{Id: "comment-1", TaskId: "task-1", UserId: "user-1", Content: "hello"}},
		},
		listOpLogsRes: &taskv1.ListOperationLogsResponse{
			Logs: []*taskv1.OperationLog{{Id: "log-1", OperatorId: "user-1", Action: "task.create"}},
		},
	}

	h := handler.NewTaskHandler(stub, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), middleware.CtxKeyRequestID, "req-123"))
		c.Next()
	})
	return h, r
}

func doTaskRequest(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	r.ServeHTTP(w, req)
	return w
}

// --- Create ---

func TestTaskCreate_Success(t *testing.T) {
	h, r := setupTaskHandler(t)
	r.POST("/tasks", h.Create)

	body := `{"project_id": "proj-1", "title": "My Task", "content": "desc"}`
	w := doTaskRequest(r, http.MethodPost, "/tasks", body)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestTaskCreate_InvalidBody(t *testing.T) {
	h, r := setupTaskHandler(t)
	r.POST("/tasks", h.Create)

	w := doTaskRequest(r, http.MethodPost, "/tasks", "not json")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaskCreate_GRPCError(t *testing.T) {
	stub := &stubTaskServiceClient{
		createTaskErr: status.Error(codes.InvalidArgument, "invalid title"),
	}
	h := handler.NewTaskHandler(stub, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), middleware.CtxKeyRequestID, "req-123"))
		c.Next()
	})
	r.POST("/tasks", h.Create)

	w := doTaskRequest(r, http.MethodPost, "/tasks", `{"project_id": "proj-1", "title": ""}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- List ---

func TestTaskList_Success(t *testing.T) {
	h, r := setupTaskHandler(t)
	r.GET("/tasks", h.List)

	w := doTaskRequest(r, http.MethodGet, "/tasks?project_id=proj-1&limit=20", "")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestTaskList_MissingProjectID(t *testing.T) {
	h, r := setupTaskHandler(t)
	r.GET("/tasks", h.List)

	w := doTaskRequest(r, http.MethodGet, "/tasks", "")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaskList_WithFilters(t *testing.T) {
	h, r := setupTaskHandler(t)
	r.GET("/tasks", h.List)

	w := doTaskRequest(r, http.MethodGet, "/tasks?project_id=proj-1&status=1&assignee_id=user-2&keyword=bug&cursor=c1&limit=10", "")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// --- Get ---

func TestTaskGet_Success(t *testing.T) {
	h, r := setupTaskHandler(t)
	r.GET("/tasks/:id", h.Get)

	w := doTaskRequest(r, http.MethodGet, "/tasks/task-1", "")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestTaskGet_NotFound(t *testing.T) {
	stub := &stubTaskServiceClient{
		getTaskErr: status.Error(codes.NotFound, "not found"),
	}
	h := handler.NewTaskHandler(stub, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), middleware.CtxKeyRequestID, "req-123"))
		c.Next()
	})
	r.GET("/tasks/:id", h.Get)

	w := doTaskRequest(r, http.MethodGet, "/tasks/task-1", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- Update ---

func TestTaskUpdate_Success(t *testing.T) {
	h, r := setupTaskHandler(t)
	r.PUT("/tasks/:id", h.Update)

	body := `{"title": "Updated", "version": 1}`
	w := doTaskRequest(r, http.MethodPut, "/tasks/task-1", body)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestTaskUpdate_InvalidBody(t *testing.T) {
	h, r := setupTaskHandler(t)
	r.PUT("/tasks/:id", h.Update)

	w := doTaskRequest(r, http.MethodPut, "/tasks/task-1", "bad")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Delete ---

func TestTaskDelete_Success(t *testing.T) {
	h, r := setupTaskHandler(t)
	r.DELETE("/tasks/:id", h.Delete)

	w := doTaskRequest(r, http.MethodDelete, "/tasks/task-1", "")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// --- Assign ---

func TestTaskAssign_Success(t *testing.T) {
	h, r := setupTaskHandler(t)
	r.POST("/tasks/:id/assign", h.Assign)

	body := `{"assignee_id": "user-2"}`
	w := doTaskRequest(r, http.MethodPost, "/tasks/task-1/assign", body)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestTaskAssign_InvalidBody(t *testing.T) {
	h, r := setupTaskHandler(t)
	r.POST("/tasks/:id/assign", h.Assign)

	w := doTaskRequest(r, http.MethodPost, "/tasks/task-1/assign", "bad")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- ChangeStatus ---

func TestTaskChangeStatus_Success(t *testing.T) {
	h, r := setupTaskHandler(t)
	r.POST("/tasks/:id/status", h.ChangeStatus)

	body := `{"status": 1, "version": 0}`
	w := doTaskRequest(r, http.MethodPost, "/tasks/task-1/status", body)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestTaskChangeStatus_InvalidBody(t *testing.T) {
	h, r := setupTaskHandler(t)
	r.POST("/tasks/:id/status", h.ChangeStatus)

	w := doTaskRequest(r, http.MethodPost, "/tasks/task-1/status", "bad")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Error Responses ---

func TestTaskHandler_ErrorResponseFormat(t *testing.T) {
	stub := &stubTaskServiceClient{
		deleteTaskErr: status.Error(codes.PermissionDenied, "members can only delete their own tasks"),
	}
	h := handler.NewTaskHandler(stub, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), middleware.CtxKeyRequestID, "req-456"))
		c.Next()
	})
	r.DELETE("/tasks/:id", h.Delete)

	w := doTaskRequest(r, http.MethodDelete, "/tasks/task-1", "")

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

func TestTaskHandler_InternalError(t *testing.T) {
	stub := &stubTaskServiceClient{
		changeStatusErr: status.Error(codes.Unknown, "something weird"),
	}
	h := handler.NewTaskHandler(stub, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), middleware.CtxKeyRequestID, "req-1"))
		c.Next()
	})
	r.POST("/tasks/:id/status", h.ChangeStatus)

	w := doTaskRequest(r, http.MethodPost, "/tasks/task-1/status", `{"status": 1, "version": 0}`)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- CreateComment ---

func TestCreateComment_Success(t *testing.T) {
	h, r := setupTaskHandler(t)
	r.POST("/tasks/:id/comments", h.CreateComment)

	body := `{"content": "nice task"}`
	w := doTaskRequest(r, http.MethodPost, "/tasks/task-1/comments", body)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestCreateComment_InvalidBody(t *testing.T) {
	h, r := setupTaskHandler(t)
	r.POST("/tasks/:id/comments", h.CreateComment)

	w := doTaskRequest(r, http.MethodPost, "/tasks/task-1/comments", "bad")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- ListComments ---

func TestListComments_Success(t *testing.T) {
	h, r := setupTaskHandler(t)
	r.GET("/tasks/:id/comments", h.ListComments)

	w := doTaskRequest(r, http.MethodGet, "/tasks/task-1/comments?limit=10", "")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// --- DeleteComment ---

func TestDeleteComment_Success(t *testing.T) {
	h, r := setupTaskHandler(t)
	r.DELETE("/tasks/:id/comments/:commentId", h.DeleteComment)

	w := doTaskRequest(r, http.MethodDelete, "/tasks/task-1/comments/comment-1", "")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestDeleteComment_NotFound(t *testing.T) {
	stub := &stubTaskServiceClient{
		deleteCommentErr: status.Error(codes.NotFound, "not found"),
	}
	h := handler.NewTaskHandler(stub, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), middleware.CtxKeyRequestID, "req-1"))
		c.Next()
	})
	r.DELETE("/tasks/:id/comments/:commentId", h.DeleteComment)

	w := doTaskRequest(r, http.MethodDelete, "/tasks/task-1/comments/missing", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- ListOperationLogs ---

func TestListOperationLogs_Success(t *testing.T) {
	h, r := setupTaskHandler(t)
	r.GET("/tasks/:id/operation-logs", h.ListOperationLogs)

	w := doTaskRequest(r, http.MethodGet, "/tasks/task-1/operation-logs?limit=20", "")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestListOperationLogs_GRPCError(t *testing.T) {
	stub := &stubTaskServiceClient{
		listOpLogsErr: status.Error(codes.InvalidArgument, "invalid cursor"),
	}
	h := handler.NewTaskHandler(stub, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), middleware.CtxKeyRequestID, "req-1"))
		c.Next()
	})
	r.GET("/tasks/:id/operation-logs", h.ListOperationLogs)

	w := doTaskRequest(r, http.MethodGet, "/tasks/task-1/operation-logs?cursor=bad", "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
