package rmq

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"sync"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/ssoeasy-dev/pkg/logger"
)

const (
	headerRetryCount  = "x-retry-count"
	headerTraceID     = "x-trace-id"
	headerRequestID   = "x-request-id"
	headerErrorReason = "x-error-reason"
	headerDLQReason   = "x-dlq-reason"
	headerOriginalKey = "x-original-routing-key"
)

// MessageHandler обрабатывает входящее сообщение.
// Возврат nil → ack. Возврат error → retry или DLQ в зависимости от конфигурации.
type MessageHandler func(ctx context.Context, body []byte, routingKey string) error

// QueueConfig описывает параметры одной очереди.
type QueueConfig struct {
	Queue          string
	BindingPattern string
	Exchange       string
	// TTL — время жизни сообщения в миллисекундах. 0 или отрицательное — без TTL.
	TTL int
}

// DelayQueueConfig описывает очередь задержки для retry.
// Exchange, BindingPattern, Queue, TTL — обязательны. MaxRetry — обязателен.
type DelayQueueConfig struct {
	QueueConfig
	MaxRetry int
}

// ConsumerConfig описывает полную топологию consumer-а.
type ConsumerConfig struct {
	Main    QueueConfig
	Delay   *DelayQueueConfig // nil — без retry, ошибки сразу в DLQ
	Dead    *QueueConfig      // nil — nack вместо DLQ
	Handler MessageHandler
}

// Consumer подписывается на Main-очередь и обрабатывает сообщения через Handler.
// При ошибках поддерживает retry через Delay-очередь и отправку в Dead-очередь.
type Consumer struct {
	log            logger.Logger
	client         *Client
	cfg            *ConsumerConfig
	publishChannel *amqp091.Channel
	stopped        chan struct{}
	stopOnce       sync.Once
}

// NewConsumer создаёт Consumer. Возвращает ErrInvalidConfig при неполной конфигурации.
func NewConsumer(log logger.Logger, client *Client, cfg *ConsumerConfig) (*Consumer, error) {
	if err := validateConsumerConfig(cfg); err != nil {
		return nil, err
	}
	return &Consumer{
		log:     log,
		client:  client,
		cfg:     cfg,
		stopped: make(chan struct{}),
	}, nil
}

func validateConsumerConfig(cfg *ConsumerConfig) error {
	if cfg.Main.Queue == "" {
		return newError(ErrInvalidConfig, "main queue name is required", nil)
	}
	if cfg.Main.Exchange == "" {
		return newError(ErrInvalidConfig, "main exchange is required", nil)
	}
	if cfg.Main.BindingPattern == "" {
		return newError(ErrInvalidConfig, "main binding pattern is required", nil)
	}
	if cfg.Handler == nil {
		return newError(ErrInvalidConfig, "handler is required", nil)
	}

	if d := cfg.Delay; d != nil {
		switch {
		case d.Queue == "":
			return newError(ErrInvalidConfig, "delay queue name is required", nil)
		case d.Exchange == "":
			return newError(ErrInvalidConfig, "delay exchange is required", nil)
		case d.BindingPattern == "":
			return newError(ErrInvalidConfig, "delay binding pattern is required", nil)
		case d.TTL <= 0:
			return newError(ErrInvalidConfig, "delay TTL must be positive", nil)
		case d.MaxRetry <= 0:
			return newError(ErrInvalidConfig, "delay MaxRetry must be positive", nil)
		}
	}

	if dlq := cfg.Dead; dlq != nil {
		switch {
		case dlq.Queue == "":
			return newError(ErrInvalidConfig, "dead queue name is required", nil)
		case dlq.Exchange == "":
			return newError(ErrInvalidConfig, "dead exchange is required", nil)
		case dlq.BindingPattern == "":
			return newError(ErrInvalidConfig, "dead binding pattern is required", nil)
		}
	}

	return nil
}

// isStopped возвращает true если Stop был вызван.
func (c *Consumer) isStopped() bool {
	select {
	case <-c.stopped:
		return true
	default:
		return false
	}
}

