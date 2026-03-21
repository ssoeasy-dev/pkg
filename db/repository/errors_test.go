package repository_test

import (
	"errors"
	"testing"

	"github.com/ssoeasy-dev/pkg/db/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestNewRepositoryError_Nil(t *testing.T) {
	assert.NoError(t, repository.NewRepositoryError(nil, "user", repository.ErrGetFailed))
}

func TestNewRepositoryError_RecordNotFound(t *testing.T) {
	err := repository.NewRepositoryError(gorm.ErrRecordNotFound, "user", repository.ErrGetFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrNotFound)
	assert.Contains(t, err.Error(), "user")
}

func TestNewRepositoryError_DefaultKind(t *testing.T) {
	cause := errors.New("connection refused")
	err := repository.NewRepositoryError(cause, "order", repository.ErrCreationFailed)
	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrCreationFailed)
	assert.Contains(t, err.Error(), "order")
	assert.Contains(t, err.Error(), "connection refused")
}

// Оригинальная причина доступна через Unwrap — для логирования.
func TestNewRepositoryError_Unwrap_ReturnsOriginalCause(t *testing.T) {
	cause := errors.New("disk full")
	err := repository.NewRepositoryError(cause, "item", repository.ErrGetFailed)
	assert.ErrorIs(t, errors.Unwrap(err), cause)
}

func TestNewRepositoryError_Unwrap_RecordNotFound(t *testing.T) {
	err := repository.NewRepositoryError(gorm.ErrRecordNotFound, "article", repository.ErrGetFailed)
	assert.ErrorIs(t, errors.Unwrap(err), gorm.ErrRecordNotFound)
}

// ─── errors.Is совместимость ─────────────────────────────────────────────────

func TestSentinels_ErrorsIs(t *testing.T) {
	cases := []struct {
		name     string
		sentinel error
		def      error
	}{
		{"NotFound via RecordNotFound", repository.ErrNotFound, repository.ErrGetFailed},
		{"CreationFailed", repository.ErrCreationFailed, repository.ErrCreationFailed},
		{"UpdateFailed", repository.ErrUpdateFailed, repository.ErrUpdateFailed},
		{"DeleteFailed", repository.ErrDeleteFailed, repository.ErrDeleteFailed},
		{"GetFailed", repository.ErrGetFailed, repository.ErrGetFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var source error
			if tc.sentinel == repository.ErrNotFound {
				source = gorm.ErrRecordNotFound
			} else {
				source = errors.New("some db error")
			}
			err := repository.NewRepositoryError(source, "entity", tc.def)
			assert.ErrorIs(t, err, tc.sentinel)
		})
	}
}

func TestSentinels_Distinct(t *testing.T) {
	sentinels := []error{
		repository.ErrNotFound,
		repository.ErrAlreadyExists,
		repository.ErrForeignKey,
		repository.ErrCreationFailed,
		repository.ErrUpdateFailed,
		repository.ErrDeleteFailed,
		repository.ErrGetFailed,
	}
	for i, a := range sentinels {
		for j, b := range sentinels {
			if i != j {
				assert.False(t, errors.Is(a, b), "%v should not match %v", a, b)
			}
		}
	}
}

// ─── Сообщения ────────────────────────────────────────────────────────────────

func TestNewRepositoryError_Message_NotFound(t *testing.T) {
	err := repository.NewRepositoryError(gorm.ErrRecordNotFound, "article", repository.ErrGetFailed)
	assert.Equal(t, "not found: article", err.Error())
}

func TestNewRepositoryError_Message_DefaultWrapsOriginal(t *testing.T) {
	cause := errors.New("timeout")
	err := repository.NewRepositoryError(cause, "payment", repository.ErrCreationFailed)
	assert.Equal(t, "creation failed: payment: timeout", err.Error())
}
