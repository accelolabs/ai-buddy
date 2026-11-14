# Backend Tracker

Independent Go backend (`ai-buddy/backend-tracker`), copied from `backend-template`.
PostgreSQL stores profiles, mentor assignments, tasks, and OIDC sessions.
No UI or administrative CRUD is included.
Backend directories, Go modules and deployment services use `backend-[service]`
(for example, `backend-tracker`). The existing Authentik OIDC client remains
`tracker`, with its registered issuer and HTTPS callback.

## Run

```sh
cp .env.example .env
# Fill database password, two different random service keys (openssl rand -hex 32),
# and the existing tracker OIDC client secret and public HTTPS callback.
docker network create ai-buddy-shared # once, shared with Authentik
# If the network already exists, continue with the next command.
docker compose up --build -d backend-tracker
```

The server binds to `127.0.0.1:8083`. Put it behind the HTTPS reverse proxy at
the existing `TRACKER_PUBLIC_URL`; use the exact callback registered by
`deploy/authentik/blueprints/01-oidc.yaml`, `/api/auth/callback`.
`OIDC_CLIENT_ID=tracker`, issuer `/application/o/tracker/`. Secure session cookies
require HTTPS. Login: `GET /api/auth/login`; identity: `GET /api/auth/me`;
logout: `POST /api/auth/logout`. Each service needs its own public hostname.

Compose runs migrations once before starting `backend-tracker`. The database has its own
volume and no published port. Only `backend-tracker` joins `ai-buddy-shared`; migrations,
seed and the database use the private network. Seed must reach the public
Authentik URL through that network's outbound access.

### Demo seed

In Authentik's admin interface, create a service account, grant it **view user**
permission (including the six demo users), and create an API token for that
account under Directory → Tokens and app passwords. Use the token's key, not
its identifier. Seed only needs read access; do not use a superuser token.
The [Authentik users API](https://api.goauthentik.io/reference/core-users-list/)
provides `pk`, username, activity, and attributes used to verify identities.

```sh
# Export AUTHENTIK_URL and AUTHENTIK_API_TOKEN in your shell for this command.
# Avoid putting a real token in shell history or committing it.
docker compose --profile tools run --rm seed
unset AUTHENTIK_API_TOKEN
```

Seed resolves all six usernames from the Authentik demo blueprint before writing
anything, checks exact usernames, active status, role and team, then writes one
transaction. Repeating it inserts missing data without changing profiles,
assignments, task dates or completions. `intern.platform.2` has no mentor.
The Authentik API token is passed only to seed; the server neither needs it nor
calls the administrative API. Seed configuration is separate from server config.

## API

All domain responses and errors are JSON. Task IDs and intern IDs are strings;
intern IDs are the string representation of Authentik `pk`.

| Method and path | Access |
| --- | --- |
| `GET /api/tasks` | Intern: own tasks, `{ "tasks": [...] }` |
| `POST /api/tasks` | Intern: create own task |
| `POST /api/tasks/{id}/complete` | Intern: complete own task |
| `GET /api/progress` | Intern: own progress |
| `GET /api/interns` | Mentor: assigned profiles, `{ "interns": [...] }` |
| `GET /api/interns/{id}/progress` | Mentor: assigned intern's progress |
| `GET /internal/interns/{id}/progress` | Buddy |
| `POST /internal/interns/{id}/tasks` | Buddy |
| `POST /internal/interns/{id}/tasks/{taskID}/complete` | Buddy |
| `GET /internal/interns/{id}/mentor` | Buddy or Ticket; `{ "mentor": null }` if unassigned |

Internal routes require `Authorization: Bearer <BUDDY_API_KEY>` or
`<TICKET_API_KEY>`. Sessions do not authorize internal calls. Ticket can only
look up mentors. Browser ownership comes from the session, never request input.

Create body: `{"title":"First task","description":"Optional","deadline":"2026-12-01"}`.
Title, description and deadline are trimmed. Title and a valid `YYYY-MM-DD`
deadline are required; past dates are allowed. All creates require an
`Idempotency-Key` (nonblank, at most 256 bytes). Use a new key per intended task.
Creation returns the task with `201`; normalized replay returns it with `200`;
changed content with the same intern/key returns `409`. Browser and Buddy share
the same key namespace. Parallel requests create one task. Completion returns
`200` and preserves the first completion timestamp on retries. Buddy handles
user confirmation before submitting a create request.

Tasks contain `id`, `intern_id`, `title`, `description`, `deadline`, `created_at`,
`completed_at`, `status` (`pending`/`completed`) and `overdue`. Order is ascending
creation time then ID. `APP_TIMEZONE` defaults to `Europe/Moscow`; today's
calendar deadline is not overdue. Completed tasks are never overdue.

Progress includes `intern`, nullable `mentor`, `start_date`, `goals`,
`expectations`, `tasks`, `completed`, `total`, integer `percent` and nullable
`next_task`. The next task is the first pending overdue task in list order,
otherwise the pending task with earliest deadline (ties preserve list order).
Empty progress is 0%; empty or fully completed progress has no next task.

Errors: `400` invalid input, `401` missing/invalid credentials, `403` role or
key permission denied, `404` unknown intern, foreign task or inaccessible
profile, `409` idempotency conflict, `500` storage failure without SQL/secrets.
Health endpoints: `/health/live`, `/health/ready` (204 on success).

## Checks

```sh
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
go vet ./...
go build -o /tmp/backend-tracker-bin/ ./cmd/...
docker compose --env-file .env.example config --quiet
docker build -t ai-buddy-backend-tracker:local .
# Dedicated reachable PostgreSQL, permissions to create/drop temporary schemas:
TEST_DATABASE_URL='postgres://user:password@localhost:5432/backend-tracker_test?sslmode=disable' go test -count=1 ./...
```

PostgreSQL tests use isolated temporary schemas and drop them after each run.
Without `TEST_DATABASE_URL` those tests explicitly skip. They check migrations,
assignment constraints, persistence, concurrent idempotency, seed repeats and
rollback. Unit/HTTP tests run without PostgreSQL and include OIDC/session tests
inherited from the template. Coverage must remain strictly above 80%.
