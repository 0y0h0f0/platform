package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"task-platform/internal/gateway/middleware"
	"task-platform/pkg/xerr"
)

func handleGRPCError(c *gin.Context, err error) {
	st, ok := status.FromError(err)
	if !ok {
		c.JSON(http.StatusInternalServerError, &xerr.HTTPResponse{
			Code:      xerr.CodeInternal,
			Message:   "internal server error",
			RequestID: middleware.GetRequestID(c.Request.Context()),
		})
		return
	}

	msg := st.Message()
	if st.Code() == codes.Internal || st.Code() == codes.Unknown {
		msg = "internal server error"
	}

	c.JSON(xerr.GRPCStatusToHTTP(st.Code()), &xerr.HTTPResponse{
		Code:      xerr.GRPCStatusToCode(st),
		Message:   msg,
		RequestID: middleware.GetRequestID(c.Request.Context()),
	})
}
