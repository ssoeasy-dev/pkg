package errors

import (
	"errors"
	"fmt"
)

type Error struct {
	kind error
	err  error
}

func (e Error) Error() string {
	return e.kind.Error()
}

func (e Error) Kind() error {
	return e.kind
}

func (e Error) Unwrap() error {
	return e.err
}

func (e Error) Is(target error) bool {
	return target == e.kind
}

func New(kind error, err string, details ...any) error {
	return Error{
		err:  fmt.Errorf(err, details...),
		kind: kind,
	}
}

func Is(err error, target error) bool {
	return errors.Is(err, target)
}

func As(err error, target any) bool {
	return errors.As(err, target)
}

func Unwrap(err error) error {
	return errors.Unwrap(err)
}
