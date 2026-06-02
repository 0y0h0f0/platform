package handler

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"task-platform/internal/gateway/middleware"
	"task-platform/pkg/xerr"
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

// Write captures successful handler output while still streaming it to Gin.
func (w *bodyCaptureWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func checkIdempotencyKey(ctx context.Context, rdb *redis.Client, key string) bool {
	ok, err := rdb.SetNX(ctx, idempotencyPrefix+key, idempotencyPending, idempotencyTTL).Result()
	if err != nil {
		log.Printf("WARN: idempotency key check failed, proceeding without guarantee: %v", err)
		return false
	}
	return !ok
}

func getIdempotencyResult(ctx context.Context, rdb *redis.Client, key string) string {
	val, err := rdb.Get(ctx, idempotencyPrefix+key).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		log.Printf("WARN: idempotency result retrieval failed: %v", err)
	}
	return val
}

func storeIdempotencyResult(ctx context.Context, rdb *redis.Client, key, body string) {
	if err := rdb.Set(ctx, idempotencyPrefix+key, body, idempotencyTTL).Err(); err != nil {
		log.Printf("WARN: idempotency result storage failed: %v", err)
	}
}

func clearIdempotencyKey(ctx context.Context, rdb *redis.Client, key string) {
	if err := rdb.Del(ctx, idempotencyPrefix+key).Err(); err != nil {
		log.Printf("WARN: idempotency key cleanup failed: %v", err)
	}
}

// SetupIdempotency implements per-user write idempotency for handlers. It
// returns shouldReturn when a cached/pending response has already been sent.
func SetupIdempotency(c *gin.Context, rdb *redis.Client) (shouldReturn bool, cleanup func()) {
	key := c.GetHeader("Idempotency-Key")
	if key == "" || rdb == nil {
		return false, func() {}
	}

	// Scope client-provided keys to the authenticated user so two users can reuse
	// the same UUID without sharing cached responses.
	userID := middleware.GetUserID(c.Request.Context())
	if userID != "" {
		key = userID + ":" + key
	}

	if checkIdempotencyKey(c.Request.Context(), rdb, key) {
		cached := getIdempotencyResult(c.Request.Context(), rdb, key)
		if cached == idempotencyPending {
			c.JSON(http.StatusConflict, &xerr.HTTPResponse{
				Code:      xerr.CodeAborted,
				Message:   "request is still processing, retry later",
				RequestID: middleware.GetRequestID(c.Request.Context()),
			})
			return true, func() {}
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
		// Failed writes clear the pending key so callers can retry with the same key.
		if c.Writer.Status() >= 400 {
			clearIdempotencyKey(c.Request.Context(), rdb, key)
		} else {
			storeIdempotencyResult(c.Request.Context(), rdb, key, capture.body.String())
		}
	}
	return false, cleanup
}
