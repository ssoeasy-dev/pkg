package db

import (
	"context"
	"regexp"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/ssoeasy-dev/pkg/errors"
	"gorm.io/gorm"
)

// NewError преобразует низкоуровневую ошибку БД (GORM или pgx) в доменную ошибку пакета errors.
// entity — имя сущности для контекста сообщения (например, "user").
func NewError(err error, entity string) error {
	if err == nil {
		return nil
	}

	// Контекстные ошибки (отмена/дедлайн) имеют приоритет
	if errors.Is(err, context.Canceled) {
		return errors.Newf(errors.ErrCanceled, "operation canceled: %s", entity)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.Newf(errors.ErrDeadlineExceeded, "deadline exceeded: %s", entity)
	}

	// GORM ошибки
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.NewWrapf(errors.ErrNotFound, err, "%s not found", entity)
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return errors.NewWrapf(errors.ErrAlreadyExists, err, "%s already exists", entity)
	}
	if errors.Is(err, gorm.ErrForeignKeyViolated) {
		return errors.NewWrapf(errors.ErrFailedPrecondition, err, "foreign key violation for %s", entity)
	}
	if errors.Is(err, gorm.ErrCheckConstraintViolated) {
		return errors.NewWrapf(errors.ErrFailedPrecondition, err, "check constraint violation for %s", entity)
	}
	if errors.Is(err, gorm.ErrInvalidData) {
		return errors.NewWrapf(errors.ErrInvalidArgument, err, "invalid data for %s", entity)
	}

	// Ошибки pgx (PostgreSQL)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			field, _, ok := parseUniqueViolationDetail(pgErr.Detail)
			if ok {
				return errors.NewWrapf(errors.ErrAlreadyExists, err, "%s with %s already exists", entity, field)
			}
			return errors.NewWrapf(errors.ErrAlreadyExists, err, "%s already exists", entity)
		case "23503": // foreign_key_violation
			return errors.NewWrapf(errors.ErrFailedPrecondition, err, "foreign key violation for %s", entity)
		case "23502": // not_null_violation
			return errors.NewWrapf(errors.ErrInvalidArgument, err, "not null violation for %s", entity)
		case "23514": // check_violation
			return errors.NewWrapf(errors.ErrFailedPrecondition, err, "check violation for %s", entity)
		case "40P01": // deadlock_detected
			return errors.NewWrapf(errors.ErrAborted, err, "deadlock detected for %s", entity)
		case "55P03": // lock_not_available
			return errors.NewWrapf(errors.ErrInternal, err, "lock timeout for %s", entity)
		case "53100": // disk_full
			return errors.NewWrapf(errors.ErrResourceExhausted, err, "resource exhausted for %s", entity)
		case "53200": // out_of_memory
			return errors.NewWrapf(errors.ErrResourceExhausted, err, "resource exhausted for %s", entity)
		case "57P01", "57P02": // admin_shutdown / crash_shutdown
			return errors.NewWrapf(errors.ErrUnavailable, err, "database unavailable for %s", entity)
		case "08000", "08001", "08003", "08004", "08006": // connection exceptions
			return errors.NewWrapf(errors.ErrUnavailable, err, "database connection error for %s", entity)
		case "42501": // insufficient_privilege
			return errors.NewWrapf(errors.ErrPermissionDenied, err, "permission denied for %s", entity)
		case "22001": // string too long
			return errors.NewWrapf(errors.ErrInvalidArgument, err, "string too long for %s", entity)
		case "22003", "22008", "22012", "22015", "2201E", "2202H", "2202J", "2202L",
			"2202M", "2202N", "2202P", "2202Q", "2202R", "2202S", "2202T", "2202U",
			"2202V", "2202W", "2202X", "2202Y", "2202Z":
			// Различные ошибки неверных данных
			return errors.NewWrapf(errors.ErrInvalidArgument, err, "invalid data for %s", entity)
		}
	}

	// Не распознанная ошибка — маскируем, чтобы не раскрывать детали реализации
	return errors.NewWrapf(errors.ErrInternal, err, "operation failed for %s", entity)
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
