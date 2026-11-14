# Tracker

Команды выполняются из `backend/deploy/tracker`. Сначала запустите
[Authentik](../authentik/README.md).

## Настройка и запуск

```sh
cp .env.example .env
```

В `.env` заполните пароль БД и OIDC-настройки клиента `tracker`.
`BUDDY_API_KEY` должен совпадать с ключом Buddy, `TICKET_API_KEY` — с ключом Ticket;
эти два ключа должны различаться. `APP_TIMEZONE` задаёт часовой пояс дедлайнов.

```sh
docker compose up --build -d backend-tracker
docker compose logs -f backend-tracker
```

HTTPS-прокси направьте на `127.0.0.1:8083`; порт задаётся `HTTP_PORT`.
Миграции выполняются при запуске Compose.

Для демо-данных укажите `AUTHENTIK_URL` и `AUTHENTIK_API_TOKEN` с правом чтения
демо-пользователей, затем выполните:

```sh
docker compose --profile tools run --rm seed
```

## API

[Описание API](../../api/tracker.yaml). Браузерный вход — `GET /api/auth/login`.
Внутренние запросы Buddy используют `BUDDY_API_KEY`; Ticket использует
`TICKET_API_KEY` для получения наставника.

```sh
curl http://127.0.0.1:8083/internal/interns/2/progress \
  -H "Authorization: Bearer $BUDDY_API_KEY"
curl http://127.0.0.1:8083/internal/interns/2/tasks \
  -H "Authorization: Bearer $BUDDY_API_KEY" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: task-1' \
  -d '{"title":"Изучить проект","description":"Прочитать документацию","deadline":"2026-12-01"}'
```

Замените `2` на ID стажёра в Authentik. Для новой задачи используйте новый `Idempotency-Key`.
