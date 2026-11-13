// server.go will implement the official gRPC interface


package envoyrls

import (
	"fmt"
	"net"

	ratelimitv3 "github.com/envoyproxy/go-control-plane/envoy/service/ratelimit/v3"
	"google.golang.org/grpc"
)

type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener
}

func NewServer(
	address string,
	handler ratelimitv3.RateLimitServiceServer,
) (*Server, error) {
	if address == "" {
		return nil, fmt.Errorf("gRPC server address must not be empty")
	}

	if handler == nil {
		return nil, fmt.Errorf("rate-limit handler must not be nil")
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", address, err)
	}

	grpcServer := grpc.NewServer()

	ratelimitv3.RegisterRateLimitServiceServer(grpcServer, handler)

	return &Server{
		grpcServer: grpcServer,
		listener:   listener,
	}, nil
}

func (s *Server) Serve() error {
	if err := s.grpcServer.Serve(s.listener); err != nil {
		return fmt.Errorf("gRPC server stopped: %w", err)
	}

	return nil
}

func (s *Server) GracefulStop() {
	s.grpcServer.GracefulStop()
}
