package xratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"task-platform/pkg/xratelimit"
)

func TestNew_ReturnsTokenBucket(t *testing.T) {
	tb := xratelimit.New(nil)
	if tb == nil {
		t.Fatal("New with nil should still return a TokenBucket")
	}
}

func TestAllow_FirstRequest(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	tb := xratelimit.New(rdb)
	allowed, err := tb.Allow(context.Background(), "rl:test", 10, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("first request should be allowed")
	}
}

func TestAllow_ExhaustTokens(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	tb := xratelimit.New(rdb)
	ctx := context.Background()

	burst := 10
	for i := 0; i < burst; i++ {
		allowed, err := tb.Allow(ctx, "rl:burst", 5, burst)
		if err != nil {
			t.Fatalf("request %d: unexpected error: %v", i, err)
		}
		if !allowed {
			t.Fatalf("request %d should be allowed (burst=%d)", i, burst)
		}
	}

	// Next request should be rejected
	allowed, err := tb.Allow(ctx, "rl:burst", 5, burst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("request exceeding burst should be rejected")
	}
}

func TestAllow_RefillsOverTime(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	tb := xratelimit.New(rdb)
	ctx := context.Background()

	// Exhaust the bucket
	burst := 10
	for i := 0; i < burst; i++ {
		_, _ = tb.Allow(ctx, "rl:refill", 5, burst)
	}

	// Should be rejected now
	allowed, _ := tb.Allow(ctx, "rl:refill", 5, burst)
	if allowed {
		t.Fatal("should be exhausted")
	}

	// Wait for tokens to refill
	time.Sleep(1100 * time.Millisecond) // 1.1s * 5 tokens/s = 5 tokens refilled

	allowed, err := tb.Allow(ctx, "rl:refill", 5, burst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("should be allowed after refill")
	}
}

func TestAllow_SeparateKeys(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	tb := xratelimit.New(rdb)
	ctx := context.Background()

	// Exhaust key A
	for i := 0; i < 5; i++ {
		_, _ = tb.Allow(ctx, "rl:A", 1, 5)
	}
	allowedA, _ := tb.Allow(ctx, "rl:A", 1, 5)
	if allowedA {
		t.Fatal("key A should be exhausted")
	}

	// Key B should still work
	allowedB, err := tb.Allow(ctx, "rl:B", 10, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowedB {
		t.Fatal("key B should be unaffected")
	}
}
