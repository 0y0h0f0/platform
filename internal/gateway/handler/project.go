package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	taskv1 "task-platform/gen/go/task/v1"
	"task-platform/internal/gateway/middleware"
	"task-platform/pkg/xerr"
)

type ProjectHandler struct {
	taskClient taskv1.TaskServiceClient
}

func NewProjectHandler(taskClient taskv1.TaskServiceClient) *ProjectHandler {
	return &ProjectHandler{taskClient: taskClient}
}

func (h *ProjectHandler) Create(c *gin.Context) {
	var req taskv1.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, xerr.ToHTTPResponse(
			xerr.NewError(xerr.CodeInvalidArgument, "invalid request body"),
			middleware.GetRequestID(c.Request.Context())))
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
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
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
