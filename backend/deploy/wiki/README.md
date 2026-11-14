# Wiki

Wiki serves JSON documents and up to three retrieved passages. It has its own
PostgreSQL database and OIDC sessions. Answer generation belongs to future Buddy;
there are no HTML pages or frontend in this service.

From this directory:

```sh
cp .env.example .env
# Replace all passwords, API keys and public HTTPS URLs.
docker network create ai-buddy-shared # once, shared with Authentik and Ollama
# Start Authentik and Ollama using their deployment guides first.
docker compose -f ../ollama/compose.yaml exec ollama ollama pull embeddinggemma:300m
docker compose up -d --build
# Explicit indexing after migrations complete and the model is available:
docker compose --profile tools run --rm index
```

The Wiki OIDC client is already declared in the Authentik blueprint. Its callback
must equal `https://<wiki-host>/api/auth/callback`. Proxy public HTTPS to
`127.0.0.1:8082`; keep this port bound to loopback. Configure the client secret in
Authentik and Wiki consistently. Login uses `/api/auth/login`; cookies and session
storage reuse the platform implementation.

Create a **separate service account** in Authentik. Grant only the global
`authentik_core.view_user` (view users) permission, create an API token for it and
set `AUTHENTIK_READ_TOKEN`. Do not reuse the blueprint/seed administrator token.
This token reads `/api/v3/core/users/{id}/` for each Buddy request, checks the
returned primary key, active status and `attributes.team`; failures never fall
back to cached teams. `AUTHENTIK_URL` is the Authentik origin, not the OIDC issuer.

## Knowledge base

The default read-only bind mount is repository `knowledge-base/`. Set
`KNOWLEDGE_BASE_DIR` in `.env` to an absolute host directory to override it. A
missing host path fails rather than being created. Only the index container gets
this mount; the server reads published documents from PostgreSQL.

Add `.md` files below `common/` or a team folder, including nested directories.
`common/` is mandatory. Root metadata, hidden files/directories and other
extensions are ignored; visible symbolic links fail the run. Team folders are
open-ended: add `new-team/` and use the matching Authentik `attributes.team`.
There are no extra ACL files. Moving a file changes its ID and access scope.

Run `docker compose --profile tools run --rm index` after additions, edits,
deletions or moves. The command reports added/changed/deleted/skipped documents;
unchanged documents do not call the embed endpoint. It has a 30-minute deadline,
60-second individual Ollama request timeout and an exclusive database advisory
lock. All file reads and embeddings finish before the publication transaction;
failures preserve the last index.

The index stores the model name, Ollama digest and splitter version. After a model
replacement or splitter change, explicitly run:

```sh
docker compose --profile tools run --rm index /app/wiki-index --full
```

Both indexing and search use `OLLAMA_URL` (default `http://ollama:11434`), the fixed
`embeddinggemma:300m` model, 768 dimensions, document/query task prefixes and
`truncate: false`. Oversized document inputs split recursively. Query inputs are
limited to 4000 Unicode characters; rejected model inputs return 503.
`WIKI_MIN_SIMILARITY` defaults to 0.35 and accepts values in [0,1].

For local Go execution, from `backend/`:

```sh
export DATABASE_URL='postgres://wiki:password@localhost:5432/wiki?sslmode=disable'
export OLLAMA_URL=http://localhost:11434
go run ./cmd/migrate wiki up
go run ./cmd/wiki-index # default ../knowledge-base
# Set the OIDC/AuthenTik/session environment variables from .env.example too:
go run ./cmd/wiki
```

## JSON API

Browser routes require the Wiki session and use its stored team snapshot:

- `POST /api/search` with `{"query":"таймауты"}` → an array of passages.
- `GET /api/documents` → accessible document metadata.
- `GET /api/documents/{id}` → metadata and original Markdown.

`POST /internal/users/{id}/search` uses the same body and requires
`Authorization: Bearer <BUDDY_API_KEY>`. The current team comes from Authentik;
client-supplied `team` is rejected. Each search result has `document_id`,
`document_title`, `section_title`, `anchor`, `text`, and a relative JSON API `url`.
The URL is not a browser page. Document responses have `id`, `path`, `title`,
`url`, and `markdown`; they do not include parsed sections. Sections and anchors
remain internal to the search index and appear in search results. A future React
frontend can render the source with `react-markdown` and `remark-gfm`.
No matches return `[]`; blank input is 400, missing/inaccessible documents 404,
missing credentials 401, and dependency/fingerprint failures 503. Reading the
published corpus continues when Ollama is down.

SQL filters `common + team` before selecting at most 20 lexical and 20 exact
cosine candidates. Russian and English full-text branches share a tsvector;
Reciprocal Rank Fusion uses constant 60 and returns at most three distinct chunks.
There is no approximate vector index.

## Verification

From `backend/`, run `scripts/check.sh --unit` for checks without PostgreSQL.
The full `scripts/check.sh` requires `TEST_DATABASE_URL` (PostgreSQL 16 + pgvector) and Docker CLI.
Wiki has its own coverage group including
`cmd/wiki` and `cmd/wiki-index`; every group must remain strictly above 80%.
For real PostgreSQL tests, set `TEST_DATABASE_URL` to an isolated test database on
PostgreSQL 16 with pgvector installed. The test user needs schema/extension
creation rights; tests isolate their tables in temporary schemas.

```sh
TEST_DATABASE_URL='postgres://test:password@localhost/test?sslmode=disable' \
  go test -race ./internal/wiki/... ./cmd/wiki ./cmd/wiki-index
```

For the deployment smoke test, index the demo corpus, log in as backend and
platform users and search `таймаут` / `инцидент`. Check that each receives only
`common + own team` through search, list and direct ID reads. Change one file and
remove another, re-index, and check the counts and results. Re-run unchanged and
check `skipped`. Stop Ollama: search should return 503 and documents should still
read. Stop Authentik: internal search should return 503. Restart dependencies.


## Code structure

The root `internal/wiki` package holds domain models, configuration and search.
`postgres` handles SQL and index publication; `indexer` coordinates updates and
contains corpus reading, Goldmark extraction and chunking as ordinary functions.
There is no separate Markdown parser package or parser interface. `ollama` and
`authentik` own their HTTP clients; `http` owns routes, response types and request
validation. Commands wire these dependencies together.

Query normalization and length validation happen once at the HTTP boundary before
external calls. Dependency URL/path settings are trusted and checked by the
HTTP/filesystem operations themselves; required credentials and the similarity
range remain startup checks. Database ACL filters, model identity, vector size,
network deadlines and atomic publication remain enforced.

Контракт API: [OpenAPI 3.1.1](../../api/wiki.yaml).
Общие быстрые и полные проверки, установка линтера и CI: [README бэкенда](../../README.md#api-и-проверки).
