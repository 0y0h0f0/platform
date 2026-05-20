package xtrace

import (
	"context"
	"testing"
)

func TestInit(t *testing.T) {
	t.Parallel()

	shutdown, err := Init(context.Background(), "test-service")
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
}

func TestSetLogger(t *testing.T) {
	t.Parallel()

	SetLogger(nil)
	if globalLogger != nil {
		t.Error("expected nil logger")
	}
}
