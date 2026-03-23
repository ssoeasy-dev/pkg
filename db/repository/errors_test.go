package repository_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/ssoeasy-dev/pkg/db/repository"
	"github.com/ssoeasy-dev/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---- Существующие тесты (исправлены) ----

func TestNewRepositoryError_Nil(t *testing.T) {
	assert.NoError(t, repository.NewRepositoryError(nil, "user", errors.ErrGetFailed))
}

func TestNewRepositoryError_RecordNotFound(t *testing.T) {
	err := repository.NewRepositoryError(gorm.ErrRecordNotFound, "user", errors.ErrGetFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrNotFound)
	assert.Contains(t, err.Error(), "user")
}

func TestNewRepositoryError_DefaultKind(t *testing.T) {
	cause := fmt.Errorf("connection refused")
	err := repository.NewRepositoryError(cause, "order", errors.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrCreationFailed)
	assert.Contains(t, err.Error(), "order")
	assert.Contains(t, err.Error(), "connection refused")
}

func TestNewRepositoryError_Unwrap_ReturnsOriginalCause(t *testing.T) {
	cause := fmt.Errorf("disk full")
	err := repository.NewRepositoryError(cause, "item", errors.ErrGetFailed)
	assert.ErrorIs(t, errors.Unwrap(err), cause)
}

func TestNewRepositoryError_Unwrap_RecordNotFound(t *testing.T) {
	err := repository.NewRepositoryError(gorm.ErrRecordNotFound, "article", errors.ErrGetFailed)
	assert.ErrorIs(t, errors.Unwrap(err), gorm.ErrRecordNotFound)
}

func TestSentinels_ErrorsIs(t *testing.T) {
	cases := []struct {
		name     string
		sentinel error
		def      error
	}{
		{"NotFound via RecordNotFound", errors.ErrNotFound, errors.ErrGetFailed},
		{"CreationFailed", errors.ErrCreationFailed, errors.ErrCreationFailed},
		{"UpdateFailed", errors.ErrUpdateFailed, errors.ErrUpdateFailed},
		{"DeleteFailed", errors.ErrDeleteFailed, errors.ErrDeleteFailed},
		{"GetFailed", errors.ErrGetFailed, errors.ErrGetFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var source error
			if tc.sentinel == errors.ErrNotFound {
				source = gorm.ErrRecordNotFound
			} else {
				source = fmt.Errorf("some db error")
			}
			err := repository.NewRepositoryError(source, "entity", tc.def)
			assert.ErrorIs(t, err, tc.sentinel)
		})
	}
}

func TestSentinels_Distinct(t *testing.T) {
	sentinels := []error{
		errors.ErrNotFound,
		errors.ErrAlreadyExists,
		errors.ErrForeignKey,
		errors.ErrCreationFailed,
		errors.ErrUpdateFailed,
		errors.ErrDeleteFailed,
		errors.ErrGetFailed,
	}
	for i, a := range sentinels {
		for j, b := range sentinels {
			if i != j {
				assert.False(t, errors.Is(a, b), "%v should not match %v", a, b)
			}
		}
	}
}

func TestNewRepositoryError_Message_NotFound(t *testing.T) {
	err := repository.NewRepositoryError(gorm.ErrRecordNotFound, "article", errors.ErrGetFailed)
	assert.Equal(t, "not found: article", err.Error())
}

func TestNewRepositoryError_Message_DefaultWrapsOriginal(t *testing.T) {
	cause := fmt.Errorf("timeout")
	err := repository.NewRepositoryError(cause, "payment", errors.ErrCreationFailed)
	assert.Equal(t, "creation failed: payment: timeout", err.Error())
}

// ---- Новые тесты ----

// Контекстные ошибки

