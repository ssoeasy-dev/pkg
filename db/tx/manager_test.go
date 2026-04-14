package tx_test

import (
	"context"
	"testing"

	"github.com/ssoeasy-dev/pkg/db/tx"
	"github.com/ssoeasy-dev/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Sentinel errors ──────────────────────────────────────────────────────────

func TestErrors_Distinct(t *testing.T) {
	assert.NotNil(t, errors.ErrTxBegin)
	assert.NotNil(t, errors.ErrTxCommit)
	assert.NotNil(t, errors.ErrTxRollback)

	assert.NotEqual(t, errors.ErrTxBegin, errors.ErrTxCommit)
	assert.NotEqual(t, errors.ErrTxBegin, errors.ErrTxRollback)
	assert.NotEqual(t, errors.ErrTxCommit, errors.ErrTxRollback)
}

func TestErrors_IsCompatible(t *testing.T) {
	// Обёрнутые ошибки совместимы с errors.Is.
	// wrapped := errors.New(errors.ErrTxBegin.Error())
	// _ = wrapped // просто убедимся что конструируется

	// Sentinel сами по себе работают с errors.Is.
	assert.ErrorIs(t, errors.ErrTxBegin, errors.ErrTxBegin)
	assert.ErrorIs(t, errors.ErrTxCommit, errors.ErrTxCommit)
	assert.ErrorIs(t, errors.ErrTxRollback, errors.ErrTxRollback)
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
		return errors.New(errors.ErrUnknown, "fn error")
	})

	// Хелпер настроен вернуть nil независимо от результата fn.
	require.NoError(t, err)
	mgr.AssertExpectations(t)
}

func TestMockTxManager_WithTransactionalRollback_FnCalledAndErrReturned(t *testing.T) {
	mgr := tx.NewMockTxManager(nil)
	ctx := context.Background()
	expectedErr := errors.New(errors.ErrUnknown, "service error")
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

	require.ErrorIs(t, err, errors.ErrTxBegin)
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

	require.ErrorIs(t, err, errors.ErrTxCommit)
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

	require.ErrorIs(t, err, errors.ErrTxRollback)
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
	mgr.On("Begin", ctx).Return(ctx, errors.ErrTxBegin)

	_, err := mgr.Begin(ctx)
	require.ErrorIs(t, err, errors.ErrTxBegin)
	mgr.AssertExpectations(t)
}

func TestMockTxManager_Commit_ReturnsErrWhenSetUp(t *testing.T) {
	mgr := tx.NewMockTxManager(nil)
	ctx := context.Background()
	mgr.On("Commit", ctx).Return(errors.ErrTxCommit)

	err := mgr.Commit(ctx)
	require.ErrorIs(t, err, errors.ErrTxCommit)
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
