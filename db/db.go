package db

import (
	"context"
	"fmt"

	"github.com/ssoeasy-dev/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// DB обёртка над gorm.DB с управлением пулом соединений.
// Conn публичен намеренно — используется в TxManager и репозиториях.
type DB struct {
	Conn *gorm.DB
}

// NewDB создаёт подключение к PostgreSQL.
// Уровень логирования GORM определяется через cfg.Environment.
func NewDB(cfg *Config, log *logger.Logger) (*DB, error) {
	logLevel := gormLogger.Silent
	if cfg.Environment.IsVerbose() {
		logLevel = gormLogger.Info
	}

	conn, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: gormLogger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := conn.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConnsOrDefault())
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConnsOrDefault())

	log.Info(context.Background(), "Database connected successfully", map[string]any{
		"host":     cfg.Host,
		"port":     cfg.Port,
		"user":     cfg.User,
		"database": cfg.Database,
	})

	return &DB{Conn: conn}, nil
}

// Ping проверяет доступность базы данных.
func (d *DB) Ping() error {
	sqlDB, err := d.Conn.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	return nil
}

// Close закрывает пул соединений.
func (d *DB) Close() error {
	sqlDB, err := d.Conn.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
