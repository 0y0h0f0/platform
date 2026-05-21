package server

import (
	"context"
	"net"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	userv1 "task-platform/gen/go/user/v1"
	taskservice "task-platform/internal/task/service"
)

type echoUserServer struct {
	userv1.UnimplementedUserServiceServer
}

func (s *echoUserServer) GetUser(_ context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	return &userv1.GetUserResponse{
		User: &userv1.User{Id: req.UserId, Username: "echo", Status: 0},
	}, nil
}

type notFoundUserServer struct {
	userv1.UnimplementedUserServiceServer
}

func (s *notFoundUserServer) GetUser(_ context.Context, _ *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	return nil, status.Error(codes.NotFound, "user not found")
}

func TestSingleValue(t *testing.T) {
	md := metadata.Pairs("x-key", "val1", "x-key", "val2")
	if v := singleValue(md, "x-key"); v != "val1" {
		t.Errorf("singleValue = %s, want val1", v)
	}
}

func TestSingleValue_Empty(t *testing.T) {
	if v := singleValue(metadata.MD{}, "x-key"); v != "" {
		t.Errorf("expected empty, got %s", v)
	}
}

func TestSingleValue_MissingKey(t *testing.T) {
	md := metadata.Pairs("x-other", "val")
	if v := singleValue(md, "x-key"); v != "" {
		t.Errorf("expected empty, got %s", v)
	}
}

func TestValidateSecret_Success(t *testing.T) {
	if err := validateSecret("TEST", "valid-secret-at-least-16", 16); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSecret_Empty(t *testing.T) {
	if err := validateSecret("TEST", "", 16); err == nil {
		t.Error("expected error for empty secret")
	}
}

func TestValidateSecret_Placeholder(t *testing.T) {
	if err := validateSecret("TEST", "replace-with-a-long-random-internal-token", 16); err == nil {
		t.Error("expected error for placeholder")
	}
}

func TestValidateSecret_TooShort(t *testing.T) {
	if err := validateSecret("TEST", "short", 8); err == nil {
		t.Error("expected error for short secret")
	}
}

func TestEnvOrDefault_WithValue(t *testing.T) {
	t.Setenv("TEST_ENV_12345", "custom")
	if v := envOrDefault("TEST_ENV_12345", "default"); v != "custom" {
		t.Errorf("envOrDefault = %s, want custom", v)
	}
}

func TestEnvOrDefault_Default(t *testing.T) {
	if v := envOrDefault("NONEXISTENT_ENV_VAR_XYZ", "fallback"); v != "fallback" {
		t.Errorf("envOrDefault = %s, want fallback", v)
	}
}

func TestAuthInterceptor_MissingMetadata(t *testing.T) {
	interceptor := newAuthInterceptor("test-token-123456")
	handler := func(_ context.Context, _ any) (any, error) { return "ok", nil }

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("code = %s, want Unauthenticated", status.Code(err))
	}
}

func TestAuthInterceptor_InvalidToken(t *testing.T) {
	interceptor := newAuthInterceptor("test-token-123456")
	handler := func(_ context.Context, _ any) (any, error) { return "ok", nil }

	md := metadata.Pairs("x-internal-token", "wrong-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("code = %s, want Unauthenticated", status.Code(err))
	}
}

func TestAuthInterceptor_MissingUserIdentity(t *testing.T) {
	interceptor := newAuthInterceptor("test-token-123456")
	handler := func(_ context.Context, _ any) (any, error) { return "ok", nil }

	md := metadata.Pairs("x-internal-token", "test-token-123456")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/task.v1.TaskService/CreateProject"}, handler)
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("code = %s, want Unauthenticated", status.Code(err))
	}
}

func TestAuthInterceptor_MissingUserIDOnly(t *testing.T) {
	interceptor := newAuthInterceptor("test-token-123456")
	handler := func(_ context.Context, _ any) (any, error) { return "ok", nil }

	md := metadata.Pairs(
		"x-internal-token", "test-token-123456",
		"x-username", "alice",
	)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/task.v1.TaskService/CreateProject"}, handler)
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("code = %s, want Unauthenticated", status.Code(err))
	}
}

