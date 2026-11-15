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

На текущем этапе разработки при изменении миграций нужно сбрасывать БД,
удаляя volume, и затем заново применять миграции и сид:

```sh
docker compose down -v
docker compose up --build -d
docker compose run --rm --build seed
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
`cmd/migrate` нужно запускать из `backend/`. Первая миграция
`00001_users.sql` создаёт таблицу `users`.

## Пользователи и сидирование

Пакет `internal/users` содержит сущность `User` с полями `ID`, `Name`, `Role`,
`Team` и PostgreSQL-репозиторий: `Add`, `Get`, `List`. ID задаёт вызывающий код;
повторное добавление того же ID возвращает ошибку. Для отсутствующего ID `Get`
возвращает ошибку, которую можно проверить через `errors.Is(err, sql.ErrNoRows)`.
`List` возвращает пользователей в порядке ID, для пустой таблицы — пустой срез.
Допустимые роли: `intern` и `mentor`.

Сидирование запускается явно после миграций:

```sh
go run ./cmd/seed
```

Либо через Compose (сервис сам дождётся миграций):

```sh
docker compose run --rm --build seed
```

| ID | Имя | Роль | Команда |
| --- | --- | --- | --- |
| `intern.backend.1` | Фёдор Добронравов | Стажёр | backend |
| `intern.backend.2` | Эдуард Радзюкевич | Стажёр | backend |
| `mentor.backend` | Сергей Дорогов | Ментор | backend |
| `intern.platform.1` | Ирина Медведева | Стажёр | platform |
| `intern.platform.2` | Галина Данилова | Стажёр | platform |
| `mentor.platform` | Андрей Кайков | Ментор | platform |

Все шесть пользователей добавляются одной транзакцией. Повторный запуск
пропускает существующие ID, не меняя их данные и других пользователей.
HTTP-операции и отдельная связь стажёра с ментором пока не реализованы.

## Проверки

```sh
go test ./...
go vet ./...
go build ./...
golangci-lint fmt --diff
golangci-lint run ./...
docker compose config --quiet
```

Интеграционные тесты пользователей применяют миграции в отдельных временных
схемах и удаляют эти схемы после проверки. Нужна тестовая PostgreSQL-база
с правом создания схем; без `TEST_DATABASE_URL` эти тесты пропускаются:

```sh
TEST_DATABASE_URL=postgres://buddy:buddy@localhost:5432/buddy_test?sslmode=disable go test -race -count=1 ./...
```
