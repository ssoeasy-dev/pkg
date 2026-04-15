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
    └── kind.go
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

## Управление версиями

Репозиторий использует полностью автоматизированный процесс выпуска версий на основе Git-тегов и GitHub Actions. Поддерживаются три вида версий:

- **dev** — временные версии для тестирования в Pull Request (PR)
- **beta** — предрелизные версии, создаваемые при мерже в `develop`
- **stable** — стабильные релизы, выпускаемые при мерже в `main`

### Жизненный цикл версий

1. **Открытие PR в `develop`**
   - Workflow `dev-version.yml` определяет изменённые пакеты.
   - Для каждого создаётся dev-тег вида `<pkg>/v<semver>-dev-<sanitized-branch>.<build>`.
   - В комментарий к PR добавляется инструкция по использованию dev-версии.

2. **Обновление PR (новые коммиты)**
   - Workflow повторно запускается на событие `synchronize` и создаёт новые dev-теги **только для пакетов, изменённых в новых коммитах**.
   - Номер сборки инкрементируется.

3. **Мерж PR в `develop`**
   - Workflow `beta-version.yml` строит граф зависимостей между изменёнными пакетами (на основе `go.mod`).
   - Последовательно, по уровням, создаются beta-теги `<pkg>/v<semver>-beta.<N>`.
   - После создания каждого уровня происходит обновление `go.mod` во всех модулях: dev-версии заменяются на свежие beta.
   - По завершении всех уровней отдельный workflow `cleanup-dev-tags.yml` удаляет dev-теги, относящиеся к смерженной ветке.

4. **Мерж `develop` в `main`**
   - Workflow `main-version.yml` аналогично строит граф и выпускает стабильные версии `<pkg>/v<semver>`.
   - После создания стабильных тегов обновляются `go.mod`: beta-версии заменяются на стабильные.
   - Создаются GitHub Releases с автоматически сгенерированным changelog.
   - Beta-теги для всех изменённых пакетов удаляются.

### Вычисление следующей версии (SemVer)

Базовая версия определяется скриптом `scripts/next-version.sh` на основе анализа коммитов после последнего стабильного тега:

| Тип коммита (префикс)               | Увеличение части |
| ----------------------------------- | ---------------- |
| `major(pkg):` или `BREAKING CHANGE` | major            |
| `feat(pkg):` или `minor(pkg):`      | minor            |
| `fix(pkg):` или любые другие        | patch            |

### Динамический граф зависимостей

Скрипт `scripts/build-release-plan.sh` парсит все `go.mod` в репозитории и строит направленный граф зависимостей между пакетами. На его основе формируется топологический порядок выпуска версий — независимые пакеты (например, `errors`, `logger`) выпускаются раньше, чем зависящие от них (`db`, `grpc`, `s3`). Это гарантирует, что в момент обновления `go.mod` все требуемые beta- или stable-теги уже существуют.

### Автоматическая очистка временных тегов

- **dev-теги** удаляются workflow `cleanup-dev-tags.yml` после успешного выпуска beta-версий для смерженной ветки. Ветка извлекается из сообщения merge-коммита (поддерживаются стандартные форматы GitHub UI и CLI).
- **beta-теги** удаляются в `main-version.yml` после выпуска стабильных версий.

Такая архитектура предотвращает накопление мусорных тегов и исключает состояния гонки при обновлении зависимостей.

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

for pkg in db logger grpc rmq s3 errors; do
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
3. Добавить зависимости (при необходимости)
4. Workflows автоматически обнаружат новый пакет через `scripts/list-packages.sh` и включат его в матрицы тестирования и релизов.

## Лицензия

MIT — см. [LICENSE](LICENSE).

## Контакты

- Email: morewiktor@yandex.ru
- Telegram: [@MoreWiktor](https://t.me/MoreWiktor)
- GitHub: [@MoreWiktor](https://github.com/MoreWiktor)
