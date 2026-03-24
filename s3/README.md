# pkg/s3

Универсальный S3‑клиент для Go‑микросервисов SSO Easy. Построен поверх aws-sdk-go-v2.
Совместим с любым S3‑совместимым хранилищем: AWS S3, Yandex Object Storage, MinIO, Tinkoff Cloud и другими.

## Возможности

- Простой и идиоматичный API
- Поддержка path‑style URL (необходимо для MinIO и большинства non‑AWS хранилищ)
- Предподписанные ссылки (pre‑signed URLs) с кэшированием клиента
- Частичная выгрузка (Range запросы) для стриминга
- Автоматическая пагинация при получении списка объектов
- Структурированные ошибки с использованием общего пакета errors

## Установка

```bash
go get github.com/ssoeasy-dev/pkg/s3@latest
```

## Использование

### Создание клиента

```go
import s3pkg "github.com/ssoeasy-dev/pkg/s3"

cfg := &s3pkg.Config{
    Endpoint:       "https://storage.yandexcloud.net",
    Region:         "ru-central1",
    AccessKey:      os.Getenv("S3_ACCESS_KEY"),
    SecretKey:      os.Getenv("S3_SECRET_KEY"),
    Bucket:         "my-bucket",
    ForcePathStyle: true, // обязательно для не-AWS хранилищ
}
client, err := s3pkg.NewClient(cfg)
if err != nil {
    log.Fatal(err)
}
```

### Загрузка объекта (Put)

```go
ctx := context.Background()
key := "videos/abc.mp4"
r := strings.NewReader("hello world")
contentType := "text/plain"

res, err := client.Put(ctx, key, r, &contentType)
if err != nil {
    log.Fatal(err)
}
fmt.Println("ETag:", res.ETag)
```

### Скачивание объекта (Get)

```go
body, meta, err := client.Get(ctx, key, nil) // nil = весь объект
if err != nil {
    log.Fatal(err)
}
defer body.Close()

data, _ := io.ReadAll(body)
fmt.Printf("Размер: %d, тип: %s\n", meta.ContentLength, meta.ContentType)
```

#### Частичная выгрузка (Range)

```go
rangeHeader := "bytes=0-1023"
body, meta, err := client.Get(ctx, key, &rangeHeader)
// ...
```

### Получение метаданных без тела (Head)

```go
meta, err := client.Head(ctx, key)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Размер: %d, ETag: %s\n", meta.ContentLength, meta.ETag)
```

### Список объектов по префиксу (List)

Автоматически обрабатывает пагинацию.

```go
items, err := client.List(ctx, "videos/")
for _, item := range items {
    fmt.Printf("%s — %d байт, обновлён %s\n", item.Key, item.Size, item.UpdatedAt)
}
```

### Генерация предподписанной ссылки (Presign)

```go
url, err := client.Presign(ctx, key, 15*time.Minute)
if err != nil {
    log.Fatal(err)
}
// ссылка действительна 15 минут
```

## Обработка ошибок

Все ошибки, возвращаемые пакетом, можно анализировать с помощью errors.Is и предопределённых типов из общего пакета github.com/ssoeasy-dev/pkg/errors:

| Sentinel                    | Когда возвращается                                      |
| --------------------------- | ------------------------------------------------------- |
| `errors.ErrInvalidArgument` | Некорректные аргументы (пустой ключ, nil reader и т.п.) |
| `errors.ErrNotFound`        | Запрашиваемый объект не существует.                     |
| `errors.ErrCreationFailed`  | Не удалось создать или загрузить объект.                |
| `errors.ErrGetFailed`       | Не удалось получить или перечислить объекты.            |

Пример:

```go
if errors.Is(err, errors.ErrNotFound) {
    // обработка отсутствия объекта
}
```

### Типы

```go
// PutResult возвращается при успешной загрузке
type PutResult struct {
    ETag string // ETag загруженного объекта
}

// ObjectMetadata содержит метаданные объекта
type ObjectMetadata struct {
    ETag          string // ETag объекта
    ContentType   string // MIME-тип
    ContentLength int64  // размер в байтах
    Range         string // возвращённый диапазон (только при Range‑запросе)
}

// ListResult представляет один объект в списке
type ListResult struct {
    Key       string    // ключ объекта
    Size      int64     // размер в байтах
    ETag      string    // ETag
    UpdatedAt time.Time // время последнего изменения
}
```

## Конфигурация

| Поле             | Тип    | Описание                                                    |
| ---------------- | ------ | ----------------------------------------------------------- |
| `Endpoint`       | string | URL хранилища. Пусто для AWS S3.                            |
| `Region`         | string | Регион. Например `"ru-central1"`, `"us-east-1"`.            |
| `AccessKey`      | string | Access key для аутентификации.                              |
| `SecretKey`      | string | Secret key для аутентификации.                              |
| `Bucket`         | string | Имя бакета.                                                 |
| `ForcePathStyle` | bool   | Path-style URLs. Требуется для MinIO и большинства non-AWS. |

## Тестирование

- Модульные тесты: `go test ./...`
Проверяют валидацию аргументов и логику без внешних зависимостей.

- Интеграционные тесты: `go test -tags=integration ./...`
Запускают контейнер MinIO через [testcontainers](https://golang.testcontainers.org/) и выполняют реальные операции.
Для стабильности CI используется `t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")`.

## Лицензия

MIT — см. [LICENSE](../LICENSE).

## Контакты

- Email: morewiktor@yandex.ru
- Telegram: [@MoreWiktor](https://t.me/MoreWiktor)
- GitHub: [@MoreWiktor](https://github.com/MoreWiktor)
