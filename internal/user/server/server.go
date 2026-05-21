package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	userv1 "task-platform/gen/go/user/v1"
	"task-platform/internal/user/biz"
	"task-platform/internal/user/data"
	"task-platform/internal/user/service"
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
		GRPCAddr:          envOrDefault("GRPC_ADDR", ":9091"),
		AdminAddr:         envOrDefault("ADMIN_ADDR", ":8081"),
		ReflectionEnabled: false,
		PostgresDSN:       os.Getenv("POSTGRES_DSN"),
		RedisAddr:         fmt.Sprintf("%s:%s", envOrDefault("REDIS_HOST", "127.0.0.1"), envOrDefault("REDIS_PORT", "6380")),
		RedisPassword:     os.Getenv("REDIS_PASSWORD"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		InternalToken:     os.Getenv("INTERNAL_TOKEN"),
		WeakPasswordsPath: envOrDefault("WEAK_PASSWORDS_PATH", "configs/security/weak_passwords.txt"),
	}
}

func NewGRPCServer(cfg Config) (*xgrpc.Server, error) {
	if cfg.PostgresDSN == "" {
		return nil, xerr.NewError(xerr.CodeFailedPrecondition, "POSTGRES_DSN is required")
	}
	if err := validateSecret("JWT_SECRET", cfg.JWTSecret, 32); err != nil {
		return nil, err
	}
	if err := validateSecret("INTERNAL_TOKEN", cfg.InternalToken, 16); err != nil {
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
	interceptor := newAuthInterceptor(cfg.InternalToken, rdb)
	srv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			xgrpc.UnaryServerMetricsInterceptor(),
			loggingInterceptor(logger),
			interceptor,
		),
	)

	grpcServer := xgrpc.NewServer(srv, cfg.ReflectionEnabled)

	userv1.RegisterUserServiceServer(grpcServer.GRPC, svc)

	return grpcServer, nil
}

func newAuthInterceptor(internalToken string, rdb *redis.Client) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		token := singleValue(md, "x-internal-token")
		got := sha256.Sum256([]byte(token))
		want := sha256.Sum256([]byte(internalToken))
		if subtle.ConstantTimeCompare(got[:], want[:]) != 1 {
			return nil, status.Error(codes.Unauthenticated, "invalid internal token")
		}

		if !anonymousMethods[info.FullMethod] {
			userID := singleValue(md, "x-user-id")
			username := singleValue(md, "x-username")
			if userID == "" || username == "" {
				return nil, status.Error(codes.Unauthenticated, "missing user identity")
			}
		}

		return handler(ctx, req)
	}
}

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

func singleValue(md metadata.MD, key string) string {
	vals := md.Get(key)
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
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

func validateSecret(name, value string, minLen int) error {
	if value == "" {
		return xerr.NewError(xerr.CodeFailedPrecondition, name+" is required")
	}
	if value == "replace-with-a-long-random-secret" || value == "replace-with-a-long-random-internal-token" {
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
