//go:build integration

package tx_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/ssoeasy-dev/pkg/db/tx"
	"github.com/ssoeasy-dev/pkg/errors"
	l "github.com/ssoeasy-dev/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testcontainerspg "github.com/testcontainers/testcontainers-go/modules/postgres"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestLogger() l.Logger {
    return l.NewLogger(l.EnvironmentTest, "tx-test")
}

// ─── Test model ───────────────────────────────────────────────────────────────

type TxArticle struct {
	ID    uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Title string    `gorm:"not null"`
}

func (TxArticle) TableName() string { return "tx_articles" }

// ─── Setup ────────────────────────────────────────────────────────────────────

func setupTxPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	ctx := context.Background()

	ctr, err := testcontainerspg.Run(ctx,
		"postgres:16-alpine",
		testcontainerspg.WithDatabase("testdb"),
		testcontainerspg.WithUsername("test"),
		testcontainerspg.WithPassword("test"),
		testcontainerspg.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(&TxArticle{}))
	return db
}

// ─── WithTransaction ──────────────────────────────────────────────────────────

func TestTxManager_WithTransaction_Commit(t *testing.T) {
	db := setupTxPostgres(t)
	log := newTestLogger()
	mgr := tx.NewTxManager(db, log)

	id := uuid.New()
	err := mgr.WithTransaction(context.Background(), func(ctx context.Context) error {
		return mgr.GetDB(ctx).Create(&TxArticle{ID: id, Title: "committed"}).Error
	})
	require.NoError(t, err)

	var a TxArticle
	require.NoError(t, db.First(&a, "id = ?", id.String()).Error)
	assert.Equal(t, "committed", a.Title)
}

func TestTxManager_WithTransaction_Rollback_OnFnError(t *testing.T) {
	db := setupTxPostgres(t)
	log := newTestLogger()
	mgr := tx.NewTxManager(db, log)

	id := uuid.New()
	expectedErr := errors.New(errors.ErrInternal, "intentional error")
	err := mgr.WithTransaction(context.Background(), func(ctx context.Context) error {
		_ = mgr.GetDB(ctx).Create(&TxArticle{ID: id, Title: "should rollback"})
		return expectedErr
	})
	require.ErrorIs(t, err, expectedErr)

	var a TxArticle
	result := db.First(&a, "id = ?", id.String())
	require.ErrorIs(t, result.Error, gorm.ErrRecordNotFound)
}

func TestTxManager_WithTransaction_Rollback_OnPanic(t *testing.T) {
	db := setupTxPostgres(t)
	log := newTestLogger()
	mgr := tx.NewTxManager(db, log)

	id := uuid.New()
	assert.Panics(t, func() {
		_ = mgr.WithTransaction(context.Background(), func(ctx context.Context) error {
			_ = mgr.GetDB(ctx).Create(&TxArticle{ID: id, Title: "should rollback"})
			panic("test panic")
		})
	})

	var a TxArticle
	result := db.First(&a, "id = ?", id.String())
	require.ErrorIs(t, result.Error, gorm.ErrRecordNotFound)
}

func TestTxManager_WithTransaction_NestedWorkIsVisible_InsideTx(t *testing.T) {
	// Создаём две записи в одной транзакции — обе должны быть видны внутри той же tx.
	db := setupTxPostgres(t)
	log := newTestLogger()
	mgr := tx.NewTxManager(db, log)

	id1, id2 := uuid.New(), uuid.New()
	err := mgr.WithTransaction(context.Background(), func(ctx context.Context) error {
		txDB := mgr.GetDB(ctx)
		if err := txDB.Create(&TxArticle{ID: id1, Title: "first"}).Error; err != nil {
			return err
		}
		// Вторая запись видит первую внутри той же транзакции.
		var count int64
		txDB.Model(&TxArticle{}).Where("id = ?", id1.String()).Count(&count)
		assert.Equal(t, int64(1), count)
		return txDB.Create(&TxArticle{ID: id2, Title: "second"}).Error
	})
	require.NoError(t, err)

	var count int64
	db.Model(&TxArticle{}).Where("id IN ?", []string{id1.String(), id2.String()}).Count(&count)
	assert.Equal(t, int64(2), count)
}

