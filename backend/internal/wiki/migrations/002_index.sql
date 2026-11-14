-- +goose Up
CREATE EXTENSION IF NOT EXISTS vector WITH SCHEMA public;
CREATE TABLE wiki_state (singleton BOOLEAN PRIMARY KEY DEFAULT true CHECK(singleton), fingerprint TEXT NOT NULL);
CREATE TABLE documents (id TEXT PRIMARY KEY, path TEXT UNIQUE NOT NULL, team TEXT NOT NULL, title TEXT NOT NULL, markdown TEXT NOT NULL, hash TEXT NOT NULL);
CREATE TABLE sections (document_id TEXT REFERENCES documents ON DELETE CASCADE, ordinal INTEGER NOT NULL, title TEXT NOT NULL, anchor TEXT NOT NULL, text TEXT NOT NULL, PRIMARY KEY(document_id, ordinal));
CREATE TABLE chunks (document_id TEXT NOT NULL, section INTEGER NOT NULL, ordinal INTEGER NOT NULL, text TEXT NOT NULL, embedding public.vector(768) NOT NULL, search tsvector GENERATED ALWAYS AS (to_tsvector('russian',text) || to_tsvector('english',text)) STORED, PRIMARY KEY(document_id, ordinal), FOREIGN KEY(document_id, section) REFERENCES sections ON DELETE CASCADE);
CREATE INDEX chunks_search ON chunks USING gin(search);
CREATE INDEX documents_team ON documents(team);
-- +goose Down
DROP TABLE chunks;
DROP TABLE sections;
DROP TABLE documents;
DROP TABLE wiki_state;
