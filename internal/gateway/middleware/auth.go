package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"task-platform/pkg/xerr"
	"task-platform/pkg/xjwt"
)

// Auth validates bearer tokens, checks logout blacklists and propagates identity
// into both Gin and request contexts for handlers and RPC clients.
func Auth(jwtManager *xjwt.Manager, rdb *redis.Client, publicPaths []string) gin.HandlerFunc {
	publicSet := make(map[string]bool, len(publicPaths))
	for _, p := range publicPaths {
		publicSet[p] = true
	}

	return func(c *gin.Context) {
		if publicSet[c.Request.URL.Path] {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			resp := xerr.NewError(xerr.CodeUnauthenticated, "missing or malformed authorization header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, xerr.ToHTTPResponse(resp, GetRequestID(c.Request.Context())))
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := jwtManager.Validate(token)
		if err != nil {
			resp := xerr.NewError(xerr.CodeUnauthenticated, "invalid or expired token")
			c.AbortWithStatusJSON(http.StatusUnauthorized, xerr.ToHTTPResponse(resp, GetRequestID(c.Request.Context())))
			return
		}

		if rdb != nil {
			// Logout stores the JWT ID in Redis; a hit means the token was revoked
			// before its natural expiration.
			blacklistKey := "blacklist:" + claims.ID
			exists, err := rdb.Exists(c.Request.Context(), blacklistKey).Result()
			if err != nil {
				resp := xerr.NewError(xerr.CodeInternal, "check token blacklist failed")
				c.AbortWithStatusJSON(http.StatusInternalServerError, xerr.ToHTTPResponse(resp, GetRequestID(c.Request.Context())))
				return
			}
			if exists > 0 {
				resp := xerr.NewError(xerr.CodeUnauthenticated, "token has been revoked")
				c.AbortWithStatusJSON(http.StatusUnauthorized, xerr.ToHTTPResponse(resp, GetRequestID(c.Request.Context())))
				return
			}
		}

		c.Set(string(CtxKeyUserID), claims.Subject)
		c.Set(string(CtxKeyUsername), claims.Username)
		c.Set(string(CtxKeyJTI), claims.ID)

		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, CtxKeyUserID, claims.Subject)
		ctx = context.WithValue(ctx, CtxKeyUsername, claims.Username)
		ctx = context.WithValue(ctx, CtxKeyJTI, claims.ID)
		ctx = context.WithValue(ctx, CtxKeyTokenExp, claims.ExpiresAt.Time)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
