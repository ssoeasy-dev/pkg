package db

import "fmt"

// Environment определяет окружение запуска приложения.
type Environment string

const (
	EnvironmentLocal       Environment = "local"
	EnvironmentTest        Environment = "test"
	EnvironmentDevelopment Environment = "development"
	EnvironmentProduction  Environment = "production"
)

// IsVerbose возвращает true для окружений где нужен подробный лог запросов.
func (e Environment) IsVerbose() bool {
	return e == EnvironmentLocal || e == EnvironmentDevelopment
}

// Config содержит параметры подключения к базе данных.
type Config struct {
	Environment Environment
	Host        string
	Port        string
	User        string
	Password    string
	Database    string
	SSLMode     string
	// MaxIdleConns — максимальное количество простаивающих соединений в пуле.
	// 0 означает использование значения по умолчанию (10).
	MaxIdleConns int
	// MaxOpenConns — максимальное количество открытых соединений.
	// 0 означает использование значения по умолчанию (100).
	MaxOpenConns int
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

func (c *Config) MaxIdleConnsOrDefault() int {
	if c.MaxIdleConns > 0 {
		return c.MaxIdleConns
	}
	return 10
}

func (c *Config) MaxOpenConnsOrDefault() int {
	if c.MaxOpenConns > 0 {
		return c.MaxOpenConns
	}
	return 100
}
