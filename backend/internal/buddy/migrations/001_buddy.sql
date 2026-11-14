-- +goose Up
CREATE TABLE sessions (token text PRIMARY KEY, data bytea NOT NULL, expiry timestamptz NOT NULL);
CREATE INDEX sessions_expiry_idx ON sessions(expiry);
CREATE TABLE chats (id text PRIMARY KEY, owner_id text NOT NULL, title text NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now());
CREATE INDEX chats_owner_idx ON chats(owner_id, created_at DESC);
CREATE TABLE messages (id text PRIMARY KEY, chat_id text NOT NULL REFERENCES chats(id), run_id text NOT NULL, role text NOT NULL CHECK(role IN ('user','assistant')), content text NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE runs (id text PRIMARY KEY, chat_id text NOT NULL REFERENCES chats(id), owner_id text NOT NULL, message_id text NOT NULL REFERENCES messages(id), status text NOT NULL CHECK(status IN ('queued','running','done','error')), error text NOT NULL DEFAULT '', request_key text NOT NULL, retry_of text UNIQUE REFERENCES runs(id), created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(chat_id,request_key));
CREATE UNIQUE INDEX runs_active_idx ON runs(chat_id) WHERE status IN ('queued','running');
CREATE TABLE events (run_id text NOT NULL REFERENCES runs(id), sequence bigint NOT NULL, kind text NOT NULL, data jsonb NOT NULL, PRIMARY KEY(run_id,sequence));
CREATE TABLE actions (id text PRIMARY KEY, run_id text NOT NULL REFERENCES runs(id), owner_id text NOT NULL, kind text NOT NULL CHECK(kind IN ('create_task','create_ticket')), status text NOT NULL CHECK(status IN ('draft','executing','done','error','cancelled')), fields jsonb NOT NULL, result jsonb NOT NULL DEFAULT 'null', error text NOT NULL DEFAULT '');
-- +goose Down
DROP TABLE actions, events, runs, messages, chats, sessions;
