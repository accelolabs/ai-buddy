# Backend AI Buddy

Один бинарник `cmd/server`, один HTTP mux и один пул PostgreSQL с pgvector.
Buddy, Tracker, Ticket и Wiki — внутренние пакеты. MCP SDK вызывает Go-сервисы
через `internal/local`; роль, команда и назначения читаются из общей БД.

## Запуск

Из `backend/`:

```sh
docker compose -f deploy/compose.yaml --env-file deploy/.env.example up -d --build
curl http://localhost:8080/api/users
curl --cookie demo_user=intern.backend.1 http://localhost:8080/api/me
```

Постоянные контейнеры: приложение, PostgreSQL и Ollama; наружу открыт только `8080`.
При старте приложение применяет миграции, транзакционно добавляет недостающие
демо-данные, восстанавливает Buddy и начинает принимать HTTP. Повторный старт
сохраняет изменения пользователей. База знаний смонтирована read-only.
Readiness `/health/ready` проверяет только БД; `/health/live` проверяет процесс.
Отсутствие моделей и индекса не мешает запуску, но поиск и генерация возвращают ошибки.

Модели и Wiki подготавливаются явно:

```sh
docker compose -f deploy/compose.yaml --env-file deploy/.env.example --profile tools run --rm models
docker compose -f deploy/compose.yaml --env-file deploy/.env.example run --rm app index
# Полный пересчёт после смены модели:
docker compose -f deploy/compose.yaml --env-file deploy/.env.example run --rm app index --full
```

Модели: `qwen3.5:4b` и `embeddinggemma:300m`. Для Buddy остаются контекст 4096,
`think:false` и максимум шесть вызовов инструментов за запуск.
Для переиспользования кэша моделей задайте `OLLAMA_VOLUME_NAME` именем существующего
Docker volume. Не удаляйте этот volume при переходе или пересоздании БД.

## Локальный бинарник и конфигурация

Нужны Go из `go.mod`, PostgreSQL с pgvector и переменная `DATABASE_URL`.
Compose не публикует PostgreSQL на хост; для локального бинарника используйте отдельно
подготовленную БД. Один пример переменных: [deploy/.env.example](deploy/.env.example).
Приложение читает переменные окружения; `.env` автоматически не загружается.

```sh
go build -o /tmp/ai-buddy-server ./cmd/server
export DATABASE_URL='postgres://buddy:buddy@localhost:5432/buddy?sslmode=disable'
/tmp/ai-buddy-server                  # serve по умолчанию
/tmp/ai-buddy-server serve
/tmp/ai-buddy-server migrate up
/tmp/ai-buddy-server migrate status
/tmp/ai-buddy-server seed
/tmp/ai-buddy-server index --full
# Удаляет все таблицы приложения; используется только для сброса демо:
/tmp/ai-buddy-server migrate down
```

| Переменная | Значение по умолчанию |
|---|---|
| `DATABASE_URL` | обязательная |
| `HTTP_ADDR` | `:8080` |
| `OLLAMA_URL` | `http://localhost:11434` (Compose: `http://ollama:11434`) |
| `APP_TIMEZONE` | `Europe/Moscow` |
| `KNOWLEDGE_BASE_DIR` | `../knowledge-base` |
| `WIKI_MIN_SIMILARITY` | `0.35` |

Миграции находятся в `internal/migrations`. Старая схема не поддерживается:
переход выполняется на пустую БД. Перед переходом остановите старые Compose-проекты
этого репозитория, удалите их БД и данные прежнего провайдера пользователей.
Идентифицируйте ресурсы по Compose labels с путём этого репозитория; удаляйте только
проверенные контейнеры и volumes, сохраняя Ollama. Глобальный Docker prune не нужен.

## Контракт переключателя пользователей

`GET /api/users` публичен, возвращает `{"users":[...]}` с сортировкой по `id`.
Профиль содержит `id`, `name`, `username`, `email`, `role`, `team`, `start_date`,
`goals`, `expectations`. `start_date` — дата или `null`.

Frontend и API размещаются на одном origin. После выбора:

```js
document.cookie = `demo_user=${encodeURIComponent(id)}; Path=/; SameSite=Lax`;
const me = await fetch('/api/me').then(r => r.json());
const events = new EventSource(`/api/runs/${runId}/events`);
```

Это выбор демо-профиля без проверки подлинности. Роль и команда всегда берутся из БД.
Отсутствующая, пустая или неизвестная cookie даёт JSON `400`, сбой чтения профиля —
JSON `503`. Пользователь по умолчанию не выбирается. При переключении UI закрывает
старый EventSource, очищает показанные данные и перезагружает данные нового профиля.
Уже начавшаяся генерация продолжает работу от прежнего пользователя; чужой профиль
не получает её историю или события. Cookie автоматически передаётся обычными
запросами и EventSource; при переподключении используйте сохранённый `Last-Event-ID`.

Демо-профили: `mentor.backend`, `intern.backend.1`, `intern.backend.2`,
`mentor.platform`, `intern.platform.1`, `intern.platform.2`. Последний стажёр без
наставника. Seed создаёт примеры задач и тикетов, не перезаписывая существующие записи.

Стажёр видит собственные задачи, диалоги и отправленные тикеты. Наставник видит
закреплённых стажёров и свои входящие тикеты. Wiki фильтруется по команде и `common`.
Ссылки на документы относительные: `/api/documents/{id}`.

## API и проверки

[OpenAPI 3.1.1](api/openapi.yaml): 22 операции, включая две проверки здоровья;
группы Users, Buddy, Tracker, Ticket, Wiki. Задачи читаются через `/api/progress`,
поиск доступен через Buddy/MCP. Тела, envelopes, ключи идемпотентности, подтверждения
черновиков и SSE сохраняются. Повтор подтверждения использует
`buddy-action:<id>` и зафиксированные при первом подтверждении поля.
Тикет сохраняет получателя на момент создания, даже после смены наставника.

```sh
# Лёгкий вариант без БД:
env -u TEST_DATABASE_URL go test -race -count=1 ./...
# Интеграционные проверки с отдельной тестовой PostgreSQL/pgvector:
TEST_DATABASE_URL='postgres://postgres:test@localhost:5432/buddy_test?sslmode=disable' go test -race -count=1 ./...
go vet ./...
golangci-lint fmt --diff
golangci-lint run ./...
go build -o /tmp/ai-buddy-server ./cmd/server
docker compose -f deploy/compose.yaml --env-file deploy/.env.example config --quiet
```

Интеграционные тесты создают изолированные схемы и проверяют миграции
up/status/down/up, повторный seed, внешние ключи, конкурентную идемпотентность,
прямые MCP-вызовы, разграничение данных, SSE replay, восстановление и shutdown.
Индекс Wiki публикуется атомарно под advisory lock; ошибка сохраняет старый индекс.
Проверка структуры OpenAPI и всех ссылок входит в Go-тесты. CI запускает проверки с БД
и без неё. При shutdown Buddy завершает сохранение фоновых операций до закрытия БД.
