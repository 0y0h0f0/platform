package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	userv1 "task-platform/gen/go/user/v1"
	"task-platform/internal/gateway/middleware"
	"task-platform/pkg/xerr"
)

type UserHandler struct {
	userClient userv1.UserServiceClient
}

func NewUserHandler(userClient userv1.UserServiceClient) *UserHandler {
	return &UserHandler{userClient: userClient}
}

func (h *UserHandler) Me(c *gin.Context) {
	userID := middleware.GetUserID(c.Request.Context())

	res, err := h.userClient.GetUser(c.Request.Context(), &userv1.GetUserRequest{
		UserId: userID,
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
