package rmq

import (
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
	RoutingKey     string
	Exchange       string
	TTL            int
}

type DelayQueueConfig struct {
	Queue    string
	MaxRetry int
	TTL      int
}

type ConsumerConfig struct {
	Main    QueueConfig
	Delay   *DelayQueueConfig
	Dead    *QueueConfig
	Handler MessageHandler
}
type Consumer struct {
	log       *logger.Logger
	client    *Client
	cfg       *ConsumerConfig
	isStopped bool
	stopOnce  sync.Once
	stopped   chan struct{}
	mu        sync.RWMutex
}

func NewConsumer(log *logger.Logger, client *Client, consumerCfg *ConsumerConfig) *Consumer {
	return &Consumer{
		log:       log,
		client:    client,
		cfg:       consumerCfg,
		stopped:   make(chan struct{}),
		isStopped: false,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	if err := c.initializeInfrastructure(ctx); err != nil {
		return fmt.Errorf("failed to initialize RabbitMQ infrastructure: %w", err)
	}

	hostname, _ := os.Hostname()
	consumerTag := fmt.Sprintf("%s-%d", hostname, os.Getpid())

	msgs, err := c.client.Channel().Consume(
		c.cfg.Main.Queue, // queue
		consumerTag,      // consumer tag
		false,            // auto-ack
		false,            // exclusive
		false,            // no-local
		false,            // no-wait
		nil,              // args
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
			if err := c.publishToDLQ(ctx, msg, err, "delay_queue_not_configured"); err != nil {
				c.log.Error(ctx, "Failed to publish message to DLQ, using standard dead-letter", map[string]any{
					"error":      err,
					"message_id": msg.MessageId,
				})
				if err := msg.Nack(false, false); err != nil {
					c.log.Error(ctx, "Failed to nack message", map[string]any{
						"error":      err,
						"message_id": msg.MessageId,
					})
				}

			} else {
				if err := msg.Ack(false); err != nil {
					c.log.Error(ctx, "Failed to ack message", map[string]any{
						"error":      err,
						"message_id": msg.MessageId,
					})
				}
			}
			return
		}

		retryCount := c.getRetryCount(msg.Headers)

		if retryCount >= c.cfg.Delay.MaxRetry {
			c.log.Error(ctx, "Max retries exceeded, sending to DLQ", map[string]any{
				"message_id":  msg.MessageId,
				"retry_count": retryCount,
			})
			if err := c.publishToDLQ(ctx, msg, err, fmt.Sprintf("max_retries_exceeded:%d", retryCount)); err != nil {
				c.log.Error(ctx, "Failed to publish message to DLQ, using standard dead-letter", map[string]any{
					"error":      err,
					"message_id": msg.MessageId,
				})
				if err := msg.Nack(false, false); err != nil {
					c.log.Error(ctx, "Failed to nack message", map[string]any{
						"error":      err,
						"message_id": msg.MessageId,
					})
				}
			} else {
				if err := msg.Ack(false); err != nil {
					c.log.Error(ctx, "Failed to ack message", map[string]any{
						"error":      err,
						"message_id": msg.MessageId,
					})
				}
			}
			return
		}

		newRetryCount := retryCount + 1
		c.log.Warn(ctx, "Retrying message via delay queue", map[string]any{
			"message_id":  msg.MessageId,
			"retry_count": newRetryCount,
		})

		if err := c.publishToDelayQueue(msg, newRetryCount, err); err != nil {
			c.log.Error(ctx, "Failed to publish message to delay queue", map[string]any{
				"error":      err,
				"message_id": msg.MessageId,
			})
			if err := c.publishToDLQ(ctx, msg, err, "failed_to_publish_to_delay_queue"); err != nil {
				c.log.Error(ctx, "Failed to publish message to DLQ, using standard dead-letter", map[string]any{
					"error":      err,
					"message_id": msg.MessageId,
				})
				if err := msg.Nack(false, false); err != nil {
					c.log.Error(ctx, "Failed to nack message", map[string]any{
						"error":      err,
						"message_id": msg.MessageId,
					})
				}
			} else {
				if err := msg.Ack(false); err != nil {
					c.log.Error(ctx, "Failed to ack message", map[string]any{
						"error":      err,
						"message_id": msg.MessageId,
					})
				}
			}
			return
		}

		if err := msg.Ack(false); err != nil {
			c.log.Error(ctx, "Failed to ack message", map[string]any{
				"error":      err,
				"message_id": msg.MessageId,
			})
		}
		return
	}

	if err := msg.Ack(false); err != nil {
		c.log.Error(ctx, "Failed to acknowledge message", map[string]any{
			"error":      err,
			"message_id": msg.MessageId,
		})
	} else {
		c.log.Info(ctx, "Message processed successfully", map[string]any{
			"routing_key": msg.RoutingKey,
			"message_id":  msg.MessageId,
		})
	}
}

