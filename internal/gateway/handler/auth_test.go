package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	userv1 "task-platform/gen/go/user/v1"
	"task-platform/internal/gateway/handler"
	"task-platform/internal/gateway/middleware"
	"task-platform/pkg/xerr"
	"task-platform/pkg/xjwt"
)

type mockUserClient struct {
	registerFn func(ctx context.Context, in *userv1.RegisterRequest, opts ...grpc.CallOption) (*userv1.RegisterResponse, error)
	loginFn    func(ctx context.Context, in *userv1.LoginRequest, opts ...grpc.CallOption) (*userv1.LoginResponse, error)
	getUserFn  func(ctx context.Context, in *userv1.GetUserRequest, opts ...grpc.CallOption) (*userv1.GetUserResponse, error)
	batchFn    func(ctx context.Context, in *userv1.BatchGetUsersRequest, opts ...grpc.CallOption) (*userv1.BatchGetUsersResponse, error)
}

func (m *mockUserClient) Register(ctx context.Context, in *userv1.RegisterRequest, opts ...grpc.CallOption) (*userv1.RegisterResponse, error) {
	return m.registerFn(ctx, in, opts...)
}

func (m *mockUserClient) Login(ctx context.Context, in *userv1.LoginRequest, opts ...grpc.CallOption) (*userv1.LoginResponse, error) {
	return m.loginFn(ctx, in, opts...)
}

func (m *mockUserClient) GetUser(ctx context.Context, in *userv1.GetUserRequest, opts ...grpc.CallOption) (*userv1.GetUserResponse, error) {
	return m.getUserFn(ctx, in, opts...)
}

func (m *mockUserClient) BatchGetUsers(ctx context.Context, in *userv1.BatchGetUsersRequest, opts ...grpc.CallOption) (*userv1.BatchGetUsersResponse, error) {
	return m.batchFn(ctx, in, opts...)
}

func setupTestEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.RequestID())
	jwtMgr := xjwt.NewManager("test-secret")
	publicPaths := []string{
		"/api/v1/auth/register",
		"/api/v1/auth/login",
	}
	engine.Use(middleware.Auth(jwtMgr, nil, publicPaths))
	return engine
}

func TestRegister_Success(t *testing.T) {
	client := &mockUserClient{
		registerFn: func(ctx context.Context, in *userv1.RegisterRequest, opts ...grpc.CallOption) (*userv1.RegisterResponse, error) {
			return &userv1.RegisterResponse{
				AccessToken: "test-token",
				User:        &userv1.User{Id: "user-1", Username: in.Username},
			}, nil
		},
	}

	engine := setupTestEngine()
	h := handler.NewAuthHandler(client, nil)
	engine.POST("/api/v1/auth/register", h.Register)

	body := `{"username":"newuser","email":"new@test.com","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var resp xerr.HTTPResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != xerr.CodeOK {
		t.Errorf("code = %s", resp.Code)
	}
}

func TestRegister_InvalidBody(t *testing.T) {
	engine := setupTestEngine()
	h := handler.NewAuthHandler(nil, nil)
	engine.POST("/api/v1/auth/register", h.Register)

	body := `not-json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLogin_Success(t *testing.T) {
	client := &mockUserClient{
		loginFn: func(ctx context.Context, in *userv1.LoginRequest, opts ...grpc.CallOption) (*userv1.LoginResponse, error) {
			return &userv1.LoginResponse{
				AccessToken: "test-token",
				User:        &userv1.User{Id: "user-1"},
			}, nil
		},
	}

	engine := setupTestEngine()
	h := handler.NewAuthHandler(client, nil)
	engine.POST("/api/v1/auth/login", h.Login)

	body := `{"account":"testuser","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestLogin_GRPCError(t *testing.T) {
	client := &mockUserClient{
		loginFn: func(ctx context.Context, in *userv1.LoginRequest, opts ...grpc.CallOption) (*userv1.LoginResponse, error) {
			return nil, xerr.NewError(xerr.CodeNotFound, "user not found")
		},
	}

	engine := setupTestEngine()
	h := handler.NewAuthHandler(client, nil)
	engine.POST("/api/v1/auth/login", h.Login)

	body := `{"account":"nouser","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
