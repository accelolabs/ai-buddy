# Backend Ticket

Go-модуль `ai-buddy/backend-ticket`: вопросы стажёров наставникам, отправленные тикеты и входящие. Использует отдельную PostgreSQL с таблицами `sessions` и `tickets`. Профили и назначения остаются в Tracker. Ответы на тикеты, статусы, редактирование и удаление в эту версию не входят.

## Запуск

Скопируйте `.env.example` в `.env`, задайте пароли и ключи. Сохраните существующий OIDC-клиент **ticket**, его secret и зарегистрированный публичный **HTTPS callback**. Не создавайте новый клиент и не меняйте callback на localhost. Для Authentik нужна доступная HTTPS-сеть и доверенный сертификат.

```sh
docker network create ai-buddy-shared # один раз для всех сервисов
docker compose up --build -d backend-ticket
```

Сервер доступен на `127.0.0.1:8084`, БД не публикуется. Compose сначала запускает одноразовые миграции. Сервер и seed подключены к `ai-buddy-shared`; БД и миграции — только к частной сети. Tracker должен быть подключён к той же общей сети под именем `backend-tracker`. Его доступность при старте Ticket не требуется. `/health/live` проверяет процесс, `/health/ready` — только БД.

`TRACKER_URL` по умолчанию в Compose — `http://backend-tracker:8080`; `TRACKER_TIMEOUT` — положительная Go duration, по умолчанию `5s`. `TICKET_API_KEY` должен совпадать с ключом Tracker. `BUDDY_API_KEY` используется для входящих запросов Buddy. Сервер не получает административный токен Authentik.

Локальные команды используют `DATABASE_URL` из окружения; Compose строит собственный адрес БД. БД Compose не доступна через localhost — для локальных Go-команд используйте отдельно доступную PostgreSQL.

```sh
go run ./cmd/migrate up     # также down и status
go run ./cmd/server
go run ./cmd/seed
```

## API

Предметные ответы и ошибки возвращаются в JSON. Браузерная авторизация — сессия OIDC, cookie и защита от межсайтовых запросов. Авторизация и выход: `/api/auth/login`, `/api/auth/callback`, `/api/auth/me`, `/api/auth/logout` (как в шаблоне).

| Метод и путь | Доступ |
| --- | --- |
| `GET /api/tickets` | intern: свои отправленные |
| `POST /api/tickets` | intern: создать |
| `GET /api/tickets/incoming` | mentor: полученные им тикеты |
| `GET /internal/interns/{id}/tickets` | Buddy: отправленные стажёром |
| `POST /internal/interns/{id}/tickets` | Buddy: создать от имени стажёра |

Внутренние запросы требуют `Authorization: Bearer <BUDDY_API_KEY>`. Браузерный ID берётся только из сессии. JSON создания содержит только `question` и `description`, оба обязательны после удаления пробелов по краям. Неизвестные поля отклоняются; размер тела ограничен 1 MiB.

```sh
curl -X POST http://127.0.0.1:8084/internal/interns/2/tickets \
  -H "Authorization: Bearer $BUDDY_API_KEY" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: confirmed-question-1' \
  -d '{"question":"Как запустить проект?","description":"Нужна помощь с окружением"}'
```

`Idempotency-Key` обязателен: непустой, до 256 байт. Ключи общие для браузера и Buddy в пределах отправителя. Новый тикет: `201`; повтор с теми же нормализованными полями: `200`; изменённые поля с прежним ключом: `409`. Повтор сохранённого запроса не вызывает Tracker. Параллельные создания защищены уникальным ограничением PostgreSQL и перечитыванием победившей записи.

Получатель определяется через `GET /internal/interns/{id}/mentor` Tracker с `TICKET_API_KEY` и сохраняется навсегда. Смена назначения не переносит существующие тикеты. Без наставника: `422`, «Ментор не назначен». Неизвестный Tracker стажёр: `404`; сбой, таймаут или некорректный ответ Tracker: `503`. Остальные ошибки: `400` неверный запрос, `401` авторизация, `403` роль, `500` хранилище без SQL и секретов. Подтверждение отправки выполняет Buddy или интерфейс до вызова Ticket.

Тикет возвращается непосредственно: `id`, `sender_id`, `recipient_id`, `question`, `description`, `created_at`. Ключ идемпотентности в ответ не входит. Списки: `{"tickets":[...]}`; пустой список — `[]`, сортировка по `created_at`, затем `id` по возрастанию. Списки работают без Tracker.

## Демо-данные

Порядок: **Authentik → seed Tracker → seed Ticket**. В `.env` задайте `AUTHENTIK_URL` и `AUTHENTIK_API_TOKEN`; они передаются только seed.

```sh
docker compose --profile tools run --rm seed
```

Seed проверяет шесть активных демо-пользователей (username, ID, роль и команда), затем запрашивает наставников четырёх стажёров через Tracker. Для стажёра без наставника тикет не создаётся. Все внешние чтения завершаются до транзакции. При сбое API записей нет; ошибка вставки откатывает всю транзакцию. Постоянные ID и ключи с префиксом `demo:ticket:` обеспечивают повторный запуск без перезаписи живых данных.

## Проверки

```sh
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
go vet ./...
go build -o /tmp/backend-ticket-bin/ ./cmd/...
docker compose --env-file .env.example config --quiet
docker compose build
```

Для интеграционных тестов задайте `TEST_DATABASE_URL` отдельной тестовой PostgreSQL. Тестам нужны права создания и удаления схем; они создают изолированные схемы, применяют миграции и удаляют схемы после проверки. Проверяются конкурентная идемпотентность, фильтрация и сортировка списков, сохранение получателя, повтор seed и откат. Без переменной эти тесты явно пропускаются. Тест сессий также использует изолированную схему; не направляйте тесты на рабочую БД.

При реализации успешно проверены `go test -race`, `go vet`, сборка трёх команд и Compose config. Покрытие сервиса тикетов — 100%. Docker-сборка и реальные PostgreSQL-проверки требуют доступного Docker daemon или тестовой БД; в среде реализации daemon и `TEST_DATABASE_URL` отсутствовали.
