package xgrpc

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	grpcServerRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_server_requests_total",
			Help: "Total number of gRPC requests handled by the server.",
		},
		[]string{"grpc_method", "grpc_code"},
	)
	grpcServerRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_server_request_duration_seconds",
			Help:    "gRPC server request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"grpc_method", "grpc_code"},
	)
	grpcServerRequestsInflight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "grpc_server_requests_inflight",
			Help: "Current number of inflight gRPC requests.",
		},
	)

	grpcClientRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_client_requests_total",
			Help: "Total number of gRPC client requests.",
		},
		[]string{"grpc_method", "grpc_code"},
	)
	grpcClientRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_client_request_duration_seconds",
			Help:    "gRPC client request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"grpc_method", "grpc_code"},
	)

	grpcMetricsOnce sync.Once
)

func registerGrpcMetrics() {
	grpcMetricsOnce.Do(func() {
		prometheus.MustRegister(grpcServerRequestsTotal)
		prometheus.MustRegister(grpcServerRequestDuration)
		prometheus.MustRegister(grpcServerRequestsInflight)
		prometheus.MustRegister(grpcClientRequestsTotal)
		prometheus.MustRegister(grpcClientRequestDuration)
	})
}

func UnaryServerMetricsInterceptor() grpc.UnaryServerInterceptor {
	registerGrpcMetrics()
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		grpcServerRequestsInflight.Inc()

		resp, err := handler(ctx, req)

		grpcServerRequestsInflight.Dec()
		code := status.Code(err).String()
		grpcServerRequestsTotal.WithLabelValues(info.FullMethod, code).Inc()
		grpcServerRequestDuration.WithLabelValues(info.FullMethod, code).Observe(time.Since(start).Seconds())

		return resp, err
	}
}

func UnaryClientMetricsInterceptor() grpc.UnaryClientInterceptor {
	registerGrpcMetrics()
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := time.Now()

		err := invoker(ctx, method, req, reply, cc, opts...)

		code := status.Code(err).String()
		grpcClientRequestsTotal.WithLabelValues(method, code).Inc()
		grpcClientRequestDuration.WithLabelValues(method, code).Observe(time.Since(start).Seconds())

		return err
	}
}
