package xhttp

import (
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	bootCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "task_platform_boot_total",
			Help: "Total number of service boots.",
		},
		[]string{"service"},
	)
	metricsOnce sync.Once
)

func NewEngine(service string, ready *atomic.Bool) *gin.Engine {
	metricsOnce.Do(func() {
		prometheus.MustRegister(bootCounter)
	})
	bootCounter.WithLabelValues(service).Inc()

	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": service,
			"status":  "bootstrapped",
		})
	})
	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": service,
			"status":  "ok",
		})
	})
	engine.GET("/readyz", func(c *gin.Context) {
		if ready.Load() {
			c.JSON(http.StatusOK, gin.H{
				"service": service,
				"status":  "ready",
			})
			return
		}

		c.JSON(http.StatusServiceUnavailable, gin.H{
			"service": service,
			"status":  "not_ready",
		})
	})
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	return engine
}
