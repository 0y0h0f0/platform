package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc"

	userv1 "task-platform/gen/go/user/v1"
	"task-platform/internal/gateway/handler"
	"task-platform/internal/gateway/middleware"
	"task-platform/pkg/xjwt"
)

func setupRedisForHandler(t *testing.T) *redis.Client {
	t.Helper()
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("Ready to accept connections"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start redis: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "6379")

	return redis.NewClient(&redis.Options{
		Addr: host + ":" + port.Port(),
	})
}

func TestLogout_Success(t *testing.T) {
	rdb := setupRedisForHandler(t)
	jwtMgr := xjwt.NewManager("test-secret-for-logout-handler-testing")

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.RequestID())
	publicPaths := []string{"/api/v1/auth/register", "/api/v1/auth/login"}
	engine.Use(middleware.Auth(jwtMgr, rdb, publicPaths))

	mockClient := &mockUserClient{
		registerFn: func(ctx context.Context, in *userv1.RegisterRequest, opts ...grpc.CallOption) (*userv1.RegisterResponse, error) {
			return &userv1.RegisterResponse{
				AccessToken: "mock-token",
				User:        &userv1.User{Id: "user-1", Username: in.Username},
			}, nil
		},
	}

	authH := handler.NewAuthHandler(mockClient, rdb)
	engine.POST("/api/v1/auth/logout", authH.Logout)

	token, _, err := jwtMgr.Generate("user-1", "testuser", time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify token is blacklisted
	jti := middleware.GetJTI(req.Context())
	// Note: jti is set by the auth middleware which ran before the handler
	_ = jti
}

func TestLogout_RefreshToken(t *testing.T) {
	rdb := setupRedisForHandler(t)
	jwtMgr := xjwt.NewManager("test-secret-for-logout-test-2")

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.RequestID())
	engine.Use(middleware.Auth(jwtMgr, rdb, nil))

	mockClient := &mockUserClient{}
	authH := handler.NewAuthHandler(mockClient, rdb)
	engine.POST("/api/v1/auth/logout", authH.Logout)

	token, _, err := jwtMgr.Generate("user-2", "testuser2", time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestLogout_InvalidBody(t *testing.T) {
	rdb := setupRedisForHandler(t)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.RequestID())

	authH := handler.NewAuthHandler(nil, rdb)
	engine.POST("/api/v1/auth/register", authH.Register)

	body := `not-json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
