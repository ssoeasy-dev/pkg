package tx

import (
	"context"

	"github.com/ssoeasy-dev/pkg/logger"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockTxManager struct {
	mock.Mock
	log *logger.Logger
}

func NewMockTxManager(log *logger.Logger) *MockTxManager {
	return &MockTxManager{
		log: log,
	}
}

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

func (m *MockTxManager) WithTransactionalSuccess(ctx context.Context) {
	m.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(context.Context) error)
		err := fn(ctx)
		if err != nil {
			m.log.Error(ctx, "WithTransactionalSuccess Function Error", map[string]any{
				"error": err.Error(),
			})
		}
	}).Return(nil)
}

func (m *MockTxManager) WithTransactionalRollback(ctx context.Context, err error) {
	m.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(context.Context) error)
		err := fn(ctx)
		if err != nil {
			m.log.Error(ctx, "WithTransactionalRollback Function Error", map[string]any{
				"error": err.Error(),
			})
		}
	}).Return(err)
}

func (m *MockTxManager) WithTransactionErrBegin(ctx context.Context) {
	m.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).Return(ErrTxBegin)
}

func (m *MockTxManager) WithTransactionErrCommit(ctx context.Context) {
	m.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(context.Context) error)
		err := fn(ctx)
		if err != nil {
			m.log.Error(ctx, "WithTransactionErrCommit Function Error", map[string]any{
				"error": err.Error(),
			})
		}
	}).Return(ErrTxCommit)
}

func (m *MockTxManager) WithTransactionErrRollback(ctx context.Context) {
	m.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(context.Context) error)
		err := fn(ctx)
		if err != nil {
			m.log.Error(ctx, "WithTransactionErrRollback Function Error", map[string]any{
				"error": err.Error(),
			})
		}
	}).Return(ErrTxRollback)
}
