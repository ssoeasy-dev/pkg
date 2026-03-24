package errors_test

import (
	"testing"

	"github.com/ssoeasy-dev/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestVerboseError_WrapAndUnwrap(t *testing.T) {
	kind := errors.ErrNotFound
	wrapped := errors.NewVerbose(kind, "user with id %d not found", 42)

	assert.ErrorIs(t, wrapped, kind)
	assert.NotErrorIs(t, wrapped, errors.ErrAlreadyExists)

	unwrapped := errors.Unwrap(wrapped)
	assert.EqualError(t, unwrapped, "user with id 42 not found")
}

func TestVerboseError_ErrorString(t *testing.T) {
    kind := errors.ErrNotFound
    wrapped := errors.NewVerbose(kind, "user %s not found", "john")
    // Ожидаем, что Error() возвращает полное сообщение с деталями
    assert.Equal(t, "user john not found", wrapped.Error())
}

func TestVerboseError_Kind(t *testing.T) {
    kind := errors.ErrNotFound
    wrapped := errors.NewVerbose(kind, "details")
    assert.Equal(t, kind, wrapped.(errors.Kinded).Kind())
}

func TestVerboseError_Is(t *testing.T) {
	kind1 := errors.ErrNotFound
	kind2 := errors.ErrAlreadyExists

	e1 := errors.NewVerbose(kind1, "test")
	e2 := errors.NewVerbose(kind2, "test")

	assert.True(t, errors.Is(e1, kind1))
	assert.False(t, errors.Is(e1, kind2))
	assert.True(t, errors.Is(e2, kind2))
}

func TestVerboseError_As(t *testing.T) {
    kind := errors.ErrNotFound
    wrapped := errors.NewVerbose(kind, "test")

    var target errors.Kinded
    assert.True(t, errors.As(wrapped, &target))
    assert.Equal(t, kind, target.Kind())
}
