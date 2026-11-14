package main

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"log/slog"
	"os"

	buddy "ai-buddy/internal/buddy/migrations"
	"ai-buddy/internal/platform/cli"
	"ai-buddy/internal/platform/database"
	"ai-buddy/internal/platform/migrations"
	ticket "ai-buddy/internal/ticket/migrations"
	tracker "ai-buddy/internal/tracker/migrations"
	wiki "ai-buddy/internal/wiki/migrations"
)

var openDatabase = database.Open

func prepare(service, command string) (cli.Operation, error) {
	var files fs.FS
	switch service {
	case "tracker":
		files = tracker.Files
	case "buddy":
		files = buddy.Files
	case "ticket":
		files = ticket.Files
	case "wiki":
		files = wiki.Files
	default:
		return nil, errors.New("unknown service")
	}

	return func(ctx context.Context, db *sql.DB) error { return migrations.Run(ctx, db, files, command) }, nil
}
func run() error { return cli.Run(os.Args[1:], true, prepare, openDatabase) }
func main() {
	if run() != nil {
		slog.Error("migrate failed; check service, command and configuration")
		os.Exit(1)
	}
}
