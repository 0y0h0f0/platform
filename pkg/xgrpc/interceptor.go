package xgrpc

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// SingleValue returns the first value for a key from gRPC metadata,
// or an empty string if the key is absent.
func SingleValue(md metadata.MD, key string) string {
	vals := md.Get(key)
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

// LoggingInterceptor returns a gRPC unary interceptor that logs every request
// with method, latency, request_id, trace_id, and span_id using a Zap logger.
func LoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		requestID := ""
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			requestID = SingleValue(md, "x-request-id")
		}

		span := trace.SpanFromContext(ctx)
		sc := span.SpanContext()

		resp, err := handler(ctx, req)

		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("latency", time.Since(start)),
			zap.String("request_id", requestID),
			zap.String("trace_id", sc.TraceID().String()),
			zap.String("span_id", sc.SpanID().String()),
		}
		if err != nil {
			fields = append(fields, zap.Error(err))
			logger.Error("grpc request failed", fields...)
		} else {
			logger.Info("grpc request", fields...)
		}

		return resp, err
	}
}
