# pkg/logger

Структурированный логгер для Go-микросервисов SSO Easy.

## Установка

```bash
go get github.com/ssoeasy-dev/pkg/logger@latest
```

## Использование

```go
import "github.com/ssoeasy-dev/pkg/logger"

log := logger.NewLogger(logger.EnvironmentDevelopment, "auth.svc")

log.Info(ctx, "User registered", map[string]any{"user_id": id})
log.Warn(ctx, "Retry attempt", map[string]any{"retry": 2})
log.Error(ctx, "DB error", map[string]any{"error": err})
log.Debug(ctx, "Token parsed", map[string]any{"claims": claims}) // только в dev/local
```

## API

```go
func NewLogger(environment Environment, prefix string) *Logger

func (l *Logger) Info(ctx context.Context, msg string, fields map[string]any)
func (l *Logger) Warn(ctx context.Context, msg string, fields map[string]any)
func (l *Logger) Error(ctx context.Context, msg string, fields map[string]any)
func (l *Logger) Debug(ctx context.Context, msg string, fields map[string]any)
```

## Окружения

| Константа                | Значение        | Debug |
| ------------------------ | --------------- | ----- |
| `EnvironmentDevelopment` | `"development"` | ✅    |
| `EnvironmentLocal`       | `"local"`       | ✅    |
| `EnvironmentProduction`  | `"production"`  | ❌    |
| `EnvironmentTest`        | `"test"`        | ❌    |

`Debug` выводится только в `development` и `local`. В остальных окружениях вызовы `Debug` игнорируются.

## Контекст

`trace_id` и `request_id` автоматически извлекаются из контекста и добавляются в каждую запись:

```go
ctx = context.WithValue(ctx, logger.TraceIDKey, "abc-123")
ctx = context.WithValue(ctx, logger.RequestIDKey, "req-456")

log.Info(ctx, "handled", nil)
// [INFO] handled | trace_id=abc-123 request_id=req-456
```

Ключи для записи в контекст:

```go
logger.TraceIDKey   // ctxKey("trace_id")
logger.RequestIDKey // ctxKey("request_id")
```

## Формат вывода

```
auth.svc 2025/01/01 12:00:00 logger.go:42: [INFO] User registered | trace_id=abc request_id=xyz user_id=1
```

`trace_id` и `request_id` всегда выводятся первыми, остальные поля — в произвольном порядке.

## Лицензия

MIT — см. [LICENSE](../LICENSE).

## Контакты

- Email: morewiktor@yandex.ru
- Telegram: [@MoreWiktor](https://t.me/MoreWiktor)
- GitHub: [@MoreWiktor](https://github.com/MoreWiktor)
