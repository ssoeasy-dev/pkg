package grpc

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/ssoeasy-dev/pkg/logger"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Header names used to propagate trace and request identifiers.
const (
	HeaderTraceID   = "x-trace-id"
	HeaderRequestID = "x-request-id"
)

// ErrorHandler преобразует ошибку приложения в gRPC-ошибку (обычно *status.Status).
// Если возвращается nil, считается, что ошибка обработана и клиенту вернётся OK.
// Если handler вернёт ту же ошибку или другую, она будет возвращена клиенту.
type ErrorHandler func(error) error
type UnaryServerInterceptor = grpc.UnaryServerInterceptor
type StreamServerInterceptor = grpc.StreamServerInterceptor

// ----- Unary ----------------------------------------------------------------

// TraceIDInterceptor reads x-trace-id from incoming gRPC metadata and stores
// it in the context via logger.TraceIDKey. A new UUID is generated if the
// header is absent or empty.
func TraceIDInterceptor() grpc.UnaryServerInterceptor {
	return metadataInterceptor(HeaderTraceID, logger.TraceIDKey)
}

// RequestIDInterceptor reads x-request-id from incoming gRPC metadata and
// stores it in the context via logger.RequestIDKey. A new UUID is generated
// if the header is absent or empty.
func RequestIDInterceptor() grpc.UnaryServerInterceptor {
	return metadataInterceptor(HeaderRequestID, logger.RequestIDKey)
}

// LoggingInterceptor logs each unary RPC with the full method name, duration,
// and gRPC status code. Failed calls are logged at Error level; successful
// calls at Info level.
func LoggingInterceptor(log logger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
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
func RecoveryInterceptor(log logger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
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

// ErrorHandlerInterceptor оборачивает unary RPC и применяет переданный обработчик ошибок.
func ErrorHandlerInterceptor(log logger.Logger, handler ErrorHandler) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
		resp, err := next(ctx, req)
		if err != nil {
			if _, ok := status.FromError(err); ok {
				return resp, err
			}
			if handler != nil {
				err = handler(err)
			}
			if _, ok := status.FromError(err); ok {
				return resp, err
			}
			err = errorHandler(ctx, log, err)
		}
		return resp, err
	}
}

// ----- Stream ----------------------------------------------------------------

// streamWithContext позволяет подменить контекст в ServerStream.
type streamWithContext struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *streamWithContext) Context() context.Context {
	return s.ctx
}

// StreamTraceIDInterceptor добавляет x-trace-id в контекст stream.
func StreamTraceIDInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		traceID := extractMetadata(ss.Context(), HeaderTraceID)
		ctx := context.WithValue(ss.Context(), logger.TraceIDKey, traceID)
		wrapped := &streamWithContext{ServerStream: ss, ctx: ctx}
		return handler(srv, wrapped)
	}
}

// StreamRequestIDInterceptor добавляет x-request-id в контекст stream.
func StreamRequestIDInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		reqID := extractMetadata(ss.Context(), HeaderRequestID)
		ctx := context.WithValue(ss.Context(), logger.RequestIDKey, reqID)
		wrapped := &streamWithContext{ServerStream: ss, ctx: ctx}
		return handler(srv, wrapped)
	}
}

// StreamLoggingInterceptor логирует начало и конец stream-метода,
// а также ошибку, если она возникла.
func StreamLoggingInterceptor(log logger.Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		err := handler(srv, ss)

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
			log.Error(ss.Context(), "gRPC stream failed", fields)
		} else {
			log.Info(ss.Context(), "gRPC stream completed", fields)
		}

		return err
	}
}

// StreamRecoveryInterceptor catches panics in stream handlers, logs the value
// and stack trace, and returns codes.Internal to the caller.
func StreamRecoveryInterceptor(log logger.Logger) grpc.StreamServerInterceptor {
	return func(
		srv any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
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

// StreamErrorHandlerInterceptor оборачивает stream RPC и применяет переданный обработчик ошибок.
func StreamErrorHandlerInterceptor(log logger.Logger, handler ErrorHandler) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, next grpc.StreamHandler) error {
		err := next(srv, ss)
		if err != nil {
			if _, ok := status.FromError(err); ok {
				return err
			}
			if handler != nil {
				err = handler(err)
			}
			if _, ok := status.FromError(err); ok {
				return err
			}
			err = errorHandler(ss.Context(), log, err)
			if handler != nil {
				err = handler(err)
			}
		}
		return err
	}
}

// ----- Helpers ----------------------------------------------------------------

// metadataInterceptor is the shared implementation for TraceIDInterceptor and
// RequestIDInterceptor. It extracts a single header value from incoming
// metadata and injects it into the context under ctxKey.
func metadataInterceptor(header string, ctxKey any) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
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
