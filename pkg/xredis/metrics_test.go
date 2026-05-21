package xredis

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestRegisterRedisMetrics_Idempotent(t *testing.T) {
	registerRedisMetrics()
	registerRedisMetrics()
}

func TestAddMetricsHook_NilClient(t *testing.T) {
	AddMetricsHook(nil)
}

func TestIncrCacheHitMiss(t *testing.T) {
	IncrCacheHit()
	IncrCacheMiss()
}

func TestMetricsHook_ProcessHook_Success(t *testing.T) {
	h := &metricsHook{}
	next := func(ctx context.Context, cmd redis.Cmder) error {
		return nil
	}
	wrapped := h.ProcessHook(next)

	cmd := redis.NewStringCmd(context.Background())
	err := wrapped(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMetricsHook_ProcessHook_RedisNil(t *testing.T) {
	h := &metricsHook{}
	next := func(ctx context.Context, cmd redis.Cmder) error {
		return redis.Nil
	}
	wrapped := h.ProcessHook(next)

	cmd := redis.NewStringCmd(context.Background())
	err := wrapped(context.Background(), cmd)
	if err != redis.Nil {
		t.Fatalf("expected redis.Nil, got: %v", err)
	}
}

func TestMetricsHook_ProcessHook_Error(t *testing.T) {
	h := &metricsHook{}
	testErr := errors.New("connection refused")
	next := func(ctx context.Context, cmd redis.Cmder) error {
		return testErr
	}
	wrapped := h.ProcessHook(next)

	cmd := redis.NewStringCmd(context.Background())
	err := wrapped(context.Background(), cmd)
	if err != testErr {
		t.Fatalf("expected testErr, got: %v", err)
	}
}

func TestMetricsHook_ProcessPipelineHook_Success(t *testing.T) {
	h := &metricsHook{}
	next := func(ctx context.Context, cmds []redis.Cmder) error {
		return nil
	}
	wrapped := h.ProcessPipelineHook(next)

	cmds := []redis.Cmder{
		redis.NewStringCmd(context.Background()),
		redis.NewIntCmd(context.Background()),
	}
	err := wrapped(context.Background(), cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMetricsHook_ProcessPipelineHook_WithNilError(t *testing.T) {
	h := &metricsHook{}
	next := func(ctx context.Context, cmds []redis.Cmder) error {
		return nil
	}
	wrapped := h.ProcessPipelineHook(next)

	cmdWithNil := redis.NewStringCmd(context.Background())
	cmdWithNil.SetErr(redis.Nil)
	cmdOk := redis.NewIntCmd(context.Background())

	cmds := []redis.Cmder{cmdWithNil, cmdOk}
	err := wrapped(context.Background(), cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMetricsHook_DialHook(t *testing.T) {
	h := &metricsHook{}
	next := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return nil, nil
	}
	wrapped := h.DialHook(next)

	_, err := wrapped(context.Background(), "tcp", "127.0.0.1:6379")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