func TestAuthInterceptor_ValidRequest(t *testing.T) {
	interceptor := newAuthInterceptor("test-token-123456")
	handler := func(ctx context.Context, _ any) (any, error) {
		return "ok", nil
	}

	md := metadata.Pairs(
		"x-internal-token", "test-token-123456",
		"x-user-id", "user-1",
		"x-username", "alice",
	)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/task.v1.TaskService/CreateProject"}, handler)
	if err != nil {
		t.Fatalf("interceptor: %v", err)
	}
	if resp != "ok" {
		t.Errorf("resp = %v", resp)
	}
}

func TestAuthInterceptor_UserIDPropagatedToContext(t *testing.T) {
	interceptor := newAuthInterceptor("test-token-123456")
	handler := func(ctx context.Context, _ any) (any, error) {
		if taskservice.GetUserID(ctx) != "user-1" {
			t.Errorf("user ID not propagated to context")
		}
		if taskservice.GetUsername(ctx) != "alice" {
			t.Errorf("username not propagated to context")
		}
		return "ok", nil
	}

	md := metadata.Pairs(
		"x-internal-token", "test-token-123456",
		"x-user-id", "user-1",
		"x-username", "alice",
	)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/task.v1.TaskService/CreateProject"}, handler)
	if err != nil {
		t.Fatalf("interceptor: %v", err)
	}
}

func TestLoggingInterceptor_Success(t *testing.T) {
	logger := zap.NewNop()
	interceptor := loggingInterceptor(logger)
	handler := func(_ context.Context, _ any) (any, error) { return "ok", nil }

	md := metadata.Pairs("x-request-id", "req-123")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test/Method"}, handler)
	if err != nil {
		t.Fatalf("interceptor: %v", err)
	}
	if resp != "ok" {
		t.Errorf("resp = %v", resp)
	}
}

func TestLoggingInterceptor_Error(t *testing.T) {
	logger := zap.NewNop()
	interceptor := loggingInterceptor(logger)
	handler := func(_ context.Context, _ any) (any, error) {
		return nil, status.Error(codes.Internal, "something went wrong")
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-request-id", "req-err"))

	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test/ErrorMethod"}, handler)
	if err == nil {
		t.Error("expected error from handler")
	}
	_ = resp
}

func TestLoggingInterceptor_NoMetadata(t *testing.T) {
	logger := zap.NewNop()
	interceptor := loggingInterceptor(logger)
	handler := func(_ context.Context, _ any) (any, error) { return "no-md", nil }

	resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test/NoMD"}, handler)
	if err != nil {
		t.Fatalf("interceptor: %v", err)
	}
	if resp != "no-md" {
		t.Errorf("resp = %v", resp)
	}
}

func TestUserClientAdapter_GetUser_Success(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer()
	userv1.RegisterUserServiceServer(srv, &echoUserServer{})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	client, conn, err := newUserServiceClient(lis.Addr().String(), "test-token-123456")
	if err != nil {
		t.Fatalf("newUserServiceClient: %v", err)
	}
	defer func() { _ = conn.Close() }()

	adapter := &userClientAdapter{client: client}
	exists, active, err := adapter.GetUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if !exists {
		t.Error("expected user to exist")
	}
	if !active {
		t.Error("expected user to be active")
	}
}

func TestUserClientAdapter_GetUser_NotFound(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer()
	userv1.RegisterUserServiceServer(srv, &notFoundUserServer{})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	client, conn, err := newUserServiceClient(lis.Addr().String(), "test-token-123456")
	if err != nil {
		t.Fatalf("newUserServiceClient: %v", err)
	}
	defer func() { _ = conn.Close() }()

	adapter := &userClientAdapter{client: client}
	exists, active, err := adapter.GetUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if exists {
		t.Error("expected user to not exist")
	}
	if active {
		t.Error("expected user to not be active")
	}
}
