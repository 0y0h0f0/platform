package xgrpc

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryServerMetricsInterceptor_Success(t *testing.T) {
	interceptor := UnaryServerMetricsInterceptor()

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
	resp, err := interceptor(context.Background(), "request", info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "ok" {
		t.Fatalf("unexpected response: %v", resp)
	}
}

func TestUnaryServerMetricsInterceptor_Error(t *testing.T) {
	interceptor := UnaryServerMetricsInterceptor()

	testErr := status.Error(codes.Internal, "internal error")
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, testErr
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
	resp, err := interceptor(context.Background(), "request", info, handler)
	if err != testErr {
		t.Fatalf("expected testErr, got: %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil response, got: %v", resp)
	}
}

func TestUnaryClientMetricsInterceptor_Success(t *testing.T) {
	interceptor := UnaryClientMetricsInterceptor()

	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return nil
	}

	err := interceptor(context.Background(), "/test.Service/Method", "req", "reply", nil, invoker)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnaryClientMetricsInterceptor_Error(t *testing.T) {
	interceptor := UnaryClientMetricsInterceptor()

	testErr := errors.New("connection error")
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return testErr
	}

	err := interceptor(context.Background(), "/test.Service/Method", "req", "reply", nil, invoker)
	if err != testErr {
		t.Fatalf("expected testErr, got: %v", err)
	}
}
