package middleware_test

import (
	"context"
	"testing"
	"time"

	"task-platform/internal/gateway/middleware"
)

func TestGetUserID(t *testing.T) {
	ctx := context.WithValue(context.Background(), middleware.CtxKeyUserID, "user-1")
	if v := middleware.GetUserID(ctx); v != "user-1" {
		t.Errorf("GetUserID = %s, want user-1", v)
	}
}

func TestGetUserID_Missing(t *testing.T) {
	if v := middleware.GetUserID(context.Background()); v != "" {
		t.Errorf("expected empty string, got %s", v)
	}
}

func TestGetUsername(t *testing.T) {
	ctx := context.WithValue(context.Background(), middleware.CtxKeyUsername, "alice")
	if v := middleware.GetUsername(ctx); v != "alice" {
		t.Errorf("GetUsername = %s, want alice", v)
	}
}

func TestGetUsername_Missing(t *testing.T) {
	if v := middleware.GetUsername(context.Background()); v != "" {
		t.Errorf("expected empty string, got %s", v)
	}
}

func TestGetJTI(t *testing.T) {
	ctx := context.WithValue(context.Background(), middleware.CtxKeyJTI, "jti-123")
	if v := middleware.GetJTI(ctx); v != "jti-123" {
		t.Errorf("GetJTI = %s, want jti-123", v)
	}
}

func TestGetJTI_Missing(t *testing.T) {
	if v := middleware.GetJTI(context.Background()); v != "" {
		t.Errorf("expected empty string, got %s", v)
	}
}

func TestGetRequestID(t *testing.T) {
	ctx := context.WithValue(context.Background(), middleware.CtxKeyRequestID, "req-1")
	if v := middleware.GetRequestID(ctx); v != "req-1" {
		t.Errorf("GetRequestID = %s, want req-1", v)
	}
}

func TestGetRequestID_Missing(t *testing.T) {
	if v := middleware.GetRequestID(context.Background()); v != "" {
		t.Errorf("expected empty string, got %s", v)
	}
}

func TestGetTokenExpiry(t *testing.T) {
	exp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := context.WithValue(context.Background(), middleware.CtxKeyTokenExp, exp)
	if v := middleware.GetTokenExpiry(ctx); !v.Equal(exp) {
		t.Errorf("GetTokenExpiry = %v, want %v", v, exp)
	}
}

func TestGetTokenExpiry_Missing(t *testing.T) {
	if v := middleware.GetTokenExpiry(context.Background()); !v.IsZero() {
		t.Errorf("expected zero time, got %v", v)
	}
}

func TestGetTokenExpiry_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), middleware.CtxKeyTokenExp, "not-a-time")
	if v := middleware.GetTokenExpiry(ctx); !v.IsZero() {
		t.Errorf("expected zero time for wrong type, got %v", v)
	}
}
