package grpc

import (
	"context"
	"time"

	"github.com/ssoeasy-dev/pkg/logger"
	"github.com/google/uuid"
	goGrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	HeaderTraceID = "x-trace-id"
	HeaderRequestID = "x-request-id"
)

func traceIDInterceptor() goGrpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *goGrpc.UnaryServerInfo,
		handler goGrpc.UnaryHandler,
	) (interface{}, error) {
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
		req interface{},
		info *goGrpc.UnaryServerInfo,
		handler goGrpc.UnaryHandler,
	) (interface{}, error) {
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
		req interface{},
		info *goGrpc.UnaryServerInfo,
		handler goGrpc.UnaryHandler,
	) (interface{}, error) {
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
