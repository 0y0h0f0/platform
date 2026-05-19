//go:build integration

package integration

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	userv1 "task-platform/gen/go/user/v1"
	taskv1 "task-platform/gen/go/task/v1"
	"task-platform/internal/user/server"
	taskserver "task-platform/internal/task/server"
)

const (
	testJWTSecret     = "integration-test-secret-at-least-32-chars-long"
	testInternalToken = "integration-test-internal-token"
)

var (
	grpcClient      userv1.UserServiceClient
	taskGrpcClient  taskv1.TaskServiceClient
	grpcConn        *grpc.ClientConn
	taskGrpcConn    *grpc.ClientConn
	pgContainer     *tcpostgres.PostgresContainer
	redisC          testcontainers.Container
	grpcSrv         *grpc.Server
	taskGrpcSrv     *grpc.Server
	grpcLisAddr     string
	taskGrpcLisAddr string
	redisAddr       string
)

func TestMain(m *testing.M) {
	code, err := runTests(m)
	if err != nil {
		fmt.Printf("setup error: %v\n", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func runTests(m *testing.M) (int, error) {
	ctx := context.Background()

	// Start PostgreSQL
	var err error
	pgContainer, err = tcpostgres.Run(ctx, "postgres:16",
		tcpostgres.WithDatabase("task_platform"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return 0, fmt.Errorf("start postgres: %w", err)
	}
	defer pgContainer.Terminate(ctx)

	pgDSN, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return 0, fmt.Errorf("pg connection string: %w", err)
	}

	// Run migrations
	db, err := gorm.Open(postgres.Open(pgDSN), &gorm.Config{})
	if err != nil {
		return 0, fmt.Errorf("connect postgres: %w", err)
	}

	rootDir := findProjectRoot()
	migrationPath := filepath.Join(rootDir, "migrations", "user_svc", "000001_init.up.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		return 0, fmt.Errorf("read migration at %s: %w", migrationPath, err)
	}

	if err := db.Exec(string(migrationSQL)).Error; err != nil {
		return 0, fmt.Errorf("run user_svc migration: %w", err)
	}

	taskMigrationPath := filepath.Join(rootDir, "migrations", "task_svc", "000001_init.up.sql")
	taskMigrationSQL, err := os.ReadFile(taskMigrationPath)
	if err != nil {
		return 0, fmt.Errorf("read task_svc migration at %s: %w", taskMigrationPath, err)
	}
	if err := db.Exec(string(taskMigrationSQL)).Error; err != nil {
		return 0, fmt.Errorf("run task_svc migration: %w", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.Close()

	// Start Redis
	redisC, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("Ready to accept connections").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		return 0, fmt.Errorf("start redis: %w", err)
	}
	defer redisC.Terminate(ctx)

	redisHost, err := redisC.Host(ctx)
	if err != nil {
		return 0, fmt.Errorf("redis host: %w", err)
	}
	redisPort, err := redisC.MappedPort(ctx, "6379")
	if err != nil {
		return 0, fmt.Errorf("redis port: %w", err)
	}
	redisAddr = fmt.Sprintf("%s:%s", redisHost, redisPort.Port())

	// Set up user-service gRPC server
	srvBundle, err := server.NewGRPCServer(server.Config{
		GRPCAddr:          "127.0.0.1:0",
		AdminAddr:         "127.0.0.1:0",
		ReflectionEnabled: false,
		PostgresDSN:       pgDSN,
		RedisAddr:         redisAddr,
		RedisPassword:     "",
		JWTSecret:         testJWTSecret,
		InternalToken:     testInternalToken,
		WeakPasswordsPath: filepath.Join(rootDir, "configs", "security", "weak_passwords.txt"),
	})
	if err != nil {
		return 0, fmt.Errorf("create grpc server: %w", err)
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("listen: %w", err)
	}
	grpcSrv = srvBundle.GRPC
	grpcLisAddr = lis.Addr().String()

	go grpcSrv.Serve(lis)

	// Create gRPC client with internal token interceptor
	grpcConn, err = grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			md, _ := metadata.FromOutgoingContext(ctx)
			if md == nil {
				md = metadata.New(nil)
			}
			md.Set("x-internal-token", testInternalToken)
			return invoker(metadata.NewOutgoingContext(ctx, md), method, req, reply, cc, opts...)
		}),
	)
	if err != nil {
		return 0, fmt.Errorf("create grpc client: %w", err)
	}
	defer grpcConn.Close()
	grpcClient = userv1.NewUserServiceClient(grpcConn)

	// Start task-service gRPC server
	taskSrvBundle, err := taskserver.NewGRPCServer(taskserver.Config{
		GRPCAddr:          "127.0.0.1:0",
		AdminAddr:         "127.0.0.1:0",
		ReflectionEnabled: false,
		PostgresDSN:       pgDSN,
		InternalToken:     testInternalToken,
		UserServiceAddr:   grpcLisAddr,
	})
	if err != nil {
		return 0, fmt.Errorf("create task grpc server: %w", err)
	}

	taskLis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("listen task: %w", err)
	}
	taskGrpcSrv = taskSrvBundle.GRPC
	taskGrpcLisAddr = taskLis.Addr().String()

	go taskGrpcSrv.Serve(taskLis)

	// Create task-service gRPC client
	taskGrpcConn, err = grpc.NewClient(taskLis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			md, _ := metadata.FromOutgoingContext(ctx)
			if md == nil {
				md = metadata.New(nil)
			}
			md.Set("x-internal-token", testInternalToken)
			return invoker(metadata.NewOutgoingContext(ctx, md), method, req, reply, cc, opts...)
		}),
	)
	if err != nil {
		return 0, fmt.Errorf("create task grpc client: %w", err)
	}
	defer taskGrpcConn.Close()
	taskGrpcClient = taskv1.NewTaskServiceClient(taskGrpcConn)

	return m.Run(), nil
}

