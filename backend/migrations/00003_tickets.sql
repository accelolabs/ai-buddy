-- +goose Up
CREATE TABLE tickets (
    id text PRIMARY KEY,
    user_id text NOT NULL REFERENCES users(id),
    title text NOT NULL,
    description text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX tickets_user_id_idx ON tickets (user_id);

-- +goose Down
DROP TABLE tickets;
