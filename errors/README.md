# pkg/errors

Стандартизированная обработка ошибок в Go-микросервисах SSO Easy с поддержкой типизированных видов ошибок (error kinds) и сохранением деталей.

## Установка

```bash
go get github.com/ssoeasy-dev/pkg/errors@latest
```

## Быстрый старт

```go
import "github.com/ssoeasy-dev/pkg/errors"

// Создание ошибки с видом "not found" и форматированным сообщением
err := errors.New(errors.ErrNotFound, "user with id %d not found", 42)

// Проверка вида ошибки
if errors.Is(err, errors.ErrNotFound) {
    // Обработка ситуации "не найдено"
}

// Извлечение исходной ошибки (без вида)
unwrapped := errors.Unwrap(err) // *fmt.Errorf("user with id 42 not found")
```

## Виды ошибок (Error Kinds)

Пакет предоставляет предопределённые виды ошибок, сгруппированные по категориям. Все они реализуют интерфейс `error`.

### CRUD-операции

```go
errors.ErrCreationFailed   // "creation failed"
errors.ErrUpdateFailed     // "update failed"
errors.ErrDeleteFailed     // "delete failed"
errors.ErrGetFailed        // "get failed"
```

### Базы данных

```go
errors.ErrNotFound         // "not found"
errors.ErrAlreadyExists    // "already exists"
errors.ErrForeignKey       // "foreign key violation"
errors.ErrCheckViolation   // "check constraint violation"
errors.ErrNotNullViolation // "not null violation"
errors.ErrDeadlock         // "deadlock detected"
errors.ErrLockTimeout      // "lock timeout"
errors.ErrResourceExhausted// "resource exhausted"
errors.ErrPermissionDenied // "permission denied"
errors.ErrUnavailable      // "database unavailable"
errors.ErrInvalidArgument  // "invalid argument"
```

### Транзакции

```go
errors.ErrTxBegin    // "failed to begin transaction"
errors.ErrTxCommit   // "failed to commit transaction"
errors.ErrTxRollback // "failed to rollback transaction"
```

### Контекст и валидация

```go
errors.ErrCanceled         // контекст отменён (аналог context.Canceled)
errors.ErrDeadlineExceeded // превышен дедлайн (аналог context.DeadlineExceeded)
errors.ErrUnauthenticated  // "Unauthenticated"
errors.ErrValidation       // "validation error"
errors.ErrUnknown          // "unknown error"
```

## Подробное использование

### Создание ошибок

Пакет предоставляет два способа создания ошибок:

1. Скрытые детали (по умолчанию)

```go
err := errors.New(errors.ErrNotFound, "user %s not found", username)

fmt.Println(err.Error()) // "not found"
```

Метод `Error()` возвращает только строку вида — детали скрыты от клиента. Детали сохраняются и доступны через `Unwrap()`.

2. Явные детали (verbose)

```go
err := errors.NewVerbose(errors.ErrUnauthenticated, "payment required for subscription %d", 42)

fmt.Println(err.Error()) // "payment required for subscription 42"
```

Метод `Error()` возвращает полное форматированное сообщение. Используйте этот вариант, когда клиенту необходимо получить поясняющие детали (например, доменные ошибки).

### Проверка вида ошибки

Используйте `errors.Is`, как в стандартной библиотеке:

```go
if errors.Is(err, errors.ErrNotFound) {
    // ошибка относится к типу "не найдено"
}
```

### Извлечение деталей

Детали ошибки доступны через стандартный `errors.Unwrap`:

```go
details := errors.Unwrap(err) // ошибка с форматированным сообщением
```

### Получение вида ошибки

Для получения вида ошибки используйте интерфейс `errors.Kinded`:

```go
if kinded, ok := err.(errors.Kinded); ok {
    kind := kinded.Kind()
    fmt.Println(kind) // "not found"
}
```

### Встраивание в цепочки ошибок

Ошибки, созданные через `New` или `NewVerbose`, реализуют интерфейс `Unwrap`, поэтому их можно комбинировать с другими ошибками:

```go
err := errors.New(errors.ErrNotFound, "user not found")
err = fmt.Errorf("failed to get user: %w", err)

// Проверка работает сквозь обёртки
if errors.Is(err, errors.ErrNotFound) {
    // true
}
```

### Пользовательские виды

Можно определять собственные виды, если предопределённых недостаточно:

```go
var ErrMyCustom = errors.New("my custom error")

err := errors.New(ErrMyCustom, "something went wrong")

if errors.Is(err, ErrMyCustom) {
    // ...
}
```

## API

`func New(kind error, format string, args ...any) error`
Создаёт ошибку со скрытыми деталями (метод `Error()` возвращает только строку вида). Реализует интерфейсы `Kinded`, `Unwrap`, `Is`.

`func NewVerbose(kind error, format string, args ...any) error`
Создаёт ошибку с явными деталями (метод `Error()` возвращает полное сообщение). Реализует те же интерфейсы, что и `New`.

`type Kinded interface { Kind() error }`
Интерфейс для получения вида ошибки.

`func Is(err error, target error) bool`
Делегирует вызов `errors.Is` из стандартной библиотеки.

`func As(err error, target any) bool`
Делегирует вызов `errors.As` из стандартной библиотеки.

`func Unwrap(err error) error`
Делегирует вызов `errors.Unwrap` из стандартной библиотеки.

## Тесты

Пакет покрыт модульными тестами:

```bash
go test -v -race ./...
```

Пример тестов из документации:

```go
package errors_test

import (
    "testing"
    "github.com/ssoeasy-dev/pkg/errors"
    "github.com/stretchr/testify/assert"
)

func TestError_WrapAndUnwrap(t *testing.T) {
    kind := errors.ErrNotFound
    wrapped := errors.New(kind, "user with id %d not found", 42)

    assert.ErrorIs(t, wrapped, kind)
    assert.NotErrorIs(t, wrapped, errors.ErrAlreadyExists)

    unwrapped := errors.Unwrap(wrapped)
    assert.EqualError(t, unwrapped, "user with id 42 not found")
}
```

## Лицензия

MIT — см. [LICENSE](../LICENSE).

## Контакты

- Email: [morewiktor@yandex.ru](mailto:morewiktor@yandex.ru)
- Telegram: [@MoreWiktor](https://t.me/MoreWiktor)
- GitHub: [@MoreWiktor](https://github.com/MoreWiktor)

