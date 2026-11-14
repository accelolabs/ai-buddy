# Ticket

Команды выполняются из `backend/deploy/ticket`. Сначала запустите
[Authentik](../authentik/README.md) и [Tracker](../tracker/README.md).

## Настройка и запуск

```sh
cp .env.example .env
```

В `.env` заполните пароль БД и OIDC-настройки клиента `ticket`.
`BUDDY_API_KEY` должен совпадать с ключом Buddy, `TICKET_API_KEY` — с ключом Tracker.
`TRACKER_URL` задаёт адрес Tracker, `TRACKER_TIMEOUT` — таймаут запроса (по умолчанию `5s`).

```sh
docker compose up --build -d backend-ticket
docker compose logs -f backend-ticket
```

HTTPS-прокси направьте на `127.0.0.1:8084`; порт задаётся `HTTP_PORT`.
Миграции выполняются при запуске Compose.

Для демо-данных сначала выполните seed Tracker. Укажите `AUTHENTIK_URL`
и `AUTHENTIK_API_TOKEN` с правом чтения демо-пользователей, затем выполните:

```sh
docker compose --profile tools run --rm seed
```

## API

[Описание API](../../api/ticket.yaml). Браузерный вход — `GET /api/auth/login`.
Внутренние запросы требуют `BUDDY_API_KEY`. Получателя определяет Tracker;
для создания тикета у стажёра должен быть назначен наставник.

```sh
curl http://127.0.0.1:8084/internal/interns/2/tickets \
  -H "Authorization: Bearer $BUDDY_API_KEY" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: question-1' \
  -d '{"question":"Как запустить проект?","description":"Нужна помощь с окружением"}'
```

Замените `2` на ID стажёра в Authentik. Для нового тикета используйте новый `Idempotency-Key`.
