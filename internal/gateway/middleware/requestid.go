package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const headerRequestID = "X-Request-ID"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(headerRequestID)
		if reqID == "" {
			reqID = uuid.NewString()
		}
		c.Header(headerRequestID, reqID)
		c.Set(string(CtxKeyRequestID), reqID)

		ctx := context.WithValue(c.Request.Context(), CtxKeyRequestID, reqID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
