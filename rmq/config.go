package rmq

import "fmt"

// Config содержит параметры подключения к RabbitMQ.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	VHost    string
}

// URL возвращает AMQP URL для подключения.
func (c *Config) URL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%s/%s",
		c.User, c.Password, c.Host, c.Port, c.VHost,
	)
}
