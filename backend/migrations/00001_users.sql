-- +goose Up
CREATE TABLE users (
    id text PRIMARY KEY,
    name text NOT NULL,
    role text NOT NULL CHECK (role IN ('intern', 'mentor')),
    team text NOT NULL
);

-- +goose Down
DROP TABLE users;
