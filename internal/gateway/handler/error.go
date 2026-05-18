package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/status"

	"task-platform/internal/gateway/middleware"
	"task-platform/pkg/xerr"
)

func handleGRPCError(c *gin.Context, err error) {
	st, ok := status.FromError(err)
	if !ok {
		resp := xerr.NewError(xerr.CodeInternal, err.Error())
		c.JSON(http.StatusInternalServerError, xerr.ToHTTPResponse(resp, middleware.GetRequestID(c.Request.Context())))
		return
	}

	c.JSON(xerr.GRPCStatusToHTTP(st.Code()), &xerr.HTTPResponse{
		Code:      xerr.GRPCStatusToCode(st),
		Message:   st.Message(),
		RequestID: middleware.GetRequestID(c.Request.Context()),
	})
}
