# pkg/grpc

Настройка gRPC-сервера со стандартным набором интерцепторов для Go-микросервисов SSO Easy.

## Установка

```bash
go get github.com/ssoeasy-dev/pkg/grpc@latest
```

## Использование

```go
import pkggrpc "github.com/ssoeasy-dev/pkg/grpc"

// Создание сервера
srv := pkggrpc.NewServer(":50052", log)

// Регистрация gRPC хендлеров
pb.RegisterAuthServiceServer(srv.GetGRPCServer(), authHandler)
pb.RegisterVerificationServiceServer(srv.GetGRPCServer(), verificationHandler)

// Опционально: gRPC reflection (для grpcurl и отладки)
srv.RegisterReflection()

// Запуск (блокирующий)
go func() {
    if err := srv.Start(); err != nil {
        log.Error(ctx, "gRPC server error", map[string]any{"error": err})
    }
}()

// Graceful shutdown
srv.Stop()
```

## API

```go
func NewServer(addr string, log *logger.Logger) *Server

func (s *Server) GetGRPCServer() *grpc.Server  // регистрация сервисов
func (s *Server) RegisterReflection()           // gRPC reflection
func (s *Server) Start() error                  // блокирующий запуск
func (s *Server) Stop()                         // graceful shutdown
```

## Встроенные интерцепторы

Применяются автоматически при `NewServer`:

| Интерцептор                 | Описание                                                                                                               |
| --------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| `traceIDInterceptor`        | Читает `x-trace-id` из metadata; генерирует UUID если отсутствует; записывает в контекст через `logger.TraceIDKey`     |
| `requestIDInterceptor`      | Читает `x-request-id` из metadata; генерирует UUID если отсутствует; записывает в контекст через `logger.RequestIDKey` |
| `loggingInterceptor`        | Логирует каждый вызов: метод, длительность, gRPC код ответа                                                            |
| `recoveryInterceptor`       | Перехватывает panic, логирует stack trace, возвращает `codes.Internal`                                                 |
| `streamRecoveryInterceptor` | Recovery для stream вызовов                                                                                            |

Trace ID и Request ID из metadata автоматически становятся доступны через `logger` в любом месте обработки запроса.

## Интеграция с api-gateway

`api-gateway` пробрасывает `x-trace-id` и `x-request-id` в gRPC metadata через `contextInterceptor` при каждом исходящем вызове. Таким образом trace-контекст сквозной: HTTP → gRPC → лог.

## Лицензия

MIT — см. [LICENSE](../LICENSE).

## Контакты

- Email: morewiktor@yandex.ru
- Telegram: [@MoreWiktor](https://t.me/MoreWiktor)
- GitHub: [@MoreWiktor](https://github.com/MoreWiktor)
