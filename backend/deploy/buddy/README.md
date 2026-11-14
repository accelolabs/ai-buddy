# Buddy

Команды выполняются из `backend/deploy/buddy`. Сначала запустите
[Authentik](../authentik/README.md), Tracker, Ticket, Wiki и [Ollama](../ollama/README.md).

## Настройка и запуск

```sh
cp .env.example .env
```

В `.env` заполните:

- `POSTGRES_PASSWORD` — пароль БД.
- `OIDC_ISSUER`, `OIDC_CLIENT_SECRET`, `OIDC_REDIRECT_URI` — настройки клиента `buddy`.
- `BUDDY_API_KEY` — одинаковый в Buddy, Tracker, Ticket и Wiki.
- `TRACKER_URL`, `TICKET_URL`, `WIKI_URL`, `OLLAMA_URL` — адреса зависимостей в общей сети.
- `WIKI_PUBLIC_URL` — публичный HTTPS-адрес Wiki для ссылок на документы.

```sh
docker compose up -d --build
docker compose logs -f backend-buddy
```

HTTPS-прокси направьте на `127.0.0.1:8081`; порт задаётся `HTTP_PORT`.
Миграции выполняются при запуске Compose. Запускайте одну реплику Buddy.

## API

[Описание API](../../api/buddy.yaml). Вход — `GET /api/auth/login`;
для запросов нужна сессия стажёра. В примерах `cookies.txt` содержит cookie этой сессии.

```sh
curl -b cookies.txt -X POST https://buddy.example.com/api/chats
curl -b cookies.txt https://buddy.example.com/api/chats/CHAT_ID
curl -b cookies.txt https://buddy.example.com/api/chats/CHAT_ID/messages \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: message-1' \
  -d '{"content":"Как получить доступ к проекту?"}'
curl -N -b cookies.txt https://buddy.example.com/api/runs/RUN_ID/events
```

`CHAT_ID` берётся из ответа создания чата, `RUN_ID` — из ответа отправки сообщения.
При переподключении к событиям передавайте `Last-Event-ID` последнего полученного события.
Для нового сообщения используйте новый `Idempotency-Key`.

Подтверждение черновика и повтор неудачного запуска:

```sh
curl -b cookies.txt https://buddy.example.com/api/actions/ACTION_ID/confirm \
  -H 'Content-Type: application/json' -d '{}'
curl -b cookies.txt -X POST https://buddy.example.com/api/runs/RUN_ID/retry
```
