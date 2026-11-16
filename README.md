# AI Buddy

## База знаний

Файлы в корне `knowledge-base/` скопированы из Playbook Авито.
В папках `backend/` и `platform/` — синтетические тестовые материалы:
данные вымышленные, все совпадения случайны.

## Запуск

Нужны Docker и Docker Compose. Из корня проекта:

```sh
cp .env.example .env
docker compose up -d --build
```

Приложение: `http://127.0.0.1:8080`. Открывается выбор пользователя из API.
Миграции, демоданные, загрузка моделей и индексация выполняются автоматически до запуска API.
Первый запуск скачивает модели и может занять несколько минут. По умолчанию используется CPU.
При повторном запуске существующие модели и данные сохраняются в именованных томах.
Имя проекта `buddy-backend` сохранено для совместимости с прежними томами.

Переменные в корневом `.env`: `WEB_PORT` (8080), `POSTGRES_PASSWORD` (buddy),
`CHAT_MODEL` (qwen3.5:4b), `EMBEDDING_MODEL` (embeddinggemma:300m),
`CHAT_CONTEXT_LENGTH` (16384), `CHAT_HISTORY_LIMIT` (6), `RAG_CHUNK_COUNT` (3).
После смены моделей или базы знаний перезапустите подготовку:

```sh
docker compose up -d --force-recreate ollama-model ollama-chat-model index server web
```

Веб-вход привязан к `127.0.0.1:${WEB_PORT}`; API, PostgreSQL и Ollama не публикуют порты.
Nginx обслуживает SPA и проксирует `/api`, включая SSE без буферизации.

```sh
docker compose ps -a
docker compose logs -f web server index
docker compose run --rm index  # повторная индексация после изменения knowledge-base/
docker compose down           # остановка без удаления данных
```

Не используйте `down -v`, если данные нужно сохранить.

## Разработка фронтенда

```sh
cd frontend
npm ci
npm run dev
npm run build
npm run lint:check
npm run format:check
npm test
```

Vite (`http://localhost:5173`) проксирует `/api` на `http://127.0.0.1:8080`.
Можно использовать запущенный Compose или локальный бэкенд с параметрами из `backend/.env.example`.
При работе через Compose порт прокси должен совпадать с `WEB_PORT`.
Чат хранится только в памяти вкладки: переходы между разделами его сохраняют,
перезагрузка, выход и смена пользователя очищают.
Создание и завершение записей требуют отдельного нажатия кнопки в карточке.

## Покрытие бэкенда

Нужен Go 1.27+. PostgreSQL и Ollama для тестов не требуются. Из корня проекта:

```sh
cd backend
go test -race -count=1 -coverpkg=./... -coverprofile=/tmp/buddy-coverage.out ./...
go tool cover -func=/tmp/buddy-coverage.out
go tool cover -html=/tmp/buddy-coverage.out -o /tmp/buddy-coverage.html
```

`-func` показывает покрытие функций и общий процент, HTML — покрытие строк кода.

## Эндпойнты

Кроме `GET /api/users`, нужна cookie `demo_user=<id>` из списка пользователей.
Стажёр работает со своими данными; ментор видит данные назначенных ему стажёров.

| Метод | Путь | Назначение |
| --- | --- | --- |
| GET | `/api/users` | Выбор пользователя |
| GET | `/api/users/interns` | Стажёры текущего ментора |
| POST | `/api/buddy/messages` | Чат через SSE; тело: `{"messages":[{"role":"user","content":"Привет"}]}` |
| GET | `/api/wiki` | Доступные статьи |
| GET | `/api/wiki/{id}` | Исходный Markdown статьи |
| GET | `/api/tracker/tasks` | Свои задачи |
| GET | `/api/tracker/interns/{internID}/tasks` | Задачи стажёра ментора |
| POST | `/api/tracker/tasks` | Создание: `description`, `deadline` в RFC3339 с часовым поясом |
| POST | `/api/tracker/tasks/{id}/complete` | Завершение задачи; пустое тело |
| GET | `/api/tickets` | Доступные тикеты |
| GET | `/api/tickets/{id}` | Тикет |
| POST | `/api/tickets` | Создание: `title`, `description` |

Создание задач/тикетов и завершение задач доступны стажёру по отдельному нажатию кнопки.
