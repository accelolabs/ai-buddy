-- +goose Up
CREATE TABLE tickets (
 id text PRIMARY KEY,
 sender_id text NOT NULL CHECK (btrim(sender_id) <> ''),
 recipient_id text NOT NULL CHECK (btrim(recipient_id) <> ''),
 question text NOT NULL CHECK (btrim(question) <> ''),
 description text NOT NULL CHECK (btrim(description) <> ''),
 created_at timestamptz NOT NULL DEFAULT now(),
 idempotency_key text NOT NULL CHECK (btrim(idempotency_key) <> '' AND octet_length(idempotency_key) <= 256),
 UNIQUE (sender_id, idempotency_key)
);
CREATE INDEX tickets_sent ON tickets(sender_id, created_at, id);
CREATE INDEX tickets_incoming ON tickets(recipient_id, created_at, id);

-- +goose Down
DROP TABLE tickets;
