package middleware

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"

	"task-platform/pkg/xratelimit"
)

var (
	rateLimiterErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gateway_rate_limiter_errors_total",
			Help: "Total rate limiter Redis errors that caused fail-open.",
		},
	)
	rateLimitAllowed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_rate_limit_allowed_total",
			Help: "Total requests allowed by the gateway rate limiter.",
		},
		[]string{"scope"},
	)
	rateLimitRejected = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_rate_limit_rejected_total",
			Help: "Total requests rejected by the gateway rate limiter.",
		},
		[]string{"scope"},
	)
	rateLimiterMetricsOnce sync.Once
)

func registerRateLimiterMetrics() {
	rateLimiterMetricsOnce.Do(func() {
		prometheus.MustRegister(rateLimiterErrors)
		prometheus.MustRegister(rateLimitAllowed)
		prometheus.MustRegister(rateLimitRejected)
	})
}

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
		log.Printf("WARN: invalid value for %s=%q, using default %d: %v", key, s, defaultVal, err)
		return defaultVal
	}
	return v
}

var publicPaths = map[string]bool{
	"/api/v1/auth/register": true,
	"/api/v1/auth/login":    true,
}

func RateLimitByIP(rdb *redis.Client) gin.HandlerFunc {
	registerRateLimiterMetrics()
	tb := xratelimit.New(rdb)
	return func(c *gin.Context) {
		if rdb == nil {
			c.Next()
			return
		}

		ip := c.ClientIP()
		rate := ipRate
		burst := ipBurst
		scope := "ip"
		key := "ratelimit:ip:" + ip
		if publicPaths[strings.TrimSuffix(c.Request.URL.Path, "/")] {
			rate = authRate
			burst = authBurst
			scope = "auth"
			key = "ratelimit:ip:auth:" + ip
		}

		allowed, err := tb.Allow(c.Request.Context(), key, rate, burst)
		if err != nil {
			rateLimiterErrors.Inc()
			c.Next()
			return
		}
		if !allowed {
			rateLimitRejected.WithLabelValues(scope).Inc()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":       "RESOURCE_EXHAUSTED",
				"message":    "rate limit exceeded",
				"request_id": GetRequestID(c.Request.Context()),
			})
			return
		}
		rateLimitAllowed.WithLabelValues(scope).Inc()
		c.Next()
	}
}

func RateLimitByUser(rdb *redis.Client) gin.HandlerFunc {
	registerRateLimiterMetrics()
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
			rateLimiterErrors.Inc()
			c.Next()
			return
		}
		if !allowed {
			rateLimitRejected.WithLabelValues("user").Inc()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":       "RESOURCE_EXHAUSTED",
				"message":    "rate limit exceeded",
				"request_id": GetRequestID(c.Request.Context()),
			})
			return
		}
		rateLimitAllowed.WithLabelValues("user").Inc()
		c.Next()
	}
}
