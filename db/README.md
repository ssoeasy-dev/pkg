# pkg/db

Обёртка над GORM для PostgreSQL. Включает Generic Repository с типизированными CRUD-операциями, Transaction Manager и стандартизированные ошибки репозитория.

## Установка

```bash
go get github.com/ssoeasy-dev/pkg/db@latest
```

## Подключение к БД

```go
import "github.com/ssoeasy-dev/pkg/db"

database, err := db.NewDB(&db.Config{
    Environment: "development", // включает GORM query log
    Host:        "localhost",
    Port:        "5432",
    User:        "postgres",
    Password:    "postgres",
    Database:    "auth",
}, log)

// Проверка соединения
if err := database.Ping(); err != nil {
    panic(err)
}

// Получение *gorm.DB
gormDB := database.Conn
```

Pool соединений: `MaxIdleConns=10`, `MaxOpenConns=100`.

## Transaction Manager

```go
import "github.com/ssoeasy-dev/pkg/db/tx"

txManager := tx.NewTxManager(gormDB)

// Выполнение в транзакции — commit при nil, rollback при ошибке
err := txManager.WithTransaction(ctx, func(ctx context.Context) error {
    // ctx содержит активную транзакцию
    return repo.Create(ctx, &entity)
})
```

`GetDB(ctx)` возвращает транзакционный `*gorm.DB` если в контексте есть активная транзакция, иначе — обычный `*gorm.DB`:

```go
db := txManager.GetDB(ctx)
```

### Mock для тестов

```go
import "github.com/ssoeasy-dev/pkg/db/tx"

mock := tx.NewMockTxManager(log)
mock.WithTransactionalSuccess(ctx)    // имитирует успешную транзакцию
mock.WithTransactionalRollback(ctx, err) // имитирует rollback
```

## Generic Repository

```go
import "github.com/ssoeasy-dev/pkg/db/repository"

repo := repository.NewRepository[User](txManager, log, "User")
```

### Интерфейс

```go
type Repository[Model any] interface {
    Create(ctx, *Model, ...RepositoryOption) error
    Update(ctx, map[string]any, ...RepositoryOption) (int64, error)
    Delete(ctx, force bool, ...RepositoryOption) (int64, error)
    FindOne(ctx, ...RepositoryOption) (*Model, error)
    FindAll(ctx, ...RepositoryOption) ([]Model, error)
    Count(ctx, ...RepositoryOption) (int64, error)
    Exists(ctx, ...RepositoryOption) (bool, error)
    DB(ctx) *gorm.DB
}
```

`force=true` в `Delete` выполняет hard delete (`.Unscoped()`), `false` — soft delete через `deleted_at`.

### Опции запросов

| Опция                               | Описание                                                  |
| ----------------------------------- | --------------------------------------------------------- |
| `WithConditions(map[string]any...)` | WHERE условия; несколько map объединяются через OR        |
| `WithSelect(fields...)`             | Выборка конкретных полей                                  |
| `WithPreloads(relations...)`        | Eager loading; поддерживает вложенные `"Relation.Nested"` |
| `WithOrder(field, desc bool)`       | Сортировка                                                |
| `WithPagination(limit, offset int)` | Пагинация                                                 |

```go
// Примеры
repo.FindOne(ctx,
    repository.WithConditions(map[string]any{"id": id, "deleted_at": nil}),
    repository.WithPreloads("Policy.Rules", "Attributes"),
)

repo.FindAll(ctx,
    repository.WithOrder("created_at", true),
    repository.WithPagination(20, 0),
)

repo.Delete(ctx, false,
    repository.WithConditions(map[string]any{"id IN": ids}),
)
```

## Ошибки репозитория

Все ошибки приводятся к `*RepositoryError` с sentinel-значениями совместимыми с `errors.Is`:

```go
import "github.com/ssoeasy-dev/pkg/db/repository"

err := repo.FindOne(ctx, repository.WithConditions(map[string]any{"id": id}))
if errors.Is(err, repository.ErrNotFound) {
    // запись не найдена
}
if errors.Is(err, repository.ErrAlreadyExists) {
    // нарушение unique constraint (содержит Field и Value)
}
```

| Sentinel            | Postgres код | Описание                                      |
| ------------------- | ------------ | --------------------------------------------- |
| `ErrNotFound`       | —            | `gorm.ErrRecordNotFound`                      |
| `ErrAlreadyExists`  | `23505`      | Unique violation (заполняет `Field`, `Value`) |
| `ErrForeignKey`     | `23503`      | FK violation                                  |
| `ErrCreationFailed` | —            | Ошибка создания                               |
| `ErrUpdateFailed`   | —            | Ошибка обновления                             |
| `ErrDeleteFailed`   | —            | Ошибка удаления                               |

## Лицензия

MIT — см. [LICENSE](../LICENSE).

## Контакты

- Email: morewiktor@yandex.ru
- Telegram: [@MoreWiktor](https://t.me/MoreWiktor)
- GitHub: [@MoreWiktor](https://github.com/MoreWiktor)
