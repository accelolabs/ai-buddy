-- +goose Up
CREATE TABLE profiles (
    authentik_id text PRIMARY KEY CHECK (authentik_id <> ''),
    name text NOT NULL,
    username text NOT NULL UNIQUE,
    email text NOT NULL,
    role text NOT NULL CHECK (role IN ('intern', 'mentor')),
    team text NOT NULL,
    start_date date,
    goals text NOT NULL DEFAULT '',
    expectations text NOT NULL DEFAULT '',
    UNIQUE (authentik_id, role, team)
);
CREATE TABLE assignments (
    intern_id text PRIMARY KEY,
    mentor_id text NOT NULL,
    team text NOT NULL,
    intern_role text NOT NULL DEFAULT 'intern' CHECK (intern_role = 'intern'),
    mentor_role text NOT NULL DEFAULT 'mentor' CHECK (mentor_role = 'mentor'),
    FOREIGN KEY (intern_id, intern_role, team) REFERENCES profiles(authentik_id, role, team),
    FOREIGN KEY (mentor_id, mentor_role, team) REFERENCES profiles(authentik_id, role, team)
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
    FOREIGN KEY (intern_id, intern_role, team) REFERENCES profiles(authentik_id, role, team),
    UNIQUE (intern_id, idempotency_key)
);
CREATE INDEX tasks_intern_order_idx ON tasks(intern_id, created_at, id);
CREATE INDEX assignments_mentor_idx ON assignments(mentor_id);
-- +goose Down
DROP TABLE tasks;
DROP TABLE assignments;
DROP TABLE profiles;
