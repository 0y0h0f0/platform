package server_test

import (
	"context"
	"os"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	taskserver "task-platform/internal/task/server"
)

func TestDefaultConfig(t *testing.T) {
	_ = os.Setenv("INTERNAL_TOKEN", "valid-token-long-enough-for-testing")
	_ = os.Setenv("USER_SERVICE_ADDR", "localhost:9999")
	_ = os.Setenv("POSTGRES_DSN", "host=localhost port=5432 user=postgres dbname=test sslmode=disable")
	t.Cleanup(func() {
		_ = os.Unsetenv("INTERNAL_TOKEN")
		_ = os.Unsetenv("USER_SERVICE_ADDR")
		_ = os.Unsetenv("POSTGRES_DSN")
	})

	cfg := taskserver.DefaultConfig()
	if cfg.InternalToken != "valid-token-long-enough-for-testing" {
		t.Errorf("InternalToken = %s", cfg.InternalToken)
	}
	if cfg.UserServiceAddr != "localhost:9999" {
		t.Errorf("UserServiceAddr = %s", cfg.UserServiceAddr)
	}
}

func TestDefaultConfig_Defaults(t *testing.T) {
	keys := []string{"INTERNAL_TOKEN", "USER_SERVICE_ADDR", "POSTGRES_DSN"}
	saved := map[string]string{}
	for _, k := range keys {
		saved[k] = os.Getenv(k)
		_ = os.Unsetenv(k)
	}
	t.Cleanup(func() {
		for k, v := range saved {
			if v != "" {
				_ = os.Setenv(k, v)
			}
		}
	})

	cfg := taskserver.DefaultConfig()
	if cfg.GRPCAddr != ":9092" {
		t.Errorf("GRPCAddr = %s", cfg.GRPCAddr)
	}
	if cfg.AdminAddr != ":8082" {
		t.Errorf("AdminAddr = %s", cfg.AdminAddr)
	}
}

func TestNewGRPCServer_MissingInternalToken(t *testing.T) {
	_, err := taskserver.NewGRPCServer(taskserver.Config{})
	if err == nil {
		t.Error("expected error for missing internal token")
	}
}

func TestNewGRPCServer_PlaceholderToken(t *testing.T) {
	_, err := taskserver.NewGRPCServer(taskserver.Config{
		InternalToken: "replace-with-a-long-random-internal-token",
	})
	if err == nil {
		t.Error("expected error for placeholder token")
	}
}

func TestNewGRPCServer_TokenTooShort(t *testing.T) {
	_, err := taskserver.NewGRPCServer(taskserver.Config{
		InternalToken: "short",
	})
	if err == nil {
		t.Error("expected error for short token")
	}
}

// TestAuthInterceptor tests the auth interceptor via the exported test helper.
func TestNewGRPCServer_PostgresError(t *testing.T) {
	_, err := taskserver.NewGRPCServer(taskserver.Config{
		InternalToken:   "valid-token-for-testing-16chars",
		PostgresDSN:     "host=127.0.0.1 port=19999 user=test dbname=test connect_timeout=1",
		UserServiceAddr: "127.0.0.1:9091",
	})
	if err == nil {
		t.Error("expected error when postgres is unreachable")
	}
}

func TestAuthInterceptor_ValidToken(t *testing.T) {
	interceptor := taskserver.TestAuthInterceptor("test-token")
	md := metadata.Pairs("x-internal-token", "test-token", "x-user-id", "user-1", "x-username", "alice")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/task.v1.TaskService/CreateTask"}, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("interceptor: %v", err)
	}
	if resp != "ok" {
		t.Errorf("resp = %v", resp)
	}
}

func TestAuthInterceptor_MissingMetadata(t *testing.T) {
	interceptor := taskserver.TestAuthInterceptor("test-token")

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) {
		return nil, nil
	})
	st, _ := status.FromError(err)
	if st.Code() != codes.Unauthenticated {
		t.Errorf("code = %s, want Unauthenticated", st.Code())
	}
}

func TestAuthInterceptor_InvalidToken(t *testing.T) {
	interceptor := taskserver.TestAuthInterceptor("test-token")
	md := metadata.Pairs("x-internal-token", "wrong-token", "x-user-id", "user-1", "x-username", "alice")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) {
		return nil, nil
	})
	st, _ := status.FromError(err)
	if st.Code() != codes.Unauthenticated {
		t.Errorf("code = %s, want Unauthenticated", st.Code())
	}
}

func TestAuthInterceptor_MissingUserID(t *testing.T) {
	interceptor := taskserver.TestAuthInterceptor("test-token")
	md := metadata.Pairs("x-internal-token", "test-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/task.v1.TaskService/CreateTask"}, func(ctx context.Context, req any) (any, error) {
		return nil, nil
	})
	st, _ := status.FromError(err)
	if st.Code() != codes.Unauthenticated {
		t.Errorf("code = %s, want Unauthenticated", st.Code())
	}
}
