# Wiki

Команды выполняются из `backend/deploy/wiki`. Сначала запустите
[Authentik](../authentik/README.md) и [Ollama](../ollama/README.md).

## Настройка и запуск

```sh
cp .env.example .env
```

В `.env` заполните:

- `POSTGRES_PASSWORD` и OIDC-настройки клиента `wiki`.
- `BUDDY_API_KEY` — тот же ключ, что в Buddy.
- `AUTHENTIK_URL` — адрес Authentik без пути OIDC issuer.
- `AUTHENTIK_READ_TOKEN` — токен отдельной служебной учётной записи с правом `authentik_core.view_user`.
- `OLLAMA_URL` — адрес Ollama; `WIKI_MIN_SIMILARITY` — порог поиска от 0 до 1 (по умолчанию `0.35`).

```sh
docker compose up -d --build
docker compose logs -f backend-wiki
```

HTTPS-прокси направьте на `127.0.0.1:8082`; порт задаётся `HTTP_PORT`.
Миграции выполняются при запуске Compose.

## Документы и индексация

По умолчанию документы читаются из `knowledge-base/` репозитория.
Для другого каталога задайте `KNOWLEDGE_BASE_DIR` абсолютным путём на хосте.
Размещайте Markdown-файлы в `common/` или папке команды; `common/` обязательна.
Символические ссылки не поддерживаются.

После изменения файлов выполните индексацию:

```sh
docker compose --profile tools run --rm index
```

После замены модели эмбеддингов выполните полный пересчёт:

```sh
docker compose --profile tools run --rm index /app/wiki-index --full
```

## API

[Описание API](../../api/wiki.yaml). Браузерный вход — `GET /api/auth/login`.
Список и чтение документов доступны через `GET /api/documents` и `GET /api/documents/{id}`.
Для поиска от имени пользователя передайте его ID в Authentik и ключ Buddy:

```sh
curl http://127.0.0.1:8082/internal/users/2/search \
  -H "Authorization: Bearer $BUDDY_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"query":"Как получить доступ к проекту?"}'
```
