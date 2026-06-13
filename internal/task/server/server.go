package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	taskv1 "task-platform/gen/go/task/v1"
	userv1 "task-platform/gen/go/user/v1"
	"task-platform/internal/task/biz"
	"task-platform/internal/task/data"
	"task-platform/internal/task/service"
	"task-platform/pkg/xconfig"
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
		GRPCAddr:          xconfig.EnvOrDefault("GRPC_ADDR", ":9092"),
		AdminAddr:         xconfig.EnvOrDefault("ADMIN_ADDR", ":8082"),
		ReflectionEnabled: false,
		PostgresDSN:       os.Getenv("POSTGRES_DSN"),
		RedisAddr:         fmt.Sprintf("%s:%s", xconfig.EnvOrDefault("REDIS_HOST", "127.0.0.1"), xconfig.EnvOrDefault("REDIS_PORT", "6380")),
		RedisPassword:     os.Getenv("REDIS_PASSWORD"),
		InternalToken:     os.Getenv("INTERNAL_TOKEN"),
		UserServiceAddr:   xconfig.EnvOrDefault("USER_SERVICE_ADDR", "127.0.0.1:9091"),
	}
}

func NewGRPCServer(cfg Config) (*ServerBundle, error) {
	if cfg.PostgresDSN == "" {
		return nil, xerr.NewError(xerr.CodeFailedPrecondition, "POSTGRES_DSN is required")
	}
	if err := xconfig.ValidateSecret("INTERNAL_TOKEN", cfg.InternalToken, 16); err != nil {
		return nil, err
	}

	db, err := xpgsql.New(cfg.PostgresDSN)
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, fmt.Sprintf("connect postgres: %v", err))
	}

	userClient, userConn, err := newUserServiceClient(cfg.UserServiceAddr, cfg.InternalToken)
	if err != nil {
		return nil, err
	}

	projectRepo := data.NewProjectRepository(db)
	memberRepo := data.NewMemberRepository(db)
	taskRepo := data.NewTaskRepository(db)
	commentRepo := data.NewCommentRepository(db)
	opLogRepo := data.NewOperationLogRepository(db)

	logger, err := zap.NewProduction()
	if err != nil {
		log.Printf("WARN: failed to create zap logger, falling back to no-op: %v", err)
		logger = zap.NewNop()
	}
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

	interceptor := newAuthInterceptor(cfg.InternalToken)
	srv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			xgrpc.UnaryServerMetricsInterceptor(),
			xgrpc.LoggingInterceptor(logger),
			xgrpc.UnaryServerTimeoutInterceptor(xgrpc.ServerTimeoutFromEnv()),
			interceptor,
		),
	)
	grpcServer := xgrpc.NewServer(srv, cfg.ReflectionEnabled)

	taskv1.RegisterTaskServiceServer(grpcServer.GRPC, svc)

	return &ServerBundle{Server: grpcServer, LogWriter: logWriter, userConn: userConn, db: db, rdb: rdb, logger: logger}, nil
}

type ServerBundle struct {
	*xgrpc.Server
	LogWriter *biz.LogWriter
	userConn  *grpc.ClientConn
	db        *gorm.DB
	rdb       *redis.Client
	logger    *zap.Logger
}

func (b *ServerBundle) Shutdown() {
	b.LogWriter.Shutdown()
	if b.userConn != nil {
		if err := b.userConn.Close(); err != nil {
			b.logger.Warn("failed to close user-service connection", zap.Error(err))
		}
	}
	if b.db != nil {
		sqlDB, err := b.db.DB()
		if err != nil {
			b.logger.Warn("failed to get underlying sql.DB", zap.Error(err))
		} else {
			if err := sqlDB.Close(); err != nil {
				b.logger.Warn("failed to close database connection pool", zap.Error(err))
			}
		}
	}
	if b.rdb != nil {
		if err := b.rdb.Close(); err != nil {
			b.logger.Warn("failed to close redis client", zap.Error(err))
		}
	}
}

var TestAuthInterceptor = newAuthInterceptor

func newAuthInterceptor(internalToken string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		token := xgrpc.SingleValue(md, "x-internal-token")
		got := sha256.Sum256([]byte(token))
		want := sha256.Sum256([]byte(internalToken))
		if subtle.ConstantTimeCompare(got[:], want[:]) != 1 {
			return nil, status.Error(codes.Unauthenticated, "invalid internal token")
		}

		if !anonymousMethods[info.FullMethod] {
			userID := xgrpc.SingleValue(md, "x-user-id")
			username := xgrpc.SingleValue(md, "x-username")
			if userID == "" || username == "" {
				return nil, status.Error(codes.Unauthenticated, "missing user identity")
			}
			ctx = service.WithUserID(ctx, userID)
			ctx = service.WithUsername(ctx, username)
		}

		return handler(ctx, req)
	}
}

func newUserServiceClient(addr, internalToken string) (userv1.UserServiceClient, *grpc.ClientConn, error) {
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
			xgrpc.UnaryClientTimeoutInterceptor(xgrpc.ClientTimeoutFromEnv()),
			xgrpc.UnaryClientMetricsInterceptor(),
			metadataInterceptor,
		),
	)
	if err != nil {
		return nil, nil, xerr.NewError(xerr.CodeInternal, fmt.Sprintf("dial user-service: %v", err))
	}

	return userv1.NewUserServiceClient(conn), conn, nil
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
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.NotFound {
			return false, false, nil
		}
		return false, false, err
	}
	return true, res.User.Status == 0, nil
}

var _ biz.UserServiceClient = (*userClientAdapter)(nil)
