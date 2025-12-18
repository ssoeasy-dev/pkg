package rmq

import "fmt"

type Config struct {
	Host       string
	Port       string
	User       string
	Password   string
}

func (c *Config) URL() string {
	return fmt.Sprintf(
		"amqp://%s:%s@%s:%s/",
		c.User, c.Password, c.Host, c.Port,
	)
}