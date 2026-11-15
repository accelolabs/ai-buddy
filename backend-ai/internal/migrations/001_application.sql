-- +goose Up
-- Serialize extension installation across independent schemas and startup processes.
SELECT pg_advisory_xact_lock(716918462);
CREATE EXTENSION IF NOT EXISTS vector WITH SCHEMA public;
CREATE TABLE profiles (
    id text PRIMARY KEY CHECK (id <> ''),
    name text NOT NULL,
    username text NOT NULL UNIQUE,
    email text NOT NULL,
    role text NOT NULL CHECK (role IN ('intern', 'mentor')),
    team text NOT NULL,
    start_date date,
    goals text NOT NULL DEFAULT '',
    expectations text NOT NULL DEFAULT '',
    UNIQUE (id, role, team)
);
CREATE TABLE assignments (
    intern_id text PRIMARY KEY,
    mentor_id text NOT NULL,
    team text NOT NULL,
    intern_role text NOT NULL DEFAULT 'intern' CHECK (intern_role = 'intern'),
    mentor_role text NOT NULL DEFAULT 'mentor' CHECK (mentor_role = 'mentor'),
    FOREIGN KEY (intern_id, intern_role, team) REFERENCES profiles(id, role, team),
    FOREIGN KEY (mentor_id, mentor_role, team) REFERENCES profiles(id, role, team)
);
CREATE TABLE tasks (
    id text PRIMARY KEY,
    intern_id text NOT NULL,
    intern_role text NOT NULL DEFAULT 'intern' CHECK (intern_role = 'intern'),
    team text NOT NULL,
    title text NOT NULL CHECK (btrim(title) <> ''),
    description text NOT NULL DEFAULT '',
    deadline date NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    idempotency_key text NOT NULL CHECK (btrim(idempotency_key) <> ''),
    FOREIGN KEY (intern_id, intern_role, team) REFERENCES profiles(id, role, team),
    UNIQUE (intern_id, idempotency_key)
);
CREATE INDEX tasks_intern_order_idx ON tasks(intern_id, created_at, id);
CREATE INDEX assignments_mentor_idx ON assignments(mentor_id);
CREATE TABLE tickets (
 id text PRIMARY KEY,
 sender_id text NOT NULL REFERENCES profiles(id) CHECK (btrim(sender_id) <> ''),
 recipient_id text NOT NULL REFERENCES profiles(id) CHECK (btrim(recipient_id) <> ''),
 question text NOT NULL CHECK (btrim(question) <> ''),
 description text NOT NULL CHECK (btrim(description) <> ''),
 created_at timestamptz NOT NULL DEFAULT now(),
 idempotency_key text NOT NULL CHECK (btrim(idempotency_key) <> '' AND octet_length(idempotency_key) <= 256),
 UNIQUE (sender_id, idempotency_key)
);
CREATE INDEX tickets_sent ON tickets(sender_id, created_at, id);
CREATE INDEX tickets_incoming ON tickets(recipient_id, created_at, id);

CREATE TABLE wiki_state (singleton BOOLEAN PRIMARY KEY DEFAULT true CHECK(singleton), fingerprint TEXT NOT NULL);
CREATE TABLE documents (id TEXT PRIMARY KEY, path TEXT UNIQUE NOT NULL, team TEXT NOT NULL, title TEXT NOT NULL, markdown TEXT NOT NULL, hash TEXT NOT NULL);
CREATE TABLE sections (document_id TEXT REFERENCES documents ON DELETE CASCADE, ordinal INTEGER NOT NULL, title TEXT NOT NULL, anchor TEXT NOT NULL, text TEXT NOT NULL, PRIMARY KEY(document_id, ordinal));
CREATE TABLE chunks (document_id TEXT NOT NULL, section INTEGER NOT NULL, ordinal INTEGER NOT NULL, text TEXT NOT NULL, embedding public.vector(768) NOT NULL, search tsvector GENERATED ALWAYS AS (to_tsvector('russian',text) || to_tsvector('english',text)) STORED, PRIMARY KEY(document_id, ordinal), FOREIGN KEY(document_id, section) REFERENCES sections ON DELETE CASCADE);
CREATE INDEX chunks_search ON chunks USING gin(search);
CREATE INDEX documents_team ON documents(team);
CREATE TABLE chats (id text PRIMARY KEY, owner_id text NOT NULL REFERENCES profiles(id), title text NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now());
CREATE INDEX chats_owner_idx ON chats(owner_id, created_at DESC);
CREATE TABLE messages (id text PRIMARY KEY, chat_id text NOT NULL REFERENCES chats(id), run_id text NOT NULL, role text NOT NULL CHECK(role IN ('user','assistant')), content text NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE runs (id text PRIMARY KEY, chat_id text NOT NULL REFERENCES chats(id), owner_id text NOT NULL REFERENCES profiles(id), message_id text NOT NULL REFERENCES messages(id), status text NOT NULL CHECK(status IN ('queued','running','done','error')), error text NOT NULL DEFAULT '', request_key text NOT NULL, retry_of text UNIQUE REFERENCES runs(id), created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(chat_id,request_key));
CREATE UNIQUE INDEX runs_active_idx ON runs(chat_id) WHERE status IN ('queued','running');
CREATE TABLE events (run_id text NOT NULL REFERENCES runs(id), sequence bigint NOT NULL, kind text NOT NULL, data jsonb NOT NULL, PRIMARY KEY(run_id,sequence));
CREATE TABLE actions (id text PRIMARY KEY, run_id text NOT NULL REFERENCES runs(id), owner_id text NOT NULL REFERENCES profiles(id), kind text NOT NULL CHECK(kind IN ('create_task','create_ticket')), status text NOT NULL CHECK(status IN ('draft','executing','done','error','cancelled')), fields jsonb NOT NULL, result jsonb NOT NULL DEFAULT 'null', error text NOT NULL DEFAULT '');
-- +goose Down

DROP TABLE actions, events, runs, messages, chats;


DROP TABLE chunks;
DROP TABLE sections;
DROP TABLE documents;
DROP TABLE wiki_state;


DROP TABLE tickets;


DROP TABLE tasks;
DROP TABLE assignments;
DROP TABLE profiles;
