package errors_test

import (
	"testing"

	"github.com/ssoeasy-dev/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestError_WrapAndUnwrap(t *testing.T) {
	kind := errors.ErrNotFound
	wrapped := errors.New(kind, "user with id %d not found", 42)

	assert.ErrorIs(t, wrapped, kind)
	assert.NotErrorIs(t, wrapped, errors.ErrAlreadyExists)

	unwrapped := errors.Unwrap(wrapped)
	assert.EqualError(t, unwrapped, "user with id 42 not found")
}

func TestError_ErrorString(t *testing.T) {
	kind := errors.ErrNotFound
	wrapped := errors.New(kind, "user not found")
	// Ожидаем, что Error() возвращает только kind (без деталей)
	assert.Equal(t, "not found", wrapped.Error())
}

func TestError_Kind(t *testing.T) {
	kind := errors.ErrNotFound
	wrapped := errors.New(kind, "details")
	assert.Equal(t, kind, wrapped.(errors.Kinded).Kind())
}

func TestError_Is(t *testing.T) {
	kind1 := errors.ErrNotFound
	kind2 := errors.ErrAlreadyExists

	e1 := errors.New(kind1, "test")
	e2 := errors.New(kind2, "test")

	assert.True(t, errors.Is(e1, kind1))
	assert.False(t, errors.Is(e1, kind2))
	assert.True(t, errors.Is(e2, kind2))
}

func TestError_As(t *testing.T) {
	kind := errors.ErrNotFound
	wrapped := errors.New(kind, "test")

	var target errors.Error
	assert.True(t, errors.As(wrapped, &target))
	assert.Equal(t, kind, target.Kind())
}
