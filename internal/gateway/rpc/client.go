package rpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	userv1 "task-platform/gen/go/user/v1"
	"task-platform/internal/gateway/middleware"
)

type Clients struct {
	UserClient userv1.UserServiceClient
	conn       *grpc.ClientConn
}

func NewClients(_ context.Context, userServiceAddr, internalToken string) (*Clients, error) {
	interceptor := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		md, _ := metadata.FromOutgoingContext(ctx)
		if md == nil {
			md = metadata.New(nil)
		}
		md.Set("x-internal-token", internalToken)

		if reqID := middleware.GetRequestID(ctx); reqID != "" {
			md.Set("x-request-id", reqID)
		}
		if userID := middleware.GetUserID(ctx); userID != "" {
			md.Set("x-user-id", userID)
			md.Set("x-username", middleware.GetUsername(ctx))
		}

		return invoker(metadata.NewOutgoingContext(ctx, md), method, req, reply, cc, opts...)
	}

	conn, err := grpc.NewClient(userServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptor),
	)
	if err != nil {
		return nil, fmt.Errorf("dial user-service: %w", err)
	}

	return &Clients{
		UserClient: userv1.NewUserServiceClient(conn),
		conn:       conn,
	}, nil
}

func (c *Clients) Close() error {
	return c.conn.Close()
}
