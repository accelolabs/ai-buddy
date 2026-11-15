# Backend

Минимальный Go-бэкенд. Конфигурация читается из окружения через cleanenv:

| Переменная | Значение |
| --- | --- |
| `ADDR` | Адрес HTTP-сервера, по умолчанию `:8080` |
| `DATABASE_URL` | Обязательный URL PostgreSQL |
| `KNOWLEDGE_BASE_DIR` | Путь к базе знаний, по умолчанию `../knowledge-base` |

## Docker Compose

Из каталога `backend/`:

```sh
docker compose up --build -d
docker compose logs server migrate
```

Compose ждёт готовности PostgreSQL, применяет миграции отдельным сервисом
`migrate` и запускает сервер на порту 8080. Список пользователей доступен
по `GET /api/users`. База доступна локально на `127.0.0.1:5432`.
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
docker compose run --rm --build index
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
`Team` и HTTP-хендлер. PostgreSQL-репозиторий `users.Repository` находится
в `internal/users/repository/repository.go`: `Add`, `Get`, `List`. ID задаёт вызывающий код;
повторное добавление того же ID возвращает ошибку. Для отсутствующего ID `Get`
возвращает ошибку, которую можно проверить через `errors.Is(err, sql.ErrNoRows)`.
`List` возвращает пользователей в порядке ID, для пустой таблицы — пустой срез.
Допустимые роли: `intern` и `mentor`.

Сидирование реализовано функцией `users.Seed(ctx, db)` в `internal/users/seed.go`
и запускается явно после миграций:

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
Отдельная связь стажёра с ментором пока не реализована.

### Получение пользователей

`GET /api/users` возвращает HTTP 200 и JSON-массив всех пользователей
в порядке ID, без авторизации и пагинации:

```json
[
  {"id": "intern.backend.1", "name": "Фёдор Добронравов", "role": "intern", "team": "backend"}
]
```

Для пустой таблицы ответ — `[]`. При ошибке БД возвращается HTTP 500
с текстом `failed to list users`; подробности ошибки записываются в лог.

### Выбранный пользователь

Middleware `users.Middleware(userRepository)(handler)` читает cookie `demo_user`,
декодирует значение, заданное через `encodeURIComponent(user.id)`, и загружает
пользователя методом `Get`. В хендлере он доступен через
`users.UserFromContext(r.Context())`, возвращающий `(User, bool)`.

Отсутствующая или пустая cookie и неизвестный ID дают HTTP 401,
ошибка percent-кодирования — HTTP 400, ошибка репозитория — HTTP 500.
`GET /api/users` остаётся доступным без cookie для выбора пользователя.
Middleware пока не подключён к маршрутам: первым маршрутом, которому нужен
выбранный пользователь, будет список wiki.

## Wiki и индексация

`internal/wiki.Document` содержит `ID`, `Path`, `Team`, `Title` без JSON-тегов.
Миграция `00002_wiki.sql` создаёт таблицу `documents`. Полный Markdown,
чанки и эмбеддинги пока не хранятся в БД.

Общие `.md`-статьи лежат в корне `KNOWLEDGE_BASE_DIR`; для них `team` — пустая
строка. Статьи команд лежат в `<team-id>/`, вложенные каталоги поддерживаются.
Служебное описание базы знаний находится в `docs/KNOWLEDGE_BASE.md`.

Из `backend/`, после миграций:

```sh
go run ./cmd/index
```

Либо через Compose, который применит миграции и подключит `knowledge-base/`
из репозитория только для чтения:

```sh
docker compose run --rm --build index
```

Команда вызывает `indexer.Indexer.Index` из `internal/wiki/indexer/indexer.go`.
Это отдельный сервис индексации. Он обходит `.md`-файлы,
вычисляет ID как SHA-256 относительного пути, берёт команду из первого каталога
и заголовок из первой строки `# ...` (при отсутствии — имя файла без `.md`).
Путь сохраняется относительно корня, с разделителем `/`.

После успешного чтения всего каталога сервис передаёт список документов
в `wiki.Repository.Sync` из `internal/wiki/repository/repository.go`.
Репозиторий одной транзакцией добавляет новые записи,
обновляет изменившиеся метаданные и удаляет записи отсутствующих файлов.
Неизменившиеся строки не перезаписываются. При переименовании файла удаляется
старая запись и добавляется новая; успешно прочитанный пустой каталог очищает индекс.

При ошибке обхода или чтения файлов сервис не вызывает репозиторий. Ошибка БД
откатывает всю транзакцию, включая удаления. Индексация не запускается
автоматически при старте сервера; HTTP-операции wiki ещё не реализованы.

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
