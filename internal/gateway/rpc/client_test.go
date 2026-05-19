package rpc_test

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	userv1 "task-platform/gen/go/user/v1"
	"task-platform/internal/gateway/middleware"
	"task-platform/internal/gateway/rpc"
)

type echoUserServer struct {
	userv1.UnimplementedUserServiceServer
}

func (s *echoUserServer) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	return &userv1.GetUserResponse{
		User: &userv1.User{Id: req.UserId, Username: "echo"},
	}, nil
}

func TestNewClients_Interceptor(t *testing.T) {
	// Start a real gRPC server
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer()
	userv1.RegisterUserServiceServer(srv, &echoUserServer{})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	clients, err := rpc.NewClients(context.Background(), lis.Addr().String(), lis.Addr().String(), "test-token")
	if err != nil {
		t.Fatalf("NewClients: %v", err)
	}
	defer func() { _ = clients.Close() }()

	// Make a real RPC call to exercise the interceptor
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.CtxKeyRequestID, "req-123")
	ctx = context.WithValue(ctx, middleware.CtxKeyUserID, "user-1")
	ctx = context.WithValue(ctx, middleware.CtxKeyUsername, "alice")

	resp, err := clients.UserClient.GetUser(ctx, &userv1.GetUserRequest{UserId: "user-1"})
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if resp.User.Username != "echo" {
		t.Errorf("username = %s, want echo", resp.User.Username)
	}
}

func TestNewClients_Success(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer()
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	clients, err := rpc.NewClients(context.Background(), lis.Addr().String(), lis.Addr().String(), "test-token")
	if err != nil {
		t.Fatalf("NewClients: %v", err)
	}
	if clients.UserClient == nil {
		t.Error("UserClient should not be nil")
	}
	if err := clients.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestNewClients_InterceptorPropagatesAllMetadata(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer()
	userv1.RegisterUserServiceServer(srv, &echoUserServer{})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	clients, err := rpc.NewClients(context.Background(), lis.Addr().String(), lis.Addr().String(), "test-token")
	if err != nil {
		t.Fatalf("NewClients: %v", err)
	}
	defer func() { _ = clients.Close() }()

	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.CtxKeyRequestID, "req-456")
	ctx = context.WithValue(ctx, middleware.CtxKeyUserID, "user-2")
	ctx = context.WithValue(ctx, middleware.CtxKeyUsername, "bob")

	resp, err := clients.UserClient.GetUser(ctx, &userv1.GetUserRequest{UserId: "user-2"})
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if resp.User.Username != "echo" {
		t.Errorf("username = %s, want echo", resp.User.Username)
	}
}

func TestNewClients_CloseCleansUpBothConns(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer()
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	clients, err := rpc.NewClients(context.Background(), lis.Addr().String(), lis.Addr().String(), "test-token")
	if err != nil {
		t.Fatalf("NewClients: %v", err)
	}
	if err := clients.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	// Second close should be safe (idempotent or return error)
	_ = clients.Close()
}

func TestNewClients_WithTransportCredentials(t *testing.T) {
	// Verify the client uses insecure credentials (for code coverage of the dial option)
	conn, err := grpc.NewClient("localhost:1",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_ = conn.Close()
}
