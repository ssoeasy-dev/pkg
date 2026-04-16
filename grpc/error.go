package grpc

import (
	"context"

	"github.com/ssoeasy-dev/pkg/errors"
	"github.com/ssoeasy-dev/pkg/logger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func errorHandler(ctx context.Context, log logger.Logger, err error) error {
	if err == nil {
		return nil
	}
	
	log.Error(ctx, errors.FullError(err), nil)
	
	if IsGRPCStatus(err) {
		return err
	}
	code := mapKindToGRPCCode(errors.Kind(err))
	
	return status.Error(code, err.Error())
}

// IsGRPCStatus проверяет, является ли ошибка gRPC-статусом.
func IsGRPCStatus(err error) bool {
	_, ok := status.FromError(err)
	return ok
}

// mapKindToGRPCCode отображает вид ошибки в gRPC-код.
// Если вид не распознан, возвращается codes.Internal.
func mapKindToGRPCCode(kind error) codes.Code {
	switch kind {
	case errors.ErrCanceled:
		return codes.Canceled
	case errors.ErrUnknown:
		return codes.Unknown
	case errors.ErrInvalidArgument, errors.ErrNotAcceptable, errors.ErrUnprocessableEntity, errors.ErrURITooLong, errors.ErrUnsupportedMediaType, errors.ErrRequestHeaderFieldsTooLarge:
		return codes.InvalidArgument
	case errors.ErrDeadlineExceeded, errors.ErrRequestTimeout, errors.ErrGatewayTimeout:
		return codes.DeadlineExceeded
	case errors.ErrNotFound, errors.ErrGone:
		return codes.NotFound
	case errors.ErrAlreadyExists:
		return codes.AlreadyExists
	case errors.ErrPermissionDenied, errors.ErrPaymentRequired, errors.ErrUnavailableForLegalReasons:
		return codes.PermissionDenied
	case errors.ErrResourceExhausted, errors.ErrPayloadTooLarge, errors.ErrTooManyRequests, errors.ErrInsufficientStorage:
		return codes.ResourceExhausted
	case errors.ErrFailedPrecondition, errors.ErrExpectationFailed, errors.ErrLocked, errors.ErrFailedDependency, errors.ErrPreconditionRequired:
		return codes.FailedPrecondition
	case errors.ErrAborted, errors.ErrConflict:
		return codes.Aborted
	case errors.ErrUnimplemented, errors.ErrMethodNotAllowed, errors.ErrUpgradeRequired, errors.ErrHTTPVersionNotSupported, errors.ErrNotExtended:
		return codes.Unimplemented
	case errors.ErrInternal, errors.ErrVariantAlsoNegotiates, errors.ErrLoopDetected:
		return codes.Internal
	case errors.ErrUnavailable, errors.ErrTooEarly, errors.ErrBadGateway:
		return codes.Unavailable
	case errors.ErrDataLoss:
		return codes.DataLoss
	case errors.ErrUnauthenticated, errors.ErrNetworkAuthenticationRequired:
		return codes.Unauthenticated
	case errors.ErrRangeNotSatisfiable:
		return codes.OutOfRange
	default:
		return codes.Internal
	}
}


