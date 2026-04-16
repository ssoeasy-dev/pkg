// Package errors предоставляет структурированную обработку ошибок с разделением
// на виды (kinds), публичные и технические сообщения, а также бесшовную интеграцию
// со стандартным пакетом errors.
//
// Основные возможности:
//   - Создание ошибок с заранее определёнными видами (ErrNotFound, ErrAlreadyExists и т.д.)
//   - Оборачивание ошибок с добавлением контекста без потери вида
//   - Разделение публичного сообщения (для клиента) и полного технического (для логов)
//   - Совместимость с errors.Is, errors.As и errors.Unwrap
package errors

import (
	"errors"
	"fmt"
)

// Error — основная структура ошибки, оборачивающая вид и детали.
type Error struct {
	kind error  // ошибка вида
	msg  string // публичное сообщение этого уровня (только если есть next)
	next error  // следующая ошибка в цепочке (может быть *Error или любая другая)
}

// Error возвращает цепочку публичных сообщений.
// Корневая ошибка (next == nil) отдаёт только строку вида.
func (e *Error) Error() string {
	if e.next == nil {
		return Kind(e).Error()
	}
	var nextErr *Error
	if errors.As(e.next, &nextErr) {
		// Если msg пустое, пропускаем этот уровень (используется в WithKind для обычных ошибок)
		if e.msg == "" {
			return nextErr.Error()
		}
		return e.msg + ": " + nextErr.Error()
	}
	// Если next — обычная ошибка и msg пустое, возвращаем только вид
	if e.msg == "" {
		return Kind(e).Error()
	}
	return e.msg
}

// Unwrap возвращает следующую ошибку в цепочке.
func (e *Error) Unwrap() error {
	return e.next
}

// Is реализует интерфейс errors.Is для сравнения по виду.
func (e *Error) Is(target error) bool {
	return target == e.kind
}

// Kind возвращает вид ошибки.
func (e *Error) Kind() error {
	return e.kind
}

// FullError возвращает полное сообщение с техническими деталями для логирования.
func (e *Error) FullError() string {
	if e.next == nil {
		if e.msg == "" {
			return Kind(e).Error()
		}
		return e.msg
	}
	var nextErr *Error
	if errors.As(e.next, &nextErr) {
		if e.msg == "" {
			return nextErr.FullError()
		}
		return e.msg + ": " + nextErr.FullError()
	}
	// next — обычная ошибка
	if e.msg == "" {
		return e.next.Error()
	}
	return e.msg + ": " + e.next.Error()
}

// New создаёт корневую ошибку с указанным видом.
// Параметры:
//   - kind: ошибка вида.
//   - msg: Сообщение. Считается техническим и будет видно только в FullError().
func New(kind error, msg string) error {
	return &Error{
		kind: kind,
		msg:  msg,
	}
}

// Newf – форматированный вариант New.
func Newf(kind error, format string, args ...any) error {
	return New(kind, fmt.Sprintf(format, args...))
}

// Wrap оборачивает любую ошибку, добавляя новый уровень публичного сообщения.
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}

	return &Error{
		msg:  msg,
		next: err,
	}
}

// Wrapf – форматированный вариант Wrap.
func Wrapf(err error, format string, args ...any) error {
	return Wrap(err, fmt.Sprintf(format, args...))
}

// WithKind возвращает ошибку с изменённым видом, сохраняя цепочку.
func WithKind(err error, kind error) error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return &Error{
			kind: kind,
			msg:  e.msg,
			next: e.next,
		}
	}
	// Для обычной ошибки: задаём kind, оставляем msg пустым,
	// чтобы вид не влиял на техническое сообщение (FullError).
	return &Error{
		kind: kind,
		msg:  "",
		next: err,
	}
}

// NewWrap создаёт обёрнутую ошибку за один вызов.
// Устанавливает явный вид kind и оборачивает причину cause с публичным сообщением msg.
func NewWrap(kind error, cause error, msg string) error {
	return &Error{
		kind: kind,
		msg:  msg,
		next: cause,
	}
}

// NewWrapf – форматированный вариант NewWrap.
func NewWrapf(kind error, cause error, format string, args ...any) error {
	return &Error{
		kind: kind,
		msg:  fmt.Sprintf(format, args...),
		next: cause,
	}
}

// Kind извлекает вид ошибки. Если ошибка не создана через этот пакет, возвращает ErrUnknown.
func Kind(err error) error {
	var e *Error
	if errors.As(err, &e) {
		if e.kind == nil && e.next != nil {
			if k := e.Kind(); k != nil {
				return k
			}
			// Если kind == nil и next == nil, возвращаем ErrUnknown как безопасное значение
			return ErrUnknown
		}
		return e.Kind()
	}
	return ErrUnknown
}

// Is проверяет, соответствует ли ошибка целевому виду.
// Аналог errors.Is из стандартной библиотеки.
func Is(err error, target error) bool {
	return errors.Is(err, target)
}

// As находит первую ошибку в цепочке, соответствующую типу target.
// Аналог errors.As из стандартной библиотеки.
func As(err error, target any) bool {
	return errors.As(err, target)
}

// Unwrap возвращает следующую ошибку в цепочке.
// Работает как с ошибками пакета, так и с обычными.
func Unwrap(err error) error {
	if e, ok := err.(*Error); ok {
		return e.Unwrap()
	}
	return errors.Unwrap(err)
}

// FullError возвращает полную строку ошибки с техническими деталями.
// Используйте только для логирования.
func FullError(err error) string {
	if err == nil {
		return ""
	}
	if e, ok := err.(*Error); ok {
		return e.FullError()
	}
	return err.Error()
}
