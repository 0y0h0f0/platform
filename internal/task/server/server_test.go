package server_test

import (
	"os"
	"testing"

	taskserver "task-platform/internal/task/server"
)

func TestDefaultConfig(t *testing.T) {
	_ = os.Setenv("INTERNAL_TOKEN", "valid-token-long-enough-for-testing")
	_ = os.Setenv("USER_SERVICE_ADDR", "localhost:9999")
	_ = os.Setenv("POSTGRES_DSN", "host=localhost port=5432 user=postgres dbname=test sslmode=disable")
	t.Cleanup(func() {
		_ = os.Unsetenv("INTERNAL_TOKEN")
		_ = os.Unsetenv("USER_SERVICE_ADDR")
		_ = os.Unsetenv("POSTGRES_DSN")
	})

	cfg := taskserver.DefaultConfig()
	if cfg.InternalToken != "valid-token-long-enough-for-testing" {
		t.Errorf("InternalToken = %s", cfg.InternalToken)
	}
	if cfg.UserServiceAddr != "localhost:9999" {
		t.Errorf("UserServiceAddr = %s", cfg.UserServiceAddr)
	}
}

func TestDefaultConfig_Defaults(t *testing.T) {
	keys := []string{"INTERNAL_TOKEN", "USER_SERVICE_ADDR", "POSTGRES_DSN"}
	saved := map[string]string{}
	for _, k := range keys {
		saved[k] = os.Getenv(k)
		_ = os.Unsetenv(k)
	}
	t.Cleanup(func() {
		for k, v := range saved {
			if v != "" {
				_ = os.Setenv(k, v)
			}
		}
	})

	cfg := taskserver.DefaultConfig()
	if cfg.GRPCAddr != ":9092" {
		t.Errorf("GRPCAddr = %s", cfg.GRPCAddr)
	}
	if cfg.AdminAddr != ":8082" {
		t.Errorf("AdminAddr = %s", cfg.AdminAddr)
	}
}

func TestNewGRPCServer_MissingInternalToken(t *testing.T) {
	_, err := taskserver.NewGRPCServer(taskserver.Config{})
	if err == nil {
		t.Error("expected error for missing internal token")
	}
}

func TestNewGRPCServer_PlaceholderToken(t *testing.T) {
	_, err := taskserver.NewGRPCServer(taskserver.Config{
		InternalToken: "replace-with-a-long-random-internal-token",
	})
	if err == nil {
		t.Error("expected error for placeholder token")
	}
}

func TestNewGRPCServer_TokenTooShort(t *testing.T) {
	_, err := taskserver.NewGRPCServer(taskserver.Config{
		InternalToken: "short",
	})
	if err == nil {
		t.Error("expected error for short token")
	}
}
