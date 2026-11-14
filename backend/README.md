# AI Buddy

Архитектура и требования: [SDD](../docs/SDD.md). В каталоге `backend/` расположен один Go-модуль `ai-buddy`, отдельные процессы и базы данных сервисов.

- [Authentik: OIDC и демо-пользователи](deploy/authentik/README.md).
- [Ollama: локальные модели на CPU](deploy/ollama/README.md).
- [Buddy: чат, SSE и подтверждения](deploy/buddy/README.md).
- [Tracker: задачи, прогресс, наставники](deploy/tracker/README.md).
- [Ticket: вопросы наставникам](deploy/ticket/README.md).
- [Wiki: документы и гибридный поиск](deploy/wiki/README.md).

## Структура

```text
cmd/buddy, cmd/tracker, cmd/ticket, cmd/wiki серверные команды
cmd/wiki-index                  явное обновление индекса Wiki
cmd/migrate, cmd/seed           общие команды с обязательным выбором сервиса
api/                           четыре самостоятельных OpenAPI 3.1.1
internal/buddy                 чаты, агент, MCP, SSE и подтверждения
internal/tracker/dto            общие DTO Tracker
internal/tracker               задачи, стажёры, прогресс, миграции и seed Tracker
internal/ticket                тикеты, HTTP-клиент Tracker, миграции и seed Ticket
internal/wiki                  модели, поиск и конфигурация Wiki
internal/wiki/{postgres,indexer} SQL и координация обновления индекса
internal/wiki/{ollama,authentik}  внешние клиенты Wiki
internal/wiki/http              маршруты, DTO и проверка запросов Wiki
internal/platform              auth, config, database, server, httpx, migrations, demousers, cli
internal/testutil              только тестовые HTTP-заглушки и временные схемы PostgreSQL
deploy/                       Compose, окружение и инструкции сервисов и Ollama
```

Предметные пакеты сохраняют подпакеты `http` и `postgres`; DTO находятся у обработчиков,
доменные модели не содержат JSON-тегов. Общая инфраструктура не импортирует сервисы.
Зависимости собираются в серверных командах, соединения с БД передаются явно.
Tracker хранит профили, назначения, задачи и свои сессии; Ticket — тикеты и свои сессии.
Ticket получает наставника только через HTTP Tracker. Общий модуль не объединяет данные.

## Локальные команды

Требуется Go 1.27. Команды выполняются из `backend/`; окружение должно содержать настройки
выбранного сервиса из `deploy/<service>/.env.example`. Go-команды не читают `.env` автоматически.
Для них нужна доступная PostgreSQL: базы Compose не публикуют порты.

```sh
cd backend # из корня репозитория
go run ./cmd/migrate tracker          # по умолчанию up
go run ./cmd/migrate tracker status  # также поддерживается down
go run ./cmd/tracker
go run ./cmd/seed tracker
# В другом окружении с DATABASE_URL и OIDC-настройками Ticket:
go run ./cmd/migrate ticket up
go run ./cmd/ticket
go run ./cmd/seed ticket
```

`migrate <tracker|ticket|wiki|buddy> [up|down|status]` и `seed <tracker|ticket>` требуют сервис;
неизвестные и лишние аргументы отклоняются до подключения к БД. Каждый запуск работает
только с переданным `DATABASE_URL` и миграциями выбранного сервиса. SQL и номера миграций сохранены.
Seed выполняется в порядке **Authentik → Tracker → Ticket**. Административный токен Authentik
нужен только seed. Внешние чтения заканчиваются до транзакции, повтор seed сохраняет живые данные.

## Контейнеры и существующие данные

Из каталога `backend/`:

```sh
docker build --build-arg SERVICE=tracker -t ai-buddy-backend-tracker:local .
docker build --build-arg SERVICE=ticket -t ai-buddy-backend-ticket:local .
# После настройки deploy/tracker/.env:
docker compose -f deploy/tracker/compose.yaml up --build -d backend-tracker
# После настройки deploy/ticket/.env:
docker compose -f deploy/ticket/compose.yaml up --build -d backend-ticket
```

Dockerfile каталога `backend/` собирает выбранный сервер и обе административные команды.
Образы, процессы, БД, учётные данные и приватные сети остаются отдельными.
Compose явно сохраняет имена проектов `backend-tracker` и `backend-ticket`, сервисы и тома:
`backend-tracker_backend-tracker-data`, `backend-ticket_backend-ticket-data`.
Для существующего развёртывания с нестандартным `-p` продолжайте передавать то же имя проекта.
Скопируйте прежние `.env` в `deploy/tracker/.env` и `deploy/ticket/.env`.
Сброс БД и перенос данных не нужны; не удаляйте существующие тома.
Порты остаются 8083/8084. Серверы используют общую сеть `ai-buddy-shared`, БД — только приватные сети.
OIDC-клиенты `tracker` и `ticket` и зарегистрированные HTTPS callback остаются прежними.

## Локальные модели

После запуска Authentik, создавшего `ai-buddy-shared`, из каталога `backend/`:

```sh
cp deploy/ollama/.env.example deploy/ollama/.env
docker compose -f deploy/ollama/compose.yaml up -d
docker compose -f deploy/ollama/compose.yaml logs -f models
```

