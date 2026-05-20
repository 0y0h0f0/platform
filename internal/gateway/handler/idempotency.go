package handler

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"task-platform/internal/gateway/middleware"
)

const (
	idempotencyTTL     = 24 * time.Hour
	idempotencyPrefix  = "idempotency:"
	idempotencyPending = "pending"
)

type bodyCaptureWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyCaptureWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func checkIdempotencyKey(ctx context.Context, rdb *redis.Client, key string) bool {
	ok, err := rdb.SetNX(ctx, idempotencyPrefix+key, idempotencyPending, idempotencyTTL).Result()
	if err != nil {
		return false
	}
	return !ok
}

func getIdempotencyResult(ctx context.Context, rdb *redis.Client, key string) string {
	val, _ := rdb.Get(ctx, idempotencyPrefix+key).Result()
	return val
}

func storeIdempotencyResult(ctx context.Context, rdb *redis.Client, key, body string) {
	rdb.Set(ctx, idempotencyPrefix+key, body, idempotencyTTL)
}

func clearIdempotencyKey(ctx context.Context, rdb *redis.Client, key string) {
	rdb.Del(ctx, idempotencyPrefix+key)
}

func SetupIdempotency(c *gin.Context, rdb *redis.Client) (shouldReturn bool, cleanup func()) {
	key := c.GetHeader("Idempotency-Key")
	if key == "" || rdb == nil {
		return false, func() {}
	}

	if checkIdempotencyKey(c.Request.Context(), rdb, key) {
		cached := getIdempotencyResult(c.Request.Context(), rdb, key)
		if cached == idempotencyPending {
			requestID := middleware.GetRequestID(c.Request.Context())
			cached = fmt.Sprintf(`{"code":"OK","message":"request already processed","request_id":"%s"}`, requestID)
		}
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.String(http.StatusOK, cached)
		return true, func() {}
	}

	capture := &bodyCaptureWriter{
		ResponseWriter: c.Writer,
		body:           &bytes.Buffer{},
	}
	c.Writer = capture

	cleanup = func() {
		if c.Writer.Status() >= 400 {
			clearIdempotencyKey(c.Request.Context(), rdb, key)
		} else {
			storeIdempotencyResult(c.Request.Context(), rdb, key, capture.body.String())
		}
	}
	return false, cleanup
}
