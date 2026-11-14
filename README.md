# AI Buddy

Архитектура и требования: [docs/SDD.md](docs/SDD.md).

Первый этап — [Authentik: Compose, OIDC и демо-пользователи](deploy/authentik/README.md).

Шаблон Go-сервиса: [backend-template/README.md](backend-template/README.md).

Backend Tracker: [backend-tracker/README.md](backend-tracker/README.md) — задачи, прогресс, наставники, внутренний API и seed.

Каталоги, Go-модули и сервисы развёртывания создавать с именами `backend-[service]`
(например, `backend-buddy`). Существующие OIDC-клиенты Authentik сохраняют имена
`buddy`, `wiki`, `tracker`, `ticket` и зарегистрированные callback URL.

## План

1. Создать `backend-buddy`, `backend-wiki`, `backend-tracker` и `backend-ticket`: модули, БД, Compose, OIDC.
2. `backend-tracker`: задачи, дедлайны, прогресс, наставники и API.
3. `backend-ticket`: вопросы наставнику, списки тикетов, идемпотентность.
4. `backend-wiki`: документы, права доступа, гибридный поиск с pgvector.
5. `backend-buddy`: диалоги, модель, MCP-инструменты и подтверждение действий.
6. SSE: поток ответов, восстановление без дублей, ошибки и повтор.
7. React: интерфейсы четырёх сервисов.
8. Настроить межсервисные API-ключи, права и повторяемые демо-данные.
