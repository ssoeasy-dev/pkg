# Go Shared Packages

Монорепозиторий общих пакетов для Go-сервисов компании.

## Пакеты

| Пакет | Описание | Версия |
|-------|----------|---------|
| [`pkg/db`](./pkg/db) | Работа с БД | ![DB Version](https://img.shields.io/github/v/tag/yourcompany/go-packages?filter=pkg/db/*) |
| [`pkg/logger`](./pkg/logger) | Логирование | ![Logger Version](https://img.shields.io/github/v/tag/yourcompany/go-packages?filter=pkg/logger/*) |
| [`pkg/grpc`](./pkg/grpc) | gRPC клиент | ![gRPC Version](https://img.shields.io/github/v/tag/yourcompany/go-packages?filter=pkg/grpc/*) |
| [`pkg/rmq`](./pkg/rmq) | RabbitMQ клиент | ![RMQ Version](https://img.shields.io/github/v/tag/yourcompany/go-packages?filter=pkg/rmq/*) |

## Использование

```go
import (
    "github.com/ssoeasy-dev/pkg/db/v1.0.0"
    "github.com/ssoeasy-dev/pkg/logger/v0.1.0"
)