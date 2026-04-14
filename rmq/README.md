# pkg/rmq

RabbitMQ клиент и consumer с retry через delay-очередь и DLQ для Go-микросервисов SSO Easy.

## Установка

```bash
go get github.com/ssoeasy-dev/pkg/rmq@latest
```

## Быстрый старт

```go
import "github.com/ssoeasy-dev/pkg/rmq"

// Клиент — одно соединение с RabbitMQ.
client, err := rmq.NewClient(log, &rmq.Config{
    Host:     "localhost",
    Port:     "5672",
    User:     "guest",
    Password: "guest",
    VHost:    "/",
})
if err != nil {
    // errors.Is(err, rmq.ErrConnect) == true
}
defer client.Close()

// Consumer — обработка сообщений с retry и DLQ.
consumer, err := rmq.NewConsumer(log, client, &rmq.ConsumerConfig{
    Main: rmq.QueueConfig{
        Exchange:       "notifications",
        BindingPattern: "notification.email.*",
        Queue:          "notification.email.queue",
        TTL:            15 * 60 * 1000, // 15m в мс
    },
    Dead: &rmq.QueueConfig{
        Exchange:       "notifications.dlx",
        BindingPattern: "notification.email.*",
        Queue:          "notification.email.dlq",
    },
    Delay: &rmq.DelayQueueConfig{
        QueueConfig: rmq.QueueConfig{
            Exchange:       "notifications.delay",
            BindingPattern: "notification.email.*",
            Queue:          "notification.email.delay.queue",
            TTL:            5 * 1000, // 5s между попытками
        },
        MaxRetry: 3,
    },
    Handler: func(ctx context.Context, body []byte, routingKey string) error {
        var msg MyMessage
        if err := rmq.UnmarshalMessage(body, &msg); err != nil {
            return fmt.Errorf("failed to unmarshal: %w", err)
        }
        // Возврат error → retry; nil → ack.
        return nil
    },
})
if err != nil {
    // errors.Is(err, rmq.ErrInvalidConfig) == true при неполной конфигурации
}

// Запуск (неблокирующий).
if err := consumer.Start(ctx); err != nil {
    log.Fatal(err)
}

// Graceful shutdown.
consumer.Stop()
```

## API

```go
// Клиент
func NewClient(log logger.Logger, cfg *Config) (*Client, error)
func (c *Client) Close() error
func (c *Client) Reconnect() error

// Consumer
func NewConsumer(log logger.Logger, client *Client, cfg *ConsumerConfig) (*Consumer, error)
func (c *Consumer) Start(ctx context.Context) error  // неблокирующий
func (c *Consumer) Stop()                            // идемпотентный

// Хелпер
func UnmarshalMessage(data []byte, v any) error
```

## Типы конфигурации

```go
type Config struct {
    Host     string
    Port     string
    User     string
    Password string
    VHost    string
}

type QueueConfig struct {
    Queue          string
    BindingPattern string
    Exchange       string
    TTL            int // мс; 0 или отрицательное — без TTL
}

// DelayQueueConfig встраивает QueueConfig.
// Exchange, BindingPattern, Queue, TTL и MaxRetry — обязательны.
type DelayQueueConfig struct {
    QueueConfig
    MaxRetry int
}

type ConsumerConfig struct {
    Main    QueueConfig
    Delay   *DelayQueueConfig // nil → без retry, ошибки сразу в DLQ
    Dead    *QueueConfig      // nil → nack вместо публикации в DLQ
    Handler MessageHandler
}
```

## Sentinel-ошибки

Все публичные ошибки совместимы с `errors.Is`. Оригинальная причина (amqp-ошибка) доступна через `errors.Unwrap` для логирования.

| Sentinel           | Когда возвращается                                          |
| ------------------ | ----------------------------------------------------------- |
| `ErrInvalidConfig` | `NewConsumer` — отсутствует обязательное поле конфигурации  |
| `ErrConnect`       | `NewClient`, `Reconnect` — невозможно установить соединение |
| `ErrPublish`       | Ошибка публикации в delay или dead-letter exchange          |
| `ErrStopped`       | Попытка reconnect на остановленном consumer-е               |

