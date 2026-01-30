package tx

import (
	"context"
	"gorm.io/gorm"
)

type txKey struct{}

type TxManager interface {
	Begin(ctx context.Context) (context.Context, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
	GetDB(ctx context.Context) *gorm.DB
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type txManager struct {
	db *gorm.DB
}

func NewTxManager(db *gorm.DB) TxManager {
	return &txManager{db: db}
}

func (m *txManager) Begin(ctx context.Context) (context.Context, error) {
	tx := m.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return ctx, ErrTxBegin
	}
	return context.WithValue(ctx, txKey{}, tx), nil
}

func (m *txManager) Commit(ctx context.Context) error {
	tx, ok := ctx.Value(txKey{}).(*gorm.DB)
	if !ok {
		return ErrTxCommit
	}
	if err := tx.Commit().Error; err != nil {
		return ErrTxCommit
	}
	return nil
}

func (m *txManager) Rollback(ctx context.Context) error {
	tx, ok := ctx.Value(txKey{}).(*gorm.DB)
	if !ok {
		return ErrTxRollback
	}
	if err := tx.Rollback().Error; err != nil {
		return ErrTxRollback
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
	ctx, err := m.Begin(ctx)
	if err != nil {
		return err
	}
	
	defer func() {
		if p := recover(); p != nil {
			_ = m.Rollback(ctx)
			panic(p)
		}
	}()
	
	err = fn(ctx)
	if err != nil {
		if rbErr := m.Rollback(ctx); rbErr != nil {
			return rbErr
		}
		return err
	}
	
	return m.Commit(ctx)
}
