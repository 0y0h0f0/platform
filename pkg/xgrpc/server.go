package xgrpc

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	GRPC   *grpc.Server
	Health *health.Server
}

func NewServer(enableReflection bool) *Server {
	grpcServer := grpc.NewServer()
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	if enableReflection {
		reflection.Register(grpcServer)
	}

	return &Server{
		GRPC:   grpcServer,
		Health: healthServer,
	}
}
