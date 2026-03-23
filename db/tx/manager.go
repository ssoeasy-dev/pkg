package tx

import (
	"context"
	"fmt"

	"github.com/ssoeasy-dev/pkg/errors"
	"gorm.io/gorm"
)

type txKey struct{}

// TxManager управляет транзакциями базы данных.
type TxManager interface {
	// Begin начинает транзакцию и возвращает контекст с встроенным tx.
	Begin(ctx context.Context) (context.Context, error)
	// Commit фиксирует транзакцию из контекста.
	Commit(ctx context.Context) error
	// Rollback откатывает транзакцию из контекста.
	Rollback(ctx context.Context) error
	// GetDB возвращает *gorm.DB из транзакционного контекста,
	// или базовый *gorm.DB если транзакции нет.
	GetDB(ctx context.Context) *gorm.DB
	// WithTransaction выполняет fn в транзакции.
	// Если fn вернул nil — коммит. Если fn вернул ошибку или случилась паника — откат.
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type txManager struct {
	db *gorm.DB
}

// NewTxManager создаёт новый TxManager поверх *gorm.DB.
func NewTxManager(db *gorm.DB) TxManager {
	return &txManager{db: db}
}

func (m *txManager) Begin(ctx context.Context) (context.Context, error) {
	tx := m.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return ctx, fmt.Errorf("%w: %v", errors.ErrTxBegin, tx.Error)
	}
	return context.WithValue(ctx, txKey{}, tx), nil
}

func (m *txManager) Commit(ctx context.Context) error {
	tx, ok := ctx.Value(txKey{}).(*gorm.DB)
	if !ok {
		return errors.ErrTxCommit
	}
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("%w: %v", errors.ErrTxCommit, err)
	}
	return nil
}

func (m *txManager) Rollback(ctx context.Context) error {
	tx, ok := ctx.Value(txKey{}).(*gorm.DB)
	if !ok {
		return errors.ErrTxRollback
	}
	if err := tx.Rollback().Error; err != nil {
		return fmt.Errorf("%w: %v", errors.ErrTxRollback, err)
	}
	return nil
}

func (m *txManager) GetDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return m.db.WithContext(ctx)
}

func (m *txManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	txCtx, err := m.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = m.Rollback(txCtx)
			panic(p)
		}
	}()

	if err = fn(txCtx); err != nil {
		if rbErr := m.Rollback(txCtx); rbErr != nil {
			return fmt.Errorf("rollback failed: %w; original error: %w", rbErr, err)
		}
		return err
	}

	return m.Commit(txCtx)
}
