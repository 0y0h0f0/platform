package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"task-platform/internal/gateway/middleware"
	"task-platform/pkg/xjwt"
)

func TestAuth_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtMgr := xjwt.NewManager("test-secret")

	engine := gin.New()
	engine.Use(middleware.RequestID())
	engine.Use(middleware.Auth(jwtMgr, nil, nil))
	engine.GET("/protected", func(c *gin.Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtMgr := xjwt.NewManager("test-secret")

	engine := gin.New()
	engine.Use(middleware.RequestID())
	engine.Use(middleware.Auth(jwtMgr, nil, nil))
	engine.GET("/protected", func(c *gin.Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuth_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtMgr := xjwt.NewManager("test-secret")
	token, _, err := jwtMgr.Generate("user-1", "testuser", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	engine := gin.New()
	engine.Use(middleware.RequestID())
	// nil Redis means we skip the blacklist check in this test
	engine.Use(middleware.Auth(jwtMgr, nil, nil))
	engine.GET("/protected", func(c *gin.Context) {
		userID := middleware.GetUserID(c.Request.Context())
		username := middleware.GetUsername(c.Request.Context())
		if userID != "user-1" {
			t.Errorf("userID = %s", userID)
		}
		if username != "testuser" {
			t.Errorf("username = %s", username)
		}
		c.String(200, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d", w.Code)
	}
}

func TestAuth_PublicPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtMgr := xjwt.NewManager("test-secret")

	engine := gin.New()
	engine.Use(middleware.RequestID())
	engine.Use(middleware.Auth(jwtMgr, nil, []string{"/public"}))
	engine.GET("/public", func(c *gin.Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/public", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d", w.Code)
	}
}
