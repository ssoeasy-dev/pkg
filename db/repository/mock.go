package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockRepository[M any] struct {
	mock.Mock
}

func NewMockRepository[M any]() *MockRepository[M] {
	return &MockRepository[M]{}
}

func (m *MockRepository[M]) Create(ctx context.Context, value *M) error {
	args := m.Called(ctx, value)

	return args.Error(0)
}

func (m *MockRepository[Model]) Update(ctx context.Context, value *Model) error {
	args := m.Called(ctx, value)
	return args.Error(0)
}

func (m *MockRepository[Model]) Delete(ctx context.Context, id uuid.UUID, force bool) error {
	args := m.Called(ctx, id, force)
	return args.Error(0)
}

func (m *MockRepository[Model]) GetByID(ctx context.Context, id uuid.UUID) (*Model, error) {
	args := m.Called(ctx, id)

	if args.Get(0) != nil {
		return args.Get(0).(*Model), args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockRepository[Model]) FindOne(ctx context.Context, conditions map[string]any) (*Model, error) {
	args := m.Called(ctx, conditions)

	if args.Get(0) != nil {
		return args.Get(0).(*Model), args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockRepository[Model]) FindAll(ctx context.Context, conditions map[string]any, limit, offset int) ([]Model, error) {
	args := m.Called(ctx, conditions, limit, offset)

	if args.Get(0) != nil {
		return args.Get(0).([]Model), args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockRepository[Model]) Count(ctx context.Context, conditions map[string]any) (int64, error) {
	args := m.Called(ctx, conditions)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRepository[Model]) Exists(ctx context.Context, conditions map[string]any) (bool, error) {
	args := m.Called(ctx, conditions)
	return args.Bool(0), args.Error(1)
}

// Helpers

func (m *MockRepository[Model]) CreateErrAlreadyExists(ctx context.Context) {
	m.On("Create", ctx, mock.Anything).Return(ErrAlreadyExists)
}

func (m *MockRepository[Model]) CreateErrCreateFailed(ctx context.Context) {
	m.On("Create", ctx, mock.Anything).Return(ErrCreationFailed)
}

func (m *MockRepository[Model]) UpdateErrNotFound(ctx context.Context) {
	m.On("Update", ctx, mock.Anything).Return(ErrNotFound)
}

func (m *MockRepository[Model]) UpdateErrUpdateFailed(ctx context.Context) {
	m.On("Update", ctx, mock.Anything).Return(ErrUpdateFailed)
}

func (m *MockRepository[Model]) DeleteErrNotFound(ctx context.Context) {
	m.On("Delete", ctx, mock.Anything).Return(ErrNotFound)
}

func (m *MockRepository[Model]) DeleteErrDeleteFailed(ctx context.Context) {
	m.On("Delete", ctx, mock.Anything).Return(ErrDeleteFailed)
}

func (m *MockRepository[Model]) GetByIDErrNotFound(ctx context.Context) {
	m.On("GetByID", ctx, mock.Anything).Return(ErrNotFound)
}

func (m *MockRepository[Model]) GetByIDErrGetFailed(ctx context.Context) {
	m.On("GetByID", ctx, mock.Anything).Return(ErrGetFailed)
}

func (m *MockRepository[Model]) FindOneErrNotFound(ctx context.Context) {
	m.On("FindOne", ctx, mock.Anything).Return(ErrNotFound)
}

func (m *MockRepository[Model]) FindOneErrGetFailed(ctx context.Context) {
	m.On("FindOne", ctx, mock.Anything).Return(ErrGetFailed)
}

func (m *MockRepository[Model]) FindAllErrGetFailed(ctx context.Context) {
	m.On("FindAll", ctx, mock.Anything).Return(ErrGetFailed)
}

func (m *MockRepository[Model]) CountErrGetFailed(ctx context.Context) {
	m.On("Count", ctx, mock.Anything).Return(ErrGetFailed)
}

func (m *MockRepository[Model]) ExistsErrGetFailed(ctx context.Context) {
	m.On("Exists", ctx, mock.Anything).Return(ErrGetFailed)
}
