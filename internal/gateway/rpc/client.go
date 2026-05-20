package rpc

import (
	"context"
	"fmt"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	taskv1 "task-platform/gen/go/task/v1"
	userv1 "task-platform/gen/go/user/v1"
	"task-platform/internal/gateway/middleware"
	"task-platform/pkg/xerr"
	"task-platform/pkg/xgrpc"
)

type Clients struct {
	UserClient userv1.UserServiceClient
	TaskClient taskv1.TaskServiceClient
	userConn   *grpc.ClientConn
	taskConn   *grpc.ClientConn
}

func NewClients(_ context.Context, userServiceAddr, taskServiceAddr, internalToken string) (*Clients, error) {
	userConn, err := dial(userServiceAddr, internalToken)
	if err != nil {
		return nil, xerr.NewError(xerr.CodeInternal, fmt.Sprintf("dial user-service: %v", err))
	}

	taskConn, err := dial(taskServiceAddr, internalToken)
	if err != nil {
		_ = userConn.Close()
		return nil, xerr.NewError(xerr.CodeInternal, fmt.Sprintf("dial task-service: %v", err))
	}

	return &Clients{
		UserClient: userv1.NewUserServiceClient(userConn),
		TaskClient: taskv1.NewTaskServiceClient(taskConn),
		userConn:   userConn,
		taskConn:   taskConn,
	}, nil
}

func dial(addr, internalToken string) (*grpc.ClientConn, error) {
	metadataInterceptor := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
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

	return grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithChainUnaryInterceptor(
			xgrpc.UnaryClientMetricsInterceptor(),
			metadataInterceptor,
		),
	)
}

func (c *Clients) Close() error {
	if err := c.userConn.Close(); err != nil {
		_ = c.taskConn.Close()
		return err
	}
	return c.taskConn.Close()
}
