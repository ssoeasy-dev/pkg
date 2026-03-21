//go:build integration

package db_test

import (
	"context"
	"testing"

	"github.com/ssoeasy-dev/pkg/db"
	"github.com/ssoeasy-dev/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testcontainerspg "github.com/testcontainers/testcontainers-go/modules/postgres"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

func setupDB(t *testing.T) *db.DB {
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

	// Открываем через gorm напрямую по DSN — не нужно парсить URL обратно в поля Config.
	conn, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Silent),
	})
	require.NoError(t, err)

	database := &db.DB{Conn: conn}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestNewDB_ConnectsSuccessfully(t *testing.T) {
	database := setupDB(t)
	require.NotNil(t, database.Conn)
}

func TestNewDB_InvalidDSN_ReturnsError(t *testing.T) {
	cfg := &db.Config{
		Environment: db.EnvironmentTest,
		Host:        "localhost",
		Port:        "1", // заведомо недоступный порт
		User:        "user",
		Password:    "pass",
		Database:    "db",
		SSLMode:     "disable",
	}
	log := logger.NewLogger(logger.EnvironmentTest, "test")

	_, err := db.NewDB(cfg, log)
	require.Error(t, err)
}

func TestDB_Ping(t *testing.T) {
	database := setupDB(t)
	require.NoError(t, database.Ping())
}

func TestDB_Close(t *testing.T) {
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

	conn, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Silent),
	})
	require.NoError(t, err)
	database := &db.DB{Conn: conn}

	require.NoError(t, database.Close())
	// После закрытия Ping должен вернуть ошибку.
	assert.Error(t, database.Ping())
}

func TestNewDB_PoolSettings_Applied(t *testing.T) {
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

	// Получаем хост и порт из контейнера напрямую.
	host, err := ctr.Host(ctx)
	require.NoError(t, err)
	port, err := ctr.MappedPort(ctx, "5432")
	require.NoError(t, err)

	cfg := &db.Config{
		Environment:  db.EnvironmentTest,
		Host:         host,
		Port:         port.Port(),
		User:         "test",
		Password:     "test",
		Database:     "testdb",
		SSLMode:      "disable",
		MaxIdleConns: 3,
		MaxOpenConns: 7,
	}
	log := logger.NewLogger(logger.EnvironmentTest, "test")

	database, err := db.NewDB(cfg, log)
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })

	sqlDB, err := database.Conn.DB()
	require.NoError(t, err)
	assert.Equal(t, 7, sqlDB.Stats().MaxOpenConnections)
}
