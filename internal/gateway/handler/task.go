package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	userv1 "task-platform/gen/go/user/v1"
	taskv1 "task-platform/gen/go/task/v1"
	"task-platform/internal/gateway/middleware"
	"task-platform/pkg/xerr"
)

type TaskHandler struct {
	taskClient taskv1.TaskServiceClient
	userClient userv1.UserServiceClient
	rdb        *redis.Client
}

func NewTaskHandler(taskClient taskv1.TaskServiceClient, userClient userv1.UserServiceClient, rdb *redis.Client) *TaskHandler {
	return &TaskHandler{taskClient: taskClient, userClient: userClient, rdb: rdb}
}

func (h *TaskHandler) Create(c *gin.Context) {
	var req taskv1.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, xerr.ToHTTPResponse(
			xerr.NewError(xerr.CodeInvalidArgument, "invalid request body"),
			middleware.GetRequestID(c.Request.Context())))
		return
	}

	shouldReturn, cleanup := SetupIdempotency(c, h.rdb)
	defer cleanup()
	if shouldReturn {
		return
	}

	res, err := h.taskClient.CreateTask(c.Request.Context(), &req)
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusCreated, &xerr.HTTPResponse{
		Code:      xerr.CodeOK,
		Message:   "ok",
		RequestID: middleware.GetRequestID(c.Request.Context()),
		Data:      res,
	})
}

func (h *TaskHandler) List(c *gin.Context) {
	projectID := c.Query("project_id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, xerr.ToHTTPResponse(
			xerr.NewError(xerr.CodeInvalidArgument, "project_id is required"),
			middleware.GetRequestID(c.Request.Context())))
		return
	}

	var req taskv1.ListTasksRequest
	req.ProjectId = projectID
	req.Cursor = c.Query("cursor")
	req.AssigneeId = c.Query("assignee_id")
	req.Keyword = c.Query("keyword")
	req.Status = -1

	if statusStr := c.Query("status"); statusStr != "" {
		var status int32
		if _, err := fmt.Sscanf(statusStr, "%d", &status); err != nil {
			c.JSON(http.StatusBadRequest, &xerr.HTTPResponse{
				Code: xerr.CodeInvalidArgument, Message: "invalid status",
				RequestID: middleware.GetRequestID(c.Request.Context()),
			})
			return
		}
		req.Status = status
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		var limit int32
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil {
			c.JSON(http.StatusBadRequest, &xerr.HTTPResponse{
				Code: xerr.CodeInvalidArgument, Message: "invalid limit",
				RequestID: middleware.GetRequestID(c.Request.Context()),
			})
			return
		}
		req.Limit = limit
	} else {
		req.Limit = 20
	}

	res, err := h.taskClient.ListTasks(c.Request.Context(), &req)
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, &xerr.HTTPResponse{
		Code:      xerr.CodeOK,
		Message:   "ok",
		RequestID: middleware.GetRequestID(c.Request.Context()),
		Data:      res,
	})
}

func (h *TaskHandler) Get(c *gin.Context) {
	taskID := c.Param("id")

	res, err := h.taskClient.GetTask(c.Request.Context(), &taskv1.GetTaskRequest{
		TaskId: taskID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, &xerr.HTTPResponse{
		Code:      xerr.CodeOK,
		Message:   "ok",
		RequestID: middleware.GetRequestID(c.Request.Context()),
		Data:      res,
	})
}

func (h *TaskHandler) Update(c *gin.Context) {
	taskID := c.Param("id")

	var req taskv1.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, xerr.ToHTTPResponse(
			xerr.NewError(xerr.CodeInvalidArgument, "invalid request body"),
			middleware.GetRequestID(c.Request.Context())))
		return
	}
	req.TaskId = taskID

	shouldReturn, cleanup := SetupIdempotency(c, h.rdb)
	defer cleanup()
	if shouldReturn {
		return
	}

	res, err := h.taskClient.UpdateTask(c.Request.Context(), &req)
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, &xerr.HTTPResponse{
		Code:      xerr.CodeOK,
		Message:   "ok",
		RequestID: middleware.GetRequestID(c.Request.Context()),
		Data:      res,
	})
}

func (h *TaskHandler) Delete(c *gin.Context) {
	taskID := c.Param("id")

	shouldReturn, cleanup := SetupIdempotency(c, h.rdb)
	defer cleanup()
	if shouldReturn {
		return
	}

	res, err := h.taskClient.DeleteTask(c.Request.Context(), &taskv1.DeleteTaskRequest{
		TaskId: taskID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, &xerr.HTTPResponse{
		Code:      xerr.CodeOK,
		Message:   "ok",
		RequestID: middleware.GetRequestID(c.Request.Context()),
		Data:      res,
	})
}

func (h *TaskHandler) Assign(c *gin.Context) {
	taskID := c.Param("id")

	var req taskv1.AssignTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, xerr.ToHTTPResponse(
			xerr.NewError(xerr.CodeInvalidArgument, "invalid request body"),
			middleware.GetRequestID(c.Request.Context())))
		return
	}
	req.TaskId = taskID

	shouldReturn, cleanup := SetupIdempotency(c, h.rdb)
	defer cleanup()
	if shouldReturn {
		return
	}

	res, err := h.taskClient.AssignTask(c.Request.Context(), &req)
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, &xerr.HTTPResponse{
		Code:      xerr.CodeOK,
		Message:   "ok",
		RequestID: middleware.GetRequestID(c.Request.Context()),
		Data:      res,
	})
}

