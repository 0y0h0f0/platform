package xredis

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

var (
	redisCommandDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "redis_command_duration_seconds",
			Help:    "Redis command duration in seconds.",
			Buckets: []float64{.0001, .0005, .001, .0025, .005, .01, .025, .05, .1},
		},
		[]string{"command"},
	)
	redisCommandErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redis_command_errors_total",
			Help: "Total number of Redis command errors.",
		},
		[]string{"command"},
	)
	redisCacheHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "redis_cache_hits_total",
			Help: "Total number of Redis cache hits.",
		},
	)
	redisCacheMisses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "redis_cache_misses_total",
			Help: "Total number of Redis cache misses.",
		},
	)
	redisMetricsOnce sync.Once
)

func registerRedisMetrics() {
	redisMetricsOnce.Do(func() {
		prometheus.MustRegister(redisCommandDuration)
		prometheus.MustRegister(redisCommandErrors)
		prometheus.MustRegister(redisCacheHits)
		prometheus.MustRegister(redisCacheMisses)
	})
}

type metricsHook struct{}

func (h *metricsHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

func (h *metricsHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmd)
		duration := time.Since(start).Seconds()

		redisCommandDuration.WithLabelValues(cmd.Name()).Observe(duration)
		if err != nil && err != redis.Nil {
			redisCommandErrors.WithLabelValues(cmd.Name()).Inc()
		}

		return err
	}
}

func (h *metricsHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmds)
		duration := time.Since(start).Seconds()

		for _, cmd := range cmds {
			redisCommandDuration.WithLabelValues(cmd.Name()).Observe(duration)
			if cmd.Err() != nil && cmd.Err() != redis.Nil {
				redisCommandErrors.WithLabelValues(cmd.Name()).Inc()
			}
		}

		return err
	}
}

func AddMetricsHook(rdb *redis.Client) {
	if rdb != nil {
		registerRedisMetrics()
		rdb.AddHook(&metricsHook{})
	}
}

func IncrCacheHit()  { redisCacheHits.Inc() }
func IncrCacheMiss() { redisCacheMisses.Inc() }
