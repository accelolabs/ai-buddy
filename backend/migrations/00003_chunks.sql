-- +goose Up
CREATE TABLE chunks (
    document_id text NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    ordinal integer NOT NULL,
    anchor text NOT NULL,
    text text NOT NULL,
    PRIMARY KEY (document_id, ordinal)
);

-- +goose Down
DROP TABLE chunks;
