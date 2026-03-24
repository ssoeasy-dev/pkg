package errors

import (
	"errors"
	"fmt"
)

// VerboseError — ошибка, у которой Error() возвращает полное сообщение.
type VerboseError struct {
	kind error
	msg  string
}

func (e VerboseError) Error() string {
	return e.msg
}

func (e VerboseError) Kind() error {
	return e.kind
}

func (e VerboseError) Unwrap() error {
	// Возвращаем ту же ошибку, чтобы обработчик мог залогировать детали.
	return errors.New(e.msg)
}

func (e VerboseError) Is(target error) bool {
	return target == e.kind
}

// NewVerbose создаёт ошибку с явными деталями (полное сообщение в Error()).
func NewVerbose(kind error, format string, args ...any) error {
	return VerboseError{
		kind: kind,
		msg:  fmt.Sprintf(format, args...),
	}
}
