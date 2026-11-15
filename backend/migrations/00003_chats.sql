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
    attachment boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX messages_chat_id_id_idx ON messages (chat_id, id);

CREATE TABLE message_sources (
    message_id bigint PRIMARY KEY REFERENCES messages(id) ON DELETE CASCADE,
    document_id text NOT NULL,
    ordinal integer NOT NULL,
    title text NOT NULL,
    anchor text NOT NULL,
    quote text NOT NULL
);

-- +goose Down
DROP TABLE message_sources;
DROP TABLE messages;
DROP TABLE chats;
