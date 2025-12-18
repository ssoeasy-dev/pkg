package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/ssoeasy-dev/pkg/logger"

	goGrpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	server *goGrpc.Server
	addr   string
	log    *logger.Logger
}

func NewServer(addr string, log *logger.Logger) *Server {
	server := goGrpc.NewServer(
		goGrpc.ChainUnaryInterceptor(
			traceIDInterceptor(),
			requestIDInterceptor(),
			loggingInterceptor(log),
		),
	)

	return &Server{
		server: server,
		addr:   addr,
		log:    log,
	}
}

func (s *Server) RegisterReflection() {
	reflection.Register(s.server)
	s.log.Info(context.Background(), "gRPC server reflection registered", nil)
}

func (s *Server) GetGRPCServer() *goGrpc.Server {
	return s.server
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.addr, err)
	}

	s.log.Info(context.Background(), "gRPC server started", map[string]any{
		"addr": s.addr,
	})

	if err := s.server.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

func (s *Server) Stop() {
	s.log.Info(context.Background(), "Stopping gRPC server", nil)
	s.server.GracefulStop()
}