// Start объявляет топологию (идемпотентно) и запускает обработку сообщений в фоне.
// Неблокирующий. Для остановки вызовите Stop или отмените ctx.
func (c *Consumer) Start(ctx context.Context) error {
	initCh, err := c.client.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open init channel: %w", err)
	}
	defer func() {
		if err := initCh.Close(); err != nil {
			c.log.Error(ctx, "Failed to close init channel", map[string]any{"error": err})
		}
	}()

	if err := c.initializeInfrastructure(ctx, initCh); err != nil {
		return fmt.Errorf("failed to initialize RabbitMQ infrastructure: %w", err)
	}

	// Закрываем старый канал при переподключении.
	if c.publishChannel != nil {
		_ = c.publishChannel.Close()
	}
	c.publishChannel, err = c.client.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open publish channel: %w", err)
	}

	consumeCh, err := c.client.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open consume channel: %w", err)
	}

	if err := consumeCh.Qos(1, 0, false); err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	hostname, _ := os.Hostname()
	consumerTag := fmt.Sprintf("%s-%d", hostname, os.Getpid())

	msgs, err := consumeCh.Consume(
		c.cfg.Main.Queue,
		consumerTag,
		false, // manual ack
		false, // not exclusive
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

// Stop инициирует graceful shutdown consumer-а. Безопасно вызывать повторно.
func (c *Consumer) Stop() {
	c.stopOnce.Do(func() {
		close(c.stopped)
		c.log.Info(context.Background(), "Consumer stop requested", nil)
	})
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
				if c.isStopped() {
					c.log.Info(ctx, "Channel closed after stop, not reconnecting", nil)
					return
				}
				select {
				case <-ctx.Done():
					return
				case <-c.stopped:
					return
				default:
					c.log.Warn(ctx, "Message channel closed, attempting reconnect", nil)
					if err := c.reconnectAndRestart(ctx); err != nil {
						c.log.Error(ctx, "Failed to reconnect", map[string]any{"error": err})
					}
					return
				}
			}
			c.handleMessage(ctx, msg)
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, msg amqp091.Delivery) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	ctx = c.extractTraceContext(ctx, msg.Headers)

	c.log.Info(ctx, "Received message", map[string]any{
		"routing_key": msg.RoutingKey,
		"message_id":  msg.MessageId,
	})

	if err := c.cfg.Handler(ctx, msg.Body, msg.RoutingKey); err != nil {
		c.log.Error(ctx, "Handler failed", map[string]any{
			"error":       err,
			"routing_key": msg.RoutingKey,
			"message_id":  msg.MessageId,
		})
		c.handleFailure(ctx, msg, err)
		return
	}

	c.ack(ctx, msg)
	c.log.Info(ctx, "Message processed successfully", map[string]any{
		"routing_key": msg.RoutingKey,
		"message_id":  msg.MessageId,
	})
}

func (c *Consumer) handleFailure(ctx context.Context, msg amqp091.Delivery, handlerErr error) {
	if c.cfg.Delay == nil {
		c.log.Error(ctx, "Delay queue not configured, sending to DLQ", map[string]any{
			"message_id": msg.MessageId,
		})
		c.sendToDLQOrNack(ctx, msg, handlerErr, "delay_queue_not_configured")
		return
	}

	retry := parseRetryCount(msg.Headers)
	if retry >= c.cfg.Delay.MaxRetry {
		c.log.Error(ctx, "Max retries exceeded, sending to DLQ", map[string]any{
			"message_id":  msg.MessageId,
			"retry_count": retry,
			"max_retry":   c.cfg.Delay.MaxRetry,
		})
		c.sendToDLQOrNack(ctx, msg, handlerErr, fmt.Sprintf("max_retries_exceeded:%d", retry))
		return
	}

	next := retry + 1
	c.log.Warn(ctx, "Retrying message via delay queue", map[string]any{
		"message_id":  msg.MessageId,
		"retry_count": next,
		"max_retry":   c.cfg.Delay.MaxRetry,
	})

	if err := c.publishToDelay(msg, next, handlerErr); err != nil {
		c.log.Error(ctx, "Failed to publish to delay queue, falling back to DLQ", map[string]any{
			"error":      err,
			"message_id": msg.MessageId,
		})
		c.sendToDLQOrNack(ctx, msg, handlerErr, "failed_to_publish_to_delay_queue")
		return
	}

	c.ack(ctx, msg)
}

