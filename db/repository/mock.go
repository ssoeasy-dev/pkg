package repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/ssoeasy-dev/pkg/errors"
	"gorm.io/gorm"
)

// MockRepository — mock-реализация интерфейса Repository[Model].
// Используется в unit-тестах сервисов.
//
// Пример:
//
//	mockRepo := repository.NewMockRepository[models.User]()
//	mockRepo.OnFindOneReturn(ctx, &user)
//	// или
//	mockRepo.FindOneErrNotFound(ctx)
type MockRepository[Model any] struct {
	mock.Mock
}

func NewMockRepository[Model any]() *MockRepository[Model] {
	return &MockRepository[Model]{}
}

// ─── Interface implementation ─────────────────────────────────────────────────

func (m *MockRepository[Model]) DB(ctx context.Context) *gorm.DB {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).(*gorm.DB)
	}
	return nil
}

func (m *MockRepository[Model]) Create(ctx context.Context, value *Model, opts ...RepositoryOption) error {
	args := m.Called(ctx, value, opts)
	return args.Error(0)
}

func (m *MockRepository[Model]) Update(ctx context.Context, value map[string]any, opts ...RepositoryOption) (int64, error) {
	args := m.Called(ctx, value, opts)
	return toInt64(args.Get(0)), args.Error(1)
}

func (m *MockRepository[Model]) Delete(ctx context.Context, force bool, opts ...RepositoryOption) (int64, error) {
	args := m.Called(ctx, force, opts)
	return toInt64(args.Get(0)), args.Error(1)
}

func (m *MockRepository[Model]) FindOne(ctx context.Context, opts ...RepositoryOption) (*Model, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) != nil {
		return args.Get(0).(*Model), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRepository[Model]) FindAll(ctx context.Context, opts ...RepositoryOption) ([]Model, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) != nil {
		return args.Get(0).([]Model), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRepository[Model]) Count(ctx context.Context, opts ...RepositoryOption) (int64, error) {
	args := m.Called(ctx, opts)
	return toInt64(args.Get(0)), args.Error(1)
}

func (m *MockRepository[Model]) Exists(ctx context.Context, opts ...RepositoryOption) (bool, error) {
	args := m.Called(ctx, opts)
	return args.Bool(0), args.Error(1)
}

// ─── Create helpers ───────────────────────────────────────────────────────────

func (m *MockRepository[Model]) OnCreate(ctx context.Context) {
	m.On("Create", ctx, mock.Anything, mock.Anything).Return(nil)
}

func (m *MockRepository[Model]) CreateErrAlreadyExists(ctx context.Context) {
	m.On("Create", ctx, mock.Anything, mock.Anything).Return(errors.ErrAlreadyExists)
}

func (m *MockRepository[Model]) CreateErrCreateFailed(ctx context.Context) {
	m.On("Create", ctx, mock.Anything, mock.Anything).Return(errors.ErrCreationFailed)
}

// ─── Update helpers ───────────────────────────────────────────────────────────

func (m *MockRepository[Model]) OnUpdate(ctx context.Context, affected int64) {
	m.On("Update", ctx, mock.Anything, mock.Anything).Return(affected, nil)
}

func (m *MockRepository[Model]) UpdateErrNotFound(ctx context.Context) {
	m.On("Update", ctx, mock.Anything, mock.Anything).Return(int64(0), errors.ErrNotFound)
}

func (m *MockRepository[Model]) UpdateErrUpdateFailed(ctx context.Context) {
	m.On("Update", ctx, mock.Anything, mock.Anything).Return(int64(0), errors.ErrUpdateFailed)
}

// ─── Delete helpers ───────────────────────────────────────────────────────────

func (m *MockRepository[Model]) OnDelete(ctx context.Context, affected int64) {
	m.On("Delete", ctx, mock.Anything, mock.Anything).Return(affected, nil)
}

func (m *MockRepository[Model]) DeleteErrNotFound(ctx context.Context) {
	m.On("Delete", ctx, mock.Anything, mock.Anything).Return(int64(0), errors.ErrNotFound)
}

func (m *MockRepository[Model]) DeleteErrDeleteFailed(ctx context.Context) {
	m.On("Delete", ctx, mock.Anything, mock.Anything).Return(int64(0), errors.ErrDeleteFailed)
}

// ─── FindOne helpers ──────────────────────────────────────────────────────────

func (m *MockRepository[Model]) OnFindOneReturn(ctx context.Context, model *Model) {
	m.On("FindOne", ctx, mock.Anything).Return(model, nil)
}

func (m *MockRepository[Model]) FindOneErrNotFound(ctx context.Context) {
	m.On("FindOne", ctx, mock.Anything).Return(nil, errors.ErrNotFound)
}

func (m *MockRepository[Model]) FindOneErrGetFailed(ctx context.Context) {
	m.On("FindOne", ctx, mock.Anything).Return(nil, errors.ErrGetFailed)
}

// ─── FindAll helpers ──────────────────────────────────────────────────────────

func (m *MockRepository[Model]) OnFindAllReturn(ctx context.Context, models []Model) {
	m.On("FindAll", ctx, mock.Anything).Return(models, nil)
}

func (m *MockRepository[Model]) FindAllErrGetFailed(ctx context.Context) {
	m.On("FindAll", ctx, mock.Anything).Return(nil, errors.ErrGetFailed)
}

// ─── Count helpers ────────────────────────────────────────────────────────────

func (m *MockRepository[Model]) OnCountReturn(ctx context.Context, count int64) {
	m.On("Count", ctx, mock.Anything).Return(count, nil)
}

func (m *MockRepository[Model]) CountErrGetFailed(ctx context.Context) {
	m.On("Count", ctx, mock.Anything).Return(int64(0), errors.ErrGetFailed)
}

// ─── Exists helpers ───────────────────────────────────────────────────────────

func (m *MockRepository[Model]) OnExistsReturn(ctx context.Context, exists bool) {
	m.On("Exists", ctx, mock.Anything).Return(exists, nil)
}

func (m *MockRepository[Model]) ExistsErrGetFailed(ctx context.Context) {
	m.On("Exists", ctx, mock.Anything).Return(false, errors.ErrGetFailed)
}

// ─── DB helpers ───────────────────────────────────────────────────────────────

func (m *MockRepository[Model]) OnDB(ctx context.Context, db *gorm.DB) {
	m.On("DB", ctx).Return(db)
}

// ─── Internal ─────────────────────────────────────────────────────────────────

// toInt64 безопасно приводит значение к int64.
// Нужен потому что testify хранит аргументы как interface{},
// и прямой type assertion паникует если Return получил nil или int вместо int64.
func toInt64(v any) int64 {
	if v == nil {
		return 0
	}
	if n, ok := v.(int64); ok {
		return n
	}
	return 0
}
