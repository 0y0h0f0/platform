package xredis

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestNew_Unreachable(t *testing.T) {
	_, err := New("127.0.0.1:19999", "", 0)
	if err == nil {
		t.Fatal("expected error for unreachable redis")
	}
}

func TestNew_UsesEnvOptions(t *testing.T) {
	mr := miniredis.RunT(t)
	t.Setenv("REDIS_POOL_SIZE", "17")
	t.Setenv("REDIS_MIN_IDLE_CONNS", "3")
	t.Setenv("REDIS_DIAL_TIMEOUT_SECONDS", "4")
	t.Setenv("REDIS_READ_TIMEOUT_SECONDS", "5")
	t.Setenv("REDIS_WRITE_TIMEOUT_SECONDS", "6")
	t.Setenv("REDIS_POOL_TIMEOUT_SECONDS", "7")

	rdb, err := New(mr.Addr(), "", 0)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	opts := rdb.Options()
	if opts.PoolSize != 17 {
		t.Fatalf("PoolSize = %d, want 17", opts.PoolSize)
	}
	if opts.MinIdleConns != 3 {
		t.Fatalf("MinIdleConns = %d, want 3", opts.MinIdleConns)
	}
	if opts.DialTimeout != 4*time.Second || opts.ReadTimeout != 5*time.Second || opts.WriteTimeout != 6*time.Second || opts.PoolTimeout != 7*time.Second {
		t.Fatalf("unexpected timeouts: dial=%s read=%s write=%s pool=%s", opts.DialTimeout, opts.ReadTimeout, opts.WriteTimeout, opts.PoolTimeout)
	}
}
