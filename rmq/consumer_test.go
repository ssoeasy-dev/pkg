package rmq_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ssoeasy-dev/pkg/logger"
	"github.com/ssoeasy-dev/pkg/rmq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Sentinel errors ──────────────────────────────────────────────────────────

func TestErrors_Distinct(t *testing.T) {
	sentinels := []error{
		rmq.ErrInvalidConfig,
		rmq.ErrConnect,
		rmq.ErrPublish,
		rmq.ErrStopped,
	}
	for i, a := range sentinels {
		for j, b := range sentinels {
			if i != j {
				assert.False(t, errors.Is(a, b), "%v should not match %v", a, b)
			}
		}
	}
}

func TestErrors_NotNil(t *testing.T) {
	assert.NotNil(t, rmq.ErrInvalidConfig)
	assert.NotNil(t, rmq.ErrConnect)
	assert.NotNil(t, rmq.ErrPublish)
	assert.NotNil(t, rmq.ErrStopped)
}

// ─── Config.URL ───────────────────────────────────────────────────────────────

func TestConfig_URL_DefaultVHost(t *testing.T) {
	cfg := &rmq.Config{
		Host:     "localhost",
		Port:     "5672",
		User:     "guest",
		Password: "guest",
		VHost:    "/",
	}
	assert.Equal(t, "amqp://guest:guest@localhost:5672//", cfg.URL())
}

func TestConfig_URL_CustomVHost(t *testing.T) {
	cfg := &rmq.Config{
		Host:     "rabbitmq.example.com",
		Port:     "5672",
		User:     "admin",
		Password: "secret",
		VHost:    "myapp",
	}
	assert.Equal(t, "amqp://admin:secret@rabbitmq.example.com:5672/myapp", cfg.URL())
}

// ─── NewConsumer — happy path ─────────────────────────────────────────────────

var testHandler rmq.MessageHandler = func(_ context.Context, _ []byte, _ string) error {
	return nil
}

func TestNewConsumer_MinimalValidConfig(t *testing.T) {
	log := logger.NewLogger(logger.EnvironmentTest, "test")
	// client=nil допустим для NewConsumer: он только проверяет конфиг и сохраняет ссылки.
	consumer, err := rmq.NewConsumer(log, nil, &rmq.ConsumerConfig{
		Main: rmq.QueueConfig{
			Exchange:       "events",
			BindingPattern: "event.#",
			Queue:          "events.queue",
		},
		Handler: testHandler,
	})
	require.NoError(t, err)
	assert.NotNil(t, consumer)
}

func TestNewConsumer_FullConfig(t *testing.T) {
	log := logger.NewLogger(logger.EnvironmentTest, "test")
	consumer, err := rmq.NewConsumer(log, nil, &rmq.ConsumerConfig{
		Main: rmq.QueueConfig{
			Exchange:       "events",
			BindingPattern: "event.#",
			Queue:          "events.queue",
			TTL:            60_000,
		},
		Delay: &rmq.DelayQueueConfig{
			QueueConfig: rmq.QueueConfig{
				Exchange:       "events.delay",
				BindingPattern: "event.#",
				Queue:          "events.delay.queue",
				TTL:            5_000,
			},
			MaxRetry: 3,
		},
		Dead: &rmq.QueueConfig{
			Exchange:       "events.dlx",
			BindingPattern: "event.#",
			Queue:          "events.dlq",
		},
		Handler: testHandler,
	})
	require.NoError(t, err)
	assert.NotNil(t, consumer)
}

// ─── NewConsumer — main queue validation ─────────────────────────────────────

