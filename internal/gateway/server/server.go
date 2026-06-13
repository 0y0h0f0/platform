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
	"task-platform/pkg/xconfig"
	"task-platform/pkg/xerr"
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
		UserServiceAddr: xconfig.EnvOrDefault("USER_SERVICE_ADDR", "127.0.0.1:9091"),
		TaskServiceAddr: xconfig.EnvOrDefault("TASK_SERVICE_ADDR", "127.0.0.1:9092"),
		RedisAddr:       fmt.Sprintf("%s:%s", xconfig.EnvOrDefault("REDIS_HOST", "127.0.0.1"), xconfig.EnvOrDefault("REDIS_PORT", "6380")),
		RedisPassword:   os.Getenv("REDIS_PASSWORD"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		InternalToken:   os.Getenv("INTERNAL_TOKEN"),
	}
}

func NewEngine(service string, ready *atomic.Bool, logger *zap.Logger, cfg Config) (*gin.Engine, func() error, error) {
	if err := xconfig.ValidateSecret("JWT_SECRET", cfg.JWTSecret, 32); err != nil {
		return nil, nil, err
	}
	if err := xconfig.ValidateSecret("INTERNAL_TOKEN", cfg.InternalToken, 16); err != nil {
		return nil, nil, err
	}

	rdb, err := xredis.New(cfg.RedisAddr, cfg.RedisPassword, 0)
	if err != nil {
		return nil, nil, xerr.NewError(xerr.CodeInternal, fmt.Sprintf("connect redis: %v", err))
	}

	clients, err := rpc.NewClients(context.Background(), cfg.UserServiceAddr, cfg.TaskServiceAddr, cfg.InternalToken)
	if err != nil {
		return nil, nil, xerr.NewError(xerr.CodeInternal, fmt.Sprintf("create rpc clients: %v", err))
	}

	jwtManager := xjwt.NewManager(cfg.JWTSecret)

	engine := xhttp.NewEngine(service, ready)

	publicPaths := []string{
		"/api/v1/auth/register",
		"/api/v1/auth/login",
		"/healthz",
		"/readyz",
	}

	engine.Use(middleware.MaxBodySize(1 << 20)) // 1 MB
	engine.Use(middleware.SecurityHeaders())
	engine.Use(middleware.RequestID())
	engine.Use(middleware.HTTPTrace())
	engine.Use(middleware.HTTPMetrics())
	engine.Use(middleware.AccessLog(logger))
	engine.Use(middleware.CORS())
	engine.Use(middleware.RateLimitByIP(rdb))
	engine.Use(middleware.Auth(jwtManager, rdb, publicPaths))
	engine.Use(middleware.RateLimitByUser(rdb))

	authH := handler.NewAuthHandler(clients.UserClient, rdb)
	userH := handler.NewUserHandler(clients.UserClient)
	projectH := handler.NewProjectHandler(clients.TaskClient, clients.UserClient, rdb)
	taskH := handler.NewTaskHandler(clients.TaskClient, clients.UserClient, rdb)

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
			projects.GET("/:id/operation-logs", projectH.ListOperationLogs)
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
			tasks.POST("/:id/comments", taskH.CreateComment)
			tasks.GET("/:id/comments", taskH.ListComments)
			tasks.DELETE("/:id/comments/:commentId", taskH.DeleteComment)
			tasks.GET("/:id/operation-logs", taskH.ListOperationLogs)
		}
	}

	cleanup := func() error {
		cerr := clients.Close()
		rerr := rdb.Close()
		if cerr != nil && rerr != nil {
			return fmt.Errorf("clients: %w; redis: %w", cerr, rerr)
		}
		if cerr != nil {
			return cerr
		}
		return rerr
	}

	return engine, cleanup, nil
}
