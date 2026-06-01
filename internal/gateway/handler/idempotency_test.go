package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func TestCheckIdempotencyKey_FirstRequest(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	isDup := checkIdempotencyKey(context.Background(), rdb, "my-key")
	if isDup {
		t.Fatal("first request should not be duplicate")
	}
}

func TestCheckIdempotencyKey_DuplicateRequest(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	ctx := context.Background()

	isDup := checkIdempotencyKey(ctx, rdb, "dup-key")
	if isDup {
		t.Fatal("first request should not be duplicate")
	}

	isDup = checkIdempotencyKey(ctx, rdb, "dup-key")
	if !isDup {
		t.Fatal("duplicate request should be detected")
	}
}

func TestCheckIdempotencyKey_DifferentKeys(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	ctx := context.Background()

	checkIdempotencyKey(ctx, rdb, "key-A")
	isDup := checkIdempotencyKey(ctx, rdb, "key-B")
	if isDup {
		t.Fatal("different key should not be duplicate")
	}
}

func TestClearIdempotencyKey(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	ctx := context.Background()

	checkIdempotencyKey(ctx, rdb, "clear-me")
	isDup := checkIdempotencyKey(ctx, rdb, "clear-me")
	if !isDup {
		t.Fatal("should be duplicate before clearing")
	}

	clearIdempotencyKey(ctx, rdb, "clear-me")

	isDup = checkIdempotencyKey(ctx, rdb, "clear-me")
	if isDup {
		t.Fatal("should be new request after clearing")
	}
}

func TestStoreAndGetIdempotencyResult(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	ctx := context.Background()

	checkIdempotencyKey(ctx, rdb, "store-test")

	// Initially "pending"
	cached := getIdempotencyResult(ctx, rdb, "store-test")
	if cached != "pending" {
		t.Fatalf("expected 'pending', got '%s'", cached)
	}

	// Store real response
	storeIdempotencyResult(ctx, rdb, "store-test", `{"code":"OK","message":"ok","request_id":"req-1","data":{"id":"1"}}`)

	// Now get the real response
	cached = getIdempotencyResult(ctx, rdb, "store-test")
	if cached == "pending" {
		t.Fatal("expected real response, got pending")
	}
	var body map[string]interface{}
	if err := json.Unmarshal([]byte(cached), &body); err != nil {
		t.Fatalf("stored response should be valid JSON: %v", err)
	}
	if body["request_id"] != "req-1" {
		t.Fatalf("expected request_id 'req-1', got '%v'", body["request_id"])
	}
}

func TestSetupIdempotency_FirstRequest(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Request.Header.Set("Idempotency-Key", "setup-test-1")

	shouldReturn, cleanup := SetupIdempotency(c, rdb)
	defer cleanup()
	if shouldReturn {
		t.Fatal("first request should not short-circuit")
	}

	// Simulate successful handler via Gin's writer (which is now bodyCaptureWriter)
	c.Writer.WriteHeader(http.StatusCreated)
	_, _ = c.Writer.Write([]byte(`{"code":"OK","message":"ok","request_id":"req-123","data":{"id":"1"}}`))

	// Run cleanup (which should store the real response)
	cleanup()

	// Second request with same key should return stored response
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c2.Request.Header.Set("Idempotency-Key", "setup-test-1")

	shouldReturn, cleanup2 := SetupIdempotency(c2, rdb)
	defer cleanup2()
	if !shouldReturn {
		t.Fatal("duplicate request should short-circuit")
	}

	var body map[string]interface{}
	_ = json.Unmarshal(w2.Body.Bytes(), &body)
	if body["request_id"] != "req-123" {
		t.Fatalf("expected request_id 'req-123', got '%v'", body["request_id"])
	}
	if body["data"] == nil {
		t.Fatal("expected data in cached response")
	}
}

func TestSetupIdempotency_PendingReturnsConflict(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Request.Header.Set("Idempotency-Key", "pending-test")

	shouldReturn, cleanup := SetupIdempotency(c, rdb)
	if shouldReturn {
		t.Fatal("first request should not short-circuit")
	}

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c2.Request.Header.Set("Idempotency-Key", "pending-test")

	shouldReturn, cleanup2 := SetupIdempotency(c2, rdb)
	defer cleanup2()
	if !shouldReturn {
		t.Fatal("pending duplicate request should short-circuit")
	}
	if w2.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", w2.Code, http.StatusConflict)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w2.Body.Bytes(), &body); err != nil {
		t.Fatalf("response should be valid JSON: %v", err)
	}
	if body["code"] != "ABORTED" {
		t.Fatalf("code = %v, want ABORTED", body["code"])
	}

	c.Writer.WriteHeader(http.StatusBadRequest)
	cleanup()
}

func TestSetupIdempotency_ErrorResponseFreesKey(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Request.Header.Set("Idempotency-Key", "error-test")

	shouldReturn, cleanup := SetupIdempotency(c, rdb)
	defer cleanup()
	if shouldReturn {
		t.Fatal("first request should not short-circuit")
	}

	// Simulate error handler via Gin's writer
	c.Writer.WriteHeader(http.StatusBadRequest)
	cleanup()

	// Next request with same key should not be treated as duplicate
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c2.Request.Header.Set("Idempotency-Key", "error-test")

	shouldReturn, _ = SetupIdempotency(c2, rdb)
	if shouldReturn {
		t.Fatal("key should have been freed after error response")
	}
}

func TestSetupIdempotency_NoHeader(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	shouldReturn, cleanup := SetupIdempotency(c, rdb)
	defer cleanup()
	if shouldReturn {
		t.Fatal("should not short-circuit when no Idempotency-Key header")
	}
}

func TestSetupIdempotency_NilClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Request.Header.Set("Idempotency-Key", "nil-test")

	shouldReturn, cleanup := SetupIdempotency(c, nil)
	defer cleanup()
	if shouldReturn {
		t.Fatal("should not short-circuit when rdb is nil")
	}
}
