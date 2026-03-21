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

// Header names used to propagate trace and request identifiers.
const (
	HeaderTraceID   = "x-trace-id"
	HeaderRequestID = "x-request-id"
)

// TraceIDInterceptor reads x-trace-id from incoming gRPC metadata and stores
// it in the context via logger.TraceIDKey. A new UUID is generated if the
// header is absent or empty.
func TraceIDInterceptor() goGrpc.UnaryServerInterceptor {
	return metadataInterceptor(HeaderTraceID, logger.TraceIDKey)
}

// RequestIDInterceptor reads x-request-id from incoming gRPC metadata and
// stores it in the context via logger.RequestIDKey. A new UUID is generated
// if the header is absent or empty.
func RequestIDInterceptor() goGrpc.UnaryServerInterceptor {
	return metadataInterceptor(HeaderRequestID, logger.RequestIDKey)
}

// metadataInterceptor is the shared implementation for TraceIDInterceptor and
// RequestIDInterceptor. It extracts a single header value from incoming
// metadata and injects it into the context under ctxKey.
func metadataInterceptor(header string, ctxKey any) goGrpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		_ *goGrpc.UnaryServerInfo,
		handler goGrpc.UnaryHandler,
	) (any, error) {
		return handler(context.WithValue(ctx, ctxKey, extractMetadata(ctx, header)), req)
	}
}

// extractMetadata reads the first value of header from incoming gRPC metadata.
// Returns a fresh UUID string when the header is absent or empty.
func extractMetadata(ctx context.Context, header string) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(header); len(values) > 0 && values[0] != "" {
			return values[0]
		}
	}
	return uuid.New().String()
}

// LoggingInterceptor logs each unary RPC with the full method name, duration,
// and gRPC status code. Failed calls are logged at Error level; successful
// calls at Info level.
func LoggingInterceptor(log *logger.Logger) goGrpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *goGrpc.UnaryServerInfo,
		handler goGrpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)

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
			"duration_ms": time.Since(start).Milliseconds(),
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

// RecoveryInterceptor catches panics in unary handlers, logs the value and
// stack trace, and returns codes.Internal to the caller.
func RecoveryInterceptor(log *logger.Logger) goGrpc.UnaryServerInterceptor {
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
					"method":  info.FullMethod,
					"stack":   string(debug.Stack()),
				})
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

// StreamRecoveryInterceptor catches panics in stream handlers, logs the value
// and stack trace, and returns codes.Internal to the caller.
//
// BUG FIX: the previous implementation used an unnamed return, so after
// recover() the function silently returned nil and the client had no indication
// that a panic occurred. Named return `err` allows the deferred function to
// set the error that is actually returned to the caller.
func StreamRecoveryInterceptor(log *logger.Logger) goGrpc.StreamServerInterceptor {
	return func(
		srv any,
		stream goGrpc.ServerStream,
		info *goGrpc.StreamServerInfo,
		handler goGrpc.StreamHandler,
	) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error(stream.Context(), "panic recovered in stream", map[string]any{
					"recover": r,
					"method":  info.FullMethod,
					"stack":   string(debug.Stack()),
				})
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(srv, stream)
	}
}
