package rmq

import (
	"context"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/ssoeasy-dev/pkg/logger"
)

// Client управляет AMQP-соединением с RabbitMQ.
// Один Client — одно соединение. Consumer создаёт собственные каналы через это соединение.
type Client struct {
	conn *amqp091.Connection
	cfg  *Config
	log  logger.Logger
}

// NewClient создаёт и возвращает Client с открытым соединением.
func NewClient(log logger.Logger, cfg *Config) (*Client, error) {
	conn, err := amqp091.Dial(cfg.URL())
	if err != nil {
		return nil, newError(ErrConnect, "failed to connect to RabbitMQ", err)
	}

	log.Info(context.Background(), "RabbitMQ client connected", map[string]any{
		"host":  cfg.Host,
		"port":  cfg.Port,
		"user":  cfg.User,
		"vhost": cfg.VHost,
	})

	return &Client{conn: conn, cfg: cfg, log: log}, nil
}

// Close закрывает соединение. Безопасно вызывать повторно.
func (c *Client) Close() error {
	if c.conn != nil && !c.conn.IsClosed() {
		if err := c.conn.Close(); err != nil {
			return fmt.Errorf("failed to close connection: %w", err)
		}
	}
	return nil
}

// Reconnect закрывает текущее соединение и устанавливает новое с экспоненциальным backoff.
// Возвращает ErrConnect если все попытки исчерпаны.
func (c *Client) Reconnect() error {
	ctx := context.Background()
	c.log.Warn(ctx, "Reconnecting to RabbitMQ", nil)

	if err := c.Close(); err != nil {
		c.log.Warn(ctx, "Error closing old connection during reconnect", map[string]any{"error": err})
	}

	const (
		maxAttempts = 10
		maxBackoff  = 30 * time.Second
	)

	backoff := 2 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		c.log.Info(ctx, "Reconnect attempt", map[string]any{
			"attempt": attempt,
			"max":     maxAttempts,
		})

		conn, err := amqp091.Dial(c.cfg.URL())
		if err != nil {
			c.log.Warn(ctx, "Reconnect failed, retrying", map[string]any{
				"attempt": attempt,
				"error":   err,
				"backoff": backoff.String(),
			})
			time.Sleep(backoff)
			if backoff < maxBackoff {
				backoff *= 2
			}
			continue
		}

		c.conn = conn
		c.log.Info(ctx, "RabbitMQ reconnected successfully", map[string]any{"attempt": attempt})
		return nil
	}

	return newError(ErrConnect, fmt.Sprintf("failed to reconnect after %d attempts", maxAttempts), nil)
}