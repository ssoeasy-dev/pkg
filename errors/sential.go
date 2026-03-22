package errors

import (
	"context"
	"errors"
)

var (
	ErrCreationFailed = errors.New("creation failed")
	ErrUpdateFailed   = errors.New("update failed")
	ErrDeleteFailed   = errors.New("delete failed")
	ErrGetFailed      = errors.New("get failed")
)

var (
	ErrNotFound       = errors.New("not found")
	ErrAlreadyExists  = errors.New("already exists")
	ErrForeignKey     = errors.New("foreign key violation")
	ErrCheckViolation = errors.New("check constraint violation")
)

var (
	ErrNotNullViolation  = errors.New("not null violation")
	ErrDeadlock          = errors.New("deadlock detected")
	ErrLockTimeout       = errors.New("lock timeout")
	ErrResourceExhausted = errors.New("resource exhausted")
	ErrPermissionDenied  = errors.New("permission denied")
	ErrUnavailable       = errors.New("database unavailable")
	ErrInvalidArgument   = errors.New("invalid argument")
)

var (
	ErrTxBegin    = errors.New("failed to begin transaction")
	ErrTxCommit   = errors.New("failed to commit transaction")
	ErrTxRollback = errors.New("failed to rollback transaction")
)

var (
	ErrUnknown          = errors.New("unknown error")
	ErrCanceled         = context.Canceled
	ErrDeadlineExceeded = context.DeadlineExceeded
	ErrUnauthenticated  = errors.New("Unauthenticated")
)

var ErrValidation = errors.New("validation error")
