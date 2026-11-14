# Ollama на CPU

Docker запускает Ollama и загружает выбранные модели. GPU и отдельная установка Ollama
на хосте не требуются. Используется [официальный образ](https://docs.ollama.com/docker).

- `embeddinggemma:300m` — эмбеддинги для поиска Wiki; 768 измерений, вход до 2048 токенов.
- `qwen3.5:4b` — ответы и вызовы инструментов Buddy; контекст запросов 4096, `think: false`.

## Запуск

Сначала запустить [Authentik](../authentik/README.md), который создаёт сеть `ai-buddy-shared`.
Для отдельного запуска Ollama достаточно один раз создать эту сеть:
`docker network create ai-buddy-shared`.

Из корня репозитория:

```sh
cd backend/deploy/ollama
cp .env.example .env
docker compose up -d
docker compose logs -f models
```

Сервис `models` ждёт готовности Ollama, последовательно выполняет `ollama pull` для обеих
моделей и завершается. Первый запуск требует интернета и времени на скачивание.
Готовность моделей проверяется по завершению `models` с кодом 0 (`docker compose ps -a`)
и списку `docker compose exec ollama ollama list`. Ошибка загрузки видна в логах и коде
завершения; для повторной попытки выполнить `docker compose run --rm models`.
Healthcheck проверяет сервер Ollama, а не наличие моделей.

Веса хранятся в томе `ai-buddy-ollama_models` и сохраняются при перезапуске контейнеров.
Повторная загрузка использует существующие слои. Теги моделей могут обновляться;
неизменяемый набор весов требует отдельной фиксации версии. `OLLAMA_IMAGE` позволяет
закрепить версию или digest официального образа вместо `latest`.

Будущие Buddy и Wiki обращаются к `http://ollama:11434` через `ai-buddy-shared`.
Для команд с хоста опубликован только `127.0.0.1:11434`; публичный домен для Ollama не нужен.
Сами интеграции Buddy/Wiki пока не реализованы.

## Память и проверка

Жёсткие лимиты RAM и swap в Compose не задаются. Память ОС, Authentik, PostgreSQL
и остальных сервисов планируется отдельно. Одновременно допускаются две загруженные модели и один запрос
на модель; контекст LLM по умолчанию 4096. Buddy должен передавать `think: false` и
`options.num_ctx: 4096` в запросах. Настройки описаны в [FAQ Ollama](https://docs.ollama.com/faq).

```sh
curl --fail http://127.0.0.1:11434/api/embed -d '{
  "model": "embeddinggemma:300m",
  "input": "task: search result | query: Как получить доступ к проекту?",
  "truncate": false
}'
curl --fail http://127.0.0.1:11434/api/chat -d '{
  "model": "qwen3.5:4b",
  "messages": [{"role": "user", "content": "Кратко объясни, что такое RAG."}],
  "think": false,
  "stream": false,
  "options": {"num_ctx": 4096, "num_predict": 512}
}'
docker compose exec ollama ollama ps
docker stats --no-stream
```

После запросов `ollama ps` должен показывать `100% CPU`. Фактический расход памяти
и задержку нужно измерить на целевом процессоре.

При индексации Wiki документы делятся на фрагменты с учётом лимита 2048 токена вместе
с префиксами. Запросы: `task: search result | query: ...`; документы:
`title: <заголовок или none> | text: ...` — согласно
[карточке EmbeddingGemma](https://huggingface.co/google/embeddinggemma-300m#prompt-instructions).
При смене модели эмбеддингов требуется полная переиндексация Wiki.

## Обновление

```sh
docker compose pull
docker compose up -d ollama
docker compose run --rm models
```

`docker compose down` сохраняет веса. `docker compose down -v` удаляет том с моделями.
