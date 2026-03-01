package rmq

import (
	"maps"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ssoeasy-dev/pkg/logger"

	"github.com/rabbitmq/amqp091-go"
)

const (
	retryHeader       = "x-retry-count"
	traceIDHeader     = "x-trace-id"
	requestIDHeader   = "x-request-id"
	errorReasonHeader = "x-error-reason"
	dlqReasonHeader   = "x-dlq-reason"
)

type MessageHandler func(ctx context.Context, message []byte, routingKey string) error

type QueueConfig struct {
	Queue          string
	BindingPattern string
	Exchange       string
	TTL            int
}

type DelayQueueConfig struct {
	QueueConfig
	MaxRetry int
}

type ConsumerConfig struct {
	Main    QueueConfig
	Delay   *DelayQueueConfig
	Dead    *QueueConfig
	Handler MessageHandler
}

type Consumer struct {
	log            *logger.Logger
	client         *Client
	cfg            *ConsumerConfig
	publishChannel *amqp091.Channel
	isStopped      bool
	stopOnce       sync.Once
	stopped        chan struct{}
	mu             sync.RWMutex
}

func NewConsumer(log *logger.Logger, client *Client, cfg *ConsumerConfig) (*Consumer, error) {
	if cfg.Main.Queue == "" {
		return nil, fmt.Errorf("main queue name is required")
	}
	if cfg.Main.Exchange == "" {
		return nil, fmt.Errorf("main exchange is required")
	}
	if cfg.Main.BindingPattern == "" {
		return nil, fmt.Errorf("main binding pattern is required")
	}
	if cfg.Handler == nil {
		return nil, fmt.Errorf("handler is required")
	}
	if cfg.Delay != nil {
		if cfg.Delay.Queue == "" {
			return nil, fmt.Errorf("delay queue name is required")
		}
		if cfg.Delay.Exchange == "" {
			return nil, fmt.Errorf("delay exchange is required")
		}
		if cfg.Delay.BindingPattern == "" {
			return nil, fmt.Errorf("delay binding pattern is required")
		}
		if cfg.Delay.TTL <= 0 {
			return nil, fmt.Errorf("delay TTL must be positive")
		}
		if cfg.Delay.MaxRetry <= 0 {
			return nil, fmt.Errorf("delay MaxRetry must be positive")
		}
	}
	if cfg.Dead != nil {
		if cfg.Dead.Queue == "" {
			return nil, fmt.Errorf("dead queue name is required")
		}
		if cfg.Dead.Exchange == "" {
			return nil, fmt.Errorf("dead exchange is required")
		}
		if cfg.Dead.BindingPattern == "" {
			return nil, fmt.Errorf("dead binding pattern is required")
		}
	}

	return &Consumer{
		log:       log,
		client:    client,
		cfg:       cfg,
		stopped:   make(chan struct{}),
		isStopped: false,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	initChannel, err := c.client.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open init channel: %w", err)
	}
	defer func() {
		err := initChannel.Close()
		if err != nil {
			c.log.Error(ctx, "Close init channel", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	if err := c.initializeInfrastructure(ctx, initChannel); err != nil {
		return fmt.Errorf("failed to initialize RabbitMQ infrastructure: %w", err)
	}

	c.publishChannel, err = c.client.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open publish channel: %w", err)
	}

	consumeChannel, err := c.client.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open consume channel: %w", err)
	}

	if err := consumeChannel.Qos(1, 0, false); err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	hostname, _ := os.Hostname()
	consumerTag := fmt.Sprintf("%s-%d", hostname, os.Getpid())

	msgs, err := consumeChannel.Consume(
		c.cfg.Main.Queue,
		consumerTag,
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	c.log.Info(ctx, "Consumer started", map[string]any{
		"queue":        c.cfg.Main.Queue,
		"consumer_tag": consumerTag,
		"hostname":     hostname,
	})

	go c.processMessages(ctx, msgs)

	return nil
}

func (c *Consumer) processMessages(ctx context.Context, msgs <-chan amqp091.Delivery) {
	for {
		select {
		case <-ctx.Done():
			c.log.Info(ctx, "Consumer context cancelled, stopping", nil)
			return
		case <-c.stopped:
			c.log.Info(ctx, "Consumer stopped, exiting", nil)
			return
		case msg, ok := <-msgs:
			if !ok {
				c.mu.RLock()
				stopped := c.isStopped
				c.mu.RUnlock()

				if stopped {
					c.log.Info(ctx, "Consumer stopped, not reconnecting", nil)
					return
				}

				select {
				case <-ctx.Done():
					c.log.Info(ctx, "Context cancelled during reconnect attempt, stopping", nil)
					return
				case <-c.stopped:
					c.log.Info(ctx, "Consumer stopped during reconnect attempt, stopping", nil)
					return
				default:
					c.log.Warn(ctx, "Message channel closed, attempting reconnect", nil)
					if err := c.reconnectAndRestart(ctx); err != nil {
						c.log.Error(ctx, "Failed to reconnect", map[string]any{"error": err})
						return
					}
					return
				}
			}

			c.handleMessage(ctx, msg)
		}
	}
}

func (c *Consumer) extractTraceContext(ctx context.Context, headers amqp091.Table) context.Context {
	if headers == nil {
		return ctx
	}

	if traceID, ok := headers[traceIDHeader]; ok {
		if str, ok := traceID.(string); ok && str != "" {
			ctx = context.WithValue(ctx, logger.TraceIDKey, str)
		}
	}

	if requestID, ok := headers[requestIDHeader]; ok {
		if str, ok := requestID.(string); ok && str != "" {
			ctx = context.WithValue(ctx, logger.RequestIDKey, str)
		}
	}

	return ctx
}

func (c *Consumer) handleMessage(ctx context.Context, msg amqp091.Delivery) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	ctx = c.extractTraceContext(ctx, msg.Headers)

	c.log.Info(ctx, "Received message", map[string]any{
		"routing_key": msg.RoutingKey,
		"message_id":  msg.MessageId,
		"message":     string(msg.Body),
	})

	if err := c.cfg.Handler(ctx, msg.Body, msg.RoutingKey); err != nil {
		c.log.Error(ctx, "Failed to process message", map[string]any{
			"error":       err,
			"routing_key": msg.RoutingKey,
			"message_id":  msg.MessageId,
		})

		if c.cfg.Delay == nil {
			c.log.Error(ctx, "Delay queue not set, sending to DLQ", map[string]any{
				"message_id": msg.MessageId,
			})
			c.sendToDLQOrNack(ctx, msg, err, "delay_queue_not_configured")
			return
		}

		retryCount := c.getRetryCount(msg.Headers)

		if retryCount >= c.cfg.Delay.MaxRetry {
			c.log.Error(ctx, "Max retries exceeded, sending to DLQ", map[string]any{
				"message_id":  msg.MessageId,
				"retry_count": retryCount,
			})
			c.sendToDLQOrNack(ctx, msg, err, fmt.Sprintf("max_retries_exceeded:%d", retryCount))
			return
		}

		newRetryCount := retryCount + 1
		c.log.Warn(ctx, "Retrying message via delay queue", map[string]any{
			"message_id":  msg.MessageId,
			"retry_count": newRetryCount,
		})

		if err := c.publishToDelayQueue(msg, newRetryCount, err); err != nil {
			c.log.Error(ctx, "Failed to publish to delay queue, sending to DLQ", map[string]any{
				"error":      err,
				"message_id": msg.MessageId,
			})
			c.sendToDLQOrNack(ctx, msg, err, "failed_to_publish_to_delay_queue")
			return
		}

		c.ackMessage(ctx, msg)
		return
	}

	c.ackMessage(ctx, msg)
	c.log.Info(ctx, "Message processed successfully", map[string]any{
		"routing_key": msg.RoutingKey,
		"message_id":  msg.MessageId,
	})
}

func (c *Consumer) sendToDLQOrNack(ctx context.Context, msg amqp091.Delivery, originalErr error, reason string) {
	if err := c.publishToDLQ(ctx, msg, originalErr, reason); err != nil {
		c.log.Error(ctx, "Failed to publish to DLQ, using standard dead-letter (nack)", map[string]any{
			"error":      err,
			"message_id": msg.MessageId,
		})
		if err := msg.Nack(false, false); err != nil {
			c.log.Error(ctx, "Failed to nack message", map[string]any{
				"error":      err,
				"message_id": msg.MessageId,
			})
		}
		return
	}

	c.ackMessage(ctx, msg)
}

func (c *Consumer) ackMessage(ctx context.Context, msg amqp091.Delivery) {
	if err := msg.Ack(false); err != nil {
		c.log.Error(ctx, "Failed to ack message", map[string]any{
			"error":      err,
			"message_id": msg.MessageId,
		})
	}
}

func (c *Consumer) getRetryCount(headers amqp091.Table) int {
	if headers == nil {
		return 0
	}

	if retryCount, ok := headers[retryHeader]; ok {
		switch count := retryCount.(type) {
		case int32:
			return int(count)
		case int:
			return count
		case int64:
			return int(count)
		}
	}

	return 0
}

func (c *Consumer) publishToDelayQueue(msg amqp091.Delivery, retryCount int, err error) error {
	headers := make(amqp091.Table, len(msg.Headers))
	maps.Copy(headers, msg.Headers)
	headers[retryHeader] = retryCount
	if err != nil {
		headers["x-reason"] = err.Error()
	}

	return c.publishChannel.Publish(
		c.cfg.Delay.Exchange,
		msg.RoutingKey,
		false, false,
		amqp091.Publishing{
			Headers:      headers,
			ContentType:  msg.ContentType,
			Body:         msg.Body,
			MessageId:    msg.MessageId,
			DeliveryMode: amqp091.Persistent,
		},
	)
}

func (c *Consumer) prepareDLQHeaders(msg amqp091.Delivery, originalErr error, reason string) amqp091.Table {
	headers := make(amqp091.Table, len(msg.Headers))
	for k, v := range msg.Headers {
		headers[k] = v
	}

	if msg.RoutingKey != "" {
		headers["x-original-routing-key"] = msg.RoutingKey
	}
	if originalErr != nil {
		headers[errorReasonHeader] = originalErr.Error()
	}
	headers[dlqReasonHeader] = reason

	if retryCount := c.getRetryCount(msg.Headers); retryCount > 0 {
		headers[retryHeader] = retryCount
	}

	return headers
}

func (c *Consumer) publishToDLQ(ctx context.Context, msg amqp091.Delivery, originalErr error, reason string) error {
	if c.cfg.Dead == nil {
		return fmt.Errorf("dead letter queue not configured")
	}

	headers := c.prepareDLQHeaders(msg, originalErr, reason)

	if err := c.publishChannel.Publish(
		c.cfg.Dead.Exchange,
		msg.RoutingKey,
		false, false,
		amqp091.Publishing{
			Headers:      headers,
			ContentType:  msg.ContentType,
			Body:         msg.Body,
			MessageId:    msg.MessageId,
			DeliveryMode: amqp091.Persistent,
		},
	); err != nil {
		return fmt.Errorf("failed to publish to DLQ: %w", err)
	}

	c.log.Info(ctx, "Message published to DLQ", map[string]any{
		"message_id":     msg.MessageId,
		"routing_key":    msg.RoutingKey,
		"reason":         reason,
		"retry_count":    c.getRetryCount(msg.Headers),
		"original_error": originalErr,
	})

	return nil
}

func (c *Consumer) reconnectAndRestart(ctx context.Context) error {
	c.mu.RLock()
	stopped := c.isStopped
	c.mu.RUnlock()

	if stopped {
		return fmt.Errorf("consumer is stopped")
	}

	if err := c.client.Reconnect(); err != nil {
		return err
	}

	return c.Start(ctx)
}

func (c *Consumer) Stop() {
	c.stopOnce.Do(func() {
		c.mu.Lock()
		c.isStopped = true
		c.mu.Unlock()
		close(c.stopped)
		c.log.Info(context.Background(), "Consumer stop requested", nil)
	})
}

func (c *Consumer) initializeInfrastructure(ctx context.Context, channel *amqp091.Channel) error {
	// DEAD
	if c.cfg.Dead != nil {
		if err := channel.ExchangeDeclare(
			c.cfg.Dead.Exchange,
			"topic",
			true, false, false, false, nil,
		); err != nil {
			return fmt.Errorf("failed to declare DLX exchange %s: %w", c.cfg.Dead.Exchange, err)
		}
		c.log.Info(ctx, "DLX exchange declared", map[string]any{"exchange": c.cfg.Dead.Exchange})

		if _, err := channel.QueueDeclare(
			c.cfg.Dead.Queue,
			true, false, false, false, nil,
		); err != nil {
			return fmt.Errorf("failed to declare DLQ %s: %w", c.cfg.Dead.Queue, err)
		}
		c.log.Info(ctx, "DLQ declared", map[string]any{"queue": c.cfg.Dead.Queue})

		if err := channel.QueueBind(
			c.cfg.Dead.Queue,
			c.cfg.Dead.BindingPattern,
			c.cfg.Dead.Exchange,
			false, nil,
		); err != nil {
			return fmt.Errorf("failed to bind DLQ: %w", err)
		}
		c.log.Info(ctx, "DLQ bound to DLX", map[string]any{
			"queue":           c.cfg.Dead.Queue,
			"exchange":        c.cfg.Dead.Exchange,
			"binding_pattern": c.cfg.Dead.BindingPattern,
		})
	}

	// MAIN
	if err := channel.ExchangeDeclare(
		c.cfg.Main.Exchange,
		"topic",
		true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("failed to declare main exchange %s: %w", c.cfg.Main.Exchange, err)
	}
	c.log.Info(ctx, "Main exchange declared", map[string]any{"exchange": c.cfg.Main.Exchange})

	mainArgs := amqp091.Table{
		"x-message-ttl":  int64(c.cfg.Main.TTL),
		"x-max-priority": int32(10),
	}
	if c.cfg.Dead != nil {
		mainArgs["x-dead-letter-exchange"] = c.cfg.Dead.Exchange
	}

	if _, err := channel.QueueDeclare(
		c.cfg.Main.Queue,
		true, false, false, false, mainArgs,
	); err != nil {
		return fmt.Errorf("failed to declare main queue %s: %w", c.cfg.Main.Queue, err)
	}
	c.log.Info(ctx, "Main queue declared", map[string]any{"queue": c.cfg.Main.Queue})

	if err := channel.QueueBind(
		c.cfg.Main.Queue,
		c.cfg.Main.BindingPattern,
		c.cfg.Main.Exchange,
		false, nil,
	); err != nil {
		return fmt.Errorf("failed to bind main queue: %w", err)
	}
	c.log.Info(ctx, "Main queue bound", map[string]any{
		"queue":           c.cfg.Main.Queue,
		"exchange":        c.cfg.Main.Exchange,
		"binding_pattern": c.cfg.Main.BindingPattern,
	})

	// DELAY
	if c.cfg.Delay != nil {
		if err := channel.ExchangeDeclare(
			c.cfg.Delay.Exchange,
			"topic",
			true, false, false, false, nil,
		); err != nil {
			return fmt.Errorf("failed to declare delay exchange %s: %w", c.cfg.Delay.Exchange, err)
		}
		c.log.Info(ctx, "Delay exchange declared", map[string]any{"exchange": c.cfg.Delay.Exchange})

		delayArgs := amqp091.Table{
			"x-message-ttl":          int64(c.cfg.Delay.TTL),
			"x-dead-letter-exchange": c.cfg.Main.Exchange,
		}

		if _, err := channel.QueueDeclare(
			c.cfg.Delay.Queue,
			true, false, false, false, delayArgs,
		); err != nil {
			return fmt.Errorf("failed to declare delay queue %s: %w", c.cfg.Delay.Queue, err)
		}
		c.log.Info(ctx, "Delay queue declared", map[string]any{"queue": c.cfg.Delay.Queue})

		if err := channel.QueueBind(
			c.cfg.Delay.Queue,
			c.cfg.Delay.BindingPattern,
			c.cfg.Delay.Exchange,
			false, nil,
		); err != nil {
			return fmt.Errorf("failed to bind delay queue: %w", err)
		}
		c.log.Info(ctx, "Delay queue bound", map[string]any{
			"queue":           c.cfg.Delay.Queue,
			"exchange":        c.cfg.Delay.Exchange,
			"binding_pattern": c.cfg.Delay.BindingPattern,
		})
	}

	return nil
}

func UnmarshalMessage(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
