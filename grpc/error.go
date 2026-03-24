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

	logError := errors.Unwrap(err)
	grpcErrStr := err.Error()
	var code codes.Code

	switch {
	// repository errors 10
	// 1 --- CANCELLED ---
	case errors.Is(err, errors.ErrCanceled):
		code = codes.Canceled
	// 2 --- UNKNOWN ---
	case errors.Is(err, errors.ErrUnknown):
		code = codes.Unknown
	// 3 --- INVALID_ARGUMENT ---
	case errors.Is(err, errors.ErrCheckViolation),
	errors.Is(err, errors.ErrNotNullViolation),
	errors.Is(err, errors.ErrInvalidArgument):
		code = codes.InvalidArgument
	case errors.Is(err, errors.ErrValidation): // TODO: build validation error from github.com/go-playground/validator
		code = codes.InvalidArgument
	// 4 --- DEADLINE_EXCEEDED ---
	case errors.Is(err, errors.ErrDeadlineExceeded):
		code = codes.DeadlineExceeded
	// 5 --- NOT_FOUND ---
	case errors.Is(err, errors.ErrNotFound):
		code = codes.NotFound
	// 6 --- ALREADY_EXISTS ---
	case errors.Is(err, errors.ErrAlreadyExists):
		code = codes.AlreadyExists
	// 7 --- PERMISSION_DENIED ---
	case errors.Is(err, errors.ErrPermissionDenied):
		code = codes.PermissionDenied
	// 8 --- RESOURCE_EXHAUSTED ---
	case errors.Is(err, errors.ErrResourceExhausted):
		code = codes.ResourceExhausted
	// 9 --- FAILED_PRECONDITION ---
	case errors.Is(err, errors.ErrForeignKey),
	errors.Is(err, errors.ErrLockTimeout):
		code = codes.FailedPrecondition
	// 10 --- ABORTED ---
	case errors.Is(err, errors.ErrDeadlock):
		code = codes.Aborted
	// 11 --- OUT_OF_RANGE ---
	// 12 --- UNIMPLEMENTED ---
	// 13 --- INTERNAL ---
	case errors.Is(err, errors.ErrCreationFailed),
	errors.Is(err, errors.ErrUpdateFailed),
	errors.Is(err, errors.ErrDeleteFailed),
	errors.Is(err, errors.ErrGetFailed),
	errors.Is(err, errors.ErrTxBegin),
	errors.Is(err, errors.ErrTxCommit),
	errors.Is(err, errors.ErrTxRollback):
		code = codes.Internal
	// 14 --- UNAVAILABLE ---
	case errors.Is(err, errors.ErrUnavailable):
		code = codes.Unavailable
	// 15 --- DATA_LOSS ---
	// 16 --- UNAUTHENTICATED ---
	case errors.Is(err, errors.ErrUnauthenticated):
		code = codes.Unauthenticated
	default:
		code = codes.Internal
	}

	if logError != nil {
		log.Error(ctx, logError.Error(), nil)
	}

	return status.Error(code, grpcErrStr)
}
