package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"task-platform/internal/gateway/middleware"
)

func TestRequestID_GeneratesID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.RequestID())
	engine.GET("/test", func(c *gin.Context) {
		reqID := middleware.GetRequestID(c.Request.Context())
		if reqID == "" {
			t.Error("request ID should not be empty")
		}
		c.String(200, reqID)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") == "" {
		t.Error("X-Request-ID header should be set")
	}
	if w.Code != 200 {
		t.Errorf("status = %d", w.Code)
	}
}

func TestRequestID_PropagatesExisting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.RequestID())
	engine.GET("/test", func(c *gin.Context) {
		reqID := middleware.GetRequestID(c.Request.Context())
		if reqID != "existing-id" {
			t.Errorf("request ID = %s, want existing-id", reqID)
		}
		c.String(200, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "existing-id")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") != "existing-id" {
		t.Error("should propagate existing request ID")
	}
}
