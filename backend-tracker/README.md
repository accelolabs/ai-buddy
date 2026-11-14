# Backend Tracker

Задачи стажёров, прогресс и назначения наставников. Go, PostgreSQL, вход через Authentik OIDC.
Сервисы создаются с именами `backend-[service]`; OIDC-клиент этого сервиса — `tracker`.

## Запуск

Команды выполнять из `backend-tracker`. Сначала запустить [Authentik](../deploy/authentik/README.md), который создаёт сеть `ai-buddy-shared`.

```sh
cp .env.example .env
# Заполнить .env, затем:
docker compose up --build -d backend-tracker
```

В `.env` указать пароль БД, разные `BUDDY_API_KEY` и `TICKET_API_KEY`, OIDC issuer,
секрет существующего клиента `tracker` и зарегистрированный HTTPS callback.
Секреты можно генерировать командой `openssl rand -hex 32`.

Сервер доступен на `127.0.0.1:8083`; для браузера нужен HTTPS-прокси.
Миграции выполняются перед запуском. БД хранится в отдельном томе, её порт не публикуется.
Вход: `/api/auth/login`, текущий пользователь: `/api/auth/me`, выход: `POST /api/auth/logout`.

## Демо-данные

В Authentik создать служебную учётную запись с правом просмотра демо-пользователей и API-токен.
Указать `AUTHENTIK_URL` и `AUTHENTIK_API_TOKEN` в окружении или `.env`, затем выполнить:

```sh
docker compose --profile tools run --rm seed
```

Токен передаётся только seed. Seed проверяет шесть пользователей до записи в БД и добавляет
профили, назначения и примеры задач одной транзакцией. Повторный запуск сохраняет существующие
данные. У `intern.platform.2` наставника нет.

## API

Ответы — JSON. Браузерные маршруты требуют сессию; стажёр работает со своими задачами,
наставник читает данные закреплённых стажёров.

| Метод и путь | Назначение |
| --- | --- |
| `GET /api/tasks`, `POST /api/tasks` | Список и создание задач стажёра |
| `POST /api/tasks/{id}/complete` | Завершение задачи |
| `GET /api/progress` | Свой прогресс |
| `GET /api/interns` | Стажёры наставника |
| `GET /api/interns/{id}/progress` | Прогресс закреплённого стажёра |
| `GET /internal/interns/{id}/progress` | Прогресс для Buddy |
| `POST /internal/interns/{id}/tasks` | Создание задачи через Buddy |
| `POST /internal/interns/{id}/tasks/{taskID}/complete` | Завершение через Buddy |
| `GET /internal/interns/{id}/mentor` | Поиск наставника для Buddy и Ticket |

Внутренние маршруты требуют `Authorization: Bearer <ключ>`; Ticket доступен только поиск наставника.
Создание требует `Idempotency-Key` и тело `{"title":"Задача","description":"Описание","deadline":"2026-12-01"}`.
Название и дата обязательны. Создание возвращает `201`, повтор — `200`, конфликт содержимого — `409`.
Повторное завершение сохраняет первую дату завершения.

Прогресс содержит профиль, наставника, задачи, счётчики, процент и следующую задачу.
Следующая — первая просроченная незавершённая, иначе с ближайшим дедлайном.
Часовой пояс задаётся `APP_TIMEZONE` (по умолчанию `Europe/Moscow`); сегодняшний дедлайн не просрочен.
Проверки доступности: `/health/live`, `/health/ready`.

## Структура и проверки

`internal/tasks` — задачи, `interns` — профили и назначения, `progress` — сводка прогресса.
В каждом пакете есть сервис, подпакеты `http` и `postgres`. Интерфейсы объявлены у потребителей,
зависимости собираются в `cmd/server`. Репозиторий прогресса использует репозитории задач и стажёров.

```sh
go test -race -coverprofile=/tmp/backend-tracker.cover ./...
go tool cover -func=/tmp/backend-tracker.cover
go vet ./...
go build -o /tmp/backend-tracker-bin/ ./cmd/...
docker compose --env-file .env.example config --quiet
```

Покрытие должно превышать 80%. Для PostgreSQL-тестов задать `TEST_DATABASE_URL`:
они создают и удаляют временные схемы. Без этой переменной интеграционные тесты пропускаются.
