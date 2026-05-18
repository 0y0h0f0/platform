package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	userv1 "task-platform/gen/go/user/v1"
	"task-platform/internal/gateway/handler"
	"task-platform/internal/gateway/middleware"
	"task-platform/pkg/xjwt"
)

func TestMe_Success(t *testing.T) {
	client := &mockUserClient{
		getUserFn: func(ctx context.Context, in *userv1.GetUserRequest, opts ...grpc.CallOption) (*userv1.GetUserResponse, error) {
			return &userv1.GetUserResponse{
				User: &userv1.User{Id: in.UserId, Username: "testuser", Email: "test@example.com"},
			}, nil
		},
	}

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.RequestID())

	jwtMgr := xjwt.NewManager("test-secret")
	engine.Use(middleware.Auth(jwtMgr, nil, nil))

	h := handler.NewUserHandler(client)
	engine.GET("/api/v1/users/me", h.Me)

	token, _, err := jwtMgr.Generate("user-1", "testuser", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
