package server

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
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
	"task-platform/pkg/xerr"
	"task-platform/pkg/xgrpc"
	"task-platform/pkg/xpgsql"
	"task-platform/pkg/xredis"
)

var anonymousMethods = map[string]bool{}

type Config struct {
	GRPCAddr          string
	AdminAddr         string
	ReflectionEnabled bool
	PostgresDSN       string
	RedisAddr         string
	RedisPassword     string
	InternalToken     string
	UserServiceAddr   string
}

func DefaultConfig() Config {
	return Config{
		GRPCAddr:          envOrDefault("GRPC_ADDR", ":9092"),
		AdminAddr:         envOrDefault("ADMIN_ADDR", ":8082"),
		ReflectionEnabled: true,
		PostgresDSN:       envOrDefault("POSTGRES_DSN", "host=127.0.0.1 port=5433 user=postgres password=postgres dbname=task_platform sslmode=disable"),
		RedisAddr:         fmt.Sprintf("%s:%s", envOrDefault("REDIS_HOST", "127.0.0.1"), envOrDefault("REDIS_PORT", "6380")),
		RedisPassword:     os.Getenv("REDIS_PASSWORD"),
		InternalToken:     os.Getenv("INTERNAL_TOKEN"),
		UserServiceAddr:   envOrDefault("USER_SERVICE_ADDR", "127.0.0.1:9091"),
	}
}

func NewGRPCServer(cfg Config) (*ServerBundle, error) {
	if err := validateSecret("INTERNAL_TOKEN", cfg.InternalToken, 16); err != nil {
		return nil, err
	}

	db, err := xpgsql.New(cfg.PostgresDSN)
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, fmt.Sprintf("connect postgres: %v", err))
	}

	userClient, err := newUserServiceClient(cfg.UserServiceAddr, cfg.InternalToken)
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, fmt.Sprintf("connect user-service: %v", err))
	}

	projectRepo := data.NewProjectRepository(db)
	memberRepo := data.NewMemberRepository(db)
	taskRepo := data.NewTaskRepository(db)
	commentRepo := data.NewCommentRepository(db)
	opLogRepo := data.NewOperationLogRepository(db)

	logger, _ := zap.NewProduction()
	logWriter := biz.NewLogWriter(opLogRepo, logger)

	rdb, redisErr := xredis.New(cfg.RedisAddr, cfg.RedisPassword, 0)
	if redisErr != nil {
		logger.Warn("redis unavailable, caching disabled", zap.Error(redisErr))
	}

	userAdapter := &userClientAdapter{client: userClient}
	b := biz.NewProjectBiz(db, projectRepo, memberRepo, userAdapter, logWriter, rdb)
	tb := biz.NewTaskBiz(db, taskRepo, projectRepo, memberRepo, userAdapter, logWriter)
	cb := biz.NewCommentBiz(db, commentRepo, taskRepo, projectRepo, memberRepo, logWriter)
	ob := biz.NewOpLogBiz(opLogRepo, projectRepo, taskRepo, memberRepo)
	svc := service.NewTaskService(b, tb, cb, ob)

	grpcServer := xgrpc.NewServer(cfg.ReflectionEnabled)

	interceptor := newAuthInterceptor(cfg.InternalToken)
	grpcServer.GRPC = grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			xgrpc.UnaryServerMetricsInterceptor(),
			loggingInterceptor(logger),
			interceptor,
		),
	)

	taskv1.RegisterTaskServiceServer(grpcServer.GRPC, svc)

	return &ServerBundle{Server: grpcServer, LogWriter: logWriter}, nil
}

type ServerBundle struct {
	*xgrpc.Server
	LogWriter *biz.LogWriter
}

var TestAuthInterceptor = newAuthInterceptor

func loggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		requestID := ""
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			requestID = singleValue(md, "x-request-id")
		}

		span := trace.SpanFromContext(ctx)
		sc := span.SpanContext()

		resp, err := handler(ctx, req)

		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("latency", time.Since(start)),
			zap.String("request_id", requestID),
			zap.String("trace_id", sc.TraceID().String()),
			zap.String("span_id", sc.SpanID().String()),
		}
		if err != nil {
			fields = append(fields, zap.Error(err))
			logger.Error("grpc request failed", fields...)
		} else {
			logger.Info("grpc request", fields...)
		}

		return resp, err
	}
}

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
	metadataInterceptor := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		md, _ := metadata.FromOutgoingContext(ctx)
		if md == nil {
			md = metadata.New(nil)
		}
		md.Set("x-internal-token", internalToken)
		return invoker(metadata.NewOutgoingContext(ctx, md), method, req, reply, cc, opts...)
	}

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithChainUnaryInterceptor(
			xgrpc.UnaryClientMetricsInterceptor(),
			metadataInterceptor,
		),
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
		return xerr.NewError(xerr.CodeFailedPrecondition, name+" is required")
	}
	if value == "replace-with-a-long-random-internal-token" {
		return xerr.NewError(xerr.CodeFailedPrecondition, name+" must be changed from the default placeholder")
	}
	if len(value) < minLen {
		return xerr.NewError(xerr.CodeFailedPrecondition, name+fmt.Sprintf(" must be at least %d characters", minLen))
	}
	return nil
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
