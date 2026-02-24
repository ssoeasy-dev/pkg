package repository

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// Sentinel-ошибки, используемые для идентификации типа проблемы через errors.Is.
var (
	ErrNotFound       = errors.New("not found")
	ErrAlreadyExists  = errors.New("already exists")
	ErrForeignKey     = errors.New("foreign key violation")
	ErrCreationFailed = errors.New("creation failed")
	ErrUpdateFailed   = errors.New("update failed")
	ErrDeleteFailed   = errors.New("delete failed")
	ErrGetFailed      = errors.New("get failed")
)

// RepositoryError — общая структура ошибки репозитория.
// Содержит контекст (сущность, поле, значение) и исходную ошибку.
// Метод Is позволяет сравнивать с sentinel-ошибками.
type RepositoryError struct {
	Kind   error  // одна из sentinel-ошибок (может быть nil)
	Entity string // имя сущности, с которой работали (например, "user")
	Field  string // поле, вызвавшее конфликт (для AlreadyExists)
	Value  any    // значение, вызвавшее конфликт (для AlreadyExists)
	Err    error  // исходная ошибка (обёрнутая)
}

// Error формирует читаемое сообщение об ошибке.
func (e *RepositoryError) Error() string {
	if e.Kind != nil {
		switch e.Kind {
		case ErrAlreadyExists:
			if e.Field != "" && e.Value != nil {
				return fmt.Sprintf("%s with %s=%v %v", e.Entity, e.Field, e.Value, ErrAlreadyExists)
			}
			return fmt.Sprintf("%s %v", e.Entity, ErrAlreadyExists)
		case ErrNotFound:
			return fmt.Sprintf("%s %v", e.Entity, ErrNotFound)
		case ErrForeignKey:
			return fmt.Sprintf("%s %v", e.Entity, ErrForeignKey)
		default:
			return fmt.Sprintf("%s %v", e.Entity, e.Kind.Error())
		}
	}
	// Если Kind не указан, возвращаем общий текст с исходной ошибкой
	return fmt.Sprintf("repository error for %s: %v", e.Entity, e.Err)
}

// Unwrap возвращает исходную ошибку для раскрутки цепочки.
func (e *RepositoryError) Unwrap() error {
	return e.Err
}

// Is позволяет errors.Is идентифицировать тип ошибки по sentinel.
func (e *RepositoryError) Is(target error) bool {
	// Сравниваем с нашей sentinel-ошибкой
	if e.Kind != nil && target == e.Kind {
		return true
	}
	// Если target — тоже RepositoryError, можно сравнивать по Kind (опционально)
	var other *RepositoryError
	if errors.As(target, &other) {
		return other.Kind == e.Kind && other.Entity == e.Entity // при необходимости
	}
	return false
}

// parseUniqueViolationDetail извлекает поле и значение из Detail ошибки PostgreSQL.
// Формат Detail: "Key (email)=(test@example.com) already exists."
// Возвращает поле, значение и флаг успеха.
func parseUniqueViolationDetail(detail string) (field string, value string, ok bool) {
	re := regexp.MustCompile(`Key \(([^)]+)\)=\(([^)]+)\)`)
	matches := re.FindStringSubmatch(detail)
	if len(matches) == 3 {
		return matches[1], matches[2], true
	}
	return "", "", false
}

// NewRepositoryError анализирует ошибку от GORM и возвращает структурированную ошибку репозитория.
// Параметры:
//   - err: исходная ошибка (обычно от GORM)
//   - entity: имя сущности (например, "user")
//
// Возвращает *RepositoryError с заполненными полями.
// Ошибка корректно работает с errors.Is для sentinel-значений (ErrNotFound, ErrAlreadyExists и т.д.).
func NewRepositoryError(err error, entity string, def error) error {
	if err == nil {
		return nil
	}

	// 1. Случай "запись не найдена"
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &RepositoryError{
			Kind:   ErrNotFound,
			Entity: entity,
			Err:    err,
		}
	}

	// 2. Попытка привести к ошибке драйвера PostgreSQL (pgx)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			field, value, _ := parseUniqueViolationDetail(pgErr.Detail)
			return &RepositoryError{
				Kind:   ErrAlreadyExists,
				Entity: entity,
				Field:  field,
				Value:  value,
				Err:    err,
			}
		case "23503": // foreign_key_violation
			return &RepositoryError{
				Kind:   ErrForeignKey,
				Entity: entity,
				Err:    err,
			}
		}
		// Можно добавить обработку других кодов (например, "23502" — not_null_violation)
	}

	// 3. Все остальные ошибки возвращаем без конкретного Kind,
	//    но с сохранением контекста сущности и исходной ошибки.
	return &RepositoryError{
		Kind:   def,
		Entity: entity,
		Err:    err,
	}
}
