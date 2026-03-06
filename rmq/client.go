package rmq

import (
	"context"
	"fmt"
	"time"

	"github.com/ssoeasy-dev/pkg/logger"

	"github.com/rabbitmq/amqp091-go"
)

type Client struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
	cfg     *Config
	log     *logger.Logger
}

func NewClient(log *logger.Logger, cfg *Config) (*Client, error) {
	conn, err := amqp091.Dial(cfg.URL())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		if err := conn.Close(); err != nil {
			return nil, fmt.Errorf("failed to close channel after failing open channel: %w", err)
		}
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	log.Info(context.Background(), "RabbitMQ client connected", map[string]any{
		"host":  cfg.Host,
		"port":  cfg.Port,
		"user":  cfg.User,
		"vhost": cfg.VHost,
	})

	return &Client{
		conn:    conn,
		channel: channel,
		cfg:     cfg,
		log:     log,
	}, nil
}

func (c *Client) Channel() *amqp091.Channel {
	return c.channel
}

func (c *Client) Close() error {
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			return fmt.Errorf("failed to close channel: %w", err)
		}
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return fmt.Errorf("failed to close channel: %w", err)
		}
	}
	return nil
}

func (c *Client) Reconnect() error {
	ctx := context.Background()
	c.log.Warn(ctx, "Reconnecting to RabbitMQ", nil)

	if err := c.Close(); err != nil {
		c.log.Warn(ctx, "Error closing old connection", map[string]any{"error": err})
	}

	backoff := 2 * time.Second
	const maxBackoff = 30 * time.Second
	const maxAttempts = 10

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

		channel, err := conn.Channel()
		if err != nil {
			if err := conn.Close(); err != nil {
				return fmt.Errorf("failed to close channel after failing reconnect: %w", err)
			}
			time.Sleep(backoff)
			if backoff < maxBackoff {
				backoff *= 2
			}
			continue
		}

		c.conn = conn
		c.channel = channel
		c.log.Info(ctx, "RabbitMQ reconnected successfully", map[string]any{
			"attempt": attempt,
		})
		return nil
	}

	return fmt.Errorf("failed to reconnect after %d attempts", maxAttempts)
}
