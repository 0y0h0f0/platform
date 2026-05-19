package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"

	"task-platform/internal/task/server"
	"task-platform/pkg/xhttp"
	"task-platform/pkg/xlog"
	"task-platform/pkg/xtrace"
)

type config struct {
	ServiceName            string `mapstructure:"service_name"`
	Env                    string `mapstructure:"env"`
	GRPCAddr               string `mapstructure:"grpc_addr"`
	AdminAddr              string `mapstructure:"admin_addr"`
	ReflectionEnabled      bool   `mapstructure:"reflection_enabled"`
	ShutdownTimeoutSeconds int    `mapstructure:"shutdown_timeout_seconds"`
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	cfg, err := loadConfig("task-service")
	if err != nil {
		return err
	}

	logger, err := xlog.New(cfg.ServiceName, cfg.Env)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}
	defer syncLogger(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	shutdownTrace, err := xtrace.Init(ctx, cfg.ServiceName)
	if err != nil {
		return fmt.Errorf("init trace: %w", err)
	}

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}

	serverBundle, err := server.NewGRPCServer(server.DefaultConfig())
	if err != nil {
		return fmt.Errorf("create grpc server: %w", err)
	}
	ready := &atomic.Bool{}
	adminServer := &http.Server{
		Addr:    cfg.AdminAddr,
		Handler: xhttp.NewEngine(cfg.ServiceName, ready),
	}

	errCh := make(chan error, 2)
	go func() {
		logger.Info("grpc server listening", zap.String("addr", cfg.GRPCAddr))
		if serveErr := serverBundle.GRPC.Serve(lis); serveErr != nil {
			errCh <- serveErr
		}
	}()
	go func() {
		logger.Info("admin http server listening", zap.String("addr", cfg.AdminAddr))
		if serveErr := adminServer.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
	}()

	ready.Store(true)

	select {
	case serveErr := <-errCh:
		return fmt.Errorf("serve: %w", serveErr)
	case <-ctx.Done():
	}

	ready.Store(false)
	serverBundle.Health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownTimeoutSeconds)*time.Second)
	defer cancel()

	logger.Info("shutting down")

	if err := adminServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown admin http: %w", err)
	}

	stopped := make(chan struct{})
	go func() {
		serverBundle.GRPC.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-shutdownCtx.Done():
		serverBundle.GRPC.Stop()
	}

	if err := shutdownTrace(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown trace: %w", err)
	}

	return nil
}

func loadConfig(service string) (config, error) {
	v := viper.New()
	v.SetConfigFile(defaultConfigPath(service))
	v.AutomaticEnv()

	v.SetDefault("service_name", service)
	v.SetDefault("env", defaultEnv())
	v.SetDefault("grpc_addr", ":9092")
	v.SetDefault("admin_addr", ":8082")
	v.SetDefault("reflection_enabled", true)
	v.SetDefault("shutdown_timeout_seconds", 10)

	if err := v.ReadInConfig(); err != nil {
		return config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg config
	if err := v.Unmarshal(&cfg); err != nil {
		return config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	return cfg, nil
}

func defaultConfigPath(service string) string {
	if explicit := os.Getenv("CONFIG_FILE"); explicit != "" {
		return explicit
	}

	return filepath.Join("configs", defaultEnv(), service+".yaml")
}

func defaultEnv() string {
	if env := os.Getenv("APP_ENV"); env != "" {
		return env
	}

	return "local"
}

func syncLogger(logger *zap.Logger) {
	_ = logger.Sync()
}
