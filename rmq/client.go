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
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	log.Info(context.Background(), "RabbitMQ client connected", map[string]any{
		"host": cfg.Host,
		"port": cfg.Port,
		"user": cfg.User,
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
		c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) Reconnect() error {
	ctx := context.Background()
	c.log.Warn(ctx, "Reconnecting to RabbitMQ", nil)

	if err := c.Close(); err != nil {
		c.log.Warn(ctx, "Error closing old connection", map[string]any{"error": err})
	}

	time.Sleep(2 * time.Second)

	conn, err := amqp091.Dial(c.cfg.URL())
	if err != nil {
		return fmt.Errorf("failed to reconnect: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	c.conn = conn
	c.channel = channel

	c.log.Info(ctx, "RabbitMQ reconnected successfully", nil)
	return nil
}
