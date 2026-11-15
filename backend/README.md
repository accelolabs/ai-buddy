# Backend

Минимальный Go-бэкенд. Конфигурация читается из окружения через cleanenv:

| Переменная | Значение |
| --- | --- |
| `ADDR` | Адрес HTTP-сервера, по умолчанию `:8080` |
| `DATABASE_URL` | Обязательный URL PostgreSQL |

## Docker Compose

Из каталога `backend/`:

```sh
docker compose up --build -d
docker compose logs server migrate
```

Compose ждёт готовности PostgreSQL, применяет миграции отдельным сервисом
`migrate` и запускает сервер на порту 8080. Прикладных маршрутов пока нет:
запросы возвращают HTTP 404. База доступна локально на `127.0.0.1:5432`.
Для нового бэкенда используется отдельный volume `buddy-backend_postgres`.
Порт 8080 должен быть свободен от прежнего бэкенда.

Повторно применить миграции:

```sh
docker compose run --rm --build migrate
```

Остановить сервисы, сохранив данные:

```sh
docker compose down
```

## Локальный запуск

Из каталога `backend/` (нужен Go 1.27):

```sh
docker compose up -d postgres
set -a
. ./.env.example
set +a
go run ./cmd/migrate
go run ./cmd/server
```

Программа сама не загружает `.env`-файлы. `.env.example` содержит только локальные
значения; Compose использует `POSTGRES_PASSWORD` из окружения или `.env`
(по умолчанию `buddy`). При смене пароля нужно также обновить локальный `DATABASE_URL`.

Миграции лежат в `migrations/` и выполняются goose по порядку. Команду
`cmd/migrate` нужно запускать из `backend/`. Начальная миграция проверяет работу
механизма без создания прикладных таблиц; схема появится при разборе первой фичи.

## Проверки

```sh
go test ./...
go vet ./...
go build ./...
golangci-lint fmt --diff
golangci-lint run ./...
docker compose config --quiet
```
