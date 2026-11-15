package wiki

import (
	"context"
	"database/sql"
	"errors"

	"ai-buddy/internal/platform/database"
)

func (r *Store) List(ctx context.Context, team string) ([]Document, error) {
	const query = `
  SELECT id,path,title
  FROM documents
  WHERE team IN ('common',$1)
  ORDER BY path
 `
	return database.Query(ctx, r.DB, query, func(row database.RowScanner) (d Document, err error) {
		err = row.Scan(&d.ID, &d.Path, &d.Title)
		return
	}, team)
}

func (r *Store) Get(ctx context.Context, team, id string) (Document, error) {
	const query = `
  SELECT id,path,title,markdown
  FROM documents
  WHERE id=$1 AND team IN ('common',$2)
 `
	var document Document
	err := r.DB.QueryRowContext(ctx, query, id, team).Scan(
		&document.ID,
		&document.Path,
		&document.Title,
		&document.Markdown,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return document, ErrNotFound
	}

	return document, err
}