func (c *Consumer) sendToDLQOrNack(ctx context.Context, msg amqp091.Delivery, originalErr error, reason string) {
	if err := c.publishToDLQ(ctx, msg, originalErr, reason); err != nil {
		c.log.Error(ctx, "Failed to publish to DLQ, using nack", map[string]any{
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
	c.ack(ctx, msg)
}

func (c *Consumer) ack(ctx context.Context, msg amqp091.Delivery) {
	if err := msg.Ack(false); err != nil {
		c.log.Error(ctx, "Failed to ack message", map[string]any{
			"error":      err,
			"message_id": msg.MessageId,
		})
	}
}

func (c *Consumer) extractTraceContext(ctx context.Context, headers amqp091.Table) context.Context {
	if headers == nil {
		return ctx
	}
	if v, ok := headers[headerTraceID].(string); ok && v != "" {
		ctx = context.WithValue(ctx, logger.TraceIDKey, v)
	}
	if v, ok := headers[headerRequestID].(string); ok && v != "" {
		ctx = context.WithValue(ctx, logger.RequestIDKey, v)
	}
	return ctx
}

// parseRetryCount извлекает счётчик попыток из заголовков сообщения.
// Поддерживает int, int32, int64 — amqp091 может вернуть любой из этих типов.
func parseRetryCount(headers amqp091.Table) int {
	if headers == nil {
		return 0
	}
	switch v := headers[headerRetryCount].(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	}
	return 0
}

func (c *Consumer) publishToDelay(msg amqp091.Delivery, nextRetry int, cause error) error {
	headers := make(amqp091.Table, len(msg.Headers)+2)
	maps.Copy(headers, msg.Headers)
	headers[headerRetryCount] = nextRetry
	if cause != nil {
		headers["x-reason"] = cause.Error()
	}

	if err := c.publishChannel.Publish(
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
	); err != nil {
		return newError(ErrPublish, "failed to publish to delay queue", err)
	}
	return nil
}

func (c *Consumer) publishToDLQ(ctx context.Context, msg amqp091.Delivery, originalErr error, reason string) error {
	if c.cfg.Dead == nil {
		return newError(ErrPublish, "dead letter queue not configured", nil)
	}

	headers := make(amqp091.Table, len(msg.Headers)+4)
	maps.Copy(headers, msg.Headers)
	if msg.RoutingKey != "" {
		headers[headerOriginalKey] = msg.RoutingKey
	}
	if originalErr != nil {
		headers[headerErrorReason] = originalErr.Error()
	}
	headers[headerDLQReason] = reason

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
		return newError(ErrPublish, "failed to publish to DLQ", err)
	}

	c.log.Info(ctx, "Message published to DLQ", map[string]any{
		"message_id":     msg.MessageId,
		"routing_key":    msg.RoutingKey,
		"reason":         reason,
		"retry_count":    parseRetryCount(msg.Headers),
		"original_error": originalErr,
	})
	return nil
}

func (c *Consumer) reconnectAndRestart(ctx context.Context) error {
	if c.isStopped() {
		return ErrStopped
	}
	if err := c.client.Reconnect(); err != nil {
		return err
	}
	return c.Start(ctx)
}

func (c *Consumer) initializeInfrastructure(ctx context.Context, ch *amqp091.Channel) error {
	// Dead letter exchange и queue объявляются первыми:
	// main queue ссылается на DLX через x-dead-letter-exchange.
	if dlq := c.cfg.Dead; dlq != nil {
		if err := ch.ExchangeDeclare(dlq.Exchange, "topic", true, false, false, false, nil); err != nil {
			return fmt.Errorf("failed to declare DLX exchange %q: %w", dlq.Exchange, err)
		}
		c.log.Info(ctx, "DLX exchange declared", map[string]any{"exchange": dlq.Exchange})

		if _, err := ch.QueueDeclare(dlq.Queue, true, false, false, false, nil); err != nil {
			return fmt.Errorf("failed to declare DLQ %q: %w", dlq.Queue, err)
		}
		c.log.Info(ctx, "DLQ declared", map[string]any{"queue": dlq.Queue})

		if err := ch.QueueBind(dlq.Queue, dlq.BindingPattern, dlq.Exchange, false, nil); err != nil {
			return fmt.Errorf("failed to bind DLQ: %w", err)
		}
		c.log.Info(ctx, "DLQ bound", map[string]any{"queue": dlq.Queue, "exchange": dlq.Exchange})
	}

	// Main exchange и queue.
	if err := ch.ExchangeDeclare(c.cfg.Main.Exchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare main exchange %q: %w", c.cfg.Main.Exchange, err)
	}
	c.log.Info(ctx, "Main exchange declared", map[string]any{"exchange": c.cfg.Main.Exchange})

	mainArgs := amqp091.Table{"x-max-priority": int32(10)}
	if c.cfg.Main.TTL > 0 {
		mainArgs["x-message-ttl"] = int64(c.cfg.Main.TTL)
	}
	if c.cfg.Dead != nil {
		mainArgs["x-dead-letter-exchange"] = c.cfg.Dead.Exchange
	}

	if _, err := ch.QueueDeclare(c.cfg.Main.Queue, true, false, false, false, mainArgs); err != nil {
		return fmt.Errorf("failed to declare main queue %q: %w", c.cfg.Main.Queue, err)
	}
	c.log.Info(ctx, "Main queue declared", map[string]any{"queue": c.cfg.Main.Queue})

	if err := ch.QueueBind(c.cfg.Main.Queue, c.cfg.Main.BindingPattern, c.cfg.Main.Exchange, false, nil); err != nil {
		return fmt.Errorf("failed to bind main queue: %w", err)
	}
	c.log.Info(ctx, "Main queue bound", map[string]any{
		"queue":    c.cfg.Main.Queue,
		"exchange": c.cfg.Main.Exchange,
		"pattern":  c.cfg.Main.BindingPattern,
	})

	// Delay queue: TTL-expired сообщения возвращаются в main exchange с оригинальным ключом.
	if d := c.cfg.Delay; d != nil {
		if err := ch.ExchangeDeclare(d.Exchange, "topic", true, false, false, false, nil); err != nil {
			return fmt.Errorf("failed to declare delay exchange %q: %w", d.Exchange, err)
		}
		c.log.Info(ctx, "Delay exchange declared", map[string]any{"exchange": d.Exchange})

		delayArgs := amqp091.Table{
			"x-message-ttl":          int64(d.TTL),
			"x-dead-letter-exchange": c.cfg.Main.Exchange,
		}
		if _, err := ch.QueueDeclare(d.Queue, true, false, false, false, delayArgs); err != nil {
			return fmt.Errorf("failed to declare delay queue %q: %w", d.Queue, err)
		}
		c.log.Info(ctx, "Delay queue declared", map[string]any{"queue": d.Queue, "ttl_ms": d.TTL})

		if err := ch.QueueBind(d.Queue, d.BindingPattern, d.Exchange, false, nil); err != nil {
			return fmt.Errorf("failed to bind delay queue: %w", err)
		}
		c.log.Info(ctx, "Delay queue bound", map[string]any{"queue": d.Queue, "exchange": d.Exchange})
	}

	return nil
}

// UnmarshalMessage десериализует тело JSON-сообщения в v.
func UnmarshalMessage(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
