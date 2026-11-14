# AI Buddy

Архитектура и требования: [SDD](../docs/SDD.md). В каталоге `backend/` расположен один Go-модуль `ai-buddy`, отдельные процессы и базы данных сервисов.

- [Authentik: OIDC и демо-пользователи](deploy/authentik/README.md).
- [Ollama: локальные модели на CPU](deploy/ollama/README.md).
- [Tracker: задачи, прогресс, наставники](deploy/tracker/README.md).
- [Ticket: вопросы наставникам](deploy/ticket/README.md).

## Структура

```text
cmd/tracker, cmd/ticket         серверные команды
cmd/migrate, cmd/seed           общие команды с обязательным выбором сервиса
internal/tracker               задачи, стажёры, прогресс, миграции и seed Tracker
internal/ticket                тикеты, HTTP-клиент Tracker, миграции и seed Ticket
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

`migrate <tracker|ticket> [up|down|status]` и `seed <tracker|ticket>` требуют сервис;
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

## Проверки

Из каталога `backend/`:

```sh
./scripts/check.sh
# С реальной отдельной тестовой PostgreSQL:
TEST_DATABASE_URL=postgres://user:password@localhost/test ./scripts/check.sh
```

Скрипт проверяет форматирование, `go vet`, сборку всех четырёх команд и `go test -race`.
Три независимых профиля (`tracker.cover`, `ticket.cover`, `platform.cover`) сохраняются
в `/tmp/ai-buddy-check` или `CHECK_OUTPUT_DIR`. В каждом покрытие по операторам должно быть
**строго больше 80%**. Tracker включает только `cmd/tracker` и `internal/tracker`, Ticket —
`cmd/ticket` и `internal/ticket`; platform — `internal/platform`, `cmd/migrate`, `cmd/seed`.
Все тесты могут вызывать выбранную группу, но общий код не засчитывается сервисам.
`internal/testutil` исключён из всех знаменателей. Порог проверяется по числу операторов до округления.

PostgreSQL-тесты создают и удаляют изолированные схемы. Без `TEST_DATABASE_URL` они явно
пропускаются. Проверяются миграции, ограничения, конкурентная идемпотентность, сессии,
повтор seed и откат транзакций. Compose проверяется без daemon; сборки образов требуют Docker daemon.
Результаты и измерения рефакторинга: [docs/refactoring.md](docs/refactoring.md).

## Добавление сервиса

1. Создать `cmd/<service>` и предметные пакеты `internal/<service>` с собственными интерфейсами,
   обработчиками и PostgreSQL-репозиториями.
2. Встроить `platform/config.Config` в настройки сервиса и добавить предметную валидацию.
   Собрать обработчики через `platform/server.Run`, передав конфигурацию, подключение и зависимости явно.
3. Добавить собственные SQL-миграции через `embed.FS`, подключить набор к `cmd/migrate`;
   добавить выбранный seed к `cmd/seed` и разрешить новое имя в `platform/cli`.
4. Создать `deploy/<service>` с отдельными БД, credentials, томом и приватной сетью;
   расширить выбор `SERVICE` Dockerfile каталога `backend/` и добавить отдельную группу покрытия.

Копирование шаблона больше не требуется. Buddy и Wiki будут добавлены отдельно.
Будущий frontend — **один React-проект**, общие компоненты и **четыре самостоятельные сборки**
для Buddy, Tracker, Ticket и Wiki на своих доменах. Их реализация в этот рефакторинг не входит.
