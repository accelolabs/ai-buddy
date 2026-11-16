# AI Buddy

## База знаний

Файлы в корне `knowledge-base/` скопированы из Playbook Авито.
В папках `backend/` и `platform/` — синтетические тестовые материалы:
данные вымышленные, все совпадения случайны.

## Запуск

Нужны Docker и Docker Compose. Из корня проекта:

```sh
cd backend
docker compose up -d --build
docker compose run --rm --build seed
docker compose run --rm --build index
```

API: `http://localhost:8080`. Первый запуск скачивает модели Ollama.
Остановка: `docker compose down` из `backend/`.

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
