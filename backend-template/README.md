# Backend template

Go 1.27+, net/http, PostgreSQL, SCS и Authentik OIDC.

## Запуск

```bash
cp .env.example .env
# Заполнить пароли, существующий OIDC-клиент и HTTPS URL
chmod 600 .env
docker compose up -d --build
```

Нужна сеть `ai-buddy-shared` от Authentik. Используйте клиент buddy/wiki/tracker/ticket
и его зарегистрированный callback. HTTPS issuer доступен браузеру и контейнеру;
прокси направляет callback на сервер, сохраняя Host. PostgreSQL не публикует порт.

Compose читает `.env`; локальные команды требуют переменных в окружении и доступной БД:

```bash
go run ./cmd/migrate          # up; также down и status, только DATABASE_URL
go run ./cmd/seed             # пока пустой; только DATABASE_URL
go run ./cmd/server
```

Сервер ждёт миграции в Compose, сам их не запускает. Дополнительные команды:
`docker compose run --rm migrate status`, `docker compose run --rm seed`.
`down` сохраняет данные, `down -v` удаляет их.

## API и вход

- `GET /api/auth/login` → Authentik; `GET /api/auth/callback` → `/`.
- `GET /api/auth/me` → snapshot или 401; `POST /api/auth/logout` → 204.
- `GET /health/live` → 204; `GET /health/ready` → 204 или 503 при недоступной БД.

Scopes: `openid profile email ai_buddy`. Проверяются RS256, issuer, audience, срок,
nonce и claims: непустые `authentik_id`, `team`, роль `intern`/`mentor`.
Snapshot содержит также имя, email и username, обновляется при входе. Токены не хранятся.
Cookie `__Host-session`: Secure, HttpOnly, SameSite=Lax. Logout завершает локальную сессию.
State, nonce и PKCE хранятся в SCS на 5 минут; callback очищает их через `Pop`.
Последовательный повтор отклоняется; атомарности между параллельными запросами нет.

Защищённую группу маршрутов оборачивайте в `WithSession` один раз:

```go
mux.Handle("/api/chat/", login.WithSession(
    login.RequireRoles(chatHandler, auth.RoleIntern),
))
```

`RequireRoles` разрешает любую перечисленную роль; пустой список запрещает доступ.
Для любой роли используйте `RequireUser`, личность — `auth.CurrentUser(ctx)`.
Без входа — 401, неподходящая роль — 403, ошибка сессии — 500.
Роль берётся из snapshot и обновляется при входе. Проверки владельца ресурса
и назначения наставника остаются в конкретном сервисе.

## Проверка и копирование

```bash
go test ./... -coverprofile=/tmp/backend-template.cover
go tool cover -func=/tmp/backend-template.cover
go vet ./...
go build ./cmd/...
docker compose config -q
```

Покрытие >80%. `TEST_DATABASE_URL` включает PostgreSQL-тест во временной схеме.
При копировании замените `ai-buddy/backend-template` в модуле и импортах, имя образа
и окружение; выполните `go mod tidy` и проверки.
