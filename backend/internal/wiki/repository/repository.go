package wiki

import (
	"backend/internal/wiki"
	"context"
	"database/sql"
	"fmt"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context) ([]wiki.Document, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, path, team, title FROM documents ORDER BY path`)
	if err != nil {
		return nil, fmt.Errorf("list wiki documents: %w", err)
	}
	defer func() { _ = rows.Close() }()

	documents := make([]wiki.Document, 0)
	for rows.Next() {
		var document wiki.Document
		if err := rows.Scan(&document.ID, &document.Path, &document.Team, &document.Title); err != nil {
			return nil, fmt.Errorf("scan wiki document: %w", err)
		}
		documents = append(documents, document)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read wiki documents: %w", err)
	}
	return documents, nil
}

func (r *Repository) Get(ctx context.Context, id string) (wiki.Document, error) {
	var document wiki.Document
	err := r.db.QueryRowContext(ctx, `
		SELECT id, path, team, title FROM documents
		WHERE id = $1`, id).
		Scan(&document.ID, &document.Path, &document.Team, &document.Title)
	if err != nil {
		return wiki.Document{}, fmt.Errorf("get wiki document: %w", err)
	}
	return document, nil
}

func (r *Repository) ListChunks(ctx context.Context, documentID string) ([]wiki.Chunk, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT document_id, ordinal, anchor, text FROM chunks
		WHERE document_id = $1 ORDER BY ordinal`, documentID)
	if err != nil {
		return nil, fmt.Errorf("list wiki chunks: %w", err)
	}
	defer func() { _ = rows.Close() }()
	chunks := make([]wiki.Chunk, 0)
	for rows.Next() {
		var chunk wiki.Chunk
		if err := rows.Scan(&chunk.DocumentID, &chunk.Ordinal, &chunk.Anchor, &chunk.Text); err != nil {
			return nil, fmt.Errorf("scan wiki chunk: %w", err)
		}
		chunks = append(chunks, chunk)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read wiki chunks: %w", err)
	}
	return chunks, nil
}

func (r *Repository) Sync(ctx context.Context, documents []wiki.Document, chunks []wiki.Chunk) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin wiki sync: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	paths := make([]string, 0, len(documents))
	for _, document := range documents {
		paths = append(paths, document.Path)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM documents WHERE NOT (path = ANY($1::text[]))`, paths); err != nil {
		return fmt.Errorf("delete missing wiki documents: %w", err)
	}

	for _, document := range documents {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO documents (id, path, team, title) VALUES ($1, $2, $3, $4)
			ON CONFLICT (path) DO UPDATE SET team = EXCLUDED.team, title = EXCLUDED.title
			WHERE (documents.team, documents.title) IS DISTINCT FROM (EXCLUDED.team, EXCLUDED.title)`,
			document.ID, document.Path, document.Team, document.Title)
		if err != nil {
			return fmt.Errorf("sync wiki document %q: %w", document.Path, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM chunks`); err != nil {
		return fmt.Errorf("delete old wiki chunks: %w", err)
	}
	for _, chunk := range chunks {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO chunks (document_id, ordinal, anchor, text) VALUES ($1, $2, $3, $4)`,
			chunk.DocumentID, chunk.Ordinal, chunk.Anchor, chunk.Text); err != nil {
			return fmt.Errorf("sync wiki chunk %q/%d: %w", chunk.DocumentID, chunk.Ordinal, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit wiki sync: %w", err)
	}

	return nil
}
