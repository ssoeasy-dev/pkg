# pkg/grpc

Настройка gRPC-сервера со стандартным набором интерцепторов для Go-микросервисов SSO Easy.

## Установка

```bash
go get github.com/ssoeasy-dev/pkg/grpc@latest
```

## Быстрый старт

```go
import pkggrpc "github.com/ssoeasy-dev/pkg/grpc"

// Создание сервера со стандартными интерцепторами
srv := pkggrpc.NewServer(":50052", log)

// Регистрация gRPC-сервисов
pb.RegisterMyServiceServer(srv.GetGRPCServer(), &myHandler{})

// Опционально: gRPC reflection (для grpcurl, Evans)
srv.RegisterReflection()

// Запуск в горутине (блокирующий)
go func() {
    if err := srv.Start(); err != nil {
        log.Error(ctx, "gRPC server error", map[string]any{"error": err})
    }
}()

// Graceful shutdown
srv.Stop()
```

## Встроенные интерцепторы

`NewServer` применяет следующую цепочку **в порядке выполнения**:

| Интерцептор                      | Тип    | Описание                                                                                    |
| -------------------------------- | ------ | ------------------------------------------------------------------------------------------- |
| `TraceIDInterceptor()`           | Unary  | Читает `x-trace-id` из metadata; генерирует UUID если отсутствует → `logger.TraceIDKey`     |
| `RequestIDInterceptor()`         | Unary  | Читает `x-request-id` из metadata; генерирует UUID если отсутствует → `logger.RequestIDKey` |
| `LoggingInterceptor(log)`        | Unary  | Логирует метод, длительность и gRPC статус; ошибки — на Error, успех — на Info              |
| `RecoveryInterceptor(log)`       | Unary  | Перехватывает panic, логирует stack trace, возвращает `codes.Internal`                      |
| `StreamRecoveryInterceptor(log)` | Stream | То же для stream-хендлеров                                                                  |

### Заголовки трассировки

```go
pkggrpc.HeaderTraceID   // "x-trace-id"
pkggrpc.HeaderRequestID // "x-request-id"
```

Интерцепторы автоматически пробрасывают эти заголовки из gRPC metadata в контекст. Значения затем доступны через логгер, который читает их из контекста при каждом вызове.

## Использование интерцепторов отдельно

Все интерцепторы экспортированы — можно использовать в собственной конфигурации:

```go
import (
    pkggrpc "github.com/ssoeasy-dev/pkg/grpc"
    goGrpc "google.golang.org/grpc"
)

// Сервер без логирования (только трассировка и recovery)
srv := goGrpc.NewServer(
    goGrpc.ChainUnaryInterceptor(
        pkggrpc.TraceIDInterceptor(),
        pkggrpc.RequestIDInterceptor(),
        pkggrpc.RecoveryInterceptor(log),
        myCustomRateLimitInterceptor(),
    ),
    goGrpc.ChainStreamInterceptor(
        pkggrpc.StreamRecoveryInterceptor(log),
    ),
)
```

## Интеграция с api-gateway

`auth.api` пробрасывает `x-trace-id` и `x-request-id` в gRPC metadata через `contextInterceptor` при каждом исходящем вызове. Трейс-контекст сквозной: HTTP → gRPC metadata → context → логгер.

Клиентская сторона:

```go
// При исходящем gRPC-вызове: добавляем trace-id из AsyncLocalStorage
metadata.AppendToOutgoingContext(ctx,
    pkggrpc.HeaderTraceID,   traceId,
    pkggrpc.HeaderRequestID, requestId,
)
```

## Тесты

```bash
# Unit-тесты (без внешних зависимостей)
go test -v -race ./...

# Интеграционные тесты (запускают реальный gRPC-сервер)
go test -v -race -tags=integration ./...
```

## Лицензия

MIT — см. [LICENSE](../LICENSE).

## Контакты

- Email: morewiktor@yandex.ru
- Telegram: [@MoreWiktor](https://t.me/MoreWiktor)
- GitHub: [@MoreWiktor](https://github.com/MoreWiktor)
