package xgrpc

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"google.golang.org/grpc"
)

const (
	defaultClientTimeout = 2 * time.Second
	defaultServerTimeout = 3 * time.Second
)

func ClientTimeoutFromEnv() time.Duration {
	return envDurationSeconds("GRPC_CLIENT_TIMEOUT_SECONDS", defaultClientTimeout)
}

func ServerTimeoutFromEnv() time.Duration {
	return envDurationSeconds("GRPC_SERVER_TIMEOUT_SECONDS", defaultServerTimeout)
}

func UnaryClientTimeoutInterceptor(timeout time.Duration) grpc.UnaryClientInterceptor {
	if timeout <= 0 {
		timeout = defaultClientTimeout
	}

	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if _, ok := ctx.Deadline(); ok {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		deadlineCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return invoker(deadlineCtx, method, req, reply, cc, opts...)
	}
}

func UnaryServerTimeoutInterceptor(timeout time.Duration) grpc.UnaryServerInterceptor {
	if timeout <= 0 {
		timeout = defaultServerTimeout
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if _, ok := ctx.Deadline(); ok {
			return handler(ctx, req)
		}

		deadlineCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return handler(deadlineCtx, req)
	}
}

func envDurationSeconds(key string, defaultVal time.Duration) time.Duration {
	s := os.Getenv(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		log.Printf("WARN: invalid value for %s=%q, using default %s", key, s, defaultVal)
		return defaultVal
	}
	return time.Duration(v) * time.Second
}