func (c *Consumer) getRetryCount(headers amqp091.Table) int {
	if headers == nil {
		return 0
	}

	if retryCount, ok := headers[retryHeader]; ok {
		if count, ok := retryCount.(int32); ok {
			return int(count)
		}
		if count, ok := retryCount.(int); ok {
			return count
		}
		if count, ok := retryCount.(int64); ok {
			return int(count)
		}
	}

	return 0
}

func (c *Consumer) publishToDelayQueue(msg amqp091.Delivery, retryCount int, err error) error {
	headers := make(amqp091.Table)
	if msg.Headers != nil {
		for k, v := range msg.Headers {
			headers[k] = v
		}
	}
	headers[retryHeader] = retryCount

	if msg.RoutingKey != "" {
		headers["x-original-routing-key"] = msg.RoutingKey
	}
	if err != nil {
		headers["x-reason"] = err.Error()
	}

	err = c.client.Channel().Publish(
		"",                // exchange (пустой = прямая публикация в очередь)
		c.cfg.Delay.Queue, // routing key (при пустом exchange = имя очереди)
		false,             // mandatory
		false,             // immediate
		amqp091.Publishing{
			Headers:      headers,
			ContentType:  msg.ContentType,
			Body:         msg.Body,
			MessageId:    msg.MessageId,
			DeliveryMode: amqp091.Persistent,
		},
	)

	return err
}

func (c *Consumer) prepareDLQHeaders(msg amqp091.Delivery, originalErr error, reason string) amqp091.Table {
	headers := make(amqp091.Table)
	if msg.Headers != nil {
		for k, v := range msg.Headers {
			headers[k] = v
		}
	}

	// Сохраняем оригинальный routing key
	if msg.RoutingKey != "" {
		headers["x-original-routing-key"] = msg.RoutingKey
	}

	// Добавляем информацию об ошибке
	if originalErr != nil {
		headers[errorReasonHeader] = originalErr.Error()
	}
	headers[dlqReasonHeader] = reason

	// Сохраняем количество попыток, если было
	retryCount := c.getRetryCount(msg.Headers)
	if retryCount > 0 {
		headers[retryHeader] = retryCount
	}

	return headers
}

