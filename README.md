# AI Buddy

AI Buddy — демо для адаптации стажёров: чат с инструментами MCP, задачи и прогресс,
вопросы наставнику и поиск по базе знаний. Один Go backend, одна PostgreSQL с pgvector,
локальные модели Ollama. Пользователь выбирается cookie `demo_user`.

- [Запуск, конфигурация и проверки](backend/README.md)
- [Архитектура и требования](docs/SDD.md)
- [Единый OpenAPI](backend/api/openapi.yaml)
- [База знаний](knowledge-base/README.md)

```sh
cd backend
docker compose -f deploy/compose.yaml --env-file deploy/.env.example up -d --build
```

API доступен на `http://localhost:8080`. `GET /api/users` возвращает шесть демо-профилей.
Frontend в эту работу не входит; контракт переключателя описан в README бэкенда.