Docker запускает Ollama на CPU и загружает `embeddinggemma:300m` для Wiki и `qwen3.5:4b`
для Buddy. Веса сохраняются в томе. Жёсткие лимиты RAM и swap не задаются.
Для LLM приняты контекст 4096 и `think: false`. Память остальных компонентов
учитывается отдельно. Сервисы используют `http://ollama:11434`.
Готовность загрузки, примеры HTTP-запросов и обновление: [инструкция Ollama](deploy/ollama/README.md).

## API и проверки

Контракты поддерживаются вручную в OpenAPI 3.1.1:
[Buddy](api/buddy.yaml), [Tracker](api/tracker.yaml), [Ticket](api/ticket.yaml), [Wiki](api/wiki.yaml).
Каждый документ самостоятельный: браузерные и внутренние операции, авторизация, health,
поля и ошибки; Buddy также описывает SSE и состояния подтверждений. Генерация серверного
кода, Swagger UI и HTTP-маршруты документации не используются.

Требуется **golangci-lint 2.13.2**, собранный с Go не старше версии модуля.
[Официальная установка](https://golangci-lint.run/docs/welcome/install/):

```sh
curl -sSfL https://golangci-lint.run/install.sh -o /tmp/install-golangci.sh
sh /tmp/install-golangci.sh -b "$(go env GOPATH)/bin" v2.13.2
export PATH="$(go env GOPATH)/bin:$PATH"
```

Из каталога `backend/`:

```sh
./scripts/lint.sh                  # только форматирование и линтер, без перезаписи файлов
./scripts/check.sh --unit          # линтер, vet, сборки, race; без PostgreSQL и порогов
TEST_DATABASE_URL='postgres://postgres:test@localhost:55432/buddy_test?sslmode=disable' ./scripts/check.sh
```

Полная проверка заранее требует `TEST_DATABASE_URL` и Docker CLI с Compose. Тестовая
PostgreSQL 16 должна поддерживать pgvector; Authentik и модели для неё не нужны.
Пример отдельного тестового экземпляра (пароль только для локальных тестов):

```sh
docker run --rm -d --name ai-buddy-test-db -e POSTGRES_PASSWORD=test -e POSTGRES_DB=buddy_test \
  -p 127.0.0.1:55432:5432 pgvector/pgvector:pg16
# После проверки:
docker stop ai-buddy-test-db
```

Скрипт выполняет весь набор race-тестов **один раз**, с покрытием объединения пяти групп.
Из него формируются `tracker.cover`, `ticket.cover`, `wiki.cover`, `buddy.cover`, `platform.cover`.
Для каждой группы требуется **строго >80%** операторов до округления; ровно 80% не проходит.
Tracker/Ticket/Buddy включают свои `internal/<service>` и `cmd/<service>`; Wiki дополнительно
включает `cmd/wiki-index`; Platform — `internal/platform`, `cmd/migrate`, `cmd/seed`.
Чужие сервисы, `internal/testutil` и тестовые проверки API не входят в знаменатель группы.
Логи, общий и пять отдельных профилей сохраняются в `/tmp/ai-buddy-check` или `CHECK_OUTPUT_DIR`.
Проверка разделения покрытия: `./scripts/coverage-test.sh`.

Интеграции создают и удаляют изолированные схемы. `--unit` явно отключает их даже при
заданном `TEST_DATABASE_URL`. Полный запуск проверяет четыре спецификации через
**kin-openapi v0.153.0** и реальные запросы/ответы в httptest-сценариях, затем Compose
для Buddy, Tracker, Ticket, Wiki и Ollama. Библиотека OpenAPI используется только проверками.
GitHub Actions запускает тот же полный скрипт с PostgreSQL 16 + pgvector и всегда сохраняет артефакты.

Реальный Qwen — отдельная ручная проверка после изменений агента, MCP или клиентов:

```sh
TEST_DATABASE_URL='postgres://postgres:test@localhost:55432/buddy_test?sslmode=disable' \
BUDDY_REAL_OLLAMA_URL=http://127.0.0.1:11434 \
go test -v -timeout 30m ./internal/buddy -run '^TestRealQwenFlow$'
```

## Добавление сервиса

1. Создать `cmd/<service>` и предметные пакеты `internal/<service>` с собственными интерфейсами,
   обработчиками и PostgreSQL-репозиториями.
2. Встроить `platform/config.Config` в настройки сервиса и добавить предметную валидацию.
   Собрать обработчики через `platform/server.Run`, передав конфигурацию, подключение и зависимости явно.
3. Добавить собственные SQL-миграции через `embed.FS`, подключить набор к `cmd/migrate`;
   добавить выбранный seed к `cmd/seed` и разрешить новое имя в `platform/cli`.
4. Создать `deploy/<service>` с отдельными БД, credentials, томом и приватной сетью;
   расширить выбор `SERVICE` Dockerfile каталога `backend/` и добавить отдельную группу покрытия.

Копирование шаблона не требуется; все четыре сервиса реализованы.
Будущий frontend — **один React-проект**, общие компоненты и **четыре самостоятельные сборки**
для Buddy, Tracker, Ticket и Wiki на своих доменах. Их реализация в этот рефакторинг не входит.

Wiki: [развёртывание и JSON API](deploy/wiki/README.md), [база знаний](../knowledge-base/README.md).

Buddy: [запуск, API, SSE и подтверждения](deploy/buddy/README.md). Миграции: `go run ./cmd/migrate buddy up`; сервер: `go run ./cmd/buddy`.