func TestNewRepositoryError_ContextCanceled(t *testing.T) {
	err := repository.NewRepositoryError(context.Canceled, "task", errors.ErrGetFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Contains(t, err.Error(), "operation canceled")
}

func TestNewRepositoryError_ContextDeadlineExceeded(t *testing.T) {
	err := repository.NewRepositoryError(context.DeadlineExceeded, "task", errors.ErrGetFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Contains(t, err.Error(), "deadline exceeded")
}

// GORM ошибки

func TestNewRepositoryError_GormDuplicatedKey(t *testing.T) {
	err := repository.NewRepositoryError(gorm.ErrDuplicatedKey, "user", errors.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrAlreadyExists)
	assert.Contains(t, err.Error(), "user")
}

func TestNewRepositoryError_GormForeignKeyViolated(t *testing.T) {
	err := repository.NewRepositoryError(gorm.ErrForeignKeyViolated, "order", errors.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrForeignKey)
	assert.Contains(t, err.Error(), "order")
}

func TestNewRepositoryError_GormCheckConstraintViolated(t *testing.T) {
	err := repository.NewRepositoryError(gorm.ErrCheckConstraintViolated, "user", errors.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrCheckViolation)
	assert.Contains(t, err.Error(), "user")
}

func TestNewRepositoryError_GormInvalidData(t *testing.T) {
	err := repository.NewRepositoryError(gorm.ErrInvalidData, "user", errors.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrInvalidArgument)
	assert.Contains(t, err.Error(), "user")
}

// Вспомогательная функция для pgx ошибок
func makePgError(code, detail string) *pgconn.PgError {
	return &pgconn.PgError{Code: code, Detail: detail}
}

// pgx: уникальное нарушение

func TestNewRepositoryError_PgxUniqueViolation(t *testing.T) {
	pgErr := makePgError("23505", "Key (email)=(test@example.com) already exists.")
	err := repository.NewRepositoryError(pgErr, "user", errors.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrAlreadyExists)
	assert.Contains(t, err.Error(), "user with email=test@example.com")
}

func TestNewRepositoryError_PgxUniqueViolationNoDetail(t *testing.T) {
	pgErr := makePgError("23505", "")
	err := repository.NewRepositoryError(pgErr, "user", errors.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrAlreadyExists)
	assert.Contains(t, err.Error(), "user")
	assert.NotContains(t, err.Error(), "with")
}

// pgx: внешний ключ

func TestNewRepositoryError_PgxForeignKeyViolation(t *testing.T) {
	pgErr := makePgError("23503", "")
	err := repository.NewRepositoryError(pgErr, "order", errors.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrForeignKey)
	assert.Contains(t, err.Error(), "order")
}

// pgx: not null

func TestNewRepositoryError_PgxNotNullViolation(t *testing.T) {
	pgErr := makePgError("23502", "")
	err := repository.NewRepositoryError(pgErr, "user", errors.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrNotNullViolation)
	assert.Contains(t, err.Error(), "not null violation")
}

// pgx: check violation

func TestNewRepositoryError_PgxCheckViolation(t *testing.T) {
	pgErr := makePgError("23514", "")
	err := repository.NewRepositoryError(pgErr, "user", errors.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrCheckViolation)
	assert.Contains(t, err.Error(), "check violation")
}

// pgx: deadlock

func TestNewRepositoryError_PgxDeadlock(t *testing.T) {
	pgErr := makePgError("40P01", "")
	err := repository.NewRepositoryError(pgErr, "order", errors.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrDeadlock)
	assert.Contains(t, err.Error(), "deadlock detected")
}

// pgx: lock timeout

func TestNewRepositoryError_PgxLockTimeout(t *testing.T) {
	pgErr := makePgError("55P03", "")
	err := repository.NewRepositoryError(pgErr, "order", errors.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrLockTimeout)
	assert.Contains(t, err.Error(), "lock timeout")
}

// pgx: диск полон

func TestNewRepositoryError_PgxDiskFull(t *testing.T) {
	pgErr := makePgError("53100", "")
	err := repository.NewRepositoryError(pgErr, "user", errors.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrResourceExhausted)
	assert.Contains(t, err.Error(), "disk full")
}

// pgx: нехватка памяти

func TestNewRepositoryError_PgxOutOfMemory(t *testing.T) {
	pgErr := makePgError("53200", "")
	err := repository.NewRepositoryError(pgErr, "user", errors.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrResourceExhausted)
	assert.Contains(t, err.Error(), "out of memory")
}

// pgx: административное завершение

func TestNewRepositoryError_PgxAdminShutdown(t *testing.T) {
	pgErr := makePgError("57P01", "")
	err := repository.NewRepositoryError(pgErr, "user", errors.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrUnavailable)
	assert.Contains(t, err.Error(), "database admin shutdown")
}

// pgx: краш

func TestNewRepositoryError_PgxCrashShutdown(t *testing.T) {
	pgErr := makePgError("57P02", "")
	err := repository.NewRepositoryError(pgErr, "user", errors.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrUnavailable)
	assert.Contains(t, err.Error(), "database crash")
}

// pgx: ошибки соединения

func TestNewRepositoryError_PgxConnectionException(t *testing.T) {
	codes := []string{"08000", "08001", "08003", "08004", "08006"}
	for _, code := range codes {
		pgErr := makePgError(code, "")
		err := repository.NewRepositoryError(pgErr, "user", errors.ErrCreationFailed)
		require.Error(t, err)
		assert.ErrorIs(t, err, errors.ErrUnavailable)
		assert.Contains(t, err.Error(), "database connection error")
	}
}

// pgx: недостаток прав

func TestNewRepositoryError_PgxPermissionDenied(t *testing.T) {
	pgErr := makePgError("42501", "")
	err := repository.NewRepositoryError(pgErr, "user", errors.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrPermissionDenied)
	assert.Contains(t, err.Error(), "permission denied")
}

// pgx: слишком длинная строка

func TestNewRepositoryError_PgxStringTooLong(t *testing.T) {
	pgErr := makePgError("22001", "")
	err := repository.NewRepositoryError(pgErr, "user", errors.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrInvalidArgument)
	assert.Contains(t, err.Error(), "string too long")
}

// pgx: числовые ошибки (один из кодов 22xxx)

func TestNewRepositoryError_PgxNumericOutOfRange(t *testing.T) {
	pgErr := makePgError("22003", "")
	err := repository.NewRepositoryError(pgErr, "user", errors.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrInvalidArgument)
	assert.Contains(t, err.Error(), "invalid data")
}

// pgx: неизвестный код -> default

func TestNewRepositoryError_PgxUnknownCode(t *testing.T) {
	pgErr := makePgError("99999", "")
	err := repository.NewRepositoryError(pgErr, "user", errors.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrCreationFailed)
	assert.Contains(t, err.Error(), "user")
}

// Обёрнутые ошибки

func TestNewRepositoryError_WrappedGormError(t *testing.T) {
	wrapped := fmt.Errorf("wrap: %w", gorm.ErrRecordNotFound)
	err := repository.NewRepositoryError(wrapped, "user", errors.ErrGetFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrNotFound)
}

func TestNewRepositoryError_WrappedPgxError(t *testing.T) {
	pgErr := makePgError("23505", "")
	wrapped := fmt.Errorf("wrap: %w", pgErr)
	err := repository.NewRepositoryError(wrapped, "user", errors.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrAlreadyExists)
}

func TestNewRepositoryError_WrappedContextCanceled(t *testing.T) {
	wrapped := fmt.Errorf("wrap: %w", context.Canceled)
	err := repository.NewRepositoryError(wrapped, "task", errors.ErrGetFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// Unwrap для pgx

func TestNewRepositoryError_Unwrap_PgxError(t *testing.T) {
	pgErr := makePgError("23505", "")
	err := repository.NewRepositoryError(pgErr, "user", errors.ErrCreationFailed)
	assert.Equal(t, pgErr, errors.Unwrap(err))
}
