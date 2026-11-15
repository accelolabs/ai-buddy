# AI Buddy

AI Buddy — демо для адаптации стажёров: чат с инструментами MCP, задачи и прогресс,
вопросы наставнику и поиск по базе знаний. Один Go backend, одна PostgreSQL с pgvector,
локальные модели Ollama. Пользователь выбирается cookie `demo_user`.

Текущая реализация сохранена в `backend-ai/`. Новую структуру и упрощение
бэкенда разбираем с пользователем по фичам: [инструкция](docs/REFACTORING.md).

- [Запуск, конфигурация и проверки](backend-ai/README.md)
- [Архитектура прежней реализации](docs/SDD.md)
- [Единый OpenAPI](backend-ai/api/openapi.yaml)
- [База знаний](docs/KNOWLEDGE_BASE.md)

```sh
cd backend-ai
docker compose -f deploy/compose.yaml --env-file deploy/.env.example up -d --build
```

API доступен на `http://localhost:8080`. `GET /api/users` возвращает шесть демо-профилей.
Frontend в эту работу не входит; контракт переключателя описан в README бэкенда.
