package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"task-platform/internal/gateway/middleware"
)

func TestRateLimitByIP_WithRedis_AllowsFirstRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	engine := gin.New()
	engine.Use(middleware.RateLimitByIP(rdb))
	engine.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRateLimitByIP_WithRedis_BlocksExcessiveRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	engine := gin.New()
	engine.Use(middleware.RateLimitByIP(rdb))
	engine.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Make rapid requests to exhaust the burst (100) + minimal refill
	for i := 0; i < 200; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1"
		engine.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			return // expected
		}
	}
	t.Fatal("expected 429 after exceeding burst")
}

func TestRateLimitByIP_WithRedis_AuthPathUsesStricterLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	engine := gin.New()
	engine.Use(middleware.RateLimitByIP(rdb))
	engine.POST("/api/v1/auth/register", func(c *gin.Context) {
		c.String(http.StatusCreated, "ok")
	})

	// Auth burst is 10
	for i := 0; i < 11; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)
		req.RemoteAddr = "192.168.1.2"
		engine.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			return // expected
		}
	}
	t.Fatal("expected 429 on auth path after burst 10")
}

func TestRateLimitByIP_WithRedis_SeparateAuthAndRegularKeys(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	engine := gin.New()
	engine.Use(middleware.RateLimitByIP(rdb))
	engine.POST("/api/v1/auth/register", func(c *gin.Context) {
		c.String(http.StatusCreated, "ok")
	})
	engine.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Exhaust auth bucket
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)
		req.RemoteAddr = "192.168.1.3"
		engine.ServeHTTP(w, req)
	}

	// Auth should be blocked
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)
	req.RemoteAddr = "192.168.1.3"
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("auth path should be rate limited, got %d", w.Code)
	}

	// Regular path should NOT be affected (separate key)
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.3"
	engine.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("regular path should not be affected by auth rate limit, got %d", w2.Code)
	}
}

func TestRateLimitByUser_WithRedis_AllowsFirstRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(string(middleware.CtxKeyUserID), "user-1")
		ctx := context.WithValue(c.Request.Context(), middleware.CtxKeyUserID, "user-1")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	engine.Use(middleware.RateLimitByUser(rdb))
	engine.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRateLimitByUser_WithRedis_BlocksExcessiveUserRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(string(middleware.CtxKeyUserID), "user-2")
		ctx := context.WithValue(c.Request.Context(), middleware.CtxKeyUserID, "user-2")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	engine.Use(middleware.RateLimitByUser(rdb))
	engine.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	for i := 0; i < 250; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		engine.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			return
		}
	}
	t.Fatal("expected 429 after exceeding user burst")
}
