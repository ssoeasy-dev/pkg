//go:build integration

package rmq_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/ssoeasy-dev/pkg/logger"
	"github.com/ssoeasy-dev/pkg/rmq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ─── Setup ────────────────────────────────────────────────────────────────────

func setupRabbitMQ(t *testing.T) *rmq.Config {
	t.Helper()
	t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "rabbitmq:3-alpine",
		ExposedPorts: []string{"5672/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForLog("Server startup complete"),
			wait.ForListeningPort("5672/tcp"),
		),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5672")
	require.NoError(t, err)

	return &rmq.Config{
		Host:     host,
		Port:     port.Port(),
		User:     "guest",
		Password: "guest",
		VHost:    "/",
	}
}

func newTestLogger() logger.Logger {
	return logger.NewLogger(logger.EnvironmentTest, "rmq-test")
}

// publish публикует сообщение в exchange напрямую через amqp091 (минуя consumer).
func publish(t *testing.T, cfg *rmq.Config, exchange, routingKey string, body []byte, headers amqp091.Table) {
	t.Helper()
	conn, err := amqp091.Dial(cfg.URL())
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	ch, err := conn.Channel()
	require.NoError(t, err)
	defer ch.Close()

	err = ch.Publish(exchange, routingKey, false, false, amqp091.Publishing{
		Body:         body,
		ContentType:  "application/json",
		DeliveryMode: amqp091.Persistent,
		Headers:      headers,
	})
	require.NoError(t, err)
}

// eventually ждёт выполнения condition с таймаутом.
func eventually(t *testing.T, condition func() bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}

// ─── NewClient ────────────────────────────────────────────────────────────────

func TestNewClient_Connect(t *testing.T) {
	cfg := setupRabbitMQ(t)
	log := newTestLogger()

	client, err := rmq.NewClient(log, cfg)
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NoError(t, client.Close())
}

