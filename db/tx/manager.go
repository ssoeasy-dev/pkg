package tx

import (
	"context"

	"github.com/ssoeasy-dev/pkg/db"
	"github.com/ssoeasy-dev/pkg/errors"
	"github.com/ssoeasy-dev/pkg/logger"
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
	log logger.Logger
	db  *gorm.DB
}

// NewTxManager создаёт новый TxManager поверх *gorm.DB.
func NewTxManager(db *gorm.DB, log logger.Logger) TxManager {
	return &txManager{db: db, log: log}
}

func (m *txManager) Begin(ctx context.Context) (context.Context, error) {
	tx := m.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return ctx, db.NewError(tx.Error, "tx begin")
	}
	return context.WithValue(ctx, txKey{}, tx), nil
}

func (m *txManager) Commit(ctx context.Context) error {
	tx, ok := ctx.Value(txKey{}).(*gorm.DB)
	if !ok || tx == nil {
		return errors.New(errors.ErrInternal, "tx commit: no transaction in context")
	}
	if err := tx.Commit().Error; err != nil {
		return db.NewError(err, "tx commit")
	}
	return nil
}

func (m *txManager) Rollback(ctx context.Context) error {
	tx, ok := ctx.Value(txKey{}).(*gorm.DB)
	if !ok || tx == nil {
		return errors.New(errors.ErrInternal, "tx rollback: no transaction in context")
	}
	if err := tx.Rollback().Error; err != nil {
		return db.NewError(err, "tx rollback")
	}
	return nil
}

func (m *txManager) GetDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
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
			// Ошибка отката — внутренняя проблема БД, оборачиваем её и добавляем контекст исходной ошибки
			m.log.Error(ctx, "transaction rollback failed after business error",
				map[string]any{
					"business_error": errors.FullError(err),
					"rollback_error": rbErr.Error(),
				})
		}
		return err
	}

	return m.Commit(txCtx)
}
