-- +goose Up
CREATE TABLE documents (
    id text PRIMARY KEY,
    path text NOT NULL UNIQUE,
    team text NOT NULL,
    title text NOT NULL
);

-- +goose Down
DROP TABLE documents;
