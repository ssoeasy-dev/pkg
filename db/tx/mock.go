package tx

import (
	"context"

	"github.com/ssoeasy-dev/pkg/errors"
	"github.com/ssoeasy-dev/pkg/logger"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// MockTxManager — mock-реализация TxManager для unit-тестов.
//
// Пример базового использования (без logger):
//
//	mgr := tx.NewMockTxManager(nil)
//	mgr.WithTransactionalSuccess(ctx)
//	// ... вызов кода с mgr ...
//	mgr.AssertExpectations(t)
type MockTxManager struct {
	mock.Mock
	log logger.Logger // nil допустим
}

// NewMockTxManager создаёт MockTxManager. log может быть nil.
func NewMockTxManager(log logger.Logger) *MockTxManager {
	return &MockTxManager{log: log}
}

// logError логирует через logger если он задан, иначе молча игнорирует.
func (m *MockTxManager) logError(ctx context.Context, msg string, fields map[string]any) {
	if m.log != nil {
		m.log.Error(ctx, msg, fields)
	}
}

// ─── Interface implementation ─────────────────────────────────────────────────

func (m *MockTxManager) Begin(ctx context.Context) (context.Context, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).(context.Context), args.Error(1)
	}
	return ctx, args.Error(1)
}

func (m *MockTxManager) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTxManager) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTxManager) GetDB(ctx context.Context) *gorm.DB {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).(*gorm.DB)
	}
	return nil
}

func (m *MockTxManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	args := m.Called(ctx, fn)
	return args.Error(0)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// WithTransactionalSuccess настраивает mock так, чтобы fn был вызван,
// а WithTransaction вернул nil.
func (m *MockTxManager) WithTransactionalSuccess(ctx context.Context) {
	m.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			if fnErr := fn(ctx); fnErr != nil {
				m.logError(ctx, "WithTransactionalSuccess: fn returned error", map[string]any{
					"error": fnErr.Error(),
				})
			}
		}).
		Return(nil)
}

// WithTransactionalRollback настраивает mock так, чтобы fn был вызван,
// а WithTransaction вернул returnErr.
func (m *MockTxManager) WithTransactionalRollback(ctx context.Context, returnErr error) {
	m.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			if fnErr := fn(ctx); fnErr != nil {
				m.logError(ctx, "WithTransactionalRollback: fn returned error", map[string]any{
					"error": fnErr.Error(),
				})
			}
		}).
		Return(returnErr)
}

// WithTransactionErrBegin настраивает mock так, чтобы WithTransaction вернул
// ErrTxBegin, не вызывая fn.
func (m *MockTxManager) WithTransactionErrBegin(ctx context.Context) {
	m.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(errors.ErrTxBegin)
}

// WithTransactionErrCommit настраивает mock так, чтобы fn был вызван,
// а WithTransaction вернул ErrTxCommit.
func (m *MockTxManager) WithTransactionErrCommit(ctx context.Context) {
	m.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			if fnErr := fn(ctx); fnErr != nil {
				m.logError(ctx, "WithTransactionErrCommit: fn returned error", map[string]any{
					"error": fnErr.Error(),
				})
			}
		}).
		Return(errors.ErrTxCommit)
}

// WithTransactionErrRollback настраивает mock так, чтобы fn был вызван,
// а WithTransaction вернул ErrTxRollback.
func (m *MockTxManager) WithTransactionErrRollback(ctx context.Context) {
	m.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			if fnErr := fn(ctx); fnErr != nil {
				m.logError(ctx, "WithTransactionErrRollback: fn returned error", map[string]any{
					"error": fnErr.Error(),
				})
			}
		}).
		Return(errors.ErrTxRollback)
}
