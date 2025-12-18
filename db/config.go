package db

import "fmt"

type Environment string

const (
	EnvironmentLocal       Environment = "local"
	EnvironmentTest        Environment = "test"
	EnvironmentDevelopment Environment = "development"
	EnvironmentProduction  Environment = "production"
)

type Config struct {
	Environment string
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}