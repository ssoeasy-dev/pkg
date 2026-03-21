package repository

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// Sentinel-ошибки. Используйте errors.Is для проверки типа.
var (
	ErrNotFound       = errors.New("not found")
	ErrAlreadyExists  = errors.New("already exists")
	ErrForeignKey     = errors.New("foreign key violation")
	ErrCreationFailed = errors.New("creation failed")
	ErrUpdateFailed   = errors.New("update failed")
	ErrDeleteFailed   = errors.New("delete failed")
	ErrGetFailed      = errors.New("get failed")
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

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &repoError{
			msg:      fmt.Sprintf("%s: %s", ErrNotFound, entity),
			sentinel: ErrNotFound,
			cause:    err,
		}
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			field, value, ok := parseUniqueViolationDetail(pgErr.Detail)
			if ok {
				return &repoError{
					msg:      fmt.Sprintf("%s: %s with %s=%v", ErrAlreadyExists, entity, field, value),
					sentinel: ErrAlreadyExists,
					cause:    err,
				}
			}
			return &repoError{
				msg:      fmt.Sprintf("%s: %s", ErrAlreadyExists, entity),
				sentinel: ErrAlreadyExists,
				cause:    err,
			}
		case "23503": // foreign_key_violation
			return &repoError{
				msg:      fmt.Sprintf("%s: %s", ErrForeignKey, entity),
				sentinel: ErrForeignKey,
				cause:    err,
			}
		}
	}

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
