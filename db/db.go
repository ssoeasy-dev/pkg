package db

import (
	"context"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"github.com/ssoeasy-dev/pkg/logger"
	gormLogger "gorm.io/gorm/logger"
)

type DB struct {
	Conn *gorm.DB
	log  *logger.Logger
}

func NewDB(cfg *Config, log *logger.Logger) (*DB, error) {
	gormCfg := &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Silent),
	}

	if cfg.Environment == "development" {
		gormCfg.Logger = gormLogger.Default.LogMode(gormLogger.Info)
	}

	conn, err := gorm.Open(postgres.Open(cfg.DSN()), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := conn.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	log.Info(context.Background(), "Database connected successfully", map[string]any{
		"host":     cfg.Host,
		"port":     cfg.Port,
		"user":     cfg.User,
		"database": cfg.Database,
	})

	return &DB{
		Conn: conn,
		log:  log,
	}, nil
}

func (d *DB) Close() error {
	sqlDB, err := d.Conn.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

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
