package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"task-platform/internal/gateway/middleware"
)

func TestAccessLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, recorded := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	engine := gin.New()
	engine.Use(middleware.RequestID())
	engine.Use(middleware.AccessLog(logger))
	engine.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}

	// Verify access log was written
	logs := recorded.All()
	if len(logs) == 0 {
		t.Error("expected at least one log entry")
	}
	for _, entry := range logs {
		if entry.Message == "access" {
			if m, ok := entry.ContextMap()["method"]; !ok || m != "GET" {
				t.Errorf("method = %v", m)
			}
			return
		}
	}
	t.Error("expected 'access' log message")
}