func TestNewClient_InvalidAddress_ReturnsErrConnect(t *testing.T) {
	log := newTestLogger()
	_, err := rmq.NewClient(log, &rmq.Config{
		Host:     "127.0.0.1",
		Port:     "1", // недоступный порт
		User:     "guest",
		Password: "guest",
		VHost:    "/",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, rmq.ErrConnect)
	// Оригинальная ошибка доступна через Unwrap для логирования.
	assert.NotNil(t, errors.Unwrap(err))
}

// ─── Consumer — happy path ────────────────────────────────────────────────────

func TestConsumer_ProcessMessage_Success(t *testing.T) {
	cfg := setupRabbitMQ(t)
	log := newTestLogger()

	client, err := rmq.NewClient(log, cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	var handled atomic.Bool

	consumer, err := rmq.NewConsumer(log, client, &rmq.ConsumerConfig{
		Main: rmq.QueueConfig{
			Exchange:       "test.events",
			BindingPattern: "test.#",
			Queue:          "test.events.queue",
			TTL:            60_000,
		},
		Handler: func(_ context.Context, body []byte, _ string) error {
			handled.Store(true)
			return nil
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, consumer.Start(ctx))
	t.Cleanup(consumer.Stop)

	publish(t, cfg, "test.events", "test.created", []byte(`{"id":1}`), nil)

	eventually(t, handled.Load, 5*time.Second)
}

func TestConsumer_ProcessMessage_ReceivesCorrectBody(t *testing.T) {
	cfg := setupRabbitMQ(t)
	log := newTestLogger()

	client, err := rmq.NewClient(log, cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	received := make(chan []byte, 1)

	consumer, err := rmq.NewConsumer(log, client, &rmq.ConsumerConfig{
		Main: rmq.QueueConfig{
			Exchange:       "body.events",
			BindingPattern: "body.#",
			Queue:          "body.events.queue",
		},
		Handler: func(_ context.Context, body []byte, _ string) error {
			received <- body
			return nil
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, consumer.Start(ctx))
	t.Cleanup(consumer.Stop)

	want := []byte(`{"key":"value"}`)
	publish(t, cfg, "body.events", "body.created", want, nil)

	select {
	case got := <-received:
		assert.Equal(t, want, got)
	case <-time.After(5 * time.Second):
		t.Fatal("message not received within timeout")
	}
}

// ─── Consumer — retry через delay queue ──────────────────────────────────────

func TestConsumer_RetryOnError_SucceedsAfterRetry(t *testing.T) {
	cfg := setupRabbitMQ(t)
	log := newTestLogger()

	client, err := rmq.NewClient(log, cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	var attempts atomic.Int32

	consumer, err := rmq.NewConsumer(log, client, &rmq.ConsumerConfig{
		Main: rmq.QueueConfig{
			Exchange:       "retry.events",
			BindingPattern: "retry.#",
			Queue:          "retry.events.queue",
		},
		Delay: &rmq.DelayQueueConfig{
			QueueConfig: rmq.QueueConfig{
				Exchange:       "retry.delay",
				BindingPattern: "retry.#",
				Queue:          "retry.events.delay.queue",
				TTL:            300, // 300ms для быстрого теста
			},
			MaxRetry: 3,
		},
		Handler: func(_ context.Context, _ []byte, _ string) error {
			n := attempts.Add(1)
			if n < 2 {
				return errors.New("not ready yet")
			}
			return nil
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, consumer.Start(ctx))
	t.Cleanup(consumer.Stop)

	publish(t, cfg, "retry.events", "retry.created", []byte(`{}`), nil)

	// Ждём пока handler вызовется >= 2 раз (1 неудачная + 1 успешная).
	eventually(t, func() bool { return attempts.Load() >= 2 }, 10*time.Second)
	assert.GreaterOrEqual(t, attempts.Load(), int32(2))
}

// ─── Consumer — DLQ после исчерпания попыток ──────────────────────────────────

func TestConsumer_DLQAfterMaxRetries(t *testing.T) {
	cfg := setupRabbitMQ(t)
	log := newTestLogger()

	client, err := rmq.NewClient(log, cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	var handlerCalls atomic.Int32
	handlerErr := errors.New("always fails")

	consumer, err := rmq.NewConsumer(log, client, &rmq.ConsumerConfig{
		Main: rmq.QueueConfig{
			Exchange:       "dlq.events",
			BindingPattern: "dlq.#",
			Queue:          "dlq.events.queue",
		},
		Delay: &rmq.DelayQueueConfig{
			QueueConfig: rmq.QueueConfig{
				Exchange:       "dlq.delay",
				BindingPattern: "dlq.#",
				Queue:          "dlq.events.delay.queue",
				TTL:            200, // 200ms для быстрого теста
			},
			MaxRetry: 2,
		},
		Dead: &rmq.QueueConfig{
			Exchange:       "dlq.dlx",
			BindingPattern: "dlq.#",
			Queue:          "dlq.events.dlq",
		},
		Handler: func(_ context.Context, _ []byte, _ string) error {
			handlerCalls.Add(1)
			return handlerErr
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, consumer.Start(ctx))
	t.Cleanup(consumer.Stop)

	publish(t, cfg, "dlq.events", "dlq.created", []byte(`{}`), nil)

	// Ждём MaxRetry+1 вызовов handler-а (initial + 2 retry).
	eventually(t, func() bool { return handlerCalls.Load() >= 3 }, 15*time.Second)

	// После DLQ handler больше не вызывается.
	callsAfterDLQ := handlerCalls.Load()
	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, callsAfterDLQ, handlerCalls.Load(), "handler should not be called after DLQ")
}

// ─── Consumer — трассировка из заголовков ────────────────────────────────────

func TestConsumer_TraceContextPropagation(t *testing.T) {
	cfg := setupRabbitMQ(t)
	log := newTestLogger()

	client, err := rmq.NewClient(log, cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	traceIDs := make(chan string, 1)

	consumer, err := rmq.NewConsumer(log, client, &rmq.ConsumerConfig{
		Main: rmq.QueueConfig{
			Exchange:       "trace.events",
			BindingPattern: "trace.#",
			Queue:          "trace.events.queue",
		},
		Handler: func(ctx context.Context, _ []byte, _ string) error {
			if v, ok := ctx.Value(logger.TraceIDKey).(string); ok {
				traceIDs <- v
			} else {
				traceIDs <- ""
			}
			return nil
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, consumer.Start(ctx))
	t.Cleanup(consumer.Stop)

	headers := amqp091.Table{
		"x-trace-id":   "trace-abc-123",
		"x-request-id": "req-xyz-456",
	}
	publish(t, cfg, "trace.events", "trace.created", []byte(`{}`), headers)

	select {
	case got := <-traceIDs:
		assert.Equal(t, "trace-abc-123", got)
	case <-time.After(5 * time.Second):
		t.Fatal("message not received within timeout")
	}
}

// ─── Consumer — Stop ──────────────────────────────────────────────────────────

func TestConsumer_Stop_Idempotent(t *testing.T) {
	log := newTestLogger()
	consumer, err := rmq.NewConsumer(log, nil, &rmq.ConsumerConfig{
		Main:    validMain(),
		Handler: testHandler,
	})
	require.NoError(t, err)

	// Stop должен быть безопасен для многократного вызова.
	assert.NotPanics(t, func() {
		consumer.Stop()
		consumer.Stop()
		consumer.Stop()
	})
}