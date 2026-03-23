package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/ssoeasy-dev/pkg/logger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server wraps a gRPC server with structured logging and graceful shutdown.
// It is pre-configured with trace-ID propagation, request-ID propagation,
// structured logging, and panic recovery on both unary and stream RPCs.
type Server struct {
	server *grpc.Server
	addr   string
	log    logger.Logger
}

// Interceptors holds optional application-level interceptors that are appended
// to the built-in chain created by NewServer.
//
// Unary interceptors run after TraceID, RequestID, and Logging interceptors
// but after RecoveryInterceptor — panics in custom interceptors are therefore
// NOT caught by the built-in recovery. If you need recovery to wrap your
// interceptors, add your own RecoveryInterceptor inside Unary.
//
// Passing nil is valid and equivalent to providing an empty Interceptors.
type Interceptors struct {
	// Unary interceptors appended after the built-in unary chain.
	// Order: TraceID → RequestID → Logging → Unary[0] → Unary[1] → ... → Recovery
	Unary []grpc.UnaryServerInterceptor
	// Stream interceptors appended after StreamRecoveryInterceptor.
	// Order: Stream[0] → Stream[1] → ... → StreamRecovery
	Stream []grpc.StreamServerInterceptor
}

// NewServer creates a Server that listens on addr (e.g. "0.0.0.0:50051").
//
// The built-in unary interceptors are applied in this order:
//  1. TraceIDInterceptor   — extracts or generates x-trace-id
//  2. RequestIDInterceptor — extracts or generates x-request-id
//  3. LoggingInterceptor   — logs method, duration, and gRPC status code
//  4. RecoveryInterceptor  — catches panics and returns codes.Internal
//
// Any interceptors provided via intercepts.Unary are appended after step 4.
// StreamRecoveryInterceptor is applied to all streaming RPCs, followed by
// any interceptors in intercepts.Stream.
//
// intercepts may be nil.
func NewServer(addr string, log logger.Logger, intercepts *Interceptors) *Server {
	// Unary цепочка: встроенные → пользовательские → Recovery
	unaryIntercepts := []grpc.UnaryServerInterceptor{
		TraceIDInterceptor(),
		RequestIDInterceptor(),
		LoggingInterceptor(log),
	}
	if intercepts != nil {
		unaryIntercepts = append(unaryIntercepts, intercepts.Unary...)
	}
	// Recovery должен быть последним, чтобы ловить паники в пользовательских интерсепторах
	unaryIntercepts = append(unaryIntercepts, RecoveryInterceptor(log))

	// Stream цепочка: встроенные → пользовательские → Recovery
	streamIntercepts := []grpc.StreamServerInterceptor{
		StreamTraceIDInterceptor(),
		StreamRequestIDInterceptor(),
		StreamLoggingInterceptor(log),
	}
	if intercepts != nil {
		streamIntercepts = append(streamIntercepts, intercepts.Stream...)
	}
	streamIntercepts = append(streamIntercepts, StreamRecoveryInterceptor(log))

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(unaryIntercepts...),
		grpc.ChainStreamInterceptor(streamIntercepts...),
	)

	return &Server{
		server: srv,
		addr:   addr,
		log:    log,
	}
}

// GetGRPCServer returns the underlying *grpc.Server for service registration.
//
//	pb.RegisterMyServiceServer(srv.GetGRPCServer(), &myHandler{})
func (s *Server) GetGRPCServer() *grpc.Server {
	return s.server
}

// RegisterReflection enables gRPC server reflection so that tools like
// grpcurl and Evans can discover available services at runtime.
func (s *Server) RegisterReflection() {
	reflection.Register(s.server)
	s.log.Info(context.Background(), "gRPC reflection registered", nil)
}

// Start binds to the configured address and begins accepting connections.
// It blocks until the server is stopped; launch it in a goroutine and call
// Stop to initiate shutdown.
//
//	go func() {
//	    if err := srv.Start(); err != nil {
//	        log.Fatal(err)
//	    }
//	}()
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("grpc: listen %s: %w", s.addr, err)
	}

	s.log.Info(context.Background(), "gRPC server started", map[string]any{
		"addr": s.addr,
	})

	if err := s.server.Serve(lis); err != nil {
		return fmt.Errorf("grpc: serve: %w", err)
	}
	return nil
}

// Stop gracefully stops the server. It blocks until all in-flight RPCs have
// completed. Start will return nil once Stop returns.
func (s *Server) Stop() {
	s.log.Info(context.Background(), "gRPC server stopping", nil)
	s.server.GracefulStop()
}
