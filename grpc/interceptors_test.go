package grpc_test

import (
	"context"
	"testing"

	"github.com/ssoeasy-dev/pkg/errors"
	pkggrpc "github.com/ssoeasy-dev/pkg/grpc"
	"github.com/ssoeasy-dev/pkg/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

func testLogger() logger.Logger {
	return logger.NewLogger(logger.EnvironmentTest, "test")
}

// captureCtxHandler stores the context passed by the interceptor.
func captureCtxHandler(out *context.Context) grpc.UnaryHandler {
	return func(ctx context.Context, _ any) (any, error) {
		*out = ctx
		return nil, nil
	}
}

func okHandler() grpc.UnaryHandler {
	return func(ctx context.Context, _ any) (any, error) { return "ok", nil }
}

func errHandler(err error) grpc.UnaryHandler {
	return func(ctx context.Context, _ any) (any, error) { return nil, err }
}

func panicHandler(val any) grpc.UnaryHandler {
	return func(ctx context.Context, _ any) (any, error) { panic(val) }
}

// ctxWithMetadata builds an incoming-metadata context with a single key/value.
func ctxWithMetadata(key, value string) context.Context {
	md := metadata.Pairs(key, value)
	return metadata.NewIncomingContext(context.Background(), md)
}

// mockStream is a minimal grpc.ServerStream for stream interceptor tests.
type mockStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockStream) Context() context.Context { return m.ctx }

// ─── TraceIDInterceptor ───────────────────────────────────────────────────────

func TestTraceIDInterceptor_GeneratesIDWhenAbsent(t *testing.T) {
	interceptor := pkggrpc.TraceIDInterceptor()
	var capturedCtx context.Context

	_, err := interceptor(context.Background(), nil, nil, captureCtxHandler(&capturedCtx))
	require.NoError(t, err)

	val, ok := capturedCtx.Value(logger.TraceIDKey).(string)
	require.True(t, ok)
	assert.NotEmpty(t, val)
}

func TestTraceIDInterceptor_PropagatesExistingID(t *testing.T) {
	interceptor := pkggrpc.TraceIDInterceptor()
	ctx := ctxWithMetadata(pkggrpc.HeaderTraceID, "trace-abc")
	var capturedCtx context.Context

	_, err := interceptor(ctx, nil, nil, captureCtxHandler(&capturedCtx))
	require.NoError(t, err)

	assert.Equal(t, "trace-abc", capturedCtx.Value(logger.TraceIDKey))
}

func TestTraceIDInterceptor_GeneratesNewIDWhenHeaderEmpty(t *testing.T) {
	interceptor := pkggrpc.TraceIDInterceptor()
	// Metadata key present but value is ""
	ctx := ctxWithMetadata(pkggrpc.HeaderTraceID, "")
	var capturedCtx context.Context

	_, err := interceptor(ctx, nil, nil, captureCtxHandler(&capturedCtx))
	require.NoError(t, err)

	val, ok := capturedCtx.Value(logger.TraceIDKey).(string)
	require.True(t, ok)
	assert.NotEmpty(t, val)
}

func TestTraceIDInterceptor_EachCallGeneratesUniqueID(t *testing.T) {
	interceptor := pkggrpc.TraceIDInterceptor()
	ids := make(map[string]struct{}, 10)

	for range 10 {
		var capturedCtx context.Context
		_, err := interceptor(context.Background(), nil, nil, captureCtxHandler(&capturedCtx))
		require.NoError(t, err)
		ids[capturedCtx.Value(logger.TraceIDKey).(string)] = struct{}{}
	}

	assert.Len(t, ids, 10, "each call should produce a unique trace ID")
}

// ─── RequestIDInterceptor ─────────────────────────────────────────────────────

func TestRequestIDInterceptor_GeneratesIDWhenAbsent(t *testing.T) {
	interceptor := pkggrpc.RequestIDInterceptor()
	var capturedCtx context.Context

	_, err := interceptor(context.Background(), nil, nil, captureCtxHandler(&capturedCtx))
	require.NoError(t, err)

	val, ok := capturedCtx.Value(logger.RequestIDKey).(string)
	require.True(t, ok)
	assert.NotEmpty(t, val)
}

func TestRequestIDInterceptor_PropagatesExistingID(t *testing.T) {
	interceptor := pkggrpc.RequestIDInterceptor()
	ctx := ctxWithMetadata(pkggrpc.HeaderRequestID, "req-123")
	var capturedCtx context.Context

	_, err := interceptor(ctx, nil, nil, captureCtxHandler(&capturedCtx))
	require.NoError(t, err)

	assert.Equal(t, "req-123", capturedCtx.Value(logger.RequestIDKey))
}

// Trace and Request IDs are stored under different keys — they must not
// overwrite each other even when both interceptors run.
func TestTraceAndRequestIDInterceptors_StoreSeparateKeys(t *testing.T) {
	trace := pkggrpc.TraceIDInterceptor()
	request := pkggrpc.RequestIDInterceptor()

	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs(
			pkggrpc.HeaderTraceID, "t-1",
			pkggrpc.HeaderRequestID, "r-1",
		),
	)

	var afterRequest context.Context
	_, err := trace(ctx, nil, nil, func(ctx context.Context, req any) (any, error) {
		return request(ctx, req, nil, captureCtxHandler(&afterRequest))
	})

	require.NoError(t, err)

	assert.NotEqual(t,
		afterRequest.Value(logger.TraceIDKey),
		afterRequest.Value(logger.RequestIDKey),
		"trace and request IDs must be stored under different context keys",
	)
}

