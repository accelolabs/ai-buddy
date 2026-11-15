-- +goose Up
CREATE TABLE chats (
    id text PRIMARY KEY,
    user_id text NOT NULL REFERENCES users(id),
    title text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE messages (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    chat_id text NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    role text NOT NULL CHECK (role IN ('user', 'assistant')),
    content text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX messages_chat_id_id_idx ON messages (chat_id, id);

-- +goose Down
DROP TABLE messages;
DROP TABLE chats;
