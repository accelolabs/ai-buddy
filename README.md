# AI Buddy

AI Buddy — сервисы Buddy, Tracker, Ticket и Wiki с авторизацией через Authentik.

- [Архитектура и требования](docs/SDD.md).
- [Бэкенд: структура, запуск, сборка и проверки](backend/README.md).
- [Authentik: OIDC и демо-пользователи](backend/deploy/authentik/README.md).
- [Ollama: локальные модели на CPU](backend/deploy/ollama/README.md).
- [Tracker: задачи, прогресс и наставники](backend/deploy/tracker/README.md).
- [Ticket: вопросы наставникам](backend/deploy/ticket/README.md).

Весь код бэкенда, Go-модуль, Dockerfile, скрипты и файлы развёртывания находятся в `backend/`.
Go-команды выполняются из этого каталога:

```sh
cd backend
./scripts/check.sh
```

Tracker и Ticket запускаются отдельными процессами со своими PostgreSQL и сессиями.
Будущий frontend — один React-проект с общими компонентами и четырьмя самостоятельными
сборками для Buddy, Tracker, Ticket и Wiki на своих доменах.

[Wiki: запуск, индексация и API](backend/deploy/wiki/README.md) · [Knowledge base](knowledge-base/README.md)

[Buddy — чат, Qwen и встроенный MCP: запуск и API](backend/deploy/buddy/README.md).