// ─── LoggingInterceptor ───────────────────────────────────────────────────────

func TestLoggingInterceptor_SuccessfulCall_ReturnsResponse(t *testing.T) {
	interceptor := pkggrpc.LoggingInterceptor(testLogger())
	info := &grpc.UnaryServerInfo{FullMethod: "/pkg.Test/Method"}

	resp, err := interceptor(context.Background(), nil, info, func(ctx context.Context, _ any) (any, error) {
		return "hello", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "hello", resp)
}

func TestLoggingInterceptor_FailedCall_PropagatesError(t *testing.T) {
	interceptor := pkggrpc.LoggingInterceptor(testLogger())
	info := &grpc.UnaryServerInfo{FullMethod: "/pkg.Test/Method"}
	handlerErr := status.Error(codes.NotFound, "not found")

	_, err := interceptor(context.Background(), nil, info, errHandler(handlerErr))
	assert.ErrorIs(t, err, handlerErr)
}

func TestLoggingInterceptor_NonGRPCError_PropagatesError(t *testing.T) {
	interceptor := pkggrpc.LoggingInterceptor(testLogger())
	info := &grpc.UnaryServerInfo{FullMethod: "/pkg.Test/Method"}
	handlerErr := errors.New(errors.ErrUnknown, "raw error")

	_, err := interceptor(context.Background(), nil, info, errHandler(handlerErr))
	assert.ErrorIs(t, err, handlerErr)
}

// ─── RecoveryInterceptor ──────────────────────────────────────────────────────

func TestRecoveryInterceptor_NoPanic_PassesThrough(t *testing.T) {
	interceptor := pkggrpc.RecoveryInterceptor(testLogger())
	info := &grpc.UnaryServerInfo{FullMethod: "/pkg.Test/Method"}

	resp, err := interceptor(context.Background(), nil, info, okHandler())
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestRecoveryInterceptor_PanicWithString_ReturnsInternal(t *testing.T) {
	interceptor := pkggrpc.RecoveryInterceptor(testLogger())
	info := &grpc.UnaryServerInfo{FullMethod: "/pkg.Test/Method"}

	_, err := interceptor(context.Background(), nil, info, panicHandler("something crashed"))
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestRecoveryInterceptor_PanicWithError_ReturnsInternal(t *testing.T) {
	interceptor := pkggrpc.RecoveryInterceptor(testLogger())
	info := &grpc.UnaryServerInfo{FullMethod: "/pkg.Test/Method"}

	_, err := interceptor(context.Background(), nil, info, panicHandler(errors.New(errors.ErrUnknown, "oops")))
	require.Error(t, err)

	st, _ := status.FromError(err)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestRecoveryInterceptor_HandlerError_IsNotRecovered(t *testing.T) {
	// A non-panic error must not be swallowed by the recovery interceptor.
	interceptor := pkggrpc.RecoveryInterceptor(testLogger())
	info := &grpc.UnaryServerInfo{FullMethod: "/pkg.Test/Method"}
	handlerErr := status.Error(codes.InvalidArgument, "bad input")

	_, err := interceptor(context.Background(), nil, info, errHandler(handlerErr))
	assert.ErrorIs(t, err, handlerErr)
}

// ─── StreamRecoveryInterceptor ────────────────────────────────────────────────

func TestStreamRecoveryInterceptor_NoPanic_PassesThrough(t *testing.T) {
	interceptor := pkggrpc.StreamRecoveryInterceptor(testLogger())
	stream := &mockStream{ctx: context.Background()}
	info := &grpc.StreamServerInfo{FullMethod: "/pkg.Test/Stream"}

	err := interceptor(nil, stream, info, func(_ any, _ grpc.ServerStream) error {
		return nil
	})
	require.NoError(t, err)
}

func TestStreamRecoveryInterceptor_HandlerError_PassesThrough(t *testing.T) {
	interceptor := pkggrpc.StreamRecoveryInterceptor(testLogger())
	stream := &mockStream{ctx: context.Background()}
	info := &grpc.StreamServerInfo{FullMethod: "/pkg.Test/Stream"}
	handlerErr := status.Error(codes.Unavailable, "unavailable")

	err := interceptor(nil, stream, info, func(_ any, _ grpc.ServerStream) error {
		return handlerErr
	})
	assert.ErrorIs(t, err, handlerErr)
}

// BUG FIX: previously the stream recovery interceptor used an unnamed return,
// so after recover() the function silently returned nil. This test verifies
// that a panic is now correctly translated to codes.Internal.
func TestStreamRecoveryInterceptor_Panic_ReturnsInternal(t *testing.T) {
	interceptor := pkggrpc.StreamRecoveryInterceptor(testLogger())
	stream := &mockStream{ctx: context.Background()}
	info := &grpc.StreamServerInfo{FullMethod: "/pkg.Test/Stream"}

	err := interceptor(nil, stream, info, func(_ any, _ grpc.ServerStream) error {
		panic("stream handler exploded")
	})
	require.Error(t, err, "expected non-nil error after panic — not nil (the old bug)")

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestStreamRecoveryInterceptor_PanicWithError_ReturnsInternal(t *testing.T) {
	interceptor := pkggrpc.StreamRecoveryInterceptor(testLogger())
	stream := &mockStream{ctx: context.Background()}
	info := &grpc.StreamServerInfo{FullMethod: "/pkg.Test/Stream"}

	err := interceptor(nil, stream, info, func(_ any, _ grpc.ServerStream) error {
		panic(errors.New(errors.ErrUnknown, "wrapped panic"))
	})

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}
