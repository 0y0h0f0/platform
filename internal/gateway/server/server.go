package server

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"task-platform/internal/gateway/handler"
	"task-platform/internal/gateway/middleware"
	"task-platform/internal/gateway/rpc"
	"task-platform/pkg/xhttp"
	"task-platform/pkg/xjwt"
	"task-platform/pkg/xredis"
)

type Config struct {
	UserServiceAddr string
	TaskServiceAddr string
	RedisAddr       string
	RedisPassword   string
	JWTSecret       string
	InternalToken   string
}

func DefaultConfig() Config {
	return Config{
		UserServiceAddr: envOrDefault("USER_SERVICE_ADDR", "127.0.0.1:9091"),
		TaskServiceAddr: envOrDefault("TASK_SERVICE_ADDR", "127.0.0.1:9092"),
		RedisAddr:       fmt.Sprintf("%s:%s", envOrDefault("REDIS_HOST", "127.0.0.1"), envOrDefault("REDIS_PORT", "6380")),
		RedisPassword:   os.Getenv("REDIS_PASSWORD"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		InternalToken:   os.Getenv("INTERNAL_TOKEN"),
	}
}

func NewEngine(service string, ready *atomic.Bool, logger *zap.Logger, cfg Config) (*gin.Engine, func() error, error) {
	if err := validateSecret("JWT_SECRET", cfg.JWTSecret, 32); err != nil {
		return nil, nil, err
	}
	if err := validateSecret("INTERNAL_TOKEN", cfg.InternalToken, 16); err != nil {
		return nil, nil, err
	}

	rdb, err := xredis.New(cfg.RedisAddr, cfg.RedisPassword, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("connect redis: %w", err)
	}

	clients, err := rpc.NewClients(context.Background(), cfg.UserServiceAddr, cfg.TaskServiceAddr, cfg.InternalToken)
	if err != nil {
		return nil, nil, fmt.Errorf("create rpc clients: %w", err)
	}

	jwtManager := xjwt.NewManager(cfg.JWTSecret)

	engine := xhttp.NewEngine(service, ready)

	publicPaths := []string{
		"/api/v1/auth/register",
		"/api/v1/auth/login",
	}

	engine.Use(middleware.RequestID())
	engine.Use(middleware.AccessLog(logger))
	engine.Use(middleware.CORS())
	engine.Use(middleware.RateLimitByIP())
	engine.Use(middleware.Auth(jwtManager, rdb, publicPaths))
	engine.Use(middleware.RateLimitByUser())

	authH := handler.NewAuthHandler(clients.UserClient, rdb)
	userH := handler.NewUserHandler(clients.UserClient)
	projectH := handler.NewProjectHandler(clients.TaskClient)
	taskH := handler.NewTaskHandler(clients.TaskClient)

	v1 := engine.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authH.Register)
			auth.POST("/login", authH.Login)
			auth.POST("/logout", authH.Logout)
		}
		users := v1.Group("/users")
		{
			users.GET("/me", userH.Me)
		}
		projects := v1.Group("/projects")
		{
			projects.POST("", projectH.Create)
			projects.GET("", projectH.List)
			projects.GET("/:id", projectH.Get)
			projects.PUT("/:id", projectH.Update)
			projects.POST("/:id/archive", projectH.Archive)
			projects.POST("/:id/unarchive", projectH.Unarchive)
			projects.POST("/:id/transfer", projectH.Transfer)
			projects.POST("/:id/members", projectH.AddMember)
			projects.GET("/:id/members", projectH.ListMembers)
			projects.PUT("/:id/members/:userId", projectH.UpdateMemberRole)
			projects.DELETE("/:id/members/:userId", projectH.RemoveMember)
			projects.POST("/:id/members/me/leave", projectH.Leave)
		}
		tasks := v1.Group("/tasks")
		{
			tasks.POST("", taskH.Create)
			tasks.GET("", taskH.List)
			tasks.GET("/:id", taskH.Get)
			tasks.PUT("/:id", taskH.Update)
			tasks.DELETE("/:id", taskH.Delete)
			tasks.POST("/:id/assign", taskH.Assign)
			tasks.POST("/:id/status", taskH.ChangeStatus)
		}
	}

	cleanup := func() error {
		if err := clients.Close(); err != nil {
			return err
		}
		return rdb.Close()
	}

	return engine, cleanup, nil
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
