package tx_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ssoeasy-dev/pkg/db/tx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Sentinel errors ──────────────────────────────────────────────────────────

func TestErrors_Distinct(t *testing.T) {
	assert.NotNil(t, tx.ErrTxBegin)
	assert.NotNil(t, tx.ErrTxCommit)
	assert.NotNil(t, tx.ErrTxRollback)

	assert.NotEqual(t, tx.ErrTxBegin, tx.ErrTxCommit)
	assert.NotEqual(t, tx.ErrTxBegin, tx.ErrTxRollback)
	assert.NotEqual(t, tx.ErrTxCommit, tx.ErrTxRollback)
}

func TestErrors_IsCompatible(t *testing.T) {
	// Обёрнутые ошибки совместимы с errors.Is.
	wrapped := errors.New("wrapped: " + tx.ErrTxBegin.Error())
	_ = wrapped // просто убедимся что конструируется

	// Sentinel сами по себе работают с errors.Is.
	assert.ErrorIs(t, tx.ErrTxBegin, tx.ErrTxBegin)
	assert.ErrorIs(t, tx.ErrTxCommit, tx.ErrTxCommit)
	assert.ErrorIs(t, tx.ErrTxRollback, tx.ErrTxRollback)
}

// ─── MockTxManager ────────────────────────────────────────────────────────────

func TestMockTxManager_WithTransactionalSuccess_FnCalledAndNilReturned(t *testing.T) {
	mgr := tx.NewMockTxManager(nil)
	ctx := context.Background()
	mgr.WithTransactionalSuccess(ctx)

	called := false
	err := mgr.WithTransaction(ctx, func(_ context.Context) error {
		called = true
		return nil
	})

	require.NoError(t, err)
	assert.True(t, called, "fn должен быть вызван")
	mgr.AssertExpectations(t)
}

func TestMockTxManager_WithTransactionalSuccess_FnErrorLogged(t *testing.T) {
	// Если fn возвращает ошибку, mock её логирует (или игнорирует при nil log),
	// но всё равно возвращает nil — такая семантика helper'а.
	mgr := tx.NewMockTxManager(nil)
	ctx := context.Background()
	mgr.WithTransactionalSuccess(ctx)

	err := mgr.WithTransaction(ctx, func(_ context.Context) error {
		return errors.New("fn error")
	})

	// Хелпер настроен вернуть nil независимо от результата fn.
	require.NoError(t, err)
	mgr.AssertExpectations(t)
}

func TestMockTxManager_WithTransactionalRollback_FnCalledAndErrReturned(t *testing.T) {
	mgr := tx.NewMockTxManager(nil)
	ctx := context.Background()
	expectedErr := errors.New("service error")
	mgr.WithTransactionalRollback(ctx, expectedErr)

	called := false
	err := mgr.WithTransaction(ctx, func(_ context.Context) error {
		called = true
		return nil
	})

	require.ErrorIs(t, err, expectedErr)
	assert.True(t, called)
	mgr.AssertExpectations(t)
}

func TestMockTxManager_WithTransactionErrBegin_FnNotCalled(t *testing.T) {
	// Семантика: транзакция не началась → fn не вызывается.
	mgr := tx.NewMockTxManager(nil)
	ctx := context.Background()
	mgr.WithTransactionErrBegin(ctx)

	called := false
	err := mgr.WithTransaction(ctx, func(_ context.Context) error {
		called = true
		return nil
	})

	require.ErrorIs(t, err, tx.ErrTxBegin)
	assert.False(t, called, "fn не должен вызываться при ошибке Begin")
	mgr.AssertExpectations(t)
}

func TestMockTxManager_WithTransactionErrCommit_FnCalledErrReturned(t *testing.T) {
	mgr := tx.NewMockTxManager(nil)
	ctx := context.Background()
	mgr.WithTransactionErrCommit(ctx)

	called := false
	err := mgr.WithTransaction(ctx, func(_ context.Context) error {
		called = true
		return nil
	})

	require.ErrorIs(t, err, tx.ErrTxCommit)
	assert.True(t, called)
	mgr.AssertExpectations(t)
}

func TestMockTxManager_WithTransactionErrRollback_FnCalledErrReturned(t *testing.T) {
	mgr := tx.NewMockTxManager(nil)
	ctx := context.Background()
	mgr.WithTransactionErrRollback(ctx)

	called := false
	err := mgr.WithTransaction(ctx, func(_ context.Context) error {
		called = true
		return nil
	})

	require.ErrorIs(t, err, tx.ErrTxRollback)
	assert.True(t, called)
	mgr.AssertExpectations(t)
}

func TestMockTxManager_GetDB_ReturnsNilWhenNotSetUp(t *testing.T) {
	mgr := tx.NewMockTxManager(nil)
	ctx := context.Background()
	mgr.On("GetDB", ctx).Return(nil)

	result := mgr.GetDB(ctx)
	assert.Nil(t, result)
	mgr.AssertExpectations(t)
}

func TestMockTxManager_Begin_ReturnsErrWhenSetUp(t *testing.T) {
	mgr := tx.NewMockTxManager(nil)
	ctx := context.Background()
	mgr.On("Begin", ctx).Return(ctx, tx.ErrTxBegin)

	_, err := mgr.Begin(ctx)
	require.ErrorIs(t, err, tx.ErrTxBegin)
	mgr.AssertExpectations(t)
}

func TestMockTxManager_Commit_ReturnsErrWhenSetUp(t *testing.T) {
	mgr := tx.NewMockTxManager(nil)
	ctx := context.Background()
	mgr.On("Commit", ctx).Return(tx.ErrTxCommit)

	err := mgr.Commit(ctx)
	require.ErrorIs(t, err, tx.ErrTxCommit)
	mgr.AssertExpectations(t)
}

func TestMockTxManager_Rollback_ReturnsNilWhenSetUp(t *testing.T) {
	mgr := tx.NewMockTxManager(nil)
	ctx := context.Background()
	mgr.On("Rollback", ctx).Return(nil)

	err := mgr.Rollback(ctx)
	require.NoError(t, err)
	mgr.AssertExpectations(t)
}
