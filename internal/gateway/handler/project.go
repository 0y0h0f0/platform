package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	userv1 "task-platform/gen/go/user/v1"
	taskv1 "task-platform/gen/go/task/v1"
	"task-platform/internal/gateway/middleware"
	"task-platform/pkg/xerr"
)

type ProjectHandler struct {
	taskClient taskv1.TaskServiceClient
	userClient userv1.UserServiceClient
	rdb        *redis.Client
}

func NewProjectHandler(taskClient taskv1.TaskServiceClient, userClient userv1.UserServiceClient, rdb *redis.Client) *ProjectHandler {
	return &ProjectHandler{taskClient: taskClient, userClient: userClient, rdb: rdb}
}

func (h *ProjectHandler) Create(c *gin.Context) {
	var req taskv1.CreateProjectRequest
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

	res, err := h.taskClient.CreateProject(c.Request.Context(), &req)
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

func (h *ProjectHandler) List(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil {
		c.JSON(http.StatusBadRequest, &xerr.HTTPResponse{
			Code: xerr.CodeInvalidArgument, Message: "invalid limit",
			RequestID: middleware.GetRequestID(c.Request.Context()),
		})
		return
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil {
		c.JSON(http.StatusBadRequest, &xerr.HTTPResponse{
			Code: xerr.CodeInvalidArgument, Message: "invalid offset",
			RequestID: middleware.GetRequestID(c.Request.Context()),
		})
		return
	}
	includeArchived := c.Query("include_archived") == "true"

	res, err := h.taskClient.ListProjects(c.Request.Context(), &taskv1.ListProjectsRequest{
		Limit:           int32(limit),
		Offset:          int32(offset),
		IncludeArchived: includeArchived,
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

func (h *ProjectHandler) Get(c *gin.Context) {
	projectID := c.Param("id")

	res, err := h.taskClient.GetProject(c.Request.Context(), &taskv1.GetProjectRequest{
		ProjectId: projectID,
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

func (h *ProjectHandler) Update(c *gin.Context) {
	projectID := c.Param("id")

	var req taskv1.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, xerr.ToHTTPResponse(
			xerr.NewError(xerr.CodeInvalidArgument, "invalid request body"),
			middleware.GetRequestID(c.Request.Context())))
		return
	}
	req.ProjectId = projectID

	shouldReturn, cleanup := SetupIdempotency(c, h.rdb)
	defer cleanup()
	if shouldReturn {
		return
	}

	res, err := h.taskClient.UpdateProject(c.Request.Context(), &req)
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

func (h *ProjectHandler) Archive(c *gin.Context) {
	projectID := c.Param("id")

	shouldReturn, cleanup := SetupIdempotency(c, h.rdb)
	defer cleanup()
	if shouldReturn {
		return
	}

	res, err := h.taskClient.ArchiveProject(c.Request.Context(), &taskv1.ArchiveProjectRequest{
		ProjectId: projectID,
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

func (h *ProjectHandler) Unarchive(c *gin.Context) {
	projectID := c.Param("id")

	shouldReturn, cleanup := SetupIdempotency(c, h.rdb)
	defer cleanup()
	if shouldReturn {
		return
	}

	res, err := h.taskClient.UnarchiveProject(c.Request.Context(), &taskv1.UnarchiveProjectRequest{
		ProjectId: projectID,
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

func (h *ProjectHandler) Transfer(c *gin.Context) {
	projectID := c.Param("id")

	var req taskv1.TransferProjectOwnershipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, xerr.ToHTTPResponse(
			xerr.NewError(xerr.CodeInvalidArgument, "invalid request body"),
			middleware.GetRequestID(c.Request.Context())))
		return
	}
	req.ProjectId = projectID

	shouldReturn, cleanup := SetupIdempotency(c, h.rdb)
	defer cleanup()
	if shouldReturn {
		return
	}

	res, err := h.taskClient.TransferProjectOwnership(c.Request.Context(), &req)
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

func (h *ProjectHandler) AddMember(c *gin.Context) {
	projectID := c.Param("id")

	var req taskv1.AddProjectMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, xerr.ToHTTPResponse(
			xerr.NewError(xerr.CodeInvalidArgument, "invalid request body"),
			middleware.GetRequestID(c.Request.Context())))
		return
	}
	req.ProjectId = projectID

	shouldReturn, cleanup := SetupIdempotency(c, h.rdb)
	defer cleanup()
	if shouldReturn {
		return
	}

	res, err := h.taskClient.AddProjectMember(c.Request.Context(), &req)
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

func (h *ProjectHandler) ListMembers(c *gin.Context) {
	projectID := c.Param("id")

	res, err := h.taskClient.ListProjectMembers(c.Request.Context(), &taskv1.ListProjectMembersRequest{
		ProjectId: projectID,
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

func (h *ProjectHandler) UpdateMemberRole(c *gin.Context) {
	projectID := c.Param("id")
	userID := c.Param("userId")

	var req taskv1.UpdateProjectMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, xerr.ToHTTPResponse(
			xerr.NewError(xerr.CodeInvalidArgument, "invalid request body"),
			middleware.GetRequestID(c.Request.Context())))
		return
	}
	req.ProjectId = projectID
	req.UserId = userID

	shouldReturn, cleanup := SetupIdempotency(c, h.rdb)
	defer cleanup()
	if shouldReturn {
		return
	}

	res, err := h.taskClient.UpdateProjectMemberRole(c.Request.Context(), &req)
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

func (h *ProjectHandler) RemoveMember(c *gin.Context) {
	projectID := c.Param("id")
	userID := c.Param("userId")

	shouldReturn, cleanup := SetupIdempotency(c, h.rdb)
	defer cleanup()
	if shouldReturn {
		return
	}

	res, err := h.taskClient.RemoveProjectMember(c.Request.Context(), &taskv1.RemoveProjectMemberRequest{
		ProjectId: projectID,
		UserId:    userID,
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

func (h *ProjectHandler) Leave(c *gin.Context) {
	projectID := c.Param("id")

	shouldReturn, cleanup := SetupIdempotency(c, h.rdb)
	defer cleanup()
	if shouldReturn {
		return
	}

	res, err := h.taskClient.LeaveProject(c.Request.Context(), &taskv1.LeaveProjectRequest{
		ProjectId: projectID,
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

func (h *ProjectHandler) ListOperationLogs(c *gin.Context) {
	projectID := c.Param("id")

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
		ProjectId: projectID,
		Limit:     limit,
		Cursor:    c.Query("cursor"),
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
