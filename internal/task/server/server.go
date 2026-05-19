package server

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	userv1 "task-platform/gen/go/user/v1"
	taskv1 "task-platform/gen/go/task/v1"
	"task-platform/internal/task/biz"
	"task-platform/internal/task/data"
	"task-platform/internal/task/service"
	"task-platform/pkg/xgrpc"
	"task-platform/pkg/xpgsql"
)

var anonymousMethods = map[string]bool{}

type Config struct {
	GRPCAddr          string
	AdminAddr         string
	ReflectionEnabled bool
	PostgresDSN       string
	InternalToken     string
	UserServiceAddr   string
}

func DefaultConfig() Config {
	return Config{
		GRPCAddr:          envOrDefault("GRPC_ADDR", ":9092"),
		AdminAddr:         envOrDefault("ADMIN_ADDR", ":8082"),
		ReflectionEnabled: true,
		PostgresDSN:       envOrDefault("POSTGRES_DSN", "host=127.0.0.1 port=5433 user=postgres password=postgres dbname=task_platform sslmode=disable"),
		InternalToken:     os.Getenv("INTERNAL_TOKEN"),
		UserServiceAddr:   envOrDefault("USER_SERVICE_ADDR", "127.0.0.1:9091"),
	}
}

func NewGRPCServer(cfg Config) (*xgrpc.Server, error) {
	if err := validateSecret("INTERNAL_TOKEN", cfg.InternalToken, 16); err != nil {
		return nil, err
	}

	db, err := xpgsql.New(cfg.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	userClient, err := newUserServiceClient(cfg.UserServiceAddr, cfg.InternalToken)
	if err != nil {
		return nil, fmt.Errorf("connect user-service: %w", err)
	}

	projectRepo := data.NewProjectRepository(db)
	memberRepo := data.NewMemberRepository(db)
	taskRepo := data.NewTaskRepository(db)

	userAdapter := &userClientAdapter{client: userClient}
	b := biz.NewProjectBiz(db, projectRepo, memberRepo, userAdapter)
	tb := biz.NewTaskBiz(db, taskRepo, projectRepo, memberRepo, userAdapter)
	svc := service.NewTaskService(b, tb)

	grpcServer := xgrpc.NewServer(cfg.ReflectionEnabled)

	interceptor := newAuthInterceptor(cfg.InternalToken)
	grpcServer.GRPC = grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptor),
	)

	taskv1.RegisterTaskServiceServer(grpcServer.GRPC, svc)

	return grpcServer, nil
}

var TestAuthInterceptor = newAuthInterceptor

func newAuthInterceptor(internalToken string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		token := singleValue(md, "x-internal-token")
		if token != internalToken {
			return nil, status.Error(codes.Unauthenticated, "invalid internal token")
		}

		if !anonymousMethods[info.FullMethod] {
			userID := singleValue(md, "x-user-id")
			username := singleValue(md, "x-username")
			if userID == "" || username == "" {
				return nil, status.Error(codes.Unauthenticated, "missing user identity")
			}
			ctx = service.WithUserID(ctx, userID)
			ctx = service.WithUsername(ctx, username)
		}

		return handler(ctx, req)
	}
}

func singleValue(md metadata.MD, key string) string {
	vals := md.Get(key)
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

func newUserServiceClient(addr, internalToken string) (userv1.UserServiceClient, error) {
	interceptor := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		md, _ := metadata.FromOutgoingContext(ctx)
		if md == nil {
			md = metadata.New(nil)
		}
		md.Set("x-internal-token", internalToken)
		return invoker(metadata.NewOutgoingContext(ctx, md), method, req, reply, cc, opts...)
	}

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptor),
	)
	if err != nil {
		return nil, fmt.Errorf("dial user-service: %w", err)
	}

	return userv1.NewUserServiceClient(conn), nil
}

type userClientAdapter struct {
	client userv1.UserServiceClient
}

func (a *userClientAdapter) GetUser(ctx context.Context, userID string) (bool, bool, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	if md == nil {
		md = metadata.New(nil)
	}
	md.Set("x-user-id", service.GetUserID(ctx))
	md.Set("x-username", service.GetUsername(ctx))

	res, err := a.client.GetUser(metadata.NewOutgoingContext(ctx, md), &userv1.GetUserRequest{UserId: userID})
	if err != nil {
		st, _ := status.FromError(err)
		if st.Code() == codes.NotFound {
			return false, false, nil
		}
		return false, false, err
	}
	return true, res.User.Status == 0, nil
}

var _ biz.UserServiceClient = (*userClientAdapter)(nil)

func validateSecret(name, value string, minLen int) error {
	if value == "" {
		return fmt.Errorf("%s is required", name)
	}
	if value == "replace-with-a-long-random-internal-token" {
		return fmt.Errorf("%s must be changed from the default placeholder", name)
	}
	if len(value) < minLen {
		return fmt.Errorf("%s must be at least %d characters", name, minLen)
	}
	return nil
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
