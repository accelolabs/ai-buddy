package migrations

import (
	"context"
	"database/sql"
	"embed"

	"ai-buddy/internal/platform/migrations"
)

//go:embed *.sql
var Files embed.FS

func Run(ctx context.Context, db *sql.DB, command string) error {
	return migrations.Run(ctx, db, Files, command)
}
