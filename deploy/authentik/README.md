# Authentik

Authentik 2026.8.5 и PostgreSQL 16. Все команды выполняются из `deploy/authentik`.

## Запуск

```bash
cp .env.example .env
chmod 600 .env
```

Заполнить `.env`: пять публичных HTTPS URL без завершающего `/`, пароли и секреты.
Каждый секрет генерировать отдельно: `openssl rand -hex 32`.

```bash
docker compose config -q
docker compose up -d --wait --wait-timeout 300
curl --fail http://127.0.0.1:9000/-/health/ready/
```

Админка: `<AUTHENTIK_PUBLIC_URL>/if/admin/`, логин `akadmin`, начальный пароль —
`AUTHENTIK_BOOTSTRAP_PASSWORD`. Bootstrap-параметры действуют только при первом запуске.
В **Customization → Blueprints** проверить статус **Successful** обоих blueprints.
Логи: `docker compose logs --tail 100 server worker`. `down -v` удаляет данные.

## Сеть

Caddy → Rathole → `127.0.0.1:9000`. Адрес и порт настраиваются в `.env`.
Если Rathole в контейнере, настроить доступ к хосту. Сохранять исходный `Host`,
передавать `X-Forwarded-Proto: https` и `X-Forwarded-For`;
`AUTHENTIK_TRUSTED_PROXY_CIDRS` должен включать фактический адрес непосредственного прокси.
Публичный HTTPS Authentik должен быть доступен браузеру и Go-контейнерам.

Compose создаёт `ai-buddy-shared`; будущие сервисы подключаются к ней как к external network.

## Blueprints

- `01-oidc.yaml` — отдельные OIDC-клиенты `buddy`, `wiki`, `tracker`, `ticket`.
- `02-users.yaml` — по наставнику и два стажёра в командах `backend` и `platform`.
  Логины: `mentor.<team>`, `intern.<team>.1`, `intern.<team>.2`;
  email: `<username>@example.test`, отдельные пароли — в `.env`.

Автоприменение асинхронное. Для проверки добавить `--dry-run` к команде применения:

```bash
docker compose exec worker ak apply_blueprint /blueprints/ai-buddy/01-oidc.yaml /blueprints/ai-buddy/02-users.yaml
```

После изменения `.env` сначала выполнить `docker compose up -d`, затем применить blueprints.
`state: present` сохраняет ID, но перезаписывает объявленные поля, включая пароли и attributes.
Назначения хранит будущий Tracker: стажёры закреплены за наставником своей команды,
кроме `intern.platform.2` — у него назначения нет.

## Контракт OIDC

- Client ID: имя сервиса; client secret: соответствующий `<SERVICE>_CLIENT_SECRET`.
- Issuer: `<AUTHENTIK_PUBLIC_URL>/application/o/<service>/`.
- Discovery: `<issuer>.well-known/openid-configuration`.
- Единственный callback: `<SERVICE_PUBLIC_URL>/api/auth/callback`, строгое совпадение.
- Authorization Code, PKCE S256, state и nonce; scopes: `openid profile email ai_buddy`.
- Общий ID: `authentik_id` — строковый ID пользователя Authentik, а не `sub`.
- Claims: `role` (`intern` / `mentor`), `team` (`backend` / `platform`),
  `name`, `email`, `preferred_username`. Отсутствующие или неизвестные роль/команда означают отказ.
- Сервисы проверяют подпись RS256 через JWKS, issuer, audience, срок действия и nonce.
  Сессии и проверка прав принадлежат сервисам; наставнику закрывает чат сам Buddy.