func withUser(ctx context.Context, userID, username string) context.Context {
	md := metadata.Pairs("x-user-id", userID, "x-username", username)
	return metadata.NewOutgoingContext(ctx, md)
}

func TestIntegration_Register(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "secret123",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, res.AccessToken)
	assert.Equal(t, "alice", res.User.Username)
	assert.NotEmpty(t, res.User.Id)
}

func TestIntegration_Register_DuplicateUsername(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "bob", Email: "bob@example.com", Password: "secret123",
	})
	require.NoError(t, err)

	_, err = grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "bob", Email: "bob2@example.com", Password: "secret123",
	})
	require.Error(t, err)
}

func TestIntegration_Register_DuplicateEmail(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "carol1", Email: "carol@example.com", Password: "secret123",
	})
	require.NoError(t, err)

	_, err = grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "carol2", Email: "carol@example.com", Password: "secret123",
	})
	require.Error(t, err)
}

func TestIntegration_Login_WithUsername(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "dave", Email: "dave@example.com", Password: "secret123",
	})
	require.NoError(t, err)

	res, err := grpcClient.Login(ctx, &userv1.LoginRequest{
		Account: "dave", Password: "secret123",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, res.AccessToken)
	assert.Equal(t, "dave", res.User.Username)
}

func TestIntegration_Login_WithEmail(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "erin", Email: "erin@example.com", Password: "secret123",
	})
	require.NoError(t, err)

	res, err := grpcClient.Login(ctx, &userv1.LoginRequest{
		Account: "erin@example.com", Password: "secret123",
	})
	require.NoError(t, err)
	assert.Equal(t, "erin", res.User.Username)
}

func TestIntegration_Login_WrongPassword(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "frank", Email: "frank@example.com", Password: "secret123",
	})
	require.NoError(t, err)

	_, err = grpcClient.Login(ctx, &userv1.LoginRequest{
		Account: "frank", Password: "wrongpassword",
	})
	require.Error(t, err)
}

func TestIntegration_GetUser(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	regRes, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "grace", Email: "grace@example.com", Password: "secret123",
	})
	require.NoError(t, err)

	res, err := grpcClient.GetUser(withUser(ctx, regRes.User.Id, "grace"), &userv1.GetUserRequest{
		UserId: regRes.User.Id,
	})
	require.NoError(t, err)
	assert.Equal(t, "grace", res.User.Username)
	assert.Equal(t, "grace@example.com", res.User.Email)
}

func TestIntegration_GetUser_NotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := grpcClient.GetUser(withUser(ctx, "00000000-0000-0000-0000-000000000000", "ghost"), &userv1.GetUserRequest{
		UserId: "00000000-0000-0000-0000-000000000000",
	})
	require.Error(t, err)
}

func TestIntegration_BatchGetUsers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reg1, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "henry", Email: "henry@example.com", Password: "secret123",
	})
	require.NoError(t, err)
	reg2, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "iris", Email: "iris@example.com", Password: "secret123",
	})
	require.NoError(t, err)

	res, err := grpcClient.BatchGetUsers(withUser(ctx, reg1.User.Id, "henry"), &userv1.BatchGetUsersRequest{
		UserIds: []string{reg1.User.Id, reg2.User.Id},
	})
	require.NoError(t, err)
	assert.Len(t, res.Users, 2)
}

func TestIntegration_BatchGetUsers_EmptyList(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := grpcClient.BatchGetUsers(withUser(ctx, "any", "any"), &userv1.BatchGetUsersRequest{
		UserIds: []string{},
	})
	require.NoError(t, err)
	assert.Len(t, res.Users, 0)
}

func TestIntegration_JWT_TokenFormat(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := grpcClient.Register(ctx, &userv1.RegisterRequest{
		Username: "jack", Email: "jack@example.com", Password: "secret123",
	})
	require.NoError(t, err)

	assert.NotEmpty(t, res.AccessToken)
	assert.Contains(t, res.AccessToken, ".")
}

func findProjectRoot() string {
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
