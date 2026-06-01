package xhttp

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 10 * time.Second
	defaultWriteTimeout      = 15 * time.Second
	defaultIdleTimeout       = 60 * time.Second
)

type ServerTimeouts struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

func DefaultServerTimeouts() ServerTimeouts {
	return ServerTimeouts{
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
	}
}

func ServerTimeoutsFromSeconds(readHeader, read, write, idle int) ServerTimeouts {
	timeouts := DefaultServerTimeouts()
	if readHeader > 0 {
		timeouts.ReadHeaderTimeout = time.Duration(readHeader) * time.Second
	}
	if read > 0 {
		timeouts.ReadTimeout = time.Duration(read) * time.Second
	}
	if write > 0 {
		timeouts.WriteTimeout = time.Duration(write) * time.Second
	}
	if idle > 0 {
		timeouts.IdleTimeout = time.Duration(idle) * time.Second
	}
	return timeouts
}

func NewServer(addr string, handler http.Handler, timeouts ServerTimeouts) *http.Server {
	defaults := DefaultServerTimeouts()
	if timeouts.ReadHeaderTimeout <= 0 {
		timeouts.ReadHeaderTimeout = defaults.ReadHeaderTimeout
	}
	if timeouts.ReadTimeout <= 0 {
		timeouts.ReadTimeout = defaults.ReadTimeout
	}
	if timeouts.WriteTimeout <= 0 {
		timeouts.WriteTimeout = defaults.WriteTimeout
	}
	if timeouts.IdleTimeout <= 0 {
		timeouts.IdleTimeout = defaults.IdleTimeout
	}

	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: timeouts.ReadHeaderTimeout,
		ReadTimeout:       timeouts.ReadTimeout,
		WriteTimeout:      timeouts.WriteTimeout,
		IdleTimeout:       timeouts.IdleTimeout,
	}
}

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