```go
client, err := rmq.NewClient(log, cfg)
switch {
case errors.Is(err, rmq.ErrConnect):
    log.Fatal("cannot connect to RabbitMQ", errors.Unwrap(err)) // оригинальная amqp ошибка
}

consumer, err := rmq.NewConsumer(log, client, cfg)
switch {
case errors.Is(err, rmq.ErrInvalidConfig):
    log.Fatal("consumer misconfigured:", err)
}
```

## Топология очередей

`Start()` объявляет всю топологию через отдельный init-канал (идемпотентно). При изменении параметров (TTL, exchange) очередь нужно удалить вручную — RabbitMQ вернёт `PRECONDITION_FAILED`.

```
publish: notification.email.verification
  → notifications (topic exchange)
  → notification.email.queue  [binding: notification.email.*]
      ↓ handler вернул error, retry < MaxRetry
  → notifications.delay (topic exchange) + инкремент x-retry-count
  → notification.email.delay.queue  [binding: notification.email.*]
      ↓ TTL истёк → notifications с оригинальным ключом
      ↓ retry >= MaxRetry
  → notifications.dlx (topic exchange)
  → notification.email.dlq  [binding: notification.email.*]
```

Оригинальный routing key сохраняется на всём пути (`x-dead-letter-routing-key` не задаётся).

## Логика обработки ошибок

| Событие                                      | Действие                                                               |
| -------------------------------------------- | ---------------------------------------------------------------------- |
| `Handler` вернул `nil`                       | `Ack`                                                                  |
| `Handler` вернул `error`, `retry < MaxRetry` | Публикация в delay exchange с инкрементом `x-retry-count`, `Ack`       |
| `retry >= MaxRetry`                          | Публикация в DLQ, `Ack`                                                |
| `Delay` не настроен                          | При любой ошибке сразу DLQ                                             |
| Публикация в delay упала                     | Fallback в DLQ                                                         |
| Публикация в DLQ упала                       | `Nack(false, false)` — RabbitMQ применяет dead-letter политику очереди |
| Канал закрылся (обрыв соединения)            | Автоматический `Reconnect` и рестарт с экспоненциальным backoff        |

## Трассировка

Заголовки `x-trace-id` и `x-request-id` автоматически извлекаются из входящего сообщения и записываются в контекст через `logger.TraceIDKey` / `logger.RequestIDKey`. Это обеспечивает сквозную трассировку: `api-gateway → RabbitMQ → consumer`.

```go
Handler: func(ctx context.Context, body []byte, routingKey string) error {
    // ctx уже содержит trace_id и request_id из заголовков сообщения
    log.Info(ctx, "processing", nil) // → [INFO] processing | trace_id=... request_id=...
    return nil
},
```

## Каналы

Consumer использует три AMQP-канала с разными ролями:

| Канал            | Назначение                                               |
| ---------------- | -------------------------------------------------------- |
| `initChannel`    | Объявление топологии при старте; закрывается сразу после |
| `consumeChannel` | `channel.Consume` с `Qos(1, 0, false)`                   |
| `publishChannel` | Публикация в delay и DLQ                                 |

Разделение гарантирует, что ошибка объявления очереди не закроет consume-канал, а `Qos(1)` обеспечивает корректный round-robin при нескольких экземплярах consumer-а.

## Тесты

```bash
# Unit-тесты (без внешних зависимостей)
go test -v -race ./...

# Интеграционные тесты (требуют Docker)
go test -v -race -tags=integration ./...
```

## Лицензия

MIT — см. [LICENSE](../LICENSE).

## Контакты

- Email: morewiktor@yandex.ru
- Telegram: [@MoreWiktor](https://t.me/MoreWiktor)
- GitHub: [@MoreWiktor](https://github.com/MoreWiktor)