func TestNewConsumer_MissingMainQueue(t *testing.T) {
	log := logger.NewLogger(logger.EnvironmentTest, "test")
	_, err := rmq.NewConsumer(log, nil, &rmq.ConsumerConfig{
		Main:    rmq.QueueConfig{Exchange: "ex", BindingPattern: "key.#"},
		Handler: testHandler,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, rmq.ErrInvalidConfig)
	assert.Contains(t, err.Error(), "main queue name is required")
}

func TestNewConsumer_MissingMainExchange(t *testing.T) {
	log := logger.NewLogger(logger.EnvironmentTest, "test")
	_, err := rmq.NewConsumer(log, nil, &rmq.ConsumerConfig{
		Main:    rmq.QueueConfig{Queue: "q", BindingPattern: "key.#"},
		Handler: testHandler,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, rmq.ErrInvalidConfig)
	assert.Contains(t, err.Error(), "main exchange is required")
}

func TestNewConsumer_MissingMainBindingPattern(t *testing.T) {
	log := logger.NewLogger(logger.EnvironmentTest, "test")
	_, err := rmq.NewConsumer(log, nil, &rmq.ConsumerConfig{
		Main:    rmq.QueueConfig{Queue: "q", Exchange: "ex"},
		Handler: testHandler,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, rmq.ErrInvalidConfig)
	assert.Contains(t, err.Error(), "main binding pattern is required")
}

func TestNewConsumer_NilHandler(t *testing.T) {
	log := logger.NewLogger(logger.EnvironmentTest, "test")
	_, err := rmq.NewConsumer(log, nil, &rmq.ConsumerConfig{
		Main: rmq.QueueConfig{Queue: "q", Exchange: "ex", BindingPattern: "key.#"},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, rmq.ErrInvalidConfig)
	assert.Contains(t, err.Error(), "handler is required")
}

// ─── NewConsumer — delay queue validation ────────────────────────────────────

func validMain() rmq.QueueConfig {
	return rmq.QueueConfig{Queue: "q", Exchange: "ex", BindingPattern: "key.#"}
}

func TestNewConsumer_DelayMissingQueue(t *testing.T) {
	log := logger.NewLogger(logger.EnvironmentTest, "test")
	_, err := rmq.NewConsumer(log, nil, &rmq.ConsumerConfig{
		Main:    validMain(),
		Handler: testHandler,
		Delay: &rmq.DelayQueueConfig{
			QueueConfig: rmq.QueueConfig{Exchange: "dex", BindingPattern: "key.#", TTL: 1000},
			MaxRetry:    3,
		},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, rmq.ErrInvalidConfig)
	assert.Contains(t, err.Error(), "delay queue name is required")
}

func TestNewConsumer_DelayMissingExchange(t *testing.T) {
	log := logger.NewLogger(logger.EnvironmentTest, "test")
	_, err := rmq.NewConsumer(log, nil, &rmq.ConsumerConfig{
		Main:    validMain(),
		Handler: testHandler,
		Delay: &rmq.DelayQueueConfig{
			QueueConfig: rmq.QueueConfig{Queue: "dq", BindingPattern: "key.#", TTL: 1000},
			MaxRetry:    3,
		},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, rmq.ErrInvalidConfig)
	assert.Contains(t, err.Error(), "delay exchange is required")
}

func TestNewConsumer_DelayZeroTTL(t *testing.T) {
	log := logger.NewLogger(logger.EnvironmentTest, "test")
	_, err := rmq.NewConsumer(log, nil, &rmq.ConsumerConfig{
		Main:    validMain(),
		Handler: testHandler,
		Delay: &rmq.DelayQueueConfig{
			QueueConfig: rmq.QueueConfig{Queue: "dq", Exchange: "dex", BindingPattern: "key.#", TTL: 0},
			MaxRetry:    3,
		},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, rmq.ErrInvalidConfig)
	assert.Contains(t, err.Error(), "delay TTL must be positive")
}

func TestNewConsumer_DelayNegativeTTL(t *testing.T) {
	log := logger.NewLogger(logger.EnvironmentTest, "test")
	_, err := rmq.NewConsumer(log, nil, &rmq.ConsumerConfig{
		Main:    validMain(),
		Handler: testHandler,
		Delay: &rmq.DelayQueueConfig{
			QueueConfig: rmq.QueueConfig{Queue: "dq", Exchange: "dex", BindingPattern: "key.#", TTL: -1},
			MaxRetry:    3,
		},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, rmq.ErrInvalidConfig)
}

func TestNewConsumer_DelayZeroMaxRetry(t *testing.T) {
	log := logger.NewLogger(logger.EnvironmentTest, "test")
	_, err := rmq.NewConsumer(log, nil, &rmq.ConsumerConfig{
		Main:    validMain(),
		Handler: testHandler,
		Delay: &rmq.DelayQueueConfig{
			QueueConfig: rmq.QueueConfig{Queue: "dq", Exchange: "dex", BindingPattern: "key.#", TTL: 1000},
			MaxRetry:    0,
		},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, rmq.ErrInvalidConfig)
	assert.Contains(t, err.Error(), "delay MaxRetry must be positive")
}

// ─── NewConsumer — dead queue validation ─────────────────────────────────────

func TestNewConsumer_DeadMissingQueue(t *testing.T) {
	log := logger.NewLogger(logger.EnvironmentTest, "test")
	_, err := rmq.NewConsumer(log, nil, &rmq.ConsumerConfig{
		Main:    validMain(),
		Handler: testHandler,
		Dead:    &rmq.QueueConfig{Exchange: "dlx", BindingPattern: "key.#"},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, rmq.ErrInvalidConfig)
	assert.Contains(t, err.Error(), "dead queue name is required")
}

func TestNewConsumer_DeadMissingExchange(t *testing.T) {
	log := logger.NewLogger(logger.EnvironmentTest, "test")
	_, err := rmq.NewConsumer(log, nil, &rmq.ConsumerConfig{
		Main:    validMain(),
		Handler: testHandler,
		Dead:    &rmq.QueueConfig{Queue: "dlq", BindingPattern: "key.#"},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, rmq.ErrInvalidConfig)
	assert.Contains(t, err.Error(), "dead exchange is required")
}

func TestNewConsumer_DeadMissingBindingPattern(t *testing.T) {
	log := logger.NewLogger(logger.EnvironmentTest, "test")
	_, err := rmq.NewConsumer(log, nil, &rmq.ConsumerConfig{
		Main:    validMain(),
		Handler: testHandler,
		Dead:    &rmq.QueueConfig{Queue: "dlq", Exchange: "dlx"},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, rmq.ErrInvalidConfig)
	assert.Contains(t, err.Error(), "dead binding pattern is required")
}

// ─── ErrInvalidConfig через errors.Unwrap ────────────────────────────────────

func TestNewConsumer_ValidationError_NoUnwrappedCause(t *testing.T) {
	// Ошибки валидации конфига не имеют cause — это ошибки конфигурации, не runtime.
	log := logger.NewLogger(logger.EnvironmentTest, "test")
	_, err := rmq.NewConsumer(log, nil, &rmq.ConsumerConfig{
		Main:    rmq.QueueConfig{Exchange: "ex", BindingPattern: "key.#"},
		Handler: testHandler,
	})
	require.Error(t, err)
	// errors.Unwrap возвращает nil (нет оригинальной cause)
	assert.Nil(t, errors.Unwrap(err))
}

// ─── UnmarshalMessage ─────────────────────────────────────────────────────────

func TestUnmarshalMessage_ValidJSON(t *testing.T) {
	type payload struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	var p payload
	err := rmq.UnmarshalMessage([]byte(`{"id":42,"name":"test"}`), &p)
	require.NoError(t, err)
	assert.Equal(t, 42, p.ID)
	assert.Equal(t, "test", p.Name)
}

func TestUnmarshalMessage_InvalidJSON(t *testing.T) {
	var m map[string]any
	err := rmq.UnmarshalMessage([]byte(`not-json`), &m)
	require.Error(t, err)
}

func TestUnmarshalMessage_EmptyBody(t *testing.T) {
	var m map[string]any
	err := rmq.UnmarshalMessage([]byte{}, &m)
	require.Error(t, err)
}
