package database

import (
	"context"
	"database/sql"

	"log/slog"
)

type RowScanner interface{ Scan(...any) error }
type Queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// Query consumes and closes the cursor before returning, including on scan failure.
func Query[T any](ctx context.Context, db Queryer, query string, scan func(RowScanner) (T, error), args ...any) ([]T, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Warn("resource cleanup failed", "error", err)
		}
	}()
	out := []T{}
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
