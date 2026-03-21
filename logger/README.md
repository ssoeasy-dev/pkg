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

`NewLogger` возвращает интерфейс `Logger` — используйте его в типах функций и struct-ах:

```go
// NewLogger возвращает Logger (интерфейс), не конкретный тип.
func NewLogger(environment Environment, prefix string) Logger

// Интерфейс:
type Logger interface {
    Debug(ctx context.Context, msg string, fields map[string]any)
    Info(ctx context.Context, msg string, fields map[string]any)
    Warn(ctx context.Context, msg string, fields map[string]any)
    Error(ctx context.Context, msg string, fields map[string]any)
}
```

Принимайте `logger.Logger` (интерфейс) в сигнатурах — это позволяет подставить мок в тестах:

```go
// Правильно:
func NewServer(addr string, log logger.Logger) *Server { ... }

// Неправильно (слишком конкретный тип):
func NewServer(addr string, log *logger.Logger) *Server { ... }
```

## Окружения

| Константа                | Значение        | Debug | IsVerbose() |
| ------------------------ | --------------- | ----- | ----------- |
| `EnvironmentDevelopment` | `"development"` | ✅    | `true`      |
| `EnvironmentLocal`       | `"local"`       | ✅    | `true`      |
| `EnvironmentProduction`  | `"production"`  | ❌    | `false`     |
| `EnvironmentTest`        | `"test"`        | ❌    | `false`     |

`Debug` выводится только в `development` и `local`. Во всех остальных окружениях вызовы `Debug` игнорируются.

`IsVerbose()` доступен и снаружи — полезен для пропуска дорогого форматирования:

```go
if env.IsVerbose() {
    log.Debug(ctx, "expensive", buildDebugFields())
}
```

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

`trace_id` и `request_id` всегда выводятся первыми, остальные поля — в алфавитном порядке (детерминировано).

## Мокирование в тестах

`Logger` — интерфейс, поэтому легко мокируется без дополнительных библиотек:

```go
type noopLogger struct{}

func (noopLogger) Debug(_ context.Context, _ string, _ map[string]any) {}
func (noopLogger) Info(_ context.Context, _ string, _ map[string]any)  {}
func (noopLogger) Warn(_ context.Context, _ string, _ map[string]any)  {}
func (noopLogger) Error(_ context.Context, _ string, _ map[string]any) {}

// В тесте:
svc := NewService(noopLogger{})
```

Или через стандартный `testify/mock`:

```go
type mockLogger struct{ mock.Mock }

func (m *mockLogger) Error(ctx context.Context, msg string, fields map[string]any) {
    m.Called(ctx, msg, fields)
}
// ...
```

## Лицензия

MIT — см. [LICENSE](../LICENSE).

## Контакты

- Email: morewiktor@yandex.ru
- Telegram: [@MoreWiktor](https://t.me/MoreWiktor)
- GitHub: [@MoreWiktor](https://github.com/MoreWiktor)
