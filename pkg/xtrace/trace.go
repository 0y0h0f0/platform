package xtrace

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
)

var (
	globalLogger *zap.Logger

	tracingStatus = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "tracing_enabled",
			Help: "Whether OpenTelemetry tracing is enabled (1) or disabled (0).",
		},
	)
	tracingMetricsOnce sync.Once
)

func SetLogger(l *zap.Logger) {
	globalLogger = l
}

func otlpDialTimeout() time.Duration {
	s := os.Getenv("OTEL_EXPORTER_OTLP_TIMEOUT")
	if s == "" {
		return 5 * time.Second
	}
	d, err := strconv.Atoi(s)
	if err != nil || d <= 0 {
		return 5 * time.Second
	}
	return time.Duration(d) * time.Second
}

func Init(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "127.0.0.1:4317"
	}

	connCtx, cancel := context.WithTimeout(ctx, otlpDialTimeout())
	defer cancel()

	exporter, err := otlptracegrpc.New(connCtx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		tracingMetricsOnce.Do(func() { prometheus.MustRegister(tracingStatus) })
		tracingStatus.Set(0)
		if globalLogger != nil {
			globalLogger.Warn("failed to create OTLP trace exporter, tracing disabled",
				zap.Error(err))
		}
		return func(context.Context) error { return nil }, nil
	}

	tracingMetricsOnce.Do(func() { prometheus.MustRegister(tracingStatus) })
	tracingStatus.Set(1)

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(serviceName)),
		resource.WithFromEnv(),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	shutdown := func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	}

	return shutdown, nil
}
