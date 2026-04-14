# pkg/errors

Стандартизированная обработка ошибок в Go-микросервисах SSO Easy с поддержкой типизированных видов ошибок (error kinds) и разделением публичных и технических сообщений.

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

## Основные концепции

### Публичное vs техническое сообщение

Каждая ошибка, созданная через этот пакет, содержит два уровня информации:

- Публичное сообщение — то, что видит клиент при вызове `err.Error()`. Оно не раскрывает технических деталей реализации.
- Техническое сообщение — полная цепочка причин, доступная через `errors.FullError(err)`. Используется только для логирования.

Пример:

```go
dbErr := sql.ErrConnDone
appErr := errors.NewWrap(errors.ErrUnavailable, dbErr, "failed to fetch user")

fmt.Println(appErr.Error())           // "failed to fetch user"
fmt.Println(errors.FullError(appErr)) // "failed to fetch user: sql: connection is already closed"
```

## API

### Создание ошибок

`New(kind error, msg string) error`
Создаёт корневую ошибку с указанным видом и техническим сообщением.
`Error()` вернёт строку вида, `FullError()` — техническое сообщение.

`Newf(kind error, format string, args ...any) error`
Форматированный вариант `New`.

`Wrap(err error, msg string) error`
Оборачивает существующую ошибку, добавляя публичное сообщение.
Вид (`Kind`) берётся из обёрнутой ошибки (если она была создана через этот пакет).

`Wrapf(err error, format string, args ...any) error`
Форматированный вариант `Wrap`.

`NewWrap(kind error, cause error, msg string) error`
Создаёт ошибку с явно заданным видом, оборачивая `cause` и добавляя публичное сообщение.
Эквивалентно созданию ошибки с заданным `kind`, где `cause` является причиной.

`NewWrapf(kind error, cause error, format string, args ...any) error`
Форматированный вариант `NewWrap`.

`WithKind(err error, kind error) error`
Возвращает копию ошибки с изменённым видом, сохраняя всю цепочку.

### Инспекция ошибок

`Kind(err error) error`
Возвращает вид ошибки. Если ошибка не была создана через этот пакет, возвращает `ErrUnknown`.

`Is(err error, target error) bool`
Аналог `errors.Is` из стандартной библиотеки.

`As(err error, target any) bool`
Аналог `errors.As` из стандартной библиотеки.

`Unwrap(err error) error`
Возвращает следующую ошибку в цепочке. Работает как с ошибками пакета, так и с обычными.

`FullError(err error) string`
Возвращает полную строку ошибки с техническими деталями, включая все обёрнутые причины. Предназначена только для логирования.

## Примеры использования

### Создание и оборачивание

```go
// Корневая ошибка
err := errors.New(errors.ErrNotFound, "user id=123 not found in database")

// Добавление контекста без изменения вида
err = errors.Wrap(err, "failed to get user profile")

// Проверка вида
if errors.Is(err, errors.ErrNotFound) {
    // обработать
}

// Логирование полной цепочки
log.Error(errors.FullError(err))
```

### Работа с внешними ошибками

```go
sqlErr := gorm.ErrRecordNotFound
appErr := errors.NewWrap(errors.ErrNotFound, sqlErr, "user not found")

fmt.Println(errors.Kind(appErr))       // ErrNotFound
fmt.Println(appErr.Error())            // "user not found"
fmt.Println(errors.FullError(appErr))  // "user not found: record not found"
```

### Изменение вида ошибки

```go
err := errors.New(errors.ErrInternal, "something went wrong")
// На уровне выше выяснилось, что это ошибка валидации
err = errors.WithKind(err, errors.ErrInvalidArgument)
```

## Тесты

Пакет покрыт модульными тестами:

```bash
go test -v -race ./...
```

## Лицензия

MIT — см. [LICENSE](../LICENSE).

## Контакты

- Email: [morewiktor@yandex.ru](mailto:morewiktor@yandex.ru)
- Telegram: [@MoreWiktor](https://t.me/MoreWiktor)
- GitHub: [@MoreWiktor](https://github.com/MoreWiktor)
