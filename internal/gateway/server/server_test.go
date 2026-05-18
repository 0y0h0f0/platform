package server_test

import (
	"context"
	"net"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	gwserver "task-platform/internal/gateway/server"
)

func setupRedis(t *testing.T) string {
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
	return host + ":" + port.Port()
}

func TestDefaultConfig(t *testing.T) {
	os.Setenv("USER_SERVICE_ADDR", "localhost:9999")
	os.Setenv("JWT_SECRET", "valid-secret-long-enough-for-hs256-algorithm-ok")
	os.Setenv("INTERNAL_TOKEN", "valid-token-long-enough-for-testing")
	os.Setenv("REDIS_HOST", "redis.example.com")
	os.Setenv("REDIS_PORT", "9999")
	t.Cleanup(func() {
		os.Unsetenv("USER_SERVICE_ADDR")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("INTERNAL_TOKEN")
		os.Unsetenv("REDIS_HOST")
		os.Unsetenv("REDIS_PORT")
	})

	cfg := gwserver.DefaultConfig()
	if cfg.UserServiceAddr != "localhost:9999" {
		t.Errorf("UserServiceAddr = %s", cfg.UserServiceAddr)
	}
	if cfg.JWTSecret != "valid-secret-long-enough-for-hs256-algorithm-ok" {
		t.Errorf("JWTSecret not read from env")
	}
	if cfg.InternalToken != "valid-token-long-enough-for-testing" {
		t.Errorf("InternalToken not read from env")
	}
}

func TestDefaultConfig_Defaults(t *testing.T) {
	keys := []string{"USER_SERVICE_ADDR", "REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "JWT_SECRET", "INTERNAL_TOKEN"}
	saved := map[string]string{}
	for _, k := range keys {
		saved[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	t.Cleanup(func() {
		for k, v := range saved {
			if v != "" {
				os.Setenv(k, v)
			}
		}
	})

	cfg := gwserver.DefaultConfig()
	if cfg.UserServiceAddr != "127.0.0.1:9091" {
		t.Errorf("default UserServiceAddr = %s", cfg.UserServiceAddr)
	}
}

func TestNewEngine_MissingJWTSecret(t *testing.T) {
	logger := zap.NewNop()
	ready := &atomic.Bool{}
	_, _, err := gwserver.NewEngine("test", ready, logger, gwserver.Config{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewEngine_EmptyJWTSecret(t *testing.T) {
	logger := zap.NewNop()
	ready := &atomic.Bool{}
	_, _, err := gwserver.NewEngine("test", ready, logger, gwserver.Config{
		InternalToken: "valid-token-long-enough-for-testing",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewEngine_EmptyInternalToken(t *testing.T) {
	logger := zap.NewNop()
	ready := &atomic.Bool{}
	_, _, err := gwserver.NewEngine("test", ready, logger, gwserver.Config{
		JWTSecret: "valid-secret-long-enough-for-hs256-algorithm",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "INTERNAL_TOKEN") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewEngine_PlaceholderJWTSecret(t *testing.T) {
	logger := zap.NewNop()
	ready := &atomic.Bool{}
	_, _, err := gwserver.NewEngine("test", ready, logger, gwserver.Config{
		JWTSecret:     "replace-with-a-long-random-secret",
		InternalToken: "valid-token-long-enough-for-testing",
	})
	if err == nil {
		t.Fatal("expected error for placeholder JWT_SECRET")
	}
}

func TestNewEngine_PlaceholderInternalToken(t *testing.T) {
	logger := zap.NewNop()
	ready := &atomic.Bool{}
	_, _, err := gwserver.NewEngine("test", ready, logger, gwserver.Config{
		JWTSecret:     "valid-secret-long-enough-for-hs256-algorithm",
		InternalToken: "replace-with-a-long-random-internal-token",
	})
	if err == nil {
		t.Fatal("expected error for placeholder INTERNAL_TOKEN")
	}
}

func TestNewEngine_JWTSecretTooShort(t *testing.T) {
	logger := zap.NewNop()
	ready := &atomic.Bool{}
	_, _, err := gwserver.NewEngine("test", ready, logger, gwserver.Config{
		JWTSecret:     "short",
		InternalToken: "valid-token-long-enough-for-testing",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "at least 32") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewEngine_InternalTokenTooShort(t *testing.T) {
	logger := zap.NewNop()
	ready := &atomic.Bool{}
	_, _, err := gwserver.NewEngine("test", ready, logger, gwserver.Config{
		JWTSecret:     "valid-secret-long-enough-for-hs256-algorithm",
		InternalToken: "short",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "at least 16") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewEngine_Success(t *testing.T) {
	redisAddr := setupRedis(t)

	// Start a minimal gRPC server for the user service
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	grpcSrv := grpc.NewServer()
	go func() { _ = grpcSrv.Serve(lis) }()
	t.Cleanup(grpcSrv.Stop)

	logger := zap.NewNop()
	ready := &atomic.Bool{}
	engine, cleanup, err := gwserver.NewEngine("test", ready, logger, gwserver.Config{
		UserServiceAddr: lis.Addr().String(),
		RedisAddr:       redisAddr,
		RedisPassword:   "",
		JWTSecret:       "valid-secret-long-enough-for-hs256-algorithm-32chars",
		InternalToken:   "valid-token-long-enough-for-testing-16chars",
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	if engine == nil {
		t.Fatal("expected engine, got nil")
	}
	if err := cleanup(); err != nil {
		t.Errorf("cleanup: %v", err)
	}
}