func (h *TaskHandler) ChangeStatus(c *gin.Context) {
	taskID := c.Param("id")

	var req taskv1.ChangeTaskStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, xerr.ToHTTPResponse(
			xerr.NewError(xerr.CodeInvalidArgument, "invalid request body"),
			middleware.GetRequestID(c.Request.Context())))
		return
	}
	req.TaskId = taskID

	shouldReturn, cleanup := SetupIdempotency(c, h.rdb)
	defer cleanup()
	if shouldReturn {
		return
	}

	res, err := h.taskClient.ChangeTaskStatus(c.Request.Context(), &req)
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, &xerr.HTTPResponse{
		Code:      xerr.CodeOK,
		Message:   "ok",
		RequestID: middleware.GetRequestID(c.Request.Context()),
		Data:      res,
	})
}

func (h *TaskHandler) CreateComment(c *gin.Context) {
	taskID := c.Param("id")

	var req taskv1.CreateTaskCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, xerr.ToHTTPResponse(
			xerr.NewError(xerr.CodeInvalidArgument, "invalid request body"),
			middleware.GetRequestID(c.Request.Context())))
		return
	}
	req.TaskId = taskID

	shouldReturn, cleanup := SetupIdempotency(c, h.rdb)
	defer cleanup()
	if shouldReturn {
		return
	}

	res, err := h.taskClient.CreateTaskComment(c.Request.Context(), &req)
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusCreated, &xerr.HTTPResponse{
		Code:      xerr.CodeOK,
		Message:   "ok",
		RequestID: middleware.GetRequestID(c.Request.Context()),
		Data:      res,
	})
}

func (h *TaskHandler) ListComments(c *gin.Context) {
	taskID := c.Param("id")

	limit := int32(20)
	if limitStr := c.Query("limit"); limitStr != "" {
		var l int32
		if _, err := fmt.Sscanf(limitStr, "%d", &l); err != nil {
			c.JSON(http.StatusBadRequest, &xerr.HTTPResponse{
				Code: xerr.CodeInvalidArgument, Message: "invalid limit",
				RequestID: middleware.GetRequestID(c.Request.Context()),
			})
			return
		}
		limit = l
	}

	res, err := h.taskClient.ListTaskComments(c.Request.Context(), &taskv1.ListTaskCommentsRequest{
		TaskId:  taskID,
		Limit:   limit,
		AfterId: c.Query("after_id"),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	data := enrichComments(c.Request.Context(), h.userClient, res)

	c.JSON(http.StatusOK, &xerr.HTTPResponse{
		Code:      xerr.CodeOK,
		Message:   "ok",
		RequestID: middleware.GetRequestID(c.Request.Context()),
		Data:      data,
	})
}

func (h *TaskHandler) DeleteComment(c *gin.Context) {
	taskID := c.Param("id")
	commentID := c.Param("commentId")

	shouldReturn, cleanup := SetupIdempotency(c, h.rdb)
	defer cleanup()
	if shouldReturn {
		return
	}

	res, err := h.taskClient.DeleteTaskComment(c.Request.Context(), &taskv1.DeleteTaskCommentRequest{
		TaskId:    taskID,
		CommentId: commentID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, &xerr.HTTPResponse{
		Code:      xerr.CodeOK,
		Message:   "ok",
		RequestID: middleware.GetRequestID(c.Request.Context()),
		Data:      res,
	})
}

func (h *TaskHandler) ListOperationLogs(c *gin.Context) {
	taskID := c.Param("id")

	limit := int32(20)
	if limitStr := c.Query("limit"); limitStr != "" {
		var l int32
		if _, err := fmt.Sscanf(limitStr, "%d", &l); err != nil {
			c.JSON(http.StatusBadRequest, &xerr.HTTPResponse{
				Code: xerr.CodeInvalidArgument, Message: "invalid limit",
				RequestID: middleware.GetRequestID(c.Request.Context()),
			})
			return
		}
		limit = l
	}

	res, err := h.taskClient.ListOperationLogs(c.Request.Context(), &taskv1.ListOperationLogsRequest{
		TaskId: taskID,
		Limit:  limit,
		Cursor: c.Query("cursor"),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	data := enrichOperationLogs(c.Request.Context(), h.userClient, res)

	c.JSON(http.StatusOK, &xerr.HTTPResponse{
		Code:      xerr.CodeOK,
		Message:   "ok",
		RequestID: middleware.GetRequestID(c.Request.Context()),
		Data:      data,
	})
}
