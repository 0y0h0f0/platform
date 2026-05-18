package server

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	userv1 "task-platform/gen/go/user/v1"
	"task-platform/internal/user/biz"
	"task-platform/internal/user/data"
	"task-platform/internal/user/service"
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
		ReflectionEnabled: true,
		PostgresDSN:       envOrDefault("POSTGRES_DSN", "host=127.0.0.1 port=5433 user=postgres password=postgres dbname=task_platform sslmode=disable"),
		RedisAddr:         fmt.Sprintf("%s:%s", envOrDefault("REDIS_HOST", "127.0.0.1"), envOrDefault("REDIS_PORT", "6380")),
		RedisPassword:     os.Getenv("REDIS_PASSWORD"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		InternalToken:     os.Getenv("INTERNAL_TOKEN"),
		WeakPasswordsPath: envOrDefault("WEAK_PASSWORDS_PATH", "configs/security/weak_passwords.txt"),
	}
}

func NewGRPCServer(cfg Config) (*xgrpc.Server, error) {
	if err := validateSecret("JWT_SECRET", cfg.JWTSecret, 32); err != nil {
		return nil, err
	}
	if err := validateSecret("INTERNAL_TOKEN", cfg.InternalToken, 16); err != nil {
		return nil, err
	}

	db, err := xpgsql.New(cfg.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	rdb, err := xredis.New(cfg.RedisAddr, cfg.RedisPassword, 0)
	if err != nil {
		return nil, fmt.Errorf("connect redis: %w", err)
	}

	weakPasswords, err := loadWeakPasswords(cfg.WeakPasswordsPath)
	if err != nil {
		return nil, fmt.Errorf("load weak passwords: %w", err)
	}

	repo := data.NewUserRepository(db)
	b := biz.NewUserBiz(repo, rdb, weakPasswords)
	jwtManager := xjwt.NewManager(cfg.JWTSecret)
	svc := service.NewUserService(b, jwtManager)

	grpcServer := xgrpc.NewServer(cfg.ReflectionEnabled)

	interceptor := newAuthInterceptor(cfg.InternalToken, rdb)
	grpcServer.GRPC = grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptor),
	)

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
		if token != internalToken {
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
		return nil, fmt.Errorf("read weak passwords file: %w", err)
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
		return fmt.Errorf("%s is required", name)
	}
	if value == "replace-with-a-long-random-secret" || value == "replace-with-a-long-random-internal-token" {
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
