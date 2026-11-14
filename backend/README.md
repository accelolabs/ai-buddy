# AI Buddy — backend

Один Go-модуль с четырьмя сервисами: Buddy, Tracker, Ticket и Wiki.
У каждого сервиса отдельные процесс и PostgreSQL. Авторизация — через Authentik.
Архитектура и требования: [SDD](../docs/SDD.md).

## Структура

- `cmd/` — серверы, миграции, seed и индексатор Wiki.
- `internal/` — код сервисов и общая инфраструктура `platform`.
- `api/` — контракты OpenAPI.
- `deploy/` — Compose, примеры окружения и инструкции запуска.

## Запуск

Требуются Go 1.27, PostgreSQL и настроенный Authentik; для контейнерного запуска — Docker Compose.
Инструкции и настройки компонентов:

- [Authentik](deploy/authentik/README.md)
- [Ollama](deploy/ollama/README.md)
- [Buddy](deploy/buddy/README.md)
- [Tracker](deploy/tracker/README.md)
- [Ticket](deploy/ticket/README.md)
- [Wiki](deploy/wiki/README.md)

Все команды ниже выполняются из `backend/`. Для локального запуска экспортируйте настройки
из `deploy/<service>/.env.example`. Go-команды не читают `.env` автоматически;
PostgreSQL должна быть доступна с хоста.

Пример для Tracker:

```sh
go run ./cmd/migrate tracker up
go run ./cmd/tracker
```

Миграции: `go run ./cmd/migrate <buddy|tracker|ticket|wiki> [up|down|status]`.
Демо-данные: `go run ./cmd/seed <tracker|ticket>`; порядок подготовки — Authentik → Tracker → Ticket.
Индексация Wiki описана в инструкции сервиса.

Пример сборки и запуска контейнера после настройки `deploy/tracker/.env`:

```sh
docker build --build-arg SERVICE=tracker -t ai-buddy-backend-tracker:local .
docker compose -f deploy/tracker/compose.yaml up --build -d backend-tracker
```

## API и проверки

Контракты OpenAPI: [Buddy](api/buddy.yaml), [Tracker](api/tracker.yaml),
[Ticket](api/ticket.yaml), [Wiki](api/wiki.yaml).

Для проверок нужен [golangci-lint 2.13.2](https://golangci-lint.run/docs/welcome/install/),
собранный с Go не старше версии модуля.

```sh
golangci-lint fmt --diff
golangci-lint run ./...
go vet ./...
go build ./cmd/...
env -u TEST_DATABASE_URL go test -race ./...
```

Для интеграционных тестов используйте отдельную PostgreSQL 16 с pgvector.
Тесты создают и удаляют временные схемы; без `TEST_DATABASE_URL` они пропускаются.

```sh
TEST_DATABASE_URL='postgres://postgres:test@localhost:55432/buddy_test?sslmode=disable' go test -race ./...
```

GitHub Actions выполняет проверки с тестовой PostgreSQL и проверяет конфигурации Compose.