// ─── Begin / Commit / Rollback manual ─────────────────────────────────────────

func TestTxManager_Begin_Commit(t *testing.T) {
	db := setupTxPostgres(t)
	log := newTestLogger()
	mgr := tx.NewTxManager(db, log)
	ctx := context.Background()

	txCtx, err := mgr.Begin(ctx)
	require.NoError(t, err)

	id := uuid.New()
	require.NoError(t, mgr.GetDB(txCtx).Create(&TxArticle{ID: id, Title: "manual commit"}).Error)

	require.NoError(t, mgr.Commit(txCtx))

	var a TxArticle
	require.NoError(t, db.First(&a, "id = ?", id.String()).Error)
	assert.Equal(t, "manual commit", a.Title)
}

func TestTxManager_Begin_Rollback(t *testing.T) {
	db := setupTxPostgres(t)
	log := newTestLogger()
	mgr := tx.NewTxManager(db, log)
	ctx := context.Background()

	txCtx, err := mgr.Begin(ctx)
	require.NoError(t, err)

	id := uuid.New()
	require.NoError(t, mgr.GetDB(txCtx).Create(&TxArticle{ID: id, Title: "manual rollback"}).Error)

	require.NoError(t, mgr.Rollback(txCtx))

	var a TxArticle
	result := db.First(&a, "id = ?", id.String())
	require.ErrorIs(t, result.Error, gorm.ErrRecordNotFound)
}

// ─── GetDB ────────────────────────────────────────────────────────────────────

func TestTxManager_GetDB_WithoutTx_ReturnsBaseDB(t *testing.T) {
	db := setupTxPostgres(t)
	log := newTestLogger()
	mgr := tx.NewTxManager(db, log)

	result := mgr.GetDB(context.Background())
	require.NotNil(t, result)

	// Запись через GetDB без транзакции должна персистироваться сразу.
	id := uuid.New()
	require.NoError(t, result.Create(&TxArticle{ID: id, Title: "no tx"}).Error)

	var a TxArticle
	require.NoError(t, db.First(&a, "id = ?", id.String()).Error)
	assert.Equal(t, "no tx", a.Title)
}

func TestTxManager_GetDB_WithTx_ReturnsTxDB(t *testing.T) {
	db := setupTxPostgres(t)
	log := newTestLogger()
	mgr := tx.NewTxManager(db, log)
	ctx := context.Background()

	txCtx, err := mgr.Begin(ctx)
	require.NoError(t, err)

	txDB := mgr.GetDB(txCtx)
	assert.NotNil(t, txDB)

	// Запись сделана через txDB — до коммита не видна снаружи.
	id := uuid.New()
	require.NoError(t, txDB.Create(&TxArticle{ID: id, Title: "in tx"}).Error)

	// Снаружи транзакции запись не видна.
	var a TxArticle
	result := db.First(&a, "id = ?", id.String())
	require.ErrorIs(t, result.Error, gorm.ErrRecordNotFound)

	// После коммита — видна.
	require.NoError(t, mgr.Commit(txCtx))
	require.NoError(t, db.First(&a, "id = ?", id.String()).Error)
}

// ─── Edge cases ───────────────────────────────────────────────────────────────

func TestTxManager_Commit_NoTx_ReturnsErrCommit(t *testing.T) {
	db := setupTxPostgres(t)
	log := newTestLogger()
	mgr := tx.NewTxManager(db, log)

	err := mgr.Commit(context.Background())
	require.ErrorIs(t, err, errors.ErrInternal)
}

func TestTxManager_Rollback_NoTx_ReturnsErrRollback(t *testing.T) {
	db := setupTxPostgres(t)
	log := newTestLogger()
	mgr := tx.NewTxManager(db, log)

	err := mgr.Rollback(context.Background())
	require.ErrorIs(t, err, errors.ErrInternal)
}
