package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	userv1 "task-platform/gen/go/user/v1"
	"task-platform/internal/user/biz"
	"task-platform/internal/user/data"
	"task-platform/internal/user/service"
	"task-platform/pkg/xconfig"
	"task-platform/pkg/xerr"
	"task-platform/pkg/xgrpc"
	"task-platform/pkg/xjwt"
	"task-platform/pkg/xpgsql"
	"task-platform/pkg/xredis"
)

var anonymousMethods = map[string]bool{
	userv1.UserService_Register_FullMethodName: true,
	userv1.UserService_Login_FullMethodName:    true,
}

type Config struct {
	GRPCAddr          string
	AdminAddr         string
	ReflectionEnabled bool
	PostgresDSN       string
	RedisAddr         string
	RedisPassword     string
	JWTSecret         string
	InternalToken     string
	WeakPasswordsPath string
}

func DefaultConfig() Config {
	return Config{
		GRPCAddr:          xconfig.EnvOrDefault("GRPC_ADDR", ":9091"),
		AdminAddr:         xconfig.EnvOrDefault("ADMIN_ADDR", ":8081"),
		ReflectionEnabled: false,
		PostgresDSN:       os.Getenv("POSTGRES_DSN"),
		RedisAddr:         fmt.Sprintf("%s:%s", xconfig.EnvOrDefault("REDIS_HOST", "127.0.0.1"), xconfig.EnvOrDefault("REDIS_PORT", "6380")),
		RedisPassword:     os.Getenv("REDIS_PASSWORD"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		InternalToken:     os.Getenv("INTERNAL_TOKEN"),
		WeakPasswordsPath: xconfig.EnvOrDefault("WEAK_PASSWORDS_PATH", "configs/security/weak_passwords.txt"),
	}
}

func NewGRPCServer(cfg Config) (*ServerBundle, error) {
	if cfg.PostgresDSN == "" {
		return nil, xerr.NewError(xerr.CodeFailedPrecondition, "POSTGRES_DSN is required")
	}
	if err := xconfig.ValidateSecret("JWT_SECRET", cfg.JWTSecret, 32); err != nil {
		return nil, err
	}
	if err := xconfig.ValidateSecret("INTERNAL_TOKEN", cfg.InternalToken, 16); err != nil {
		return nil, err
	}

	db, err := xpgsql.New(cfg.PostgresDSN)
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, fmt.Sprintf("connect postgres: %v", err))
	}

	rdb, err := xredis.New(cfg.RedisAddr, cfg.RedisPassword, 0)
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, fmt.Sprintf("connect redis: %v", err))
	}

	weakPasswords, err := loadWeakPasswords(cfg.WeakPasswordsPath)
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, fmt.Sprintf("load weak passwords: %v", err))
	}

	repo := data.NewUserRepository(db)
	b := biz.NewUserBiz(repo, rdb, weakPasswords)
	jwtManager := xjwt.NewManager(cfg.JWTSecret)
	svc := service.NewUserService(b, jwtManager)

	logger, err := zap.NewProduction()
	if err != nil {
		log.Printf("WARN: failed to create zap logger, falling back to no-op: %v", err)
		logger = zap.NewNop()
	}
	b.SetLogger(logger)
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

	userv1.RegisterUserServiceServer(grpcServer.GRPC, svc)

	return &ServerBundle{Server: grpcServer, db: db, rdb: rdb, logger: logger}, nil
}

type ServerBundle struct {
	*xgrpc.Server
	db     *gorm.DB
	rdb    *redis.Client
	logger *zap.Logger
}

func (b *ServerBundle) Shutdown() {
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
		}

		return handler(ctx, req)
	}
}

func loadWeakPasswords(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, fmt.Sprintf("read weak passwords file: %v", err))
	}
	var result []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result, nil
}
