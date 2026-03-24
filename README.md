# pkg

Монорепозиторий shared Go-пакетов для микросервисов SSO Easy. Каждый пакет — отдельный Go-модуль со своим `go.mod` и независимым версионированием.

## Пакеты

| Пакет    | Модуль                              | Последняя версия | Описание                                              |
| -------- | ----------------------------------- | ---------------- | ----------------------------------------------------- |
| `db`     | `github.com/ssoeasy-dev/pkg/db`     | v1.0.10          | Generic репозиторий поверх GORM + transaction manager |
| `logger` | `github.com/ssoeasy-dev/pkg/logger` | v1.0.1           | Структурированный логгер                              |
| `grpc`   | `github.com/ssoeasy-dev/pkg/grpc`   | v1.0.2           | Настройка gRPC-сервера с интерцепторами               |
| `rmq`    | `github.com/ssoeasy-dev/pkg/rmq`    | v1.0.4           | RabbitMQ клиент и consumer с retry/DLQ логикой        |
| `s3`     | `github.com/ssoeasy-dev/pkg/s3`     | v1.0.3           | Generic S3 клиент (AWS, Tinkoff, Yandex, MinIO)       |
| `errors` | `github.com/ssoeasy-dev/pkg/errors` | v1.0.0           | Кастомная обработка ошибок с Kind                     |

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
├── s3/
│   ├── client.go        # S3 клиент: Put, Get, Head, List, Presign
│   ├── config.go        # Config struct
│   └── go.mod
└── errors/
    ├── error_test.go
    ├── error.go
    ├── go.mod
    ├── kind.go
    ├── README.md
    ├── verbose_error_test.go
    └── verbose_error.go
```

## Установка

Каждый пакет устанавливается отдельно:

```bash
go get github.com/ssoeasy-dev/pkg/db@latest
go get github.com/ssoeasy-dev/pkg/logger@latest
go get github.com/ssoeasy-dev/pkg/grpc@latest
go get github.com/ssoeasy-dev/pkg/rmq@latest
go get github.com/ssoeasy-dev/pkg/s3@latest
go get github.com/ssoeasy-dev/pkg/errors@latest
```

Или конкретную версию:

```bash
go get github.com/ssoeasy-dev/pkg/db@v1.0.10
```

## Релизы

Релизы выполняются автоматически при мерже в `main` через GitHub Actions.

**Версионирование следует [Semantic Versioning](https://semver.org/) и управляется через commit messages:**

| Prefix в сообщении коммита | Тип бампа | Пример                                 |
| -------------------------- | --------- | -------------------------------------- |
| `BREAKING` / `major:`      | major     | `BREAKING: remove Repository.RawQuery` |
| `feat:` / `minor:`         | minor     | `feat(db): add WithClauses option`     |
| всё остальное              | patch     | `fix(rmq): handle nil headers`         |

При мерже CI автоматически:

1. Определяет, какие пакеты затронуты (по diff)
2. Вычитывает commit messages и определяет тип бампа
3. Прогоняет тесты
4. Создаёт git-тег вида `<package>/vX.Y.Z`
5. Публикует GitHub Release с changelog

## Разработка

### Зависимости для разработки

```bash
# Go 1.24+
go version

# golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
  | sh -s -- -b $(go env GOPATH)/bin v2.4.0
```

### Тесты

```bash
# Один пакет
cd db && go test -v -race ./...

# Все пакеты
for pkg in db logger grpc rmq s3; do
  echo "=== $pkg ===" && cd $pkg && go test -race ./... && cd ..
done
```

### CI

На каждый push и PR GitHub Actions запускает для каждого пакета:

- `golangci-lint` — линтинг
- `go test -race` — тесты с детектором гонок

### Разработка с локальной заменой

Если правите `pkg` и сервис одновременно, используйте `replace` в `go.mod` сервиса:

```go
// auth.svc/go.mod
replace github.com/ssoeasy-dev/pkg/db => ../pkg/db
```

Убирайте `replace` перед мержем.

Если нужно протестировать изменения из `develop` до релиза — ссылайтесь по commit hash:

```bash
go get github.com/ssoeasy-dev/pkg/db@<commit-hash>
# go.mod получит псевдо-версию: v1.0.11-0.20260320143021-abc1234f8b9a
```

### Добавление нового пакета

1. Создать директорию `pkg/<name>/`
2. Инициализировать модуль: `go mod init github.com/ssoeasy-dev/pkg/<name>`
3. Добавить пакет в матрицу CI: `.github/workflows/lint.yml`, `.github/workflows/test.yml`
4. Добавить в список `ALL_PACKAGES` в `.github/workflows/release.yml`

## Лицензия

MIT — см. [LICENSE](LICENSE).

## Контакты

- Email: morewiktor@yandex.ru
- Telegram: [@MoreWiktor](https://t.me/MoreWiktor)
- GitHub: [@MoreWiktor](https://github.com/MoreWiktor)