func (c *Consumer) publishToDLQ(ctx context.Context, msg amqp091.Delivery, originalErr error, reason string) error {
	if c.cfg.Dead == nil {
		return fmt.Errorf("dead letter queue not configured")
	}

	headers := c.prepareDLQHeaders(msg, originalErr, reason)

	err := c.client.Channel().Publish(
		c.cfg.Dead.Exchange,
		c.cfg.Dead.RoutingKey,
		false,
		false,
		amqp091.Publishing{
			Headers:      headers,
			ContentType:  msg.ContentType,
			Body:         msg.Body,
			MessageId:    msg.MessageId,
			DeliveryMode: amqp091.Persistent,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish to DLQ: %w", err)
	}

	retryCount := c.getRetryCount(msg.Headers)
	c.log.Info(ctx, "Message published to DLQ", map[string]any{
		"message_id":     msg.MessageId,
		"reason":         reason,
		"retry_count":    retryCount,
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

func (c *Consumer) initializeInfrastructure(ctx context.Context) error {
	channel := c.client.Channel()

	// DEAD
	if c.cfg.Dead != nil {
		if err := channel.ExchangeDeclare(
			c.cfg.Dead.Exchange,
			"direct", // type
			true,     // durable
			false,    // auto-deleted
			false,    // internal
			false,    // no-wait
			nil,      // arguments
		); err != nil {
			return fmt.Errorf("failed to declare DLX exchange %s: %w", c.cfg.Dead.Exchange, err)
		}
		c.log.Info(ctx, "DLX Exchange declared", map[string]any{"exchange": c.cfg.Dead.Exchange})

		_, err := channel.QueueDeclare(
			c.cfg.Dead.Queue,
			true,  // durable
			false, // delete when unused
			false, // exclusive
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to declare DLQ %s: %w", c.cfg.Dead.Queue, err)
		}
		c.log.Info(ctx, "DLQ declared", map[string]any{"queue": c.cfg.Dead.Queue})

		if err := channel.QueueBind(
			c.cfg.Dead.Queue, // queue name
			c.cfg.Dead.RoutingKey, // routing key
			c.cfg.Dead.Exchange, // exchange name
			false, // no-wait
			nil,   // arguments
		); err != nil {
			return fmt.Errorf("failed to bind DLQ %s to DLX %s: %w", c.cfg.Dead.Queue, c.cfg.Dead.Exchange, err)
		}
		c.log.Info(ctx, "DLQ bound to DLX", map[string]any{
			"queue":       c.cfg.Dead.Queue,
			"exchange":    c.cfg.Dead.Exchange,
			"routing_key": c.cfg.Dead.RoutingKey,
		})
	}

	// MAIN
	if err := channel.ExchangeDeclare(
		c.cfg.Main.Exchange,
		"topic", // type
		true,    // durable
		false,   // auto-deleted
		false,   // internal
		false,   // no-wait
		nil,     // arguments
	); err != nil {
		return fmt.Errorf("failed to declare exchange %s: %w", c.cfg.Main.Exchange, err)
	}
	c.log.Info(ctx, "Exchange declared", map[string]any{"exchange": c.cfg.Main.Exchange})

	table := amqp091.Table{
		"x-message-ttl":  int64(c.cfg.Main.TTL),
		"x-max-priority": int32(10),
	}
	if c.cfg.Dead != nil {
		table["x-dead-letter-exchange"] = c.cfg.Dead.Exchange // exchange name
		table["x-dead-letter-routing-key"] = c.cfg.Dead.RoutingKey // routing key
	}

	_, err := channel.QueueDeclare(
		c.cfg.Main.Queue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		table, // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", c.cfg.Main.Queue, err)
	}
	c.log.Info(ctx, "Queue declared", map[string]any{"queue": c.cfg.Main.Queue})

	if err := channel.QueueBind(
		c.cfg.Main.Queue, // queue name
		c.cfg.Main.BindingPattern, // binding pattern
		c.cfg.Main.Exchange, // exchange name
		false, // no-wait
		nil,   // arguments
	); err != nil {
		return fmt.Errorf("failed to bind queue %s to exchange %s: %w", c.cfg.Main.Queue, c.cfg.Main.Exchange, err)
	}
	c.log.Info(ctx, "Queue bound to exchange", map[string]any{
		"queue":           c.cfg.Main.Queue,
		"exchange":        c.cfg.Main.Exchange,
		"binding_pattern": c.cfg.Main.BindingPattern,
	})

	// DELAY
	if c.cfg.Delay != nil {
		table := amqp091.Table{
			"x-message-ttl": int64(c.cfg.Delay.TTL),
		}
		if c.cfg.Dead != nil {
			table["x-dead-letter-exchange"] = c.cfg.Main.Exchange // exchange name
			table["x-dead-letter-routing-key"] = c.cfg.Main.RoutingKey // routing key
		}
		_, err = channel.QueueDeclare(
			c.cfg.Delay.Queue,
			true,  // durable
			false, // delete when unused
			false, // exclusive
			false, // no-wait
			table, // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to declare delay queue %s: %w", c.cfg.Delay.Queue, err)
		}
		c.log.Info(ctx, "Delay queue declared", map[string]any{"queue": c.cfg.Delay.Queue})
	}

	return nil
}

func UnmarshalMessage(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
