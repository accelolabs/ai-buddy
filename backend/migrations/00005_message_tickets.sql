-- +goose Up
CREATE TABLE message_tickets (
 message_id bigint PRIMARY KEY REFERENCES messages(id) ON DELETE CASCADE,
 title text NOT NULL,
 description text NOT NULL,
 recipient_id text NOT NULL,
 recipient_name text NOT NULL,
 ticket_id text UNIQUE REFERENCES tickets(id)
);

-- +goose Down
DROP TABLE message_tickets;
