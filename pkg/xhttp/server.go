package xhttp

import (
	"net/http"
	"os"
	"strings"
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

// ServerTimeouts groups HTTP server timeout knobs so services can override only
// the values they need.
type ServerTimeouts struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

// DefaultServerTimeouts returns production-safe defaults for HTTP servers.
func DefaultServerTimeouts() ServerTimeouts {
	return ServerTimeouts{
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
	}
}

// ServerTimeoutsFromSeconds converts positive config values into durations and
// leaves non-positive values at defaults.
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

// NewServer creates an http.Server with defaults filled for missing timeouts.
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

// NewEngine builds the base Gin engine shared by services, including health,
// readiness and Prometheus endpoints.
func NewEngine(service string, ready *atomic.Bool) *gin.Engine {
	metricsOnce.Do(func() {
		prometheus.MustRegister(bootCounter)
	})
	bootCounter.WithLabelValues(service).Inc()

	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()

	// Prevent X-Forwarded-For spoofing by trusting only internal proxies.
	// Override via TRUSTED_PROXIES env var (comma-separated IPs/CIDRs).
	engine.SetTrustedProxies(trustedProxies())

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

// trustedProxies returns the list of proxy IPs/CIDRs that Gin should trust
// when parsing X-Forwarded-For headers. Defaults to private network ranges;
// override via the TRUSTED_PROXIES env var (comma-separated).
func trustedProxies() []string {
	if v := os.Getenv("TRUSTED_PROXIES"); v != "" {
		return strings.Split(v, ",")
	}
	return []string{
		"127.0.0.1",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}
}
