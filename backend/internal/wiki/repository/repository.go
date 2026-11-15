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

func (r *Repository) Sync(ctx context.Context, documents []wiki.Document) error {
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
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit wiki sync: %w", err)
	}

	return nil
}
