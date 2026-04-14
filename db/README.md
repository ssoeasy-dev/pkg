# pkg/db

Обёртка над GORM для PostgreSQL. Включает Generic Repository с типизированными CRUD-операциями, Transaction Manager и стандартизированные ошибки, совместимые с пакетом `errors`.

## Установка

```bash
go get github.com/ssoeasy-dev/pkg/db@latest
```

## Подключение к БД

```go
import "github.com/ssoeasy-dev/pkg/db"

database, err := db.NewDB(&db.Config{
    Environment: db.EnvironmentDevelopment, // включает GORM query log
    Host:        "localhost",
    Port:        "5432",
    User:        "postgres",
    Password:    "postgres",
    Database:    "auth",
}, log)

if err := database.Ping(); err != nil {
    panic(err)
}

// Получение *gorm.DB
gormDB := database.Conn
```

Размер пула задаётся через конфиг (дефолты: `MaxIdleConns=10`, `MaxOpenConns=100`):

```go
&db.Config{
    // ...
    MaxIdleConns: 5,
    MaxOpenConns: 50,
}
```

### Константы Environment

| Константа                   | Значение        | GORM query log |
| --------------------------- | --------------- | -------------- |
| `db.EnvironmentDevelopment` | `"development"` | ✅             |
| `db.EnvironmentLocal`       | `"local"`       | ✅             |
| `db.EnvironmentTest`        | `"test"`        | ❌             |
| `db.EnvironmentProduction`  | `"production"`  | ❌             |

## Transaction Manager

```go
import "github.com/ssoeasy-dev/pkg/db/tx"

txManager := tx.NewTxManager(gormDB)

// commit при nil, rollback при ошибке
err := txManager.WithTransaction(ctx, func(ctx context.Context) error {
    return repo.Create(ctx, &entity)
})
```

`GetDB(ctx)` возвращает транзакционный `*gorm.DB` если в контексте есть активная транзакция, иначе — обычный:

```go
conn := txManager.GetDB(ctx)
```

### Mock для тестов

```go
mock := tx.NewMockTxManager(nil) // log опциональный, nil допустим
mock.WithTransactionalSuccess(ctx)           // имитирует успешную транзакцию
mock.WithTransactionalRollback(ctx, err)     // имитирует rollback
mock.WithTransactionErrBegin(ctx)            // fn не вызывается, возвращает ErrInternal
mock.WithTransactionErrCommit(ctx)           // fn вызывается, возвращает ErrInternal
mock.WithTransactionErrRollback(ctx)         // fn вызывается, возвращает ErrInternal
```

## Generic Repository

```go
import "github.com/ssoeasy-dev/pkg/db/repository"

repo := repository.NewRepository[User](txManager, log, "User")
```

### Интерфейс

```go
type Repository[Model any] interface {
    DB(ctx context.Context) *gorm.DB
    Create(ctx, *Model, ...RepositoryOption) error
    Update(ctx, map[string]any, ...RepositoryOption) (int64, error)
    Delete(ctx, force bool, ...RepositoryOption) (int64, error)
    FindOne(ctx, ...RepositoryOption) (*Model, error)
    FindAll(ctx, ...RepositoryOption) ([]Model, error)
    Count(ctx, ...RepositoryOption) (int64, error)
    Exists(ctx, ...RepositoryOption) (bool, error)
}
```

`force=true` в `Delete` выполняет hard delete (`.Unscoped()`), `false` — soft delete через `deleted_at`.

### Опции запросов

#### `WithConditions(conditions ...map[string]any)`

Несколько map объединяются через OR, поля внутри одной map — через AND.

```go
// AND внутри map
repo.FindOne(ctx,
    repository.WithConditions(map[string]any{
        "id":         id,
        "deleted_at": nil,
    }),
)

// OR между maps
repo.FindAll(ctx,
    repository.WithConditions(
        map[string]any{"status": "active"},
        map[string]any{"role": "admin"},
    ),
)

// LIKE
repo.FindAll(ctx,
    repository.WithConditions(map[string]any{
        "login": repository.Like("%john%"),
    }),
)

// IS NULL / IS NOT NULL
repo.FindAll(ctx,
    repository.WithConditions(map[string]any{
        "deleted_at": repository.IsNull(true),
    }),
)
```

#### `WithSelect(fields ...string)`

