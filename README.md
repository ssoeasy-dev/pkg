# pkg

Монорепозиторий shared Go-пакетов для микросервисов SSO Easy. Каждый пакет — отдельный Go-модуль со своим `go.mod` и версионированием.

## Пакеты

| Пакет    | Модуль                              | Описание                                              |
| -------- | ----------------------------------- | ----------------------------------------------------- |
| `db`     | `github.com/ssoeasy-dev/pkg/db`     | Generic репозиторий поверх GORM + transaction manager |
| `logger` | `github.com/ssoeasy-dev/pkg/logger` | Структурированный логгер                              |
| `grpc`   | `github.com/ssoeasy-dev/pkg/grpc`   | Настройка gRPC-сервера с интерцепторами               |
| `rmq`    | `github.com/ssoeasy-dev/pkg/rmq`    | RabbitMQ клиент и consumer с retry/DLQ логикой        |
| `s3`     | `github.com/ssoeasy-dev/pkg/s3`     | Generic S3 клиент (AWS, Tinkoff, Yandex, MinIO)       |

## Структура репозитория

```
pkg/
├── db/
│   ├── repository/      # Generic Repository[Model], опции запросов
│   ├── tx/              # TxManager — управление транзакциями
│   └── go.mod
├── logger/
│   └── go.mod
├── grpc/
│   └── go.mod
├── rmq/
│   ├── client.go        # RabbitMQ клиент
│   ├── consumer.go      # Consumer с retry, delay queue, DLQ
│   └── go.mod
└── s3/
    ├── client.go        # S3 клиент: Put, Get, Head
    ├── config.go        # Config struct
    └── go.mod
```

## Установка

Каждый пакет устанавливается отдельно:

```bash
go get github.com/ssoeasy-dev/pkg/db@latest
go get github.com/ssoeasy-dev/pkg/logger@latest
go get github.com/ssoeasy-dev/pkg/grpc@latest
go get github.com/ssoeasy-dev/pkg/rmq@latest
go get github.com/ssoeasy-dev/pkg/s3@latest
```

## Разработка

### CI

На каждый push GitHub Actions запускает для каждого пакета (`db`, `logger`, `grpc`, `rmq`, `s3`):

- `golangci-lint` — линтинг
- `go test -race` — тесты с детектором гонок

### Добавление нового пакета

1. Создать директорию `pkg/<n>/`
2. Инициализировать модуль: `go mod init github.com/ssoeasy-dev/pkg/<n>`
3. Добавить пакет в матрицу CI в `.github/workflows/lint.yml` и `test.yml`

### Обновление зависимости в микросервисах

```bash
# В директории микросервиса
go get github.com/ssoeasy-dev/pkg/s3@latest
go mod tidy
```

## Лицензия

MIT — см. [LICENSE](LICENSE).

## Контакты

- Email: morewiktor@yandex.ru
- Telegram: [@MoreWiktor](https://t.me/MoreWiktor)
- GitHub: [@MoreWiktor](https://github.com/MoreWiktor)
