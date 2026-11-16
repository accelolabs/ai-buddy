-- +goose Up
CREATE TABLE tasks (
    id text PRIMARY KEY,
    user_id text NOT NULL REFERENCES users(id),
    description text NOT NULL,
    deadline date NOT NULL,
    completed boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX tasks_user_id_idx ON tasks (user_id);

-- +goose Down
DROP TABLE tasks;
