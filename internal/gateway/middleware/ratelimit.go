package middleware

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"task-platform/pkg/xratelimit"
)

var (
	ipRate    = envInt("RATELIMIT_IP_RATE", 60)
	ipBurst   = envInt("RATELIMIT_IP_BURST", 100)
	authRate  = envInt("RATELIMIT_AUTH_RATE", 5)
	authBurst = envInt("RATELIMIT_AUTH_BURST", 10)
	userRate  = envInt("RATELIMIT_USER_RATE", 100)
	userBurst = envInt("RATELIMIT_USER_BURST", 200)
)

func envInt(key string, defaultVal int) int {
	s := os.Getenv(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

var publicPaths = map[string]bool{
	"/api/v1/auth/register": true,
	"/api/v1/auth/login":    true,
}

func RateLimitByIP(rdb *redis.Client) gin.HandlerFunc {
	tb := xratelimit.New(rdb)
	return func(c *gin.Context) {
		if rdb == nil {
			c.Next()
			return
		}

		ip := c.ClientIP()
		rate := ipRate
		burst := ipBurst
		key := "ratelimit:ip:" + ip
		if publicPaths[strings.TrimSuffix(c.Request.URL.Path, "/")] {
			rate = authRate
			burst = authBurst
			key = "ratelimit:ip:auth:" + ip
		}

		allowed, err := tb.Allow(c.Request.Context(), key, rate, burst)
		if err != nil {
			c.Next()
			return
		}
		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    "RESOURCE_EXHAUSTED",
				"message": "rate limit exceeded",
			})
			return
		}
		c.Next()
	}
}

func RateLimitByUser(rdb *redis.Client) gin.HandlerFunc {
	tb := xratelimit.New(rdb)
	return func(c *gin.Context) {
		if rdb == nil {
			c.Next()
			return
		}

		userID := GetUserID(c.Request.Context())
		if userID == "" {
			c.Next()
			return
		}

		key := "ratelimit:user:" + userID
		allowed, err := tb.Allow(c.Request.Context(), key, userRate, userBurst)
		if err != nil {
			c.Next()
			return
		}
		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    "RESOURCE_EXHAUSTED",
				"message": "rate limit exceeded",
			})
			return
		}
		c.Next()
	}
}
