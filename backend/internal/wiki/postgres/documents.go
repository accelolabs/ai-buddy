package postgres

import (
	"context"
	"database/sql"
	"errors"

	"ai-buddy/internal/wiki"
)

func (r *Repository) List(ctx context.Context, team string) ([]wiki.Document, error) {
	const query = `
  SELECT id,path,title
  FROM documents
  WHERE team IN ('common',$1)
  ORDER BY path
 `
	rows, err := r.db.QueryContext(ctx, query, team)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	documents := []wiki.Document{}
	for rows.Next() {
		var document wiki.Document
		if err := rows.Scan(&document.ID, &document.Path, &document.Title); err != nil {
			return nil, err
		}
		documents = append(documents, document)
	}

	return documents, rows.Err()
}

func (r *Repository) Get(ctx context.Context, team, id string) (wiki.Document, error) {
	const query = `
  SELECT id,path,title,markdown
  FROM documents
  WHERE id=$1 AND team IN ('common',$2)
 `
	var document wiki.Document
	err := r.db.QueryRowContext(ctx, query, id, team).Scan(
		&document.ID,
		&document.Path,
		&document.Title,
		&document.Markdown,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return document, wiki.ErrNotFound
	}

	return document, err
}
