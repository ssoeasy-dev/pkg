package grpc

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"github.com/ssoeasy-dev/pkg/logger"
	goGrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	HeaderTraceID   = "x-trace-id"
	HeaderRequestID = "x-request-id"
)

func traceIDInterceptor() goGrpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *goGrpc.UnaryServerInfo,
		handler goGrpc.UnaryHandler,
	) (any, error) {
		traceID := ""

		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if values := md.Get(HeaderTraceID); len(values) > 0 {
				traceID = values[0]
			}
		}

		if traceID == "" {
			traceID = uuid.New().String()
		}

		ctx = context.WithValue(ctx, logger.TraceIDKey, traceID)

		return handler(ctx, req)
	}
}

func requestIDInterceptor() goGrpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *goGrpc.UnaryServerInfo,
		handler goGrpc.UnaryHandler,
	) (any, error) {
		requestID := ""

		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if values := md.Get(HeaderRequestID); len(values) > 0 {
				requestID = values[0]
			}
		}

		if requestID == "" {
			requestID = uuid.New().String()
		}

		ctx = context.WithValue(ctx, logger.RequestIDKey, requestID)

		return handler(ctx, req)
	}
}

func loggingInterceptor(log *logger.Logger) goGrpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *goGrpc.UnaryServerInfo,
		handler goGrpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		code := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				code = st.Code()
			} else {
				code = codes.Unknown
			}
		}

		fields := map[string]any{
			"method":      info.FullMethod,
			"duration_ms": duration.Milliseconds(),
			"code":        code.String(),
		}

		if err != nil {
			fields["error"] = err.Error()
			log.Error(ctx, "gRPC request failed", fields)
		} else {
			log.Info(ctx, "gRPC request completed", fields)
		}

		return resp, err
	}
}

func recoveryInterceptor(log *logger.Logger) goGrpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *goGrpc.UnaryServerInfo,
		handler goGrpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error(ctx, "panic recovered", map[string]any{
					"recover": r,
					"stack":   string(debug.Stack()),
				})

				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

func streamRecoveryInterceptor(log *logger.Logger) goGrpc.StreamServerInterceptor {
	return func(
		srv any,
		stream goGrpc.ServerStream,
		info *goGrpc.StreamServerInfo,
		handler goGrpc.StreamHandler,
	) error {
		defer func() {
			if r := recover(); r != nil {
				log.Error(stream.Context(), "panic recovered", map[string]any{
					"recover": r,
					"stack":   string(debug.Stack()),
				})
			}
		}()
		return handler(srv, stream)
	}
}
