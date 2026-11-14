# Ollama

Команды выполняются из `backend/deploy/ollama`. Сначала запустите
[Authentik](../authentik/README.md), чтобы создать сеть `ai-buddy-shared`.

## Настройка и запуск

```sh
cp .env.example .env
docker compose up -d
docker compose logs -f models
```

Параметры `.env`: `OLLAMA_IMAGE` — образ, `OLLAMA_HTTP_PORT` — порт на хосте,
`OLLAMA_EMBEDDING_MODEL` и `OLLAMA_CHAT_MODEL` — загружаемые модели.
Первый запуск требует интернета для скачивания весов.

```sh
docker compose ps -a
docker compose exec ollama ollama list
```

Контейнер `models` должен завершиться с кодом 0. Повтор загрузки:
`docker compose run --rm models`.

Buddy и Wiki используют `http://ollama:11434`; с хоста доступен `127.0.0.1:11434`.
Модели работают на CPU, веса сохраняются в томе.

## API

Примеры для моделей из `.env.example`:

```sh
curl --fail http://127.0.0.1:11434/api/embed -d '{
  "model": "embeddinggemma:300m",
  "input": "task: search result | query: Как получить доступ к проекту?",
  "truncate": false
}'
curl --fail http://127.0.0.1:11434/api/chat -d '{
  "model": "qwen3.5:4b",
  "messages": [{"role": "user", "content": "Что такое RAG?"}],
  "think": false,
  "stream": false,
  "options": {"num_ctx": 4096, "num_predict": 512}
}'
```

## Обновление

```sh
docker compose pull
docker compose up -d ollama
docker compose run --rm models
```

После замены модели эмбеддингов выполните [полную индексацию Wiki](../wiki/README.md#документы-и-индексация).
