-- +goose Up
CREATE TABLE users (
    id text PRIMARY KEY,
    name text NOT NULL,
    role text NOT NULL CHECK (role IN ('intern', 'mentor')),
    team text NOT NULL
);

CREATE TABLE mentorships (
    intern_id text PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    mentor_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX mentorships_mentor_id_idx ON mentorships (mentor_id);

-- +goose Down
DROP TABLE mentorships;
DROP TABLE users;
