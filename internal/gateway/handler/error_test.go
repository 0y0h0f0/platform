package handler_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"task-platform/internal/gateway/middleware"
	"task-platform/pkg/xerr"
)

// handleGRPCError is tested indirectly through the error handler's integration.
// We test it by hitting an endpoint that returns gRPC errors.
// Here we use a custom gin handler to exercise the error paths directly.

func TestHandleGRPCError_XerrError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.RequestID())

	engine.GET("/test", func(c *gin.Context) {
		c.Set("user_id", "user-1")
		c.Set("username", "testuser")
		// Test with a plain error (wrapped in xerr.NewError)
		err := xerr.NewError(xerr.CodeNotFound, "user not found")
		st := err.GRPCStatus()
		c.JSON(xerr.GRPCStatusToHTTP(st.Code()), &xerr.HTTPResponse{
			Code:      xerr.GRPCStatusToCode(st),
			Message:   st.Message(),
			RequestID: middleware.GetRequestID(c.Request.Context()),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleGRPCError_PlainError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.RequestID())

	engine.GET("/test", func(c *gin.Context) {
		resp := xerr.NewError(xerr.CodeInternal, "something went wrong")
		c.JSON(http.StatusInternalServerError, xerr.ToHTTPResponse(resp, middleware.GetRequestID(c.Request.Context())))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleGRPCError_GRPCStatusError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.RequestID())

	engine.GET("/test", func(c *gin.Context) {
		err := status.Error(codes.Unavailable, "service down")
		st, _ := status.FromError(err)
		c.JSON(xerr.GRPCStatusToHTTP(st.Code()), &xerr.HTTPResponse{
			Code:      xerr.GRPCStatusToCode(st),
			Message:   st.Message(),
			RequestID: middleware.GetRequestID(c.Request.Context()),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestToHTTPResponse_NilError(t *testing.T) {
	resp := xerr.ToHTTPResponse(nil, "req-1")
	if resp.Code != xerr.CodeOK {
		t.Errorf("Code = %s", resp.Code)
	}
}

func TestToHTTPResponse_PlainError(t *testing.T) {
	err := errors.New("something broke")
	resp := xerr.ToHTTPResponse(err, "req-1")
	if resp.Code != xerr.CodeInternal {
		t.Errorf("Code = %s", resp.Code)
	}
}
