package db_test

import (
	"testing"

	"github.com/ssoeasy-dev/pkg/db"
	"github.com/stretchr/testify/assert"
)

// ─── Environment ──────────────────────────────────────────────────────────────

func TestEnvironment_IsVerbose(t *testing.T) {
	assert.True(t, db.EnvironmentLocal.IsVerbose())
	assert.True(t, db.EnvironmentDevelopment.IsVerbose())
	assert.False(t, db.EnvironmentProduction.IsVerbose())
	assert.False(t, db.EnvironmentTest.IsVerbose())
}

// ─── Config ───────────────────────────────────────────────────────────────────

func TestConfig_DSN(t *testing.T) {
	cfg := &db.Config{
		Host:     "localhost",
		Port:     "5432",
		User:     "user",
		Password: "pass",
		Database: "mydb",
		SSLMode:  "disable",
	}
	assert.Equal(t,
		"host=localhost port=5432 user=user password=pass dbname=mydb sslmode=disable",
		cfg.DSN(),
	)
}

func TestConfig_PoolDefaults(t *testing.T) {
	// Нулевые значения дают дефолты.
	cfg := &db.Config{}
	assert.Equal(t, 10, cfg.MaxIdleConnsOrDefault())
	assert.Equal(t, 100, cfg.MaxOpenConnsOrDefault())
}

func TestConfig_PoolCustom(t *testing.T) {
	cfg := &db.Config{MaxIdleConns: 5, MaxOpenConns: 50}
	assert.Equal(t, 5, cfg.MaxIdleConnsOrDefault())
	assert.Equal(t, 50, cfg.MaxOpenConnsOrDefault())
}
