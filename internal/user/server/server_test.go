package server_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	userv1 "task-platform/gen/go/user/v1"
	"task-platform/internal/user/server"
)

func setupDBForServer(t *testing.T) string {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx, "postgres:16",
		tcpostgres.WithDatabase("task_platform"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = pgContainer.Terminate(ctx) })

	pgDSN, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("pg connection string: %v", err)
	}

	// Run migration
	db, err := gorm.Open(postgres.Open(pgDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}

	rootDir := findRootDir()
	migrationSQL, err := os.ReadFile(filepath.Join(rootDir, "migrations", "user_svc", "000001_init.up.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	if err := db.Exec(string(migrationSQL)).Error; err != nil {
		t.Fatalf("run migration: %v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.Close()

	return pgDSN
}

func findRootDir() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func setupRedisForServer(t *testing.T) string {
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
	os.Setenv("GRPC_ADDR", ":9991")
	os.Setenv("JWT_SECRET", "test-jwt-secret-long-enough-for-hs256-algorithm")
	os.Setenv("INTERNAL_TOKEN", "test-internal-token-long-enough-for-testing")
	t.Cleanup(func() {
		os.Unsetenv("GRPC_ADDR")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("INTERNAL_TOKEN")
	})

	cfg := server.DefaultConfig()
	if cfg.GRPCAddr != ":9991" {
		t.Errorf("GRPCAddr = %s, want :9991", cfg.GRPCAddr)
	}
	if cfg.JWTSecret != "test-jwt-secret-long-enough-for-hs256-algorithm" {
		t.Errorf("JWTSecret not read from env")
	}
	if cfg.InternalToken != "test-internal-token-long-enough-for-testing" {
		t.Errorf("InternalToken not read from env")
	}
}

func TestDefaultConfig_Defaults(t *testing.T) {
	keys := []string{"GRPC_ADDR", "REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "JWT_SECRET", "INTERNAL_TOKEN", "WEAK_PASSWORDS_PATH"}
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

	cfg := server.DefaultConfig()
	if cfg.GRPCAddr != ":9091" {
		t.Errorf("default GRPCAddr = %s, want :9091", cfg.GRPCAddr)
	}
	if cfg.JWTSecret != "" {
		t.Errorf("JWTSecret should be empty by default, got %s", cfg.JWTSecret)
	}
}

func TestNewGRPCServer_MissingJWTSecret(t *testing.T) {
	_, err := server.NewGRPCServer(server.Config{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewGRPCServer_EmptyJWTSecret(t *testing.T) {
	_, err := server.NewGRPCServer(server.Config{
		InternalToken: "valid-token-that-is-long-enough-for-testing",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewGRPCServer_EmptyInternalToken(t *testing.T) {
	_, err := server.NewGRPCServer(server.Config{
		JWTSecret: "valid-secret-that-is-long-enough-for-hs256-algorithm",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "INTERNAL_TOKEN") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewGRPCServer_PlaceholderJWTSecret(t *testing.T) {
	_, err := server.NewGRPCServer(server.Config{
		JWTSecret:     "replace-with-a-long-random-secret",
		InternalToken: "valid-token-that-is-long-enough-for-tests",
	})
	if err == nil {
		t.Fatal("expected error for placeholder JWT_SECRET")
	}
}

func TestNewGRPCServer_PlaceholderInternalToken(t *testing.T) {
	_, err := server.NewGRPCServer(server.Config{
		JWTSecret:     "valid-secret-that-is-long-enough-for-hs256-algorithm",
		InternalToken: "replace-with-a-long-random-internal-token",
	})
	if err == nil {
		t.Fatal("expected error for placeholder INTERNAL_TOKEN")
	}
}

func TestNewGRPCServer_JWTSecretTooShort(t *testing.T) {
	_, err := server.NewGRPCServer(server.Config{
		JWTSecret:     "short",
		InternalToken: "valid-token-that-is-long-enough-for-tests",
	})
	if err == nil {
		t.Fatal("expected error for short JWT_SECRET")
	}
	if !strings.Contains(err.Error(), "at least 32") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewGRPCServer_InternalTokenTooShort(t *testing.T) {
	_, err := server.NewGRPCServer(server.Config{
		JWTSecret:     "valid-secret-that-is-long-enough-for-hs256-algorithm",
		InternalToken: "short",
	})
	if err == nil {
		t.Fatal("expected error for short INTERNAL_TOKEN")
	}
	if !strings.Contains(err.Error(), "at least 16") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewGRPCServer_DBConnectionFailure(t *testing.T) {
	_, err := server.NewGRPCServer(server.Config{
		JWTSecret:     "valid-secret-long-enough-for-hs256-testing",
		InternalToken: "valid-token-long-enough-for-testing-ok",
		PostgresDSN:   "host=127.0.0.1 port=19999 user=invalid password=invalid dbname=invalid sslmode=disable",
	})
	if err == nil {
		t.Fatal("expected db connection error")
	}
	if !strings.Contains(err.Error(), "connect postgres") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewGRPCServer_Success(t *testing.T) {
	pgDSN := setupDBForServer(t)
	redisAddr := setupRedisForServer(t)

	weakPath := createTempWeakPasswords(t)

	cfg := server.Config{
		PostgresDSN:       pgDSN,
		RedisAddr:         redisAddr,
		JWTSecret:         "valid-secret-long-enough-for-hs256-testing-ok",
		InternalToken:     "valid-token-long-enough-for-testing-ok",
		WeakPasswordsPath: weakPath,
		ReflectionEnabled: false,
	}

	srv, err := server.NewGRPCServer(cfg)
	if err != nil {
		t.Fatalf("NewGRPCServer: %v", err)
	}
	if srv == nil {
		t.Fatal("expected server, got nil")
	}

	_ = srv.GRPC.GetServiceInfo()
}

func createTempWeakPasswords(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp("", "weak-passwords-*.txt")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	content := "password\n123456\nadmin\n"
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func TestNewGRPCServer_MissingWeakPasswordsFile(t *testing.T) {
	pgDSN := setupDBForServer(t)
	redisAddr := setupRedisForServer(t)

	cfg := server.Config{
		PostgresDSN:       pgDSN,
		RedisAddr:         redisAddr,
		JWTSecret:         "valid-secret-long-enough-for-hs256-testing-ok",
		InternalToken:     "valid-token-long-enough-for-testing-ok",
		WeakPasswordsPath: "/nonexistent/path/weak_passwords.txt",
	}

	_, err := server.NewGRPCServer(cfg)
	if err == nil {
		t.Fatal("expected error for missing weak passwords file")
	}
	if !strings.Contains(err.Error(), "load weak passwords") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewAuthInterceptor_InvalidToken(t *testing.T) {
	tok := "valid-token-long-enough-for-testing-ok"
	pgDSN := setupDBForServer(t)
	redisAddr := setupRedisForServer(t)
	weakPath := createTempWeakPasswords(t)

	srv, err := server.NewGRPCServer(server.Config{
		PostgresDSN:       pgDSN,
		RedisAddr:         redisAddr,
		JWTSecret:         "valid-secret-long-enough-for-hs256-testing-ok",
		InternalToken:     tok,
		WeakPasswordsPath: weakPath,
		ReflectionEnabled: false,
	})
	if err != nil {
		t.Fatalf("NewGRPCServer: %v", err)
	}

	bLis := bufconn.Listen(1024 * 1024)
	go func() { _ = srv.GRPC.Serve(bLis) }()
	t.Cleanup(srv.GRPC.Stop)

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return bLis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc dial: %v", err)
	}
	defer conn.Close()

	// Call without x-internal-token metadata
	client := userv1.NewUserServiceClient(conn)
	_, err = client.GetUser(context.Background(), &userv1.GetUserRequest{UserId: "test"})
	if err == nil {
		t.Fatal("expected UNAUTHENTICATED error")
	}
	if !strings.Contains(err.Error(), "internal token") && !strings.Contains(err.Error(), "missing metadata") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewAuthInterceptor_InvalidInternalToken(t *testing.T) {
	tok := "valid-token-long-enough-for-testing-ok"
	pgDSN := setupDBForServer(t)
	redisAddr := setupRedisForServer(t)
	weakPath := createTempWeakPasswords(t)

	srv, err := server.NewGRPCServer(server.Config{
		PostgresDSN:       pgDSN,
		RedisAddr:         redisAddr,
		JWTSecret:         "valid-secret-long-enough-for-hs256-testing-ok",
		InternalToken:     tok,
		WeakPasswordsPath: weakPath,
		ReflectionEnabled: false,
	})
	if err != nil {
		t.Fatalf("NewGRPCServer: %v", err)
	}

	bLis := bufconn.Listen(1024 * 1024)
	go func() { _ = srv.GRPC.Serve(bLis) }()
	t.Cleanup(srv.GRPC.Stop)

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return bLis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc dial: %v", err)
	}
	defer conn.Close()

	// Call with wrong x-internal-token
	md := metadata.New(map[string]string{"x-internal-token": "wrong-token"})
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	client := userv1.NewUserServiceClient(conn)
	_, err = client.GetUser(ctx, &userv1.GetUserRequest{UserId: "test"})
	if err == nil {
		t.Fatal("expected UNAUTHENTICATED error for invalid token")
	}
	if !strings.Contains(err.Error(), "invalid internal token") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewAuthInterceptor_MissingUserIdentity(t *testing.T) {
	// This test verifies the auth interceptor's behavior through the bufconn approach
	pass := "valid-secret-long-enough-for-hs256-testing-ok"
	tok := "valid-token-long-enough-for-testing-ok"

	pgDSN := setupDBForServer(t)
	redisAddr := setupRedisForServer(t)
	weakPath := createTempWeakPasswords(t)

	srv, err := server.NewGRPCServer(server.Config{
		PostgresDSN:       pgDSN,
		RedisAddr:         redisAddr,
		JWTSecret:         pass,
		InternalToken:     tok,
		WeakPasswordsPath: weakPath,
		ReflectionEnabled: false,
	})
	if err != nil {
		t.Fatalf("NewGRPCServer: %v", err)
	}

	// Use bufconn for in-memory gRPC connection
	bLis := bufconn.Listen(1024 * 1024)
	go func() { _ = srv.GRPC.Serve(bLis) }()
	t.Cleanup(srv.GRPC.Stop)

	dialer := func(context.Context, string) (net.Conn, error) {
		return bLis.Dial()
	}

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc dial: %v", err)
	}
	defer conn.Close()

	// Call GetUser (non-anonymous) without x-user-id
	md := metadata.New(map[string]string{"x-internal-token": tok})
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	client := userv1.NewUserServiceClient(conn)
	_, err = client.GetUser(ctx, &userv1.GetUserRequest{UserId: "test"})
	if err == nil {
		t.Fatal("expected UNAUTHENTICATED error for missing user identity")
	}
	if !strings.Contains(err.Error(), "missing user identity") {
		t.Errorf("unexpected error: %v", err)
	}
}
