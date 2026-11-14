# Authentik

Команды выполняются из `backend/deploy/authentik`.

## Настройка и запуск

```sh
cp .env.example .env
chmod 600 .env
```

В `.env` заполните публичные HTTPS-адреса без завершающего `/`, пароль БД,
`AUTHENTIK_SECRET_KEY`, начальные данные администратора, секреты OIDC-клиентов
и пароли демо-пользователей. Каждый секрет генерируйте отдельно: `openssl rand -hex 32`.
`AUTHENTIK_TRUSTED_PROXY_CIDRS` должен включать адрес непосредственного прокси.

```sh
docker compose up -d --wait --wait-timeout 300
curl --fail http://127.0.0.1:9000/-/health/ready/
docker compose logs --tail 100 server worker
```

Админка: `<AUTHENTIK_PUBLIC_URL>/if/admin/`, логин `akadmin`, пароль —
`AUTHENTIK_BOOTSTRAP_PASSWORD`. Начальные данные применяются только при первом запуске.
В **Customization → Blueprints** проверьте статус **Successful**.
Compose создаёт общую сеть `ai-buddy-shared`.

Проксируйте HTTPS на `127.0.0.1:9000`, сохраняя `Host` и передавая
`X-Forwarded-Proto: https` и `X-Forwarded-For`. Публичный адрес Authentik
должен быть доступен браузеру и контейнерам сервисов.

## OIDC и пользователи

Для каждого сервиса (`buddy`, `wiki`, `tracker`, `ticket`):

- `OIDC_CLIENT_ID` — имя сервиса.
- `OIDC_CLIENT_SECRET` — соответствующий `<SERVICE>_CLIENT_SECRET` из `.env`.
- `OIDC_ISSUER` — `<AUTHENTIK_PUBLIC_URL>/application/o/<service>/`.
- `OIDC_REDIRECT_URI` — `<SERVICE_PUBLIC_URL>/api/auth/callback`.

Демо-логины: `mentor.<team>`, `intern.<team>.1`, `intern.<team>.2`,
где `<team>` — `backend` или `platform`. Пароли задаются в `.env`.

После изменения `.env` пересоздайте контейнеры и примените blueprints:

```sh
docker compose up -d
docker compose exec worker ak apply_blueprint /blueprints/ai-buddy/01-oidc.yaml /blueprints/ai-buddy/02-users.yaml
```

Применение перезаписывает объявленные поля пользователей, включая пароли.
