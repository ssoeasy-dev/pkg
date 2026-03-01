# pkg/rmq

RabbitMQ клиент и consumer с retry через delay-очередь и DLQ для Go-микросервисов SSO Easy.

## Установка

```bash
go get github.com/ssoeasy-dev/pkg/rmq@latest
```

## Клиент

```go
import "github.com/ssoeasy-dev/pkg/rmq"

client, err := rmq.NewClient(log, &rmq.Config{
    Host:     "localhost",
    Port:     5672,
    User:     "guest",
    Password: "guest",
})

defer client.Close()
```

## Consumer

```go
consumer, err := rmq.NewConsumer(log, client, &rmq.ConsumerConfig{
    Main: rmq.QueueConfig{
        Exchange:       "notifications",
        BindingPattern: "notification.email.*",
        Queue:          "notification.email.queue",
        TTL:            15 * 60 * 1000, // мс, 15m
    },
    Dead: &rmq.QueueConfig{
        Exchange:       "dlx",
        BindingPattern: "notification.email.*",
        Queue:          "notification.email-dlx.queue",
    },
    Delay: &rmq.DelayQueueConfig{
        QueueConfig: rmq.QueueConfig{
            Exchange:       "delay",
            BindingPattern: "notification.email.*",
            Queue:          "notification.email.delay.queue",
            TTL:            5 * 1000, // мс, 5s между попытками
        },
        MaxRetry: 3,
    },
    Handler: func(ctx context.Context, body []byte, routingKey string) error {
        // обработка сообщения
        // возврат error → retry; nil → ack
        return nil
    },
})
if err != nil {
    panic(err)
}

// Запуск (неблокирующий)
if err := consumer.Start(ctx); err != nil {
    panic(err)
}

// Graceful shutdown
consumer.Stop()
```

## Типы

```go
type QueueConfig struct {
    Queue          string
    BindingPattern string
    Exchange       string
    TTL            int
}

// DelayQueueConfig встраивает QueueConfig — Exchange, BindingPattern, Queue, TTL обязательны
type DelayQueueConfig struct {
    QueueConfig
    MaxRetry int
}
```

## Топология очередей

При каждом `Start()` consumer объявляет всю топологию через отдельный init-канал. Если очередь уже существует с теми же параметрами — объявление идемпотентно. При изменении параметров (TTL, exchange) нужно удалить очередь вручную — RabbitMQ вернёт `PRECONDITION_FAILED`.

```
publish: notification.email.verification
  → notifications (topic exchange)
  → notification.email.queue  [binding: notification.email.*]
      ↓ ошибка
  → delay (topic exchange) с ключом notification.email.verification
  → notification.email.delay.queue  [binding: notification.email.*]
      ↓ TTL истёк → notifications с оригинальным ключом (следующая попытка)
      ↓ retry >= MaxRetry
  → dlx (topic exchange) с ключом notification.email.verification
  → notification.email-dlx.queue  [binding: notification.email.*]
```

Оригинальный routing key сохраняется на всём пути — `x-dead-letter-routing-key` не задаётся нигде, RabbitMQ сохраняет ключ автоматически.

При масштабировании новый consumer добавляет свои очереди к тем же `delay` и `dlx` exchange через свой паттерн — без конфликтов:

```
delay (topic exchange)
  ├── notification.email.*  → notification.email.delay.queue
  └── notification.sms.*    → notification.sms.delay.queue
```

## Логика обработки ошибок

| Событие                                      | Действие                                                               |
| -------------------------------------------- | ---------------------------------------------------------------------- |
| `Handler` вернул `nil`                       | `Ack`                                                                  |
| `Handler` вернул `error`, `retry < MaxRetry` | Публикация в delay exchange с инкрементом `x-retry-count`, `Ack`       |
| `retry >= MaxRetry`                          | Публикация в DLQ, `Ack`                                                |
| Delay-очередь не настроена                   | При любой ошибке сразу DLQ                                             |
| Публикация в delay упала                     | Fallback в DLQ                                                         |
| Публикация в DLQ упала                       | `Nack(false, false)` — RabbitMQ применяет dead-letter политику очереди |
| Канал закрылся (обрыв соединения)            | Автоматический reconnect и рестарт                                     |

## Каналы

Consumer использует три отдельных AMQP-канала:

| Канал            | Назначение                                               |
| ---------------- | -------------------------------------------------------- |
| `initChannel`    | Объявление топологии при старте, закрывается сразу после |
| `consumeChannel` | `channel.Consume` с `Qos(1, 0, false)`                   |
| `publishChannel` | Публикация в delay и DLQ                                 |

Разделение гарантирует что ошибка при объявлении очереди не закроет consume-канал, а `Qos(1)` обеспечивает корректный round-robin при нескольких consumer-ах.

## Трассировка

`x-trace-id` и `x-request-id` из headers сообщения автоматически записываются в контекст через `logger.TraceIDKey` / `logger.RequestIDKey`. Это обеспечивает сквозную трассировку: `auth.api → RabbitMQ → notificator.svc`.

## Десериализация сообщений

```go
var msg MyMessage
if err := rmq.UnmarshalMessage(body, &msg); err != nil {
    return fmt.Errorf("failed to unmarshal: %w", err)
}
```

## API

```go
// Клиент
func NewClient(log *logger.Logger, cfg *Config) (*Client, error)
func (c *Client) Close() error

// Consumer — NewConsumer валидирует конфиг и возвращает ошибку при неполных данных
func NewConsumer(log *logger.Logger, client *Client, cfg *ConsumerConfig) (*Consumer, error)
func (c *Consumer) Start(ctx context.Context) error
func (c *Consumer) Stop()

// Хелпер
func UnmarshalMessage(data []byte, v any) error
```

## Лицензия

MIT — см. [LICENSE](../LICENSE).

## Контакты

- Email: morewiktor@yandex.ru
- Telegram: [@MoreWiktor](https://t.me/MoreWiktor)
- GitHub: [@MoreWiktor](https://github.com/MoreWiktor)