```go
repo.FindOne(ctx,
    repository.WithSelect("id", "login", "is_active"),
)
```

#### `WithPreloads(relation string, opts ...RepositoryOption)`

Поддерживает вложенные preload с условиями:

```go
repo.FindOne(ctx,
    repository.WithPreloads("Policy",
        repository.WithPreloads("Rules"),
        repository.WithConditions(map[string]any{"deleted_at": nil}),
    ),
)
```

#### `WithJoins(joins ...Join)`

```go
repo.FindOne(ctx,
    repository.WithJoins(repository.Join{
        Type:  repository.JoinTypeLeft,
        Table: "users",
        On: repository.JoinON{
            From: "user_id",   // автоматически квалифицируется как main_table.user_id
            To:   "id",        // автоматически квалифицируется как users.id
        },
    }),
    repository.WithConditions(map[string]any{"users.login": login}),
)
```

Доступные типы: `JoinTypeInner`, `JoinTypeLeft`, `JoinTypeRight`.

#### `WithOrder(orders ...Order)`

```go
repo.FindAll(ctx,
    repository.WithOrder(
        repository.Order{By: "created_at", Dir: repository.OrderDirDesc},
    ),
)
```

#### `WithPagination(pagination Pagination)`

```go
repo.FindAll(ctx,
    repository.WithPagination(repository.Pagination{Limit: 20, Page: 2}),
)
```

#### Прочие опции

```go
repository.WithLimit(10)
repository.WithOffset(20)
repository.WithDeleted(true)   // включает soft-deleted записи (Unscoped)
repository.WithScope(func(db *gorm.DB) *gorm.DB { ... })
repository.WithClauses(clause.OnConflict{DoNothing: true})
```

## Ошибки репозитория

Методы репозитория возвращают ошибки пакета `errors`. Используйте `errors.Is` для проверки вида ошибки.  
Оригинальная причина (pgx/gorm ошибка) доступна через `errors.Unwrap`, а полное техническое сообщение — через `errors.FullError` (только для логирования).

```go
import "github.com/ssoeasy-dev/pkg/errors"

err := repo.FindOne(ctx, repository.WithConditions(map[string]any{"id": id}))

switch {
    case errors.Is(err, errors.ErrNotFound):
        // запись не найдена
    case errors.Is(err, errors.ErrAlreadyExists):
        // нарушение unique constraint
        // сообщение содержит имя поля и значение: "already exists: user with email=x@y.com"
    case errors.Is(err, errors.ErrFailedPrecondition):
        // нарушение внешнего ключа или check constraint
    default:
        // логируем полную цепочку
    log.Error(ctx, errors.FullError(err), nil)
}
```

Полный маппинг кодов PostgreSQL на виды ошибок реализован в функции `db.NewError`.  
Основные соответствия:

| PostgreSQL код  | Вид ошибки (`errors.Kind`) | Пояснение                  |
| --------------- | -------------------------- | -------------------------- |
| `23505`         | `ErrAlreadyExists`         | Unique violation           |
| `23503`         | `ErrFailedPrecondition`    | Foreign key violation      |
| `23502`         | `ErrInvalidArgument`       | Not null violation         |
| `23514`         | `ErrFailedPrecondition`    | Check constraint violation |
| `40P01`         | `ErrAborted`               | Deadlock detected          |
| `55P03`         | `ErrInternal`              | Lock timeout               |
| `57P01`,`57P02` | `ErrUnavailable`           | Database shutdown          |
| `08000`‑`08006` | `ErrUnavailable`           | Connection exceptions      |

## MockRepository

Для unit-тестов без реальной БД:

```go
import "github.com/ssoeasy-dev/pkg/db/repository"

mock := repository.NewMockRepository[User]()

// Настраиваем ожидания через testify/mock
mock.On("Create", ctx, mock.Anything).Return(nil)
mock.On("FindOne", ctx, mock.Anything).Return(&User{ID: id}, nil)

// Готовые хелперы для типичных сценариев
mock.OnFindOneReturn(ctx, &user)
mock.FindOneErrNotFound(ctx)
```

## Лицензия

MIT — см. [LICENSE](../LICENSE).

## Контакты

- Email: morewiktor@yandex.ru
- Telegram: [@MoreWiktor](https://t.me/MoreWiktor)
- GitHub: [@MoreWiktor](https://github.com/MoreWiktor)
