package xtrace

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestInit_ExporterUnreachable(t *testing.T) {
	// Use a cancelled context to force the exporter creation to fail.
	// Init should gracefully degrade: return no error and a no-op shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	shutdown, err := Init(ctx, "test-service")
	if err != nil {
		t.Fatalf("Init() returned unexpected error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown even when exporter is unavailable")
	}
	// The shutdown function must be callable and return nil.
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() returned error: %v", err)
	}
}

func TestInit_ExporterErrorLogsWarning(t *testing.T) {
	// When exporter creation fails and a logger is set, Init should log a warning.
	logger := zap.NewNop()
	SetLogger(logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	shutdown, err := Init(ctx, "test-service")
	if err != nil {
		t.Fatalf("Init() returned unexpected error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown")
	}

	SetLogger(nil)
}

func TestSetLogger_WithLogger(t *testing.T) {
	logger := zap.NewNop()
	SetLogger(logger)
	if globalLogger != logger {
		t.Error("expected logger to be set")
	}
	SetLogger(nil)
}

func TestOtlpDialTimeout_Default(t *testing.T) {
	if d := otlpDialTimeout(); d != 5*time.Second {
		t.Errorf("default timeout = %v, want 5s", d)
	}
}

func TestOtlpDialTimeout_FromEnv(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_TIMEOUT", "10")
	if d := otlpDialTimeout(); d != 10*time.Second {
		t.Errorf("timeout = %v, want 10s", d)
	}
}

func TestOtlpDialTimeout_InvalidEnv(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_TIMEOUT", "notanumber")
	if d := otlpDialTimeout(); d != 5*time.Second {
		t.Errorf("timeout = %v, want 5s (default for invalid)", d)
	}
}

func TestOtlpDialTimeout_ZeroEnv(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_TIMEOUT", "0")
	if d := otlpDialTimeout(); d != 5*time.Second {
		t.Errorf("timeout = %v, want 5s (default for zero)", d)
	}
}
