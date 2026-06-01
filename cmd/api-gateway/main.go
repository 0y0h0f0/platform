package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	gwserver "task-platform/internal/gateway/server"
	"task-platform/pkg/xhttp"
	"task-platform/pkg/xlog"
	"task-platform/pkg/xtrace"
)

type config struct {
	ServiceName              string `mapstructure:"service_name"`
	Env                      string `mapstructure:"env"`
	HTTPAddr                 string `mapstructure:"http_addr"`
	ShutdownTimeoutSeconds   int    `mapstructure:"shutdown_timeout_seconds"`
	ReadHeaderTimeoutSeconds int    `mapstructure:"read_header_timeout_seconds"`
	ReadTimeoutSeconds       int    `mapstructure:"read_timeout_seconds"`
	WriteTimeoutSeconds      int    `mapstructure:"write_timeout_seconds"`
	IdleTimeoutSeconds       int    `mapstructure:"idle_timeout_seconds"`
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	cfg, err := loadConfig("api-gateway")
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

	ready := &atomic.Bool{}
	engine, cleanup, err := gwserver.NewEngine(cfg.ServiceName, ready, logger, gwserver.DefaultConfig())
	if err != nil {
		return fmt.Errorf("create engine: %w", err)
	}
	defer func() {
		if err := cleanup(); err != nil {
			logger.Error("cleanup failed", zap.Error(err))
		}
	}()

	server := xhttp.NewServer(cfg.HTTPAddr, engine, xhttp.ServerTimeoutsFromSeconds(
		cfg.ReadHeaderTimeoutSeconds,
		cfg.ReadTimeoutSeconds,
		cfg.WriteTimeoutSeconds,
		cfg.IdleTimeoutSeconds,
	))

	errCh := make(chan error, 1)
	go func() {
		logger.Info("http server listening", zap.String("addr", cfg.HTTPAddr))
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
	}()

	ready.Store(true)

	select {
	case serveErr := <-errCh:
		return fmt.Errorf("serve http: %w", serveErr)
	case <-ctx.Done():
	}

	ready.Store(false)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownTimeoutSeconds)*time.Second)
	defer cancel()

	logger.Info("shutting down")

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http: %w", err)
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
	v.SetDefault("http_addr", ":8080")
	v.SetDefault("shutdown_timeout_seconds", 10)
	v.SetDefault("read_header_timeout_seconds", 5)
	v.SetDefault("read_timeout_seconds", 10)
	v.SetDefault("write_timeout_seconds", 15)
	v.SetDefault("idle_timeout_seconds", 60)

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
