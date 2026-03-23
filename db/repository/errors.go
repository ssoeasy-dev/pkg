package repository

import (
	"context"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/ssoeasy-dev/pkg/errors"
	"gorm.io/gorm"
)

// repoError — минимальная обёртка: sentinel для errors.Is, причина для errors.Unwrap.
//
// errors.Is(err, ErrNotFound) → true  (через sentinel)
// errors.Unwrap(err)          → оригинальная pgx/gorm ошибка (для логирования)
type repoError struct {
	msg      string // человекочитаемое сообщение с контекстом
	sentinel error  // ErrNotFound, ErrAlreadyExists и т.д.
	cause    error  // оригинальная ошибка GORM/pgx
}

func (e *repoError) Error() string        { return e.msg }
func (e *repoError) Is(target error) bool { return target == e.sentinel }
func (e *repoError) Unwrap() error        { return e.cause }

// NewRepositoryError классифицирует ошибку GORM/pgx и возвращает repoError.
// Вызывающий код использует errors.Is для ветвления, errors.Unwrap для логирования.
func NewRepositoryError(err error, entity string, def error) error {
    if err == nil {
        return nil
    }

    // Проверка контекстных ошибок (отмена/дедлайн) — они могут быть обёрнуты в GORM
    if errors.Is(err, context.Canceled) {
        return &repoError{
            msg:      "operation canceled",
            sentinel: errors.ErrCanceled,
            cause:    err,
        }
    }
    if errors.Is(err, context.DeadlineExceeded) {
        return &repoError{
            msg:      "deadline exceeded",
            sentinel: errors.ErrDeadlineExceeded,
            cause:    err,
        }
    }

    // GORM ошибки
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return &repoError{
            msg:      fmt.Sprintf("%s: %s", errors.ErrNotFound, entity),
            sentinel: errors.ErrNotFound,
            cause:    err,
        }
    }
    if errors.Is(err, gorm.ErrDuplicatedKey) {
        return &repoError{
            msg:      fmt.Sprintf("%s: %s", errors.ErrAlreadyExists, entity),
            sentinel: errors.ErrAlreadyExists,
            cause:    err,
        }
    }
    if errors.Is(err, gorm.ErrForeignKeyViolated) {
        return &repoError{
            msg:      fmt.Sprintf("%s: %s", errors.ErrForeignKey, entity),
            sentinel: errors.ErrForeignKey,
            cause:    err,
        }
    }
    if errors.Is(err, gorm.ErrCheckConstraintViolated) {
        return &repoError{
            msg:      fmt.Sprintf("%s: check constraint violation for %s", errors.ErrCheckViolation, entity),
            sentinel: errors.ErrCheckViolation,
            cause:    err,
        }
    }
    if errors.Is(err, gorm.ErrInvalidData) {
        return &repoError{
            msg:      fmt.Sprintf("%s: invalid data for %s", errors.ErrInvalidArgument, entity),
            sentinel: errors.ErrInvalidArgument,
            cause:    err,
        }
    }

    // Ошибки pgx (PostgreSQL)
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        switch pgErr.Code {
        case "23505": // unique_violation
            field, value, ok := parseUniqueViolationDetail(pgErr.Detail)
            if ok {
                return &repoError{
                    msg:      fmt.Sprintf("%s: %s with %s=%v", errors.ErrAlreadyExists, entity, field, value),
                    sentinel: errors.ErrAlreadyExists,
                    cause:    err,
                }
            }
            return &repoError{
                msg:      fmt.Sprintf("%s: %s", errors.ErrAlreadyExists, entity),
                sentinel: errors.ErrAlreadyExists,
                cause:    err,
            }
        case "23503": // foreign_key_violation
            return &repoError{
                msg:      fmt.Sprintf("%s: %s", errors.ErrForeignKey, entity),
                sentinel: errors.ErrForeignKey,
                cause:    err,
            }
        case "23502": // not_null_violation
            return &repoError{
                msg:      fmt.Sprintf("%s: not null violation for %s", errors.ErrNotNullViolation, entity),
                sentinel: errors.ErrNotNullViolation,
                cause:    err,
            }
        case "23514": // check_violation
            return &repoError{
                msg:      fmt.Sprintf("%s: check violation for %s", errors.ErrCheckViolation, entity),
                sentinel: errors.ErrCheckViolation,
                cause:    err,
            }
        case "40P01": // deadlock_detected
            return &repoError{
                msg:      fmt.Sprintf("deadlock detected: %s", entity),
                sentinel: errors.ErrDeadlock, // нужно добавить
                cause:    err,
            }
        case "55P03": // lock_not_available
            return &repoError{
                msg:      fmt.Sprintf("lock timeout: %s", entity),
                sentinel: errors.ErrLockTimeout, // нужно добавить
                cause:    err,
            }
        case "53100": // disk_full
            return &repoError{
                msg:      fmt.Sprintf("disk full: %s", entity),
                sentinel: errors.ErrResourceExhausted,
                cause:    err,
            }
        case "53200": // out_of_memory
            return &repoError{
                msg:      fmt.Sprintf("out of memory: %s", entity),
                sentinel: errors.ErrResourceExhausted,
                cause:    err,
            }
        case "57P01": // admin_shutdown
            return &repoError{
                msg:      fmt.Sprintf("database admin shutdown: %s", entity),
                sentinel: errors.ErrUnavailable,
                cause:    err,
            }
        case "57P02": // crash_shutdown
            return &repoError{
                msg:      fmt.Sprintf("database crash: %s", entity),
                sentinel: errors.ErrUnavailable,
                cause:    err,
            }
        case "08000", "08001", "08003", "08004", "08006": // connection exceptions
            return &repoError{
                msg:      fmt.Sprintf("database connection error: %s", entity),
                sentinel: errors.ErrUnavailable,
                cause:    err,
            }
        case "42501": // insufficient_privilege
            return &repoError{
                msg:      fmt.Sprintf("permission denied: %s", entity),
                sentinel: errors.ErrPermissionDenied,
                cause:    err,
            }
        case "22001": // string too long
            return &repoError{
                msg:      fmt.Sprintf("%s: string too long for %s", errors.ErrInvalidArgument, entity),
                sentinel: errors.ErrInvalidArgument,
                cause:    err,
            }
        // Другие коды 22xxx (numeric value out of range, etc.) можно объединить:
        case "22003", "22008", "22012", "22015", "2201E", "2202H", "2202J", "2202L", "2202M", "2202N", "2202P", "2202Q", "2202R", "2202S", "2202T", "2202U", "2202V", "2202W", "2202X", "2202Y", "2202Z":
            return &repoError{
                msg:      fmt.Sprintf("%s: invalid data for %s", errors.ErrInvalidArgument, entity),
                sentinel: errors.ErrInvalidArgument,
                cause:    err,
            }
        default:
            // fallback
        }
    }

    // Если не распознали, используем def sentinel
    return &repoError{
        msg:      fmt.Sprintf("%s: %s: %v", def, entity, err),
        sentinel: def,
        cause:    err,
    }
}

// parseUniqueViolationDetail извлекает поле и значение из Detail pgx-ошибки.
// Формат: "Key (email)=(test@example.com) already exists."
func parseUniqueViolationDetail(detail string) (field, value string, ok bool) {
	re := regexp.MustCompile(`Key \(([^)]+)\)=\(([^)]+)\)`)
	m := re.FindStringSubmatch(detail)
	if len(m) == 3 {
		return m[1], m[2], true
	}
	return "", "", false
}
